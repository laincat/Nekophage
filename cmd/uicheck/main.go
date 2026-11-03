// uicheck opens the main window on a real source, searches, and saves what
// the window shows, so that the native interface is checked with real data
// rather than assumed. It is a development tool, not part of the app.
//
//	go run ./cmd/uicheck -script /path/to/manga_dex.js -query "one piece" -out shot.png
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/library"
	"github.com/laincat/nekophage/internal/probesite"
	"github.com/laincat/nekophage/internal/reader"
	"github.com/laincat/nekophage/internal/sourceapi"
	"github.com/laincat/nekophage/internal/ui"
	"github.com/laincat/veneracat-core/comic/js"
	"github.com/laincat/veneracat-core/plugin"

	_ "github.com/egoist/mygo/locales/zh-Hans"
)

func main() {
	script := flag.String("script", "", "source script to install")
	script2 := flag.String("script2", "", "a second source script to install")
	query := flag.String("query", "", "search keyword")
	out := flag.String("out", "", "where to write the capture; nothing is written when empty")
	wait := flag.Duration("wait", 10*time.Second, "how long the search may take")
	openFirst := flag.Bool("open", false, "open the first result before capturing")
	browse := flag.Bool("browse", false, "capture the browse view instead of searching")
	categoryShot := flag.Bool("category", false, "capture the source's category page")
	rank := flag.String("rank", "", "pick a ranking option after the category page opens")
	favorites := flag.Bool("favorites", false, "capture the source's own favorites")
	servePort := flag.Int("serve", 0, "also serve one PNG on this port, for a source whose chapter points at loopback")
	expect := flag.String("expect", "", "fail unless the first result's title contains this")
	expectDetail := flag.String("expect-detail", "", "fail unless the opened comic's title contains this and it lists chapters")
	callback := flag.String("callback", "", "run one of the source's setting callbacks")
	tag := flag.String("tag", "", "press a tag on the opened comic, as namespace|tag")
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
	videoScript := flag.String("video-script", "", "video source script to install")
	videoQuery := flag.String("video", "", "search the video window for this text, open the first result and play it")
	videoFrame := flag.String("video-frame", "", "where to write the frame read back from the player")
	videoMedia := flag.String("video-media", "", "MP4 or fixture directory to serve on 127.0.0.1:18732")
	videoAuth := flag.Bool("video-auth", false, "require fixture Referer and a Cookie obtained at /login for every media request")
	videoHLS := flag.Bool("video-hlsjs", false, "verify the bundled HLS.js decoder even where native HLS is available")
	videoNative := flag.Bool("video-native", false, "force the browser's own HLS decoder, to exercise the fallback")
	videoAudio := flag.Bool("video-audio", false, "require a positive decoded-audio byte counter after playback")
	videoSubs := flag.Bool("video-subs", false, "require active subtitle text on a showing track")
	videoSubsText := flag.String("video-subs-text", "", "require this text in the active subtitle cue")
	// dir reuses a profile, so the shelf and installed sources carry over
	// from an earlier import.
	dir := flag.String("dir", "", "profile directory; a fresh one under the temp directory when empty")
	flag.Parse()
	if *script == "" && *script2 == "" && *videoScript == "" && *dir == "" {
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
	if *videoScript != "" {
		rec, err := lib.InstallFile(ctx, plugin.TypeVideoJS, *videoScript)
		if err != nil {
			log.Fatalf("install video source %s: %v", *videoScript, err)
		}
		log.Printf("installed video source %s (%s)", rec.Name, rec.Key)
	}
	// The reader's scheme must be registered before its window is made, and
	// the same server must reach the window that reports the position.
	srv := reader.New(c)
	if err := mygo.Protocol.Handle(reader.Scheme, srv); err != nil {
		log.Fatalf("scheme: %v", err)
	}
	app := ui.New(c, lib, srv)
	jsui.Attach(app)
	app.SetDevOptions(ui.DevOptions{WebSession: mygo.NewEphemeralSession(), Query: *query, OpenFirst: *openFirst, Browse: *browse,
		ExploreIndex: *exploreIndex, ExplorePage: *explorePage, Shelf: *shelfMode, OpenChapter: *chapter, ShelfDemo: *shelfDemo, ScrollPercent: *scrollPercent,
		AdvanceChapter: *advance, AutoNext: *autoNext, Filters: *filters, Filter: *filterPick,
		SearchAll: *searchAll, OpenLast: *openLast, Category: *categoryShot, Rank: *rank, Favorites: *favorites, Callback: *callback,
		Tag:      *tag,
		KeepPage: *keepPage, Favourite: *favourite, OpenSettings: *settingsShot, BackupCookies: *backupCookies,
		SetSetting: *setSetting, AI: *aiShot, AITest: *aiTest, AIPrompt: *aiPrompt, AIModel: *aiModel,
		Video: *videoQuery, VideoOut: *videoFrame, VideoQuit: *videoQuery != "", VideoForceHLS: *videoHLS, VideoForceNative: *videoNative, VideoRequireAudio: *videoAudio, VideoRequireSubtitles: *videoSubs || *videoSubsText != "", VideoSubtitleText: *videoSubsText})

	if *videoMedia != "" {
		stop, err := serveMedia(*videoMedia, "127.0.0.1:18732", *videoAuth)
		if err != nil {
			log.Fatalf("serve media: %v", err)
		}
		defer stop()
	}
	// A chapter that points at loopback can be opened without any site, which is what
	// the repository's own fixture does; see internal/probesite.
	if *servePort > 0 {
		stop, err := probesite.ServeOnePage(*servePort)
		if err != nil {
			log.Fatalf("serve: %v", err)
		}
		defer stop()
	}

	mygo.App.SetLanguages("zh-Hans")
	var shotErr error
	mygo.App.WhenReady(func() {
		go func() {
			if *videoQuery != "" {
				// The probe reports and quits on its own; a capture here would
				// photograph the window before it holds a player.
				return
			}
			time.Sleep(*wait)
			data, err := app.Window().CapturePage()
			// A default file name here would leave a window.png in whatever
			// directory the tool was started from, which is how one ended up
			// committed. Nothing is written unless a path was asked for.
			if err == nil && *out != "" {
				err = os.WriteFile(*out, data, 0o644)
			}
			if err != nil {
				shotErr = err
				mygo.App.Quit()
				return
			}
			if *out != "" {
				log.Printf("capture: %s (%d bytes)", *out, len(data))
			}
			shotErr = inspect(data)
			if *openFirst {
				title, chapters, errText := app.DetailForCheck()
				log.Printf("detail: title=%q chapters=%d err=%q", title, chapters, errText)
				if *expectDetail != "" {
					if !strings.Contains(title, *expectDetail) {
						shotErr = fmt.Errorf("detail title %q does not contain %q", title, *expectDetail)
					} else if chapters == 0 {
						shotErr = fmt.Errorf("the detail page lists no chapters")
					}
				}
			}
			if *expect != "" {
				// A grid that renders and a grid that found nothing capture to
				// almost the same size, so the count and the first title are what
				// the check has to read.
				count, first, errText := app.SearchForCheck()
				log.Printf("search: count=%d first=%q err=%q", count, first, errText)
				if count == 0 {
					shotErr = fmt.Errorf("search returned nothing (%s)", errText)
				} else if !strings.Contains(first, *expect) {
					shotErr = fmt.Errorf("first result %q does not contain %q", first, *expect)
				}
			}
			if *categoryShot {
				// Two halves have to agree: the parts are drawn from `category`, the
				// results come from `categoryComics`. Counting both is how a page that
				// draws choosers and loads nothing becomes visible.
				parts, values, page, comics, errText := app.CategoryForCheck()
				log.Printf("category: parts=%d values=%d page=%d comics=%d err=%q",
					parts, values, page, comics, errText)
			}
			if *browse {
				// The chips are drawn only when a source offers more than one
				// section, so the count is reported rather than inferred from a
				// picture that has the right shape either way.
				sections, chips, comics, errText := app.BrowseForCheck()
				log.Printf("browse: sections=%d chips=%d comics=%d err=%q",
					sections, chips, comics, errText)
			}
			if *favorites {
				// The same lesson as the comment thread: a listing and an empty
				// page capture to nearly the same size, so ask the view.
				count, first, errText := app.FavoritesForCheck()
				log.Printf("favorites: count=%d first=%q err=%q", count, first, errText)
			}
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
				// What the window holds, not what it looks like: an empty thread and
				// a real one capture to almost the same number of bytes.
				count, author, body, errText := app.CommentsForCheck()
				log.Printf("comments: count=%d author=%q body=%q err=%q", count, author, body, errText)
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
	if *videoQuery != "" {
		if err := app.VideoProbeError(); err != nil {
			log.Fatal(err)
		}
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

// serveMedia supplies local MP4/HLS resources. Optional fixture authentication
// makes browser decoding fail if the proxy drops source headers or cookies.
func serveMedia(path, address string, checkAuth bool) (func(), error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	var files http.Handler
	if info.IsDir() {
		files = http.FileServer(http.Dir(path))
	} else {
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		files = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/clip.mp4" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "video/mp4")
			http.ServeContent(w, r, "clip.mp4", time.Time{}, bytes.NewReader(body))
		})
	}
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "video_fixture", Value: "signed-in", Path: "/"})
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if checkAuth {
			cookie, err := r.Cookie("video_fixture")
			if r.Header.Get("Referer") != "https://fixture.invalid/" || err != nil || cookie.Value != "signed-in" {
				log.Printf("media fixture: auth rejected path=%s", r.URL.Path)
				http.Error(w, "fixture authentication missing", http.StatusForbidden)
				return
			}
		}
		log.Printf("media fixture: path=%s range=%q auth=%v", r.URL.Path, r.Header.Get("Range"), checkAuth)
		files.ServeHTTP(w, r)
	})
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	return func() { _ = srv.Close() }, nil
}
