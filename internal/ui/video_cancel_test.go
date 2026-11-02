package ui

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/shelf"
	"github.com/laincat/veneracat-core/plugin"
	"github.com/laincat/veneracat-core/video"
)

// Real Core + JS + loopback HTTP: discarded state alone is insufficient. The
// server must observe cancellation, and a subsequent read must still succeed.
func TestVideoNavigationStopsSupersededHTTP(t *testing.T) {
	for _, tc := range []struct{ read, action string }{
		{"search", "replace"}, {"search", "source"}, {"search", "close"},
		{"page", "replace"},
		{"details", "replace"}, {"details", "source"}, {"details", "close"},
		{"resume", "close"},
		{"play", "replace"}, {"play", "source"}, {"play", "close"},
	} {
		t.Run(tc.read+"/"+tc.action, func(t *testing.T) {
			started, stopped, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
			var onceStarted, onceStopped sync.Once
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query()
				blocked := (tc.read == "search" && r.URL.Path == "/search" && q.Get("q") == "old") ||
					(tc.read == "page" && r.URL.Path == "/search" && q.Get("page") == "2") ||
					((tc.read == "details" || tc.read == "resume") && r.URL.Path == "/details" && q.Get("id") == "old") ||
					(tc.read == "play" && r.URL.Path == "/play" && q.Get("id") == "old")
				if blocked {
					onceStarted.Do(func() { close(started) })
					select {
					case <-r.Context().Done():
						onceStopped.Do(func() { close(stopped) })
					case <-release:
					}
					return
				}
				fmt.Fprint(w, "new answer")
			}))
			defer server.Close()
			defer close(release)
			c, err := core.Open(core.Options{DataDir: t.TempDir(), AppVersion: "1.6.3"})
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			for _, key := range []string{"video_cancel", "other_video"} {
				script := fmt.Sprintf(`const BASE = %q
class VideoCancel extends VideoSource {
 name = "Video cancellation"; key = %q; version = "1.0.0"
 search = {load: async (keyword, options, page) => {
  const res = await Network.get(BASE + "/search?q=" + encodeURIComponent(keyword) + "&page=" + page, {})
  return {videos: [{id: keyword, title: res.body}], maxPage: 2}
 }}
 video = {
  loadInfo: async id => {
   const res = await Network.get(BASE + "/details?id=" + encodeURIComponent(id), {})
   return {title: res.body, lines: [{id: "line", name: "Line", episodes: [{id: "old", title: "Old"}, {id: "new", title: "New"}]}]}
  },
  loadPlay: async (id, line, episode) => {
   await Network.get(BASE + "/play?id=" + encodeURIComponent(episode), {})
   return {url: BASE + "/media.mp4", format: "mp4"}
  }
 }
}`, server.URL, key)
				if _, err := c.InstallTyped(context.Background(), plugin.TypeVideoJS, script, "file", ""); err != nil {
					t.Fatal(err)
				}
			}
			a := &App{core: c, video: videoState{key: "video_cancel", query: "old"}}
			switch tc.read {
			case "search":
				a.videoSearch()
			case "page":
				a.video.query = "first"
				a.videoSearch()
				waitVideoState(t, a, func(s videoState) bool { return s.busy == "" && s.next == "2" })
				a.videoMore()
			case "details":
				a.openVideoDetails(video.Item{ID: "old", Title: "Old"})
			case "resume":
				a.videoResume(shelf.VideoHistory{Source: "video_cancel", ItemID: "old", LineID: "line", EpisodeID: "old"})
			case "play":
				a.video.detailKey = "video_cancel"
				a.video.selected = &video.Item{ID: "clip", Title: "Clip"}
				a.video.details = &video.Details{Lines: []video.Line{{ID: "line", Episodes: []video.Episode{{ID: "old", Title: "Old"}, {ID: "new", Title: "New"}}}}}
				a.playVideo(0)
			}
			select {
			case <-started:
			case <-time.After(3 * time.Second):
				t.Fatal("the old video read never reached the HTTP server")
			}
			switch tc.action {
			case "source":
				a.mu.Lock()
				a.selectVideoSourceLocked("other_video")
				a.mu.Unlock()
			case "close":
				if tc.read == "search" {
					a.closeVideoList()
				} else {
					a.closeVideoDetails()
				}
			case "replace":
				switch tc.read {
				case "search", "page":
					a.mu.Lock()
					a.video.query = "new"
					a.mu.Unlock()
					a.videoSearch()
				case "details":
					a.openVideoDetails(video.Item{ID: "new", Title: "New"})
				case "play":
					a.playVideo(1)
				}
			}
			select {
			case <-stopped:
			case <-time.After(2 * time.Second):
				t.Fatal("navigation left the upstream video HTTP request running")
			}
			if tc.action == "replace" && tc.read == "play" {
				waitVideoState(t, a, func(s videoState) bool { return s.detailBusy == "" && s.playing == "New" && s.detailErr == "" })
			} else if tc.action == "replace" && tc.read == "details" {
				waitVideoState(t, a, func(s videoState) bool {
					return s.details != nil && s.details.ID == "new" && s.details.Title == "new answer" && s.detailErr == ""
				})
			} else {
				// Drive the real next search, not just a fresh context ticket.
				a.mu.Lock()
				a.video.query = "new"
				a.mu.Unlock()
				a.videoSearch()
				waitVideoState(t, a, func(s videoState) bool {
					return s.busy == "" && s.errText == "" && len(s.results) == 1 && s.results[0].ID == "new"
				})
			}
		})
	}
}

