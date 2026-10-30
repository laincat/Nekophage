// uicheck opens the main window on a real source, searches, and saves what
// the window shows, so that the native interface is checked with real data
// rather than assumed. It is a development tool, not part of the app.
//
//	go run ./cmd/uicheck -script /path/to/manga_dex.js -query "one piece" -out shot.png
package main

import (
	"context"
	"flag"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/library"
	"github.com/laincat/nekophage/internal/reader"
	"github.com/laincat/nekophage/internal/ui"

	_ "github.com/egoist/mygo/locales/zh-Hans"
)

func main() {
	script := flag.String("script", "", "source script to install")
	script2 := flag.String("script2", "", "a second source script to install")
	query := flag.String("query", "", "search keyword")
	out := flag.String("out", "window.png", "where to write the capture")
	wait := flag.Duration("wait", 10*time.Second, "how long the search may take")
	openFirst := flag.Bool("open", false, "open the first result before capturing")
	browse := flag.Bool("browse", false, "capture the browse view instead of searching")
	exploreIndex := flag.Int("explore", 0, "which browse to open (0-based)")
	explorePage := flag.Int("explore-page", 1, "which page of that browse")
	shelfMode := flag.Bool("shelf", false, "capture the shelf instead of searching")
	chapter := flag.Int("chapter", -1, "open that chapter of the first result in the reader")
	shelfDemo := flag.Bool("shelf-demo", false, "create, move and rename a folder before capturing the shelf")
	scrollPercent := flag.Int("scroll-percent", 0, "scroll the reader that far into the chapter")
	advance := flag.Bool("next-chapter", false, "click the reader's 下一话 button once")
	filters := flag.Bool("filters", false, "open the source's search filters")
	filterPick := flag.String("filter", "", "choose a search filter as index=key, then search again")
	settingsShot := flag.Bool("settings", false, "open the source settings panel and capture the sources window")
	settingsOut := flag.String("settings-out", "", "where to write the sources-window capture")
	setSetting := flag.String("set-setting", "", "change a declared setting as key=value")
	autoNext := flag.Bool("auto-next", false, "turn on automatic next chapter and read to the end")
	searchAll := flag.Bool("all-sources", false, "search every installed source at once")
	openLast := flag.Bool("open-last", false, "open the last result, to check a source switch")
	keepPage := flag.Bool("keep-page", false, "click the reader's 收藏本页 button once")
	favourite := flag.Bool("favourite", false, "keep the first search result, to check the shelf marker")
	// dir reuses a profile, so the shelf and installed sources carry over
	// from an earlier import.
	dir := flag.String("dir", "", "profile directory; a fresh one under the temp directory when empty")
	flag.Parse()
	if *script == "" && *script2 == "" && *dir == "" {
		flag.Usage()
		os.Exit(2)
	}

	dataDir := *dir
	if dataDir == "" {
		dataDir = filepath.Join(os.TempDir(), "uicheck")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	c, err := core.Open(core.Options{DataDir: dataDir, AppVersion: "1.6.0", Locale: "zh_CN"})
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	lib := library.New(c)
	if *script == "" {
		// Reusing a profile: its sources are on disk and must be compiled
		// again before the window lists them.
		if failed, err := lib.Restore(ctx); err != nil {
			log.Fatalf("restore: %v", err)
		} else if len(failed) > 0 {
			log.Printf("restore: %d sources failed", len(failed))
		}
	}
	for _, path := range []string{*script, *script2} {
		if path == "" {
			continue
		}
		body, err := os.ReadFile(path)
		if err != nil {
			log.Fatal(err)
		}
		rec, err := lib.InstallText(ctx, string(body), path)
		if err != nil {
			log.Fatalf("install %s: %v", path, err)
		}
		log.Printf("installed %s (%s)", rec.Name, rec.Key)
	}
	// The reader's scheme must be registered before its window is made, and
	// the same server must reach the window that reports the position.
	srv := reader.New(c)
	if err := mygo.Protocol.Handle(reader.Scheme, srv); err != nil {
		log.Fatalf("scheme: %v", err)
	}
	app := ui.New(c, lib, srv)
	app.SetDevOptions(ui.DevOptions{Query: *query, OpenFirst: *openFirst, Browse: *browse,
		ExploreIndex: *exploreIndex, ExplorePage: *explorePage, Shelf: *shelfMode, OpenChapter: *chapter, ShelfDemo: *shelfDemo, ScrollPercent: *scrollPercent,
		AdvanceChapter: *advance, AutoNext: *autoNext, Filters: *filters, Filter: *filterPick,
		SearchAll: *searchAll, OpenLast: *openLast,
		KeepPage: *keepPage, Favourite: *favourite, OpenSettings: *settingsShot,
		SetSetting: *setSetting})

	mygo.App.SetLanguages("zh-Hans")
	var shotErr error
	mygo.App.WhenReady(func() {
		go func() {
			time.Sleep(*wait)
			data, err := app.Window().CapturePage()
			if err == nil {
				err = os.WriteFile(*out, data, 0o644)
			}
			if err != nil {
				shotErr = err
				mygo.App.Quit()
				return
			}
			log.Printf("capture: %s (%d bytes)", *out, len(data))
			shotErr = inspect(data)
			// The sources window is a window of its own; catching it needs
			// its own capture.
			if *settingsShot && *settingsOut != "" {
				if win := app.ManageWindow(); win != nil {
					if shot, err := win.CapturePage(); err == nil {
						if werr := os.WriteFile(*settingsOut, shot, 0o644); werr == nil {
							log.Printf("sources capture: %s (%d bytes)", *settingsOut, len(shot))
						}
					} else {
						log.Printf("sources capture: %v", err)
					}
				}
			}
			mygo.App.Quit()
		}()
	})
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
	if shotErr != nil {
		log.Fatal(shotErr)
	}
	log.Println("done")
}

// inspect reads the capture back: a blank window is one colour.
func inspect(data []byte) error {
	img, err := png.Decode(bytesReader(data))
	if err != nil {
		return err
	}
	seen := map[uint32]bool{}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y += 5 {
		for x := b.Min.X; x < b.Max.X; x += 5 {
			r, g, bl, _ := img.At(x, y).RGBA()
			seen[r<<16^g<<8^bl] = true
		}
	}
	log.Printf("capture %v, %d distinct colours", b, len(seen))
	if len(seen) < 8 {
		log.Printf("the window looks blank")
	}
	return nil
}

func bytesReader(b []byte) *os.File {
	f, err := os.CreateTemp("", "win*.png")
	if err != nil {
		panic(err)
	}
	_, _ = f.Write(b)
	_, _ = f.Seek(0, 0)
	return f
}

var _ image.Image
