package player

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/laincat/veneracat-core/video"
)

// A request can resolve a token before Close and finish receiving its body
// afterwards. It must not overwrite a newer player's saved state at that point.
func TestClosedSessionDiscardsPendingReports(t *testing.T) {
	for _, shutdown := range []bool{false, true} {
		name := "session-close"
		if shutdown {
			name = "server-close"
		}
		t.Run(name, func(t *testing.T) {
			for _, report := range []struct{ path, body string }{
				{"progress", `{"time":8,"duration":10,"paused":true}`},
				{"subtitle", `{"off":true}`},
			} {
				t.Run(report.path, func(t *testing.T) {
					server, err := NewServer()
					if err != nil {
						t.Fatal(err)
					}
					defer server.Close()
					var callbacks atomic.Int32
					server.OnProgress = func(Meta, Progress) { callbacks.Add(1) }
					server.OnSubtitle = func(Meta, Subtitle) { callbacks.Add(1) }
					session, err := server.Open(httpSource{client: http.DefaultClient}, "probe",
						testPlay(t, "https://fixture.invalid/clip.mp4", video.FormatMP4),
						Meta{Source: "probe", ItemID: "clip", EpisodeID: "ep1"})
					if err != nil {
						t.Fatal(err)
					}
					defer session.Close()
					pageURL := session.URL("Probe", 0)
					prefix := pageURL[:strings.Index(pageURL, "/play")]
					reader, writer := io.Pipe()
					defer reader.Close()
					defer writer.Close()
					req := httptest.NewRequest(http.MethodPost, prefix+"/"+report.path, reader)
					req.Header.Set("Content-Type", "application/json")
					recorder := httptest.NewRecorder()
					done := make(chan struct{})
					go func() {
						defer close(done)
						server.ServeHTTP(recorder, req)
					}()
					// Write completes only after the handler has resolved the
					// session and consumed the bytes. Withhold EOF until Close.
					if _, err := writer.Write([]byte(report.body)); err != nil {
						t.Fatal(err)
					}
					if shutdown {
						if err := server.Close(); err != nil {
							t.Fatal(err)
						}
					} else {
						session.Close()
					}
					if err := writer.Close(); err != nil {
						t.Fatal(err)
					}
					select {
					case <-done:
					case <-time.After(2 * time.Second):
						t.Fatal("pending report never completed")
					}
					if recorder.Code != http.StatusGone || callbacks.Load() != 0 {
						t.Fatalf("late %s: status=%d callbacks=%d, want 410 and no write",
							report.path, recorder.Code, callbacks.Load())
					}
				})
			}
		})
	}
}
