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
	"github.com/laincat/nekophage/internal/sourceapi"
	"github.com/laincat/nekophage/internal/ui"
	"github.com/laincat/veneracat-core/comic/js"

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
	categoryShot := flag.Bool("category", false, "capture the source's category page")
	rank := flag.String("rank", "", "pick a ranking option after the category page opens")
	favorites := flag.Bool("favorites", false, "capture the source's own favorites")
	callback := flag.String("callback", "", "run one of the source's setting callbacks")
	dialogShot := flag.Bool("dialog", false, "open a source dialog window and capture it")
	dialogOut := flag.String("dialog-out", "", "where to write the dialog capture")
	dialogInput := flag.Bool("dialog-input", false, "capture an input dialog instead of a confirm")
	dialogImage := flag.String("dialog-image", "", "an image the input dialog asks to show")
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
	aiShot := flag.Bool("ai", false, "open the AI window instead of searching")
	aiTest := flag.Bool("ai-test", false, "go with -ai: press 测试连接 and 读取模型 first")
	aiOut := flag.String("ai-out", "", "where to write the AI-window capture with -ai")
	aiPrompt := flag.String("ai-prompt", "", "go with -ai: send this text and capture the answer")
	aiModel := flag.String("ai-model", "", "go with -ai: fill the default model before sending")
	openLast := flag.Bool("open-last", false, "open the last result, to check a source switch")
	keepPage := flag.Bool("keep-page", false, "click the reader's 收藏本页 button once")
	favourite := flag.Bool("favourite", false, "keep the first search result, to check the shelf marker")
	commentsShot := flag.Bool("comments", false, "open the comment thread and capture it")
	commentsOut := flag.String("comments-out", "", "where to write the comment-window capture")
	logsShot := flag.Bool("logs", false, "open the source log panel and capture it")
	logsOut := flag.String("logs-out", "", "where to write the log-panel capture")
	backupCookies := flag.Bool("backup-cookies", false, "start the sources window with the backup cookie checkbox ticked")
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
	// A capture cannot press a button in a window it is photographing, so a
	// dialog the source opens takes its first action instead of waiting.
	jsui := ui.NewJSHandler()
	jsui.SetAutoConfirm(!*dialogShot)
	c, err := core.Open(core.Options{DataDir: dataDir, AppVersion: sourceapi.Level, Locale: "zh_CN", UI: jsui})
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
	jsui.Attach(app)
	app.SetDevOptions(ui.DevOptions{Query: *query, OpenFirst: *openFirst, Browse: *browse,
		ExploreIndex: *exploreIndex, ExplorePage: *explorePage, Shelf: *shelfMode, OpenChapter: *chapter, ShelfDemo: *shelfDemo, ScrollPercent: *scrollPercent,
		AdvanceChapter: *advance, AutoNext: *autoNext, Filters: *filters, Filter: *filterPick,
		SearchAll: *searchAll, OpenLast: *openLast, Category: *categoryShot, Rank: *rank, Favorites: *favorites, Callback: *callback,
		KeepPage: *keepPage, Favourite: *favourite, OpenSettings: *settingsShot, BackupCookies: *backupCookies,
		SetSetting: *setSetting, AI: *aiShot, AITest: *aiTest, AIPrompt: *aiPrompt, AIModel: *aiModel})

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
			// The AI window is its own window too.
			if *aiShot && *aiOut != "" {
				if win := app.AIWindow(); win != nil {
					if shot, err := win.CapturePage(); err == nil {
						if werr := os.WriteFile(*aiOut, shot, 0o644); werr == nil {
							log.Printf("AI capture: %s (%d bytes)", *aiOut, len(shot))
						}
					} else {
						log.Printf("AI capture: %v", err)
					}
				} else {
					log.Printf("AI capture: the AI window was never opened")
				}
			}
			if *commentsShot {
				app.OpenCommentsForCheck()
				time.Sleep(4000 * time.Millisecond)
				if win := app.CommentsWindow(); win != nil {
					if shot, err := win.CapturePage(); err == nil && *commentsOut != "" {
						if werr := os.WriteFile(*commentsOut, shot, 0o644); werr == nil {
							log.Printf("comments capture: %s (%d bytes)", *commentsOut, len(shot))
						}
					} else if err != nil {
						log.Printf("comments capture: %v", err)
					}
					win.Close()
				} else {
					log.Printf("comments capture: the window was never opened")
				}
			}
			if *logsShot {
				app.OpenSourceLogs()
				time.Sleep(1200 * time.Millisecond)
				if win := app.LogWindow(); win != nil {
					if shot, err := win.CapturePage(); err == nil && *logsOut != "" {
						if werr := os.WriteFile(*logsOut, shot, 0o644); werr == nil {
							log.Printf("log capture: %s (%d bytes)", *logsOut, len(shot))
						}
					} else if err != nil {
						log.Printf("log capture: %v", err)
					}
					win.Close()
				} else {
					log.Printf("log capture: the panel was never opened")
				}
			}
			// A dialog is a window of its own too. Without auto-confirm the
			// handler waits for a press, so the tool opens one and photographs
			// it before quitting.
			if *dialogShot {
				go func() {
					if *dialogInput {
						_, _ = jsui.ShowInputDialog("源需要你填一个值", *dialogImage, func(string) string { return "" })
						return
					}
					_ = jsui.ShowDialog("来自源的确认", "源要执行一个操作,是否继续?",
						[]js.DialogAction{
							{Text: "确定", Style: "filled", Invoke: func() error { return nil }},
							{Text: "取消", Invoke: func() error { return nil }},
						})
				}()
				time.Sleep(1500 * time.Millisecond)
				if win := jsui.LastWindow(); win != nil {
					if shot, err := win.CapturePage(); err == nil && *dialogOut != "" {
						if werr := os.WriteFile(*dialogOut, shot, 0o644); werr == nil {
							log.Printf("dialog capture: %s (%d bytes)", *dialogOut, len(shot))
						}
					} else if err != nil {
						log.Printf("dialog capture: %v", err)
					}
					win.Close()
				} else {
					log.Printf("dialog capture: no dialog window was opened")
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
