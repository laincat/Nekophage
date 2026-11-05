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
	picture := newDialogPicture()
	picture.url = server.URL
	go func() {
		ctx, cancel := context.WithTimeout(picture.lifetime, 30*time.Second)
		defer cancel()
		_, _, _ = a.core.FetchThumbnail(ctx, "picture_cancel", server.URL, "jsinput")
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("the dialog picture request never reached the server")
	}
	picture.close()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("closing the dialog kept its picture HTTP request running")
	}
}

// A dialog can be closed before its picture fetch gets going: the close handler
// runs on the window's thread while the fetch waits for its goroutine. The
// request must then never reach the source, or it would run for its full 30
// second limit with no dialog left to show the answer in.
func TestClosingDialogBeforeTheFetchStartsSendsNothing(t *testing.T) {
	requests, release := make(chan struct{}, 1), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- struct{}{}
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer server.Close()
	defer close(release)
	c, err := core.Open(core.Options{DataDir: t.TempDir(), AppVersion: "1.6.3"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	script := `class PictureEarly extends ComicSource {
 name = "Picture early close"; key = "picture_early"; version = "1.0.0"
 search = {load: async () => ({comics: [], maxPage: 1})}
}`
	if _, err = c.Install(context.Background(), script, "file", ""); err != nil {
		t.Fatal(err)
	}
	// The real path: the dialog closes before fetchDialogImage is called, so
	// this goes through the handler rather than a hand-made request.
	h := NewJSHandler()
	h.app = &App{core: c, lib: library.New(c), sourceKey: "picture_early"}
	picture := newDialogPicture()
	picture.url = server.URL
	picture.close()
	h.fetchDialogImage(picture, nil)
	select {
	case <-requests:
		t.Fatal("a closed dialog still sent its picture request to the source")
	case <-time.After(500 * time.Millisecond):
	}
}
