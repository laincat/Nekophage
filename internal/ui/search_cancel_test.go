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

// The server has to observe the old request going away. A discarded reply or a
// canceled UI ticket alone would not prove the source's HTTP request stopped.
func TestSearchReplacementStopsHTTPAndLeavesNewRequestUsable(t *testing.T) {
	started := make(chan struct{})
	stopped := make(chan struct{})
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") == "old" {
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
	script := fmt.Sprintf(`class SearchCancelProbe extends ComicSource {
    name = "Search cancellation probe"; key = "search_cancel_probe"; version = "1.0.0"
    search = { load: async (keyword, options, page) => {
        const res = await Network.get(%q + "/?q=" + encodeURIComponent(keyword), {})
        return {comics: [new Comic({id: keyword, title: res.body, cover: ""})], maxPage: 1}
    }}
}`, srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rec, err := c.Install(ctx, script, "file", "")
	if err != nil {
		t.Fatal(err)
	}

	var app App
	oldTicket := app.searchGen.next()
	oldCtx, cancelOld := context.WithTimeout(app.searchGen.requestContext(oldTicket), 10*time.Second)
	defer cancelOld()
	oldDone := make(chan error, 1)
	go func() {
		_, err := c.Search(oldCtx, rec.Key, "old", nil, 1)
		oldDone <- err
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("the old search never reached the HTTP server")
	}

	newTicket := app.searchGen.next()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("the superseded search kept its HTTP request running")
	}
	select {
	case err := <-oldDone:
		if err == nil {
			t.Fatal("the cancelled old search reported success")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the old Core search did not return after cancellation")
	}

	newCtx, cancelNew := context.WithTimeout(app.searchGen.requestContext(newTicket), 3*time.Second)
	defer cancelNew()
	page, err := c.Search(newCtx, rec.Key, "new", nil, 1)
	if err != nil {
		t.Fatalf("the new search failed after canceling the old one: %v", err)
	}
	if len(page.Comics) != 1 || page.Comics[0].Title != "new answer" {
		t.Fatalf("new search = %+v, want the server's new answer", page.Comics)
	}
}
