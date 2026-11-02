// videocheck plays a real clip through the desktop player, so "desktop video
// works" is something observed rather than inferred from a compile.
//
//	go run ./cmd/videocheck -media clip.mp4 -out shot.png
//
// It installs a video source that points at a loopback server, opens a player
// session through the media proxy, loads the player page in a WebView and asks
// the page how much it decoded. A window capture shows only the native layer,
// so the numbers the page reports are the evidence.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/player"
	"github.com/laincat/nekophage/internal/sourceapi"
	"github.com/laincat/veneracat-core/plugin"
	"github.com/laincat/veneracat-core/video"
)

func main() {
	mediaPath := flag.String("media", "", "local mp4 to play")
	out := flag.String("out", "video.png", "where to write the window capture")
	wait := flag.Duration("wait", 25*time.Second, "how long the clip may take to start")
	flag.Parse()
	if *mediaPath == "" {
		flag.Usage()
		os.Exit(2)
	}
	media, err := os.ReadFile(*mediaPath)
	if err != nil {
		log.Fatal(err)
	}

	// The clip is served with byte ranges, as a real CDN serves one; the
	// player fetches it through the proxy, never directly.
	mux := http.NewServeMux()
	mux.HandleFunc("/clip.mp4", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		http.ServeContent(w, r, "clip.mp4", time.Time{}, bytes.NewReader(media))
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(listener) }()
	defer srv.Close()
	mediaURL := "http://" + listener.Addr().String() + "/clip.mp4"

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	c, err := core.Open(core.Options{DataDir: filepath.Join(os.TempDir(), "videocheck"), AppVersion: sourceapi.Level, Locale: "zh_CN"})
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	script := fmt.Sprintf(`class VideoCheck extends VideoSource {
 name = "Video check"; key = "video_check"; version = "1.0.0"
 search = {load: async () => ({videos: [{id: "clip", title: "Local clip"}], maxPage: 1})}
 video = {
  loadInfo: async () => ({id: "clip", title: "Local clip", lines: [
   {id: "line1", name: "Line 1", episodes: [{id: "ep1", title: "Episode 1"}]}]}),
  loadPlay: async () => ({url: %q, format: "mp4", headers: {Referer: "https://fixture.invalid/"}})
 }
}`, mediaURL)
	if _, err := c.InstallTyped(ctx, plugin.TypeVideoJS, script, "file", ""); err != nil {
		log.Fatalf("install: %v", err)
	}
	page, err := c.VideoSearch(ctx, "video_check", "clip", "")
	if err != nil || len(page.Items) != 1 {
		log.Fatalf("search: %v, %+v", err, page)
	}
	details, err := c.VideoDetails(ctx, "video_check", "clip")
	if err != nil || len(details.Lines) != 1 || len(details.Lines[0].Episodes) != 1 {
		log.Fatalf("details: %v, %+v", err, details)
	}
	play, err := c.VideoPlay(ctx, "video_check", "clip", "line1", "ep1")
	if err != nil {
		log.Fatalf("play: %v", err)
	}

	server, err := player.NewServer()
	if err != nil {
		log.Fatal(err)
	}
	defer server.Close()
	progress := make(chan player.Progress, 8)
	server.OnProgress = func(_ player.Meta, p player.Progress) {
		select {
		case progress <- p:
		default:
		}
	}
	meta := player.Meta{Source: "video_check", ItemID: "clip", Title: "Local clip",
		LineID: "line1", EpisodeID: "ep1", EpisodeName: "Episode 1"}
	session, err := server.Open(c, "video_check", play, meta)
	if err != nil {
		log.Fatalf("open player session: %v", err)
	}
	defer session.Close()

	var (
		checkErr error
		web      *mygo.WebView
	)
	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{Title: "player", Width: 900, Height: 620,
			Content: ui.View(func(ctx *ui.Context) {
				ui.Column(ctx).Fill().Children(func() {
					if web != nil {
						ui.WebView(ctx, web).Grow(1)
					} else {
						ui.Text(ctx, "正在加载…").Padding(16)
					}
				})
			})})
		view, err := win.NewWebView(mygo.WebViewOptions{URL: session.URL("Local clip", 0)})
		if err != nil {
			checkErr = err
			mygo.App.Quit()
			return
		}
		web = view
		win.Invalidate()
		go func() {
			defer mygo.App.Quit()
			deadline := time.Now().Add(*wait)
			var ready, width, time1, time2 any
			for time.Now().Before(deadline) {
				time.Sleep(time.Second)
				ready, _ = view.Page().Eval("document.getElementById('v').readyState")
				width, _ = view.Page().Eval("document.getElementById('v').videoWidth")
				if n, ok := width.(float64); ok && n > 0 {
					break
				}
			}
			// Playback only counts if the position advances on its own.
			_, _ = view.Page().Eval("document.getElementById('v').play(); 0")
			time.Sleep(2 * time.Second)
			time1, _ = view.Page().Eval("document.getElementById('v').currentTime")
			time.Sleep(2 * time.Second)
			time2, _ = view.Page().Eval("document.getElementById('v').currentTime")
			duration, _ := view.Page().Eval("document.getElementById('v').duration")
			errText, _ := view.Page().Eval("document.getElementById('err').textContent")
			hlsMode, _ := view.Page().Eval("typeof Hls !== 'undefined' ? Hls.version : 'absent'")
			log.Printf("page: readyState=%v videoWidth=%v duration=%v time=%v -> %v hls.js=%v err=%q",
				ready, width, duration, time1, time2, hlsMode, errText)
			if data, err := win.CapturePage(); err == nil {
				if err := os.WriteFile(*out, data, 0o644); err == nil {
					log.Printf("window capture: %s (%d bytes; the web layer is not in it)", *out, len(data))
				}
			}
			start, ok1 := time1.(float64)
			end, ok2 := time2.(float64)
			if !ok1 || !ok2 || end <= start {
				checkErr = fmt.Errorf("playback did not advance: %v -> %v", time1, time2)
			}
			if text, ok := errText.(string); ok && text != "" {
				checkErr = fmt.Errorf("the page reported: %s", text)
			}
			// The page reports progress on pause; ask for one explicitly.
			_, _ = view.Page().Eval("document.getElementById('v').pause(); 0")
			select {
			case p := <-progress:
				log.Printf("progress report: time=%.2f duration=%.2f paused=%v", p.Time, p.Duration, p.Paused)
				if p.Time <= 0 || p.Duration <= 0 {
					checkErr = fmt.Errorf("progress report was empty: %+v", p)
				}
			case <-time.After(5 * time.Second):
				checkErr = fmt.Errorf("the player page never reported progress")
			}
		}()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
	if checkErr != nil {
		log.Fatal(checkErr)
	}
	if play.Format != video.FormatAuto && play.Format != video.FormatMP4 {
		log.Fatalf("unexpected play format %q", play.Format)
	}
	fmt.Println("done")
}
