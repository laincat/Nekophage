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

func TestClosingCommentsStopsTheirHTTP(t *testing.T) {
	started, stopped, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
			close(stopped)
		case <-release:
			fmt.Fprint(w, `{"comments":[],"maxPage":1}`)
		}
	}))
	defer server.Close()
	defer close(release)
	c, err := core.Open(core.Options{DataDir: t.TempDir(), AppVersion: "1.6.3"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	script := fmt.Sprintf(`class CommentCancel extends ComicSource {
 name = "Comment cancellation"; key = "comment_cancel"; version = "1.0.0"
 comic = { loadComments: async () => { const res = await Network.get(%q, {}); return JSON.parse(res.body) } }
}`, server.URL)
	if _, err = c.Install(context.Background(), script, "file", ""); err != nil {
		t.Fatal(err)
	}
	a := &App{core: c}
	a.comments.open = true
	ctx, cancel := a.commentsContext()
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := c.Comments(ctx, "comment_cancel", "comic", "", 1, "")
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("the comment request never reached the server")
	}
	a.comments.mu.Lock()
	a.comments.cancel()
	a.comments.open = false
	a.comments.mu.Unlock()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("closing comments kept its HTTP request running")
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("the cancelled comment read reported success")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Core did not return after closing comments")
	}
}
