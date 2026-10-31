// Command importcheck reads a `.venera` backup the way the import window
// does, and optionally applies it into a throwaway profile, so the flow can
// be checked without clicking through the interface.
//
//	go run ./cmd/importcheck -file backup.venera -dir profile
//	go run ./cmd/importcheck -file backup.venera -dir profile -apply
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/laincat/nekophage/internal/archive"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/library"
	"github.com/laincat/nekophage/internal/searches"
	"github.com/laincat/nekophage/internal/shelf"
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
	opts := core.Options{DataDir: *dir, Locale: "zh_CN", AppVersion: "1.6.0",
		UserAgent: "Nekophage/0.1"}
	c, err := core.Open(opts)
	if err != nil {
		fail(err)
	}
	defer c.Close()
	lib := library.New(c)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

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

	installed, fields := 0, 0
	for _, src := range b.Plan.Sources {
		if src.Script == "" {
			continue
		}
		info, err := lib.InstallText(ctx, src.Script, "backup:"+src.Key)
		if err != nil {
			fmt.Printf("  install %s: %v\n", src.Key, err)
			continue
		}
		fmt.Printf("  installed %s %s\n", info.Key, info.Version)
		installed++
	}
	sh, err := shelf.Open(*dir)
	if err != nil {
		fail(err)
	}
	recorded, kept := 0, 0
	for _, p := range b.Positions() {
		ok, err := sh.Merge(shelf.History{
			Comic:   shelf.Comic{Source: p.Source, ID: p.ID, Title: p.Title, Author: p.Author, Cover: p.Cover},
			Chapter: "第 " + itoa(p.Episode) + " 话", ChapterIndex: p.Episode - 1, Page: p.Page, Time: p.Time})
		if err != nil {
			fail(err)
		}
		if ok {
			recorded++
		}
	}
	for _, f := range b.Favourites() {
		for _, it := range f.Items {
			added, err := sh.Add(f.Name, shelf.Comic{Source: it.Source, ID: it.ID,
				Title: it.Title, Author: it.Author, Cover: it.Cover})
			if err != nil {
				fail(err)
			}
			if added {
				kept++
			}
		}
	}
	fmt.Printf("  shelf: %d history, %d favourites\n", recorded, kept)

	pages, err := b.Pages()
	if err != nil {
		fail(err)
	}
	keptPages := 0
	for _, page := range pages {
		err = sh.KeepPage(shelf.ImageFavourite{
			Source:       page.Source,
			ComicID:      page.ComicID,
			ComicTitle:   page.ComicTitle,
			ChapterID:    page.ChapterID,
			ChapterIndex: page.ChapterIndex,
			ChapterName:  page.ChapterName,
			Page:         page.Page,
			URL:          page.URL,
			Time:         page.Time,
		})
		if err != nil {
			fail(err)
		}
		keptPages++
	}
	// Kept pages arrive from two places in a backup (image favourites and the
	// episode rows that point at them), so the count read and the count stored
	// are reported separately rather than assumed equal.
	fmt.Printf("  pages: %d rows -> %d kept\n", keptPages, len(sh.Pages(0)))

	searched, err := searches.Open(*dir)
	if err != nil {
		fail(err)
	}
	if n, err := searched.Merge(b.Content.SearchHistory); err != nil {
		fail(err)
	} else if n > 0 {
		fmt.Printf("  search history: %d added\n", n)
	}

	data, err := b.SourceData()
	if err != nil {
		fail(err)
	}
	for _, d := range data {
		values := make(map[string]json.RawMessage, len(d.Fields))
		for _, f := range d.Fields {
			values[f.Name] = f.Value
		}
		n, err := lib.ImportSourceData(d.Key, values)
		if err != nil {
			fmt.Printf("  data %s: %v\n", d.Key, err)
		}
		fields += n
	}
	fmt.Printf("applied: %d sources, %d data keys\n", installed, fields)

	// Read the profile back: the source must be listed and its values must
	// still be there, with their digits intact.
	after := lib.Sources()
	fmt.Printf("profile now has %d sources:", len(after))
	for _, s := range after {
		fmt.Printf(" %s(%s)", s.Key, s.Version)
	}
	fmt.Println()
	for _, d := range data {
		for _, f := range d.Fields {
			got, ok, err := c.Store().LoadData(d.Key, f.Name)
			if err != nil || !ok {
				fmt.Printf("  missing %s/%s\n", d.Key, f.Name)
				continue
			}
			var a, b any
			_ = json.Unmarshal(got, &a)
			_ = json.Unmarshal(f.Value, &b)
			if fmt.Sprint(a) != fmt.Sprint(b) {
				fmt.Printf("  %s/%s = %s, want %s\n", d.Key, f.Name, got, f.Value)
			}
		}
	}
}

// itoa is strconv.Itoa without the import noise.
func itoa(n int) string { return strconv.Itoa(n) }

func fail(err error) {
	fmt.Fprintf(os.Stderr, "importcheck: %v\n", err)
	os.Exit(1)
}
