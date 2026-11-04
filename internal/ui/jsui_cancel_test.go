package ui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/library"
)

func TestClosingDialogStopsPictureHTTP(t *testing.T) {
	started, stopped, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
			close(stopped)
		case <-release:
			http.Error(w, "held", http.StatusOK)
		}
	}))
	defer server.Close()
	defer close(release)
	c, err := core.Open(core.Options{DataDir: t.TempDir(), AppVersion: "1.6.3"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	script := `class PictureCancel extends ComicSource {
 name = "Picture cancellation"; key = "picture_cancel"; version = "1.0.0"
 search = {load: async () => ({comics: [], maxPage: 1})}
}`
	if _, err = c.Install(context.Background(), script, "file", ""); err != nil {
		t.Fatal(err)
	}
	a := &App{core: c, lib: library.New(c)}
	picture := &dialogPicture{url: server.URL}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	picture.mu.Lock()
	picture.cancel = cancel
	picture.mu.Unlock()
	go func() {
		_, _, _ = a.core.FetchThumbnail(ctx, "picture_cancel", server.URL, "jsinput")
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("the dialog picture request never reached the server")
	}
	picture.mu.Lock()
	picture.cancel()
	picture.cancel = nil
	picture.mu.Unlock()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("closing the dialog kept its picture HTTP request running")
	}
}
