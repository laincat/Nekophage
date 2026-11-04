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

func TestLeavingShelfStopsKeptPageHTTP(t *testing.T) {
	started, stopped, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
			close(stopped)
		case <-release:
			fmt.Fprint(w, "page")
		}
	}))
	defer server.Close()
	defer close(release)
	c, err := core.Open(core.Options{DataDir: t.TempDir(), AppVersion: "1.6.3"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	script := `class PageCancel extends ComicSource {
 name = "Page cancellation"; key = "page_cancel"; version = "1.0.0"
 search = {load: async () => ({comics: [], maxPage: 1})}
}`
	if _, err = c.Install(context.Background(), script, "file", ""); err != nil {
		t.Fatal(err)
	}
	a := &App{core: c, mode: 2, images: newImageCache()}
	a.pageImage("page_cancel", "comic", "chapter", server.URL)
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("the kept-page request never reached the server")
	}
	a.mu.Lock()
	a.mode = 0
	a.shelfGen.next()
	a.mu.Unlock()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("leaving the shelf kept its page HTTP request running")
	}
}
