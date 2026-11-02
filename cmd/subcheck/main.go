// subcheck reopens an episode in the desktop player to check that a subtitle
// choice outlives the page. It is a development tool, not part of the app.
//
//	go run ./cmd/subcheck -dir internal/player/testdata/hls-subs -out shot.png
//
// Why a separate tool: the choice used to live only in the page, so choosing
// 中文 and closing the player fell back to the manifest default on reopen. A Go
// test proves what is stored; only a real WebView with a real HLS.js session
// proves the page asks for it and applies it.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/player"
	"github.com/laincat/nekophage/internal/shelf"
	"github.com/laincat/nekophage/internal/sourceapi"
	"github.com/laincat/veneracat-core/plugin"
	"github.com/laincat/veneracat-core/video"
)

type pageState struct {
	Loading      bool     `json:"loading"`
	Value        string   `json:"value"`
	Selected     string   `json:"selected"`
	Hidden       bool     `json:"hidden"`
	ShowingText  string   `json:"showingText"`
	HLS          string   `json:"hls"`
	CueLanguages []string `json:"cueLanguages"`
	Options      []string `json:"options"`
}

const stateScript = `(() => {
  const s = document.getElementById('subtitles');
  const v = document.getElementById('v');
  // The page can still be loading: the WebView exists before its document does.
  if (!s || !v) return JSON.stringify({loading: true});
  const tracks = Array.from(v.textTracks);
  const showing = tracks.filter(function (t) { return t.mode === 'showing'; });
  const text = showing.map(function (t) {
    return Array.from(t.cues || []).map(function (c) { return c.text; }).join(' ');
  }).join(' | ');
  return JSON.stringify({
    value: s.value,
    selected: s.selectedOptions[0] ? s.selectedOptions[0].textContent : '',
    hidden: document.getElementById('subtitle-choice').hidden,
    options: Array.from(s.options).map(function (o) { return o.textContent; }),
    showingText: text,
    cueLanguages: showing.map(function (t) { return (t.language || t.label || '') + ':' + (t.cues ? t.cues.length : 0); }),
    hls: typeof Hls !== 'undefined' ? Hls.version : 'absent'
  });
})()`

const selectScript = `(() => {
  const s = document.getElementById('subtitles');
  const wanted = %q;
  const options = Array.from(s.options);
  const hit = options.findIndex(function (o) { return o.textContent.trim() === wanted; });
  if (hit < 0) return JSON.stringify({error: 'no such option', options: options.map(function (o) { return o.textContent; })});
  // The option's value is the track position ("0" for the first track), not
  // the index of the option element: "关闭" takes the first slot.
  s.value = options[hit].value;
  s.dispatchEvent(new Event('change'));
  return JSON.stringify({picked: s.selectedOptions[0].textContent});
})()`

