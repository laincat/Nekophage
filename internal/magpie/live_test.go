package magpie

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A run against a Magpie that is actually installed on the machine. The
// headless tests fake the gateway, so nothing else shows that the address, the
// model catalogue, a real completion and a real stream work together.
//
//	NK_MAGPIE_LIVE=1 go test ./internal/magpie -run TestLiveMagpie -v
//
// The model is whatever the gateway offers first unless NK_MAGPIE_MODEL names
// one; a gateway with no routing configured is skipped, not failed, because
// that is a setup state rather than a defect.
func liveSession(t *testing.T) (*Session, *Client) {
	t.Helper()
	if os.Getenv("NK_MAGPIE_LIVE") == "" {
		t.Skip("NK_MAGPIE_LIVE is not set")
	}
	s, err := OpenSession(filepath.Join(t.TempDir(), "magpie.json"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := New(s.Settings().ClientConfig())
	if err != nil {
		t.Fatal(err)
	}
	return s, c
}

func TestLiveMagpie(t *testing.T) {
	s, c := liveSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	t.Logf("gateway %s, protocol %s", c.BaseURL(), c.Protocol())
	if err := c.Health(ctx); err != nil {
		t.Fatalf("health: %v", err)
	}
	models, err := c.Models(ctx)
	if err != nil {
		t.Fatalf("models: %v", err)
	}
	if len(models) == 0 {
		t.Skip("the gateway offers no model; routing is not configured")
	}
	t.Logf("%d models, first is %q", len(models), models[0].ID)

	model := os.Getenv("NK_MAGPIE_MODEL")
	if model == "" {
		model = models[0].ID
	}
	// The tier the interface offers has to resolve to something the gateway
	// actually lists, or the app would send a name it invented.
	if err := s.Update(Settings{BaseURL: s.Settings().BaseURL, Protocol: s.Settings().Protocol,
		Tiers: map[string]string{"simple": model}}); err != nil {
		t.Fatal(err)
	}
	if got := s.Settings().ModelFor(TaskSimple); got != model {
		t.Errorf("tier simple = %q, want %q", got, model)
	}

	// All three protocols are claimed; the real gateway is the only place to
	// show that each one is wired end to end rather than in a fake.
	for _, protocol := range []Protocol{Chat, Responses, Messages} {
		pc, err := New(Config{BaseURL: s.Settings().BaseURL, Token: s.Settings().Token, Protocol: protocol})
		if err != nil {
			t.Fatalf("%s: %v", protocol, err)
		}
		stream, err := pc.Chat(ctx, Request{Model: model, Messages: []Message{
			{Role: "user", Content: "Reply with the single word: ready"},
		}})
		if err != nil {
			t.Fatalf("%s: %v", protocol, err)
		}
		var text strings.Builder
		var events int
		for ev := range stream {
			events++
			if ev.Kind == EventError {
				t.Fatalf("%s stream error: %s", protocol, ev.Text)
			}
			text.WriteString(ev.Text)
		}
		if events == 0 {
			t.Fatalf("%s: the stream carried no events", protocol)
		}
		reply := strings.TrimSpace(text.String())
		if reply == "" {
			t.Fatalf("%s: the stream carried no text", protocol)
		}
		t.Logf("%s: %d events, reply %q", protocol, events, reply)
	}
}
