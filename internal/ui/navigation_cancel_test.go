package ui

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/laincat/nekophage/internal/core"
)

// These checks drive Core's actual source APIs. A request disappearing from a
// page is insufficient: the server must observe its connection being cancelled,
// and the source must still be usable for the next read.
func TestLeavingReadViewsStopsHTTPAndAllowsTheNextRead(t *testing.T) {
	for _, view := range []string{"details", "browse", "category"} {
		t.Run(view, func(t *testing.T) {
			started, stopped, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/old" {
					close(started)
					select {
					case <-r.Context().Done():
						close(stopped)
					case <-release:
					}
					return
				}
				fmt.Fprint(w, "new answer")
			}))
			defer srv.Close()
			defer close(release)
			c, err := core.Open(core.Options{DataDir: t.TempDir(), AppVersion: "1.6.3"})
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			script := fmt.Sprintf(`const BASE = %q
class ReadCancelProbe extends ComicSource {
    name = "Read cancellation probe"; key = "read_cancel_probe"; version = "1.0.0"
    comic = { loadInfo: async (id) => {
        const res = await Network.get(BASE + "/" + id, {})
        return {title: res.body}
    }}
    explore = [{title: "Probe", type: "multiPageComicList", load: async (page) => {
        const res = await Network.get(BASE + (page === 1 ? "/old" : "/new"), {})
        return {comics: [{id: "1", title: res.body}], maxPage: 2}
    }}]
    categoryComics = { load: async (name, param, options, page) => {
        const res = await Network.get(BASE + "/" + name, {})
        return {comics: [{id: "1", title: res.body}], maxPage: 1}
    }}
}`, srv.URL)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			rec, err := c.Install(ctx, script, "file", "")
			if err != nil {
				t.Fatal(err)
			}
			var app App
			var g *generation
			var read func(context.Context, string) (string, error)
			switch view {
			case "details":
				app.open, app.busy = true, "details"
				g = &app.detailGen
				read = func(ctx context.Context, id string) (string, error) {
					d, err := c.Details(ctx, rec.Key, id)
					return d.Title, err
				}
			case "browse":
				app.mode, app.exploreBusy = 1, true
				g = &app.exploreGen
				read = func(ctx context.Context, id string) (string, error) {
					page := 1
					if id == "new" {
						page = 2
					}
					result, err := c.Explore(ctx, rec.Key, 0, page, "")
					if err != nil {
						return "", err
					}
					if len(result.Sections) == 0 || len(result.Sections[0].Comics) == 0 {
						return "", fmt.Errorf("browse returned no comics")
					}
					return result.Sections[0].Comics[0].Title, nil
				}
			case "category":
				app.mode, app.catBusy = 3, true
				g = &app.catGen
				read = func(ctx context.Context, id string) (string, error) {
					result, err := c.CategoryComics(ctx, rec.Key, id, id, nil, 1)
					if err != nil {
						return "", err
					}
					if len(result.Comics) == 0 {
						return "", fmt.Errorf("category returned no comics")
					}
					return result.Comics[0].Title, nil
				}
			}
			ticket := g.next()
			oldCtx, finishOld := context.WithTimeout(g.requestContext(ticket), 10*time.Second)
			defer finishOld()
			done := make(chan error, 1)
			go func() { _, err := read(oldCtx, "old"); done <- err }()
			select {
			case <-started:
			case <-ctx.Done():
				t.Fatal("the read never reached the server")
			}
			if view == "details" {
				app.closeDetails()
			} else {
				app.setMode(2)
			}
			select {
			case <-stopped:
			case <-time.After(2 * time.Second):
				t.Fatal("leaving the view kept its HTTP request running")
			}
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("the cancelled read reported success")
				}
			case <-time.After(2 * time.Second):
				t.Fatal("Core did not return after cancelling the read")
			}
			newCtx, finishNew := context.WithTimeout(g.requestContext(g.next()), 3*time.Second)
			defer finishNew()
			text, err := read(newCtx, "new")
			if err != nil || text != "new answer" {
				t.Fatalf("next read = %q, %v, want new answer", text, err)
			}
		})
	}
}

func TestLeavingBrowsePreservesCompletedListsButResetsInterruptedOnes(t *testing.T) {
	for _, busy := range []bool{false, true} {
		app := App{mode: 1, exploreBusy: busy, catLoaded: true, catBusy: busy}
		app.explorePages = []core.ExplorePage{{Title: "saved"}}
		app.catComics = []core.Comic{{Title: "saved"}}
		app.setMode(2)
		if app.exploreBusy || app.catBusy {
			t.Fatal("the abandoned view stayed busy")
		}
		if busy && (len(app.explorePages) != 0 || app.catLoaded || len(app.catComics) != 0) {
			t.Fatal("an interrupted view kept a partial declaration instead of allowing a reload")
		}
		if !busy && (len(app.explorePages) != 1 || !app.catLoaded || len(app.catComics) != 1) {
			t.Fatal("leaving a completed view discarded its saved list")
		}
	}
}

func TestSourceSwitchCancelsReadsAndClearsOldDeclarations(t *testing.T) {
	var app App
	app.catLoaded, app.catBusy, app.exploreBusy = true, true, true
	app.explorePages = []core.ExplorePage{{Title: "old source"}}
	app.catComics = []core.Comic{{Title: "old source"}}
	var contexts []context.Context
	for _, g := range []*generation{&app.searchGen, &app.detailGen, &app.tagGen, &app.exploreGen, &app.catGen} {
		contexts = append(contexts, g.requestContext(g.next()))
	}
	app.mu.Lock()
	app.selectSourceLocked("new source")
	app.mu.Unlock()
	for i, ctx := range contexts {
		if ctx.Err() != context.Canceled {
			t.Errorf("read %d survived the source switch", i)
		}
	}
	if app.sourceKey != "new source" || app.catLoaded || app.catBusy || app.exploreBusy || len(app.explorePages) != 0 || len(app.catComics) != 0 {
		t.Fatal("the new source inherited the old source's declaration or loading state")
	}
}