func main() {
	dir := flag.String("dir", "internal/player/testdata/hls-subs", "HLS fixture directory")
	work := flag.String("work", "", "where the shelf keeps the choice (a temporary directory by default)")
	choose := flag.String("choose", "English", "the subtitle option to pick before reopening")
	out := flag.String("out", "", "where to write the final window capture")
	wait := flag.Duration("wait", 25*time.Second, "how long the page may take to load")
	flag.Parse()

	abs, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatal(err)
	}
	files := http.FileServer(http.Dir(abs))
	mux := http.NewServeMux()
	mux.Handle("/", files)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = srv.Serve(listener) }()
	defer srv.Close()
	mediaURL := "http://" + listener.Addr().String() + "/master.m3u8"

	workDir := *work
	if workDir == "" {
		if workDir, err = os.MkdirTemp("", "subcheck"); err != nil {
			log.Fatal(err)
		}
		defer os.RemoveAll(workDir)
	}
	// The store under test is the one the app uses, not a stub: the tool must
	// fail if the real shelf loses the choice.
	records, err := shelf.Open(workDir)
	if err != nil {
		log.Fatal(err)
	}
	meta := player.Meta{Source: "sub_check", ItemID: "clip", Title: "Local HLS clip",
		LineID: "local", EpisodeID: "ep1", EpisodeName: "Episode 1"}
	// A choice belongs to an episode already in history, as it does in the app.
	if err := records.RecordVideo(shelf.VideoHistory{Source: meta.Source, ItemID: meta.ItemID,
		Title: meta.Title, LineID: meta.LineID, EpisodeID: meta.EpisodeID,
		EpisodeName: meta.EpisodeName, Position: 1, Duration: 10}); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	c, err := core.Open(core.Options{DataDir: filepath.Join(workDir, "core"),
		AppVersion: sourceapi.Level, Locale: "zh_CN"})
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	script := fmt.Sprintf(`class SubCheck extends VideoSource {
 name = "Subtitle check"; key = "sub_check"; version = "1.0.0"
 search = {load: async () => ({videos: [{id: "clip", title: "Local HLS clip"}], maxPage: 1})}
 video = {
  loadInfo: async () => ({id: "clip", title: "Local HLS clip", lines: [
   {id: "local", name: "Local", episodes: [{id: "ep1", title: "Episode 1"}]}]}),
  loadPlay: async () => ({url: %q, format: "hls"})
 }
}`, mediaURL)
	if _, err := c.InstallTyped(ctx, plugin.TypeVideoJS, script, "file", ""); err != nil {
		log.Fatalf("install: %v", err)
	}
	if page, err := c.VideoSearch(ctx, "sub_check", "clip", ""); err != nil || len(page.Items) != 1 {
		log.Fatalf("search: %v, %+v", err, page)
	}
	play, err := c.VideoPlay(ctx, "sub_check", "clip", "local", "ep1")
	if err != nil {
		log.Fatalf("play: %v", err)
	}
	if play.Format != video.FormatHLS {
		log.Fatalf("unexpected play format %q", play.Format)
	}

	server, err := player.NewServer()
	if err != nil {
		log.Fatal(err)
	}
	defer server.Close()
	// The same two hooks internal/ui/video.go installs.
	server.Subtitle = func(m player.Meta) (player.Subtitle, bool) {
		record, ok := records.VideoProgress(m.Source, m.ItemID, m.EpisodeID)
		if !ok || record.Subtitle == nil {
			return player.Subtitle{}, false
		}
		return player.Subtitle{Off: record.Subtitle.Off, Name: record.Subtitle.Name,
			Language: record.Subtitle.Language}, true
	}
	server.OnSubtitle = func(m player.Meta, choice player.Subtitle) {
		if _, err := records.SetVideoSubtitle(m.Source, m.ItemID, m.EpisodeID,
			&shelf.VideoSubtitle{Off: choice.Off, Name: choice.Name, Language: choice.Language}); err != nil {
			log.Printf("subtitle write: %v", err)
		}
	}
	first, err := server.Open(c, meta.Source, play, meta)
	if err != nil {
		log.Fatal(err)
	}
	defer first.Close()

	var (
		checkErr error
		web      *mygo.WebView
	)
	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{Title: "subtitle check", Width: 900, Height: 620,
			Content: ui.View(func(ctx *ui.Context) {
				ui.Column(ctx).Fill().Children(func() {
					if web != nil {
						ui.WebView(ctx, web).Grow(1)
					} else {
						ui.Text(ctx, "正在加载…").Padding(16)
					}
				})
			})})
		view, err := win.NewWebView(mygo.WebViewOptions{URL: first.URL("Local HLS clip", 0)})
		if err != nil {
			checkErr = err
			mygo.App.Quit()
			return
		}
		web = view
		win.Invalidate()

		state := func() pageState {
			raw, err := view.Page().Eval(stateScript)
			if err != nil {
				checkErr = fmt.Errorf("read state: %w", err)
				return pageState{}
			}
			text, _ := raw.(string)
			var out pageState
			if err := json.Unmarshal([]byte(text), &out); err != nil {
				checkErr = fmt.Errorf("state is not JSON: %v (%q)", err, text)
			}
			return out
		}
		// Wait until the manifest's subtitle groups have arrived: before that
		// the select has only "关闭" and there is nothing to pick.
		deadline := time.Now().Add(*wait)
		var before pageState
		for time.Now().Before(deadline) {
			if checkErr != nil {
				mygo.App.Quit()
				return
			}
			before = state()
			if len(before.Options) > 1 && !before.Hidden {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		log.Printf("before: options=%v selected=%q hidden=%v hls=%s", before.Options, before.Selected, before.Hidden, before.HLS)
		if len(before.Options) < 2 {
			checkErr = fmt.Errorf("the manifest's subtitle groups never arrived: %+v", before)
			mygo.App.Quit()
			return
		}

		picked, err := view.Page().Eval(fmt.Sprintf(selectScript, *choose))
		log.Printf("choose %q: %v (%v)", *choose, picked, err)
		if err != nil {
			checkErr = fmt.Errorf("choose %q: %w", *choose, err)
			mygo.App.Quit()
			return
		}
		if text, _ := picked.(string); strings.Contains(text, `"error"`) {
			checkErr = fmt.Errorf("the option is missing: %s", text)
			mygo.App.Quit()
			return
		}
		// The page reports the change to the app; wait for it to land.
		stored := false
		for deadline = time.Now().Add(10 * time.Second); time.Now().Before(deadline); {
			if record, ok := records.VideoProgress(meta.Source, meta.ItemID, meta.EpisodeID); ok && record.Subtitle != nil {
				stored = true
				log.Printf("stored: %+v", *record.Subtitle)
				break
			}
			time.Sleep(300 * time.Millisecond)
		}
		if !stored {
			checkErr = fmt.Errorf("the app never stored the choice")
			mygo.App.Quit()
			return
		}
		time.Sleep(2 * time.Second)
		chosen := state()
		log.Printf("chosen: selected=%q showing=%q cues=%v", chosen.Selected, chosen.ShowingText, chosen.CueLanguages)
		if chosen.Selected != *choose {
			log.Printf("the select did not keep %q: %+v", *choose, chosen)
		}

		// Reopen the same episode in a new session, as closing and pressing
		// play again does. The page must ask for the stored choice.
		second, err := server.Open(c, meta.Source, play, meta)
		if err != nil {
			checkErr = err
			mygo.App.Quit()
			return
		}
		defer second.Close()
		if err := view.Page().LoadURL(second.URL("Local HLS clip", 0)); err != nil {
			checkErr = fmt.Errorf("reload: %w", err)
			mygo.App.Quit()
			return
		}
		var reopened pageState
		for deadline = time.Now().Add(*wait); time.Now().Before(deadline); {
			reopened = state()
			if len(reopened.Options) > 1 && reopened.Value != "" {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		log.Printf("reopened: options=%v selected=%q value=%s showing=%q cues=%v",
			reopened.Options, reopened.Selected, reopened.Value, reopened.ShowingText, reopened.CueLanguages)
		if reopened.Selected != *choose {
			checkErr = fmt.Errorf("reopening fell back to %q, not %q", reopened.Selected, *choose)
		} else if reopened.Value == "-1" {
			checkErr = fmt.Errorf("reopening shows %q but has subtitles switched off", *choose)
		} else if !strings.Contains(reopened.ShowingText, "Subtitle rendering proof") {
			checkErr = fmt.Errorf("the restored track rendered no English cue: %q", reopened.ShowingText)
		}
		if *out != "" {
			if data, err := win.CapturePage(); err == nil {
				if err := os.WriteFile(*out, data, 0o644); err == nil {
					log.Printf("window capture: %s (%d bytes; the web layer is not in it)", *out, len(data))
				}
			}
		}
		mygo.App.Quit()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
	if checkErr != nil {
		log.Fatal(checkErr)
	}
	fmt.Println("done")
}