func waitVideoState(t *testing.T, a *App, accept func(videoState) bool) {
	t.Helper()
	if !a.waitFor(40, func() bool {
		a.mu.Lock()
		defer a.mu.Unlock()
		return accept(a.video)
	}) {
		a.mu.Lock()
		defer a.mu.Unlock()
		t.Fatalf("video state never reached its expected result: %+v", a.video)
	}
}

func TestVideoPaginationUsesTheQueryThatProducedItsList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s-%s", r.URL.Query().Get("q"), r.URL.Query().Get("page"))
	}))
	defer server.Close()
	c, err := core.Open(core.Options{DataDir: t.TempDir(), AppVersion: "1.6.3"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	script := fmt.Sprintf(`class PageProbe extends VideoSource {
 name = "Page probe"; key = "video_page_probe"; version = "1.0.0"
 search = {load: async (keyword, options, page) => {
  const res = await Network.get(%q + "/?q=" + encodeURIComponent(keyword) + "&page=" + page, {})
  return {videos: [{id: res.body, title: res.body}], maxPage: 2}
 }}
}`, server.URL)
	if _, err := c.InstallTyped(context.Background(), plugin.TypeVideoJS, script, "file", ""); err != nil {
		t.Fatal(err)
	}
	a := &App{core: c, video: videoState{key: "video_page_probe", query: "first"}}
	a.videoSearch()
	waitVideoState(t, a, func(s videoState) bool { return s.busy == "" && s.next == "2" })
	a.mu.Lock()
	a.video.query = "edited-but-not-submitted"
	a.mu.Unlock()
	a.videoMore()
	waitVideoState(t, a, func(s videoState) bool { return s.busy == "" && len(s.results) == 2 })
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.video.results[1].ID != "first-2" || a.video.next != "" {
		t.Fatalf("pagination mixed queries or kept a nonexistent page: %+v", a.video)
	}
}

func TestLateVideoResultsCannotClearNewerLoadingState(t *testing.T) {
	a := &App{video: videoState{key: "provider"}}
	oldSearch := a.videoSearchGen.next()
	a.videoSearchGen.next()
	a.video.busy = "new search"
	if a.finishVideoPage(oldSearch, "provider", false, &video.Page{Items: []video.Item{{ID: "old"}}}, nil) {
		t.Fatal("a late search result was accepted")
	}
	oldDetails := a.videoDetailGen.next()
	a.videoDetailGen.next()
	a.video.selected, a.video.detailKey = &video.Item{ID: "same-id"}, "provider"
	a.video.detailBusy = "new details"
	if a.finishVideoDetails(oldDetails, "provider", "same-id", &video.Details{Title: "old"}, nil) {
		t.Fatal("late details with the same source/item ID were accepted")
	}
	if a.video.busy != "new search" || a.video.detailBusy != "new details" || len(a.video.results) != 0 || a.video.details != nil {
		t.Fatal("an old completion changed the new request's state")
	}
}

func TestVideoListAndDetailsDoNotShareBusyState(t *testing.T) {
	a := &App{video: videoState{key: "provider", busy: "search pending", detailKey: "provider", detailBusy: "details pending", selected: &video.Item{ID: "clip"}}}
	detailTicket := a.videoDetailGen.next()
	if !a.finishVideoDetails(detailTicket, "provider", "clip", &video.Details{ID: "clip", Title: "ready"}, nil) {
		t.Fatal("current details were rejected")
	}
	if !strings.Contains(a.video.busy, "search") || a.video.detailBusy != "" {
		t.Fatal("finishing details cleared the still-running search")
	}
	a.closeVideoList()
	if a.video.details == nil || a.video.details.Title != "ready" {
		t.Fatal("closing the search window discarded completed independent details")
	}
}
