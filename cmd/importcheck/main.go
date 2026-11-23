// Command importcheck reads a `.venera` backup the way the import window
// does, and optionally applies it into a throwaway profile, so the flow can
// be checked without clicking through the interface.
//
//	go run ./cmd/importcheck -file backup.venera -dir profile
//	go run ./cmd/importcheck -file backup.venera -dir profile -apply
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/laincat/nekophage/internal/archive"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/searches"
	"github.com/laincat/nekophage/internal/shelf"
	"github.com/laincat/nekophage/internal/sourceapi"
	"github.com/laincat/veneracat-core/gohost/library"
)

func main() {
	file := flag.String("file", "", "path to a .venera backup")
	dir := flag.String("dir", "", "profile directory to import into")
	apply := flag.Bool("apply", false, "apply the import, not just read it")
	withCookies := flag.Bool("cookies", false, "also restore the backup's cookies into per-source partitions")
	flag.Parse()
	if *file == "" || *dir == "" {
		fmt.Fprintln(os.Stderr, "importcheck: -file and -dir are required")
		os.Exit(2)
	}
	if err := os.MkdirAll(*dir, 0o755); err != nil {
		fail(err)
	}
	opts := core.Options{DataDir: *dir, Locale: "zh_CN", AppVersion: sourceapi.Level,
		UserAgent: "Nekophage/0.1"}
	c, err := core.Open(opts)
	if err != nil {
		fail(err)
	}
	defer c.Close()
	lib := library.New(c)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if _, err := lib.Restore(ctx); err != nil {
		fail(err)
	}

	known := make([]string, 0, len(lib.Sources()))
	for _, s := range lib.Sources() {
		known = append(known, s.Key)
	}
	b, err := archive.Read(ctx, *file, known, archive.Options{Cookies: *withCookies})
	if err != nil {
		fail(err)
	}
	sum := b.Summarize()
	fmt.Printf("backup: %s\n", filepath.Base(sum.Path))
	fmt.Printf("  sources %d (with data %d), importable %d, data %d\n",
		sum.Sources, sum.WithData, sum.Importable, sum.RestoredData)
	fmt.Printf("  history %d, folders %d/%d items, image favourites %d\n",
		sum.History, sum.FavoriteFolders, sum.FavoriteItems, sum.ImageFavorites)
	fmt.Printf("  search history %d, settings %d, unresolved %d\n",
		sum.SearchHistory, sum.Settings, sum.Unresolved)
	for _, w := range sum.Warnings {
		fmt.Printf("  warning: %s\n", w)
	}
	if !*apply {
		return
	}

	sh, err := shelf.Open(*dir)
	if err != nil {
		fail(err)
	}
	history, err := searches.Open(*dir)
	if err != nil {
		fail(err)
	}
	result, err := archive.Apply(ctx, b, lib, sh, history)
	fmt.Printf("applied: %d sources, %d data keys, %d history, %d favourites, %d pages, %d searches, %d cookies\n",
		result.Sources, result.Fields, result.History, result.Favourites, result.Pages, result.Searches, result.Cookies)
	if err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "importcheck: %v\n", err)
	os.Exit(1)
}
