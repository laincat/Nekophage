package ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/laincat/nekophage/internal/core"
)

func TestLeavingAViewStopsItsCoverHTTPRequest(t *testing.T) {
	started, stopped, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
			close(stopped)
		case <-release:
			fmt.Fprint(w, "cover")
		}
	}))
	defer server.Close()
	defer close(release)
	c, err := core.Open(core.Options{DataDir: t.TempDir(), AppVersion: "1.6.3"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	script := `class CoverCancel extends ComicSource {
 name = "Cover cancellation"; key = "cover_cancel"; version = "1.0.0"
 search = {load: async () => ({comics: [], maxPage: 1})}
}`
	if _, err = c.Install(context.Background(), script, "file", ""); err != nil {
		t.Fatal(err)
	}
	a := &App{core: c, sourceKey: "cover_cancel", mode: 0, images: newImageCache()}
	a.cover("cover_cancel", "comic", server.URL)
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("the cover request never reached the server")
	}
	a.mu.Lock()
	a.searchGen.next()
	a.mu.Unlock()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("leaving the search kept its cover HTTP request running")
	}
}

func TestImageDownloadsBoundHTTPAndCancelQueuedWork(t *testing.T) {
	var active, peak, requests atomic.Int32
	started := make(chan struct{}, maxImageDownloads*2)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		n := active.Add(1)
		defer active.Add(-1)
		for p := peak.Load(); n > p && !peak.CompareAndSwap(p, n); p = peak.Load() {
		}
		started <- struct{}{}
		select {
		case <-release:
			_, _ = w.Write([]byte("image"))
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	defer unblock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	fetch := func(ctx context.Context) ([]byte, string, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
		if err != nil {
			return nil, "", err
		}
		res, err := srv.Client().Do(req)
		if err != nil {
			return nil, "", err
		}
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		return data, res.Header.Get("Content-Type"), err
	}
	var downloads imageDownloads
	done := make(chan error, maxImageDownloads*2)
	for i := 0; i < maxImageDownloads*2; i++ {
		go func() {
			_, _, err := downloads.do(ctx, fetch)
			done <- err
		}()
	}
	for i := 0; i < maxImageDownloads; i++ {
		select {
		case <-started:
		case <-ctx.Done():
			t.Fatal("the first image downloads did not reach the server")
		}
	}

	// All connections are held by the server. Cancel another image while it
	// has no slot; its fetch must never run and it must not consume a slot.
	queuedCtx, cancelQueued := context.WithCancel(ctx)
	queuedDone := make(chan error, 1)
	var queuedFetchRan atomic.Bool
	go func() {
		_, _, err := downloads.do(queuedCtx, func(context.Context) ([]byte, string, error) {
			queuedFetchRan.Store(true)
			return nil, "", nil
		})
		queuedDone <- err
	}()
	cancelQueued()
	select {
	case err := <-queuedDone:
		if !errors.Is(err, context.Canceled) || queuedFetchRan.Load() {
			t.Fatalf("queued cancellation = %v, fetch ran = %v", err, queuedFetchRan.Load())
		}
	case <-ctx.Done():
		t.Fatal("the cancelled queued image stayed blocked")
	}

	unblock()
	for i := 0; i < maxImageDownloads*2; i++ {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-ctx.Done():
			t.Fatal("image slots were not released for later downloads")
		}
	}
	if got := peak.Load(); got != maxImageDownloads {
		t.Fatalf("peak simultaneous HTTP requests = %d, want %d", got, maxImageDownloads)
	}
	if got := requests.Load(); got != maxImageDownloads*2 {
		t.Fatalf("HTTP requests = %d, want %d", got, maxImageDownloads*2)
	}
}
