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

func TestLeavingFavoritesAndDetailsStopsTheirHTTP(t *testing.T) {
	for _, action := range []string{"list", "write"} {
		t.Run(action, func(t *testing.T) {
			started, stopped, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/"+action {
					fmt.Fprint(w, "[]")
					return
				}
				close(started)
				select {
				case <-r.Context().Done():
					close(stopped)
				case <-release:
					fmt.Fprint(w, "[]")
				}
			}))
			defer server.Close()
			defer close(release)
			c, err := core.Open(core.Options{DataDir: t.TempDir(), AppVersion: "1.6.3"})
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			script := fmt.Sprintf(`class FavoriteCancel extends ComicSource {
 name = "Favorite cancellation"; key = "favorite_cancel"; version = "1.0.0"
 favorites = {
  loadComics: async () => { await Network.get(%q + "/list", {}); return {comics: [], maxPage: 1} },
  addOrDelFavorite: async () => { await Network.get(%q + "/write", {}); return "ok" }
 }
}`, server.URL, server.URL)
			if _, err = c.Install(context.Background(), script, "file", ""); err != nil {
				t.Fatal(err)
			}
			a := &App{core: c, sourceKey: "favorite_cancel", open: true, detailComic: core.Comic{ID: "comic"}}
			if action == "list" {
				a.loadFavoritePage("favorite_cancel", "", 1)
			} else {
				a.srcFav.Supported = true
				a.toggleSourceFavorite()
			}
			select {
			case <-started:
			case <-time.After(3 * time.Second):
				t.Fatal("the favorite request never reached the server")
			}
			a.mu.Lock()
			if action == "list" {
				a.favGen.next()
			} else {
				a.detailGen.next()
			}
			a.mu.Unlock()
			select {
			case <-stopped:
			case <-time.After(2 * time.Second):
				t.Fatal("leaving the view kept the favorite HTTP request running")
			}
		})
	}
}
