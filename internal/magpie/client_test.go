package magpie

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func client(t *testing.T, server *httptest.Server, protocol Protocol) *Client {
	t.Helper()
	c, err := New(Config{BaseURL: server.URL, Protocol: protocol, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	return c
}

func TestModelsKeepTheCatalogIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer magpie" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-group","name":"强模型组"},{"id":"vendor/model-1"}]}`))
	}))
	defer server.Close()

	models, err := client(t, server, Chat).Models(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0].ID != "gpt-group" || models[0].Name != "强模型组" {
		t.Fatalf("models = %+v", models)
	}
	if models[1].Name != "vendor/model-1" {
		t.Fatalf("a missing name should fall back to the id: %+v", models[1])
	}
}

func TestChatStreamsTextAndReportsTheEnd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("accept = %q", r.Header.Get("Accept"))
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("body: %v", err)
		}
		if body["stream"] != true || body["model"] != "gpt-group" {
			t.Errorf("body = %v", body)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"你\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"好\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	events, err := client(t, server, Chat).Chat(context.Background(), Request{
		Model: "gpt-group", Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var text strings.Builder
	sawDone := false
	for event := range events {
		switch event.Kind {
		case EventText:
			text.WriteString(event.Text)
		case EventDone:
			sawDone = true
		case EventError:
			t.Fatalf("stream failed: %s", event.Text)
		}
	}
	if text.String() != "你好" {
		t.Fatalf("text = %q", text.String())
	}
	if !sawDone {
		t.Fatal("the stream never reported completion")
	}
}

func TestAnthropicAndResponsesDialectsAreDecoded(t *testing.T) {
	anthropic := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["max_tokens"] == nil {
			t.Error("Anthropic needs max_tokens")
		}
		if body["system"] != "be brief" {
			t.Errorf("system = %v", body["system"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"hi\"}}\n\n"))
		_, _ = w.Write([]byte("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
	}))
	defer anthropic.Close()
	events, err := client(t, anthropic, Messages).Chat(context.Background(), Request{
		System: "be brief", Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var text strings.Builder
	for event := range events {
		if event.Kind == EventText {
			text.WriteString(event.Text)
		}
		if event.Kind == EventError {
			t.Fatalf("error: %s", event.Text)
		}
	}
	if text.String() != "hi" {
		t.Fatalf("text = %q", text.String())
	}

	responses := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"yo\"}\n\n"))
		_, _ = w.Write([]byte("event: response.completed\ndata: {\"type\":\"response.completed\"}\n\n"))
	}))
	defer responses.Close()
	events, err = client(t, responses, Responses).Chat(context.Background(), Request{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	text.Reset()
	for event := range events {
		if event.Kind == EventText {
			text.WriteString(event.Text)
		}
	}
	if text.String() != "yo" {
		t.Fatalf("responses text = %q", text.String())
	}
}

func TestCancellingStopsTheStream(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"a\"}}]}\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
		select {
		case <-release:
		case <-r.Context().Done():
			// The client cancelled: this is the behaviour under test.
		}
	}))
	defer server.Close()
	defer close(release)

	ctx, cancel := context.WithCancel(context.Background())
	events, err := client(t, server, Chat).Chat(ctx, Request{Messages: []Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatal(err)
	}
	first := <-events
	if first.Kind != EventText || first.Text != "a" {
		t.Fatalf("first event = %+v", first)
	}
	cancel()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case _, ok := <-events:
			if !ok {
				return // closed: the reader stopped instead of hanging
			}
		case <-deadline:
			t.Fatal("the stream kept running after cancellation")
		}
	}
}

func TestAnEmptyCatalogIsNotAnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()
	models, err := client(t, server, Chat).Models(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 0 {
		t.Fatalf("models = %+v", models)
	}
}

func TestDiagnosticsAreActOnAble(t *testing.T) {
	refusing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer refusing.Close()
	if _, err := client(t, refusing, Chat).Models(context.Background()); !errors.Is(err, ErrRejected) {
		t.Fatalf("401 gave %v", err)
	}

	missing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer missing.Close()
	if _, err := client(t, missing, Chat).Models(context.Background()); !errors.Is(err, ErrModelNotFound) {
		t.Fatalf("404 gave %v", err)
	}

	// Nothing listening: the message must tell the user to start Magpie.
	dead, err := New(Config{BaseURL: "http://127.0.0.1:1", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dead.Models(context.Background()); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("connection refused gave %v", err)
	}
}

func TestAddressAndTokenAreValidated(t *testing.T) {
	for _, bad := range []string{"ftp://127.0.0.1:3425", "http://user:pw@127.0.0.1:3425", "http://127.0.0.1:3425/?x=1"} {
		if _, err := New(Config{BaseURL: bad}); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	// A bare host:port is what a user types; it gets a scheme.
	c, err := New(Config{BaseURL: "127.0.0.1:3425"})
	if err != nil {
		t.Fatal(err)
	}
	if c.BaseURL() != "http://127.0.0.1:3425" {
		t.Fatalf("base = %s", c.BaseURL())
	}
	if _, err := New(Config{Token: "bad\ntoken"}); err == nil {
		t.Fatal("accepted a token with a control character")
	}
	if _, err := New(Config{Protocol: "gemini"}); err == nil {
		t.Fatal("accepted an unknown protocol")
	}
}
