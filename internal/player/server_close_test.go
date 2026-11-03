package player

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"

	"github.com/laincat/veneracat-core/video"
)

// Late playback preparation must not receive a usable-looking page address
// after its loopback server has shut down.
func TestClosedServerRefusesNewSessions(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	session, err := server.Open(httpSource{client: http.DefaultClient}, "probe",
		testPlay(t, "https://fixture.invalid/clip.mp4", video.FormatMP4), Meta{})
	if session != nil {
		defer session.Close()
	}
	if session != nil || !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("Open after Close: session returned=%t, error=%v; want no session and http.ErrServerClosed", session != nil, err)
	}
}

// Every concurrent Open must either be included in Close's cancellation or
// rejected. Returning after Close with an uncancelled proxy is never valid.
func TestServerCloseCannotLeaveLiveSessions(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	play := testPlay(t, "https://fixture.invalid/clip.mp4", video.FormatMP4)
	type result struct {
		session *Session
		err     error
	}
	results := make(chan result, 32)
	start := make(chan struct{})
	var workers sync.WaitGroup
	for i := 0; i < cap(results); i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			session, err := server.Open(httpSource{client: http.DefaultClient}, "probe", play, Meta{})
			results <- result{session, err}
		}()
	}
	close(start)
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	workers.Wait()
	close(results)
	for result := range results {
		if result.err != nil {
			if result.session != nil || !errors.Is(result.err, http.ErrServerClosed) {
				t.Errorf("unexpected Open result: %v", result.err)
			}
			continue
		}
		if result.session == nil {
			t.Error("Open returned neither a session nor an error")
			continue
		}
		if !errors.Is(result.session.proxy.ctx.Err(), context.Canceled) {
			t.Error("Open raced past Close and returned a live media proxy")
		}
		result.session.Close()
	}
}
