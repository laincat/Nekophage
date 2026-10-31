package magpie

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// gateway records what the session sent, so a test can assert which model a
// tier actually reached.
type gateway struct {
	mu     sync.Mutex
	models []string
	server *httptest.Server
}

func newGateway(t *testing.T) *gateway {
	t.Helper()
	g := &gateway{}
	g.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			_, _ = w.Write([]byte(`{"data":[{"id":"cheap"},{"id":"strong"}]}`))
		case "/v1/chat/completions":
			var body struct {
				Model string `json:"model"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			g.mu.Lock()
			g.models = append(g.models, body.Model)
			g.mu.Unlock()
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: " + `{"choices":[{"delta":{"content":"ok"}}]}` + "\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(g.server.Close)
	return g
}

func (g *gateway) lastModel() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.models) == 0 {
		return ""
	}
	return g.models[len(g.models)-1]
}

func TestSessionStartsFromDefaultsAndTestsTheConnection(t *testing.T) {
	g := newGateway(t)
	session, err := OpenSession(filepath.Join(t.TempDir(), "magpie.json"))
	if err != nil {
		t.Fatal(err)
	}
	if session.Settings().Protocol != Chat {
		t.Fatalf("protocol = %q", session.Settings().Protocol)
	}
	next := DefaultSettings()
	next.BaseURL = g.server.URL
	if err := session.Update(next); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	message, err := session.Test(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if message == "" {
		t.Fatal("a successful connection test said nothing")
	}
}

// The tier the user configured is the model that actually reaches the gateway.
func TestSessionSendsTheTierModel(t *testing.T) {
	g := newGateway(t)
	session, err := OpenSession(filepath.Join(t.TempDir(), "magpie.json"))
	if err != nil {
		t.Fatal(err)
	}
	next := DefaultSettings()
	next.BaseURL = g.server.URL
	next.Tiers = map[string]string{"complex": "strong", "simple": "cheap"}
	if err := session.Update(next); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	events, err := session.Ask(ctx, TaskComplex, []Message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatal(err)
	}
	for range events {
	}
	if got := g.lastModel(); got != "strong" {
		t.Fatalf("gateway saw model %q, want the complex tier's group", got)
	}
	if _, err := session.Ask(ctx, TaskSimple, []Message{{Role: "user", Content: "hi"}}); err != nil {
		t.Fatal(err)
	}
	events, err = session.Ask(ctx, TaskSimple, []Message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatal(err)
	}
	for range events {
	}
	if got := g.lastModel(); got != "cheap" {
		t.Fatalf("gateway saw model %q, want the simple tier's group", got)
	}
}

// Changing the address must take effect on the next request, not after a
// restart: the client built from the old settings is dropped.
func TestSessionFollowsAnAddressChange(t *testing.T) {
	first := newGateway(t)
	second := newGateway(t)
	session, err := OpenSession(filepath.Join(t.TempDir(), "magpie.json"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i, g := range []*gateway{first, second} {
		next := DefaultSettings()
		next.BaseURL = g.server.URL
		// A task needs a model: the gateway rejects an empty one.
		next.Model = "model-" + g.server.URL
		if err := session.Update(next); err != nil {
			t.Fatal(err)
		}
		if _, err := session.Test(ctx); err != nil {
			t.Fatalf("test %d: %v", i, err)
		}
		events, err := session.Ask(ctx, TaskNormal, []Message{{Role: "user", Content: "hi"}})
		if err != nil {
			t.Fatalf("ask %d: %v", i, err)
		}
		for range events {
		}
	}
	if g := second.lastModel(); g != "model-"+second.server.URL {
		t.Fatalf("the second gateway saw %q, want its own model", g)
	}
	if g := first.lastModel(); g != "model-"+first.server.URL {
		t.Fatalf("the first gateway saw a later request: %q", g)
	}
}

// A rejected update leaves the stored settings alone, so a typo cannot wipe a
// working configuration.
func TestRejectedUpdateKeepsTheOldSettings(t *testing.T) {
	session, err := OpenSession(filepath.Join(t.TempDir(), "magpie.json"))
	if err != nil {
		t.Fatal(err)
	}
	good := DefaultSettings()
	good.BaseURL = "http://127.0.0.1:1"
	if err := session.Update(good); err != nil {
		t.Fatal(err)
	}
	bad := DefaultSettings()
	bad.Tiers = map[string]string{"hard": "x"}
	if err := session.Update(bad); err == nil {
		t.Fatal("a misspelled tier was accepted")
	}
	if session.Settings().BaseURL != good.BaseURL {
		t.Fatalf("settings changed after a rejected update: %+v", session.Settings())
	}
}

func TestSessionRefusesAnEmptyConversation(t *testing.T) {
	session, err := OpenSession(filepath.Join(t.TempDir(), "magpie.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.Ask(context.Background(), TaskNormal, nil); err == nil {
		t.Fatal("an empty conversation was sent")
	}
}

// The catalog the window shows comes through the session's own connection, so
// an address change is visible there too.
func TestSessionListsModelsThroughTheCurrentConnection(t *testing.T) {
	g := newGateway(t)
	session, err := OpenSession(filepath.Join(t.TempDir(), "magpie.json"))
	if err != nil {
		t.Fatal(err)
	}
	next := DefaultSettings()
	next.BaseURL = g.server.URL
	if err := session.Update(next); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	models, err := session.Models(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0].ID != "cheap" {
		t.Fatalf("models = %+v", models)
	}
}

// Nothing listening is reported as "start Magpie", not as a JSON error.
func TestSessionReportsAMissingGateway(t *testing.T) {
	session, err := OpenSession(filepath.Join(t.TempDir(), "magpie.json"))
	if err != nil {
		t.Fatal(err)
	}
	next := DefaultSettings()
	next.BaseURL = "http://127.0.0.1:1"
	if err := session.Update(next); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := session.Models(ctx); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("err = %v", err)
	}
}

// The Magpie installed here answers HTTP 400 for an empty model, so a task
// with nothing chosen must be refused where the reason is readable instead of
// being sent and rejected.
func TestAskRefusesAnEmptyModel(t *testing.T) {
	g := newGateway(t)
	session, err := OpenSession(filepath.Join(t.TempDir(), "magpie.json"))
	if err != nil {
		t.Fatal(err)
	}
	settings := DefaultSettings()
	settings.BaseURL = g.server.URL
	if err := session.Update(settings); err != nil {
		t.Fatal(err)
	}
	if _, err := session.Ask(context.Background(), TaskNormal, []Message{{Role: "user", Content: "hi"}}); !errors.Is(err, ErrNoModel) {
		t.Fatalf("err = %v, want ErrNoModel", err)
	}
	if got := g.lastModel(); got != "" {
		t.Errorf("the gateway was asked with model %q", got)
	}
}
