package magpie

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// A gateway that ends with the sentinel and no finish_reason is common; the
// stream must still be reported as complete, or every reply would look
// interrupted.
func TestDoneSentinelAloneCompletesTheStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: " + `{"choices":[{"delta":{"content":"x"}}]}` + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	events, err := client(t, server, Chat).Chat(context.Background(), Request{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	sawDone := false
	for event := range events {
		switch event.Kind {
		case EventDone:
			sawDone = true
		case EventError:
			t.Fatalf("stream failed: %s", event.Text)
		}
	}
	if !sawDone {
		t.Fatal("a stream ending with [DONE] was reported as interrupted")
	}
}

// A stream that stops without any completion marker is an interrupted answer,
// not a short one.
func TestAStreamWithoutCompletionIsAnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: " + `{"choices":[{"delta":{"content":"x"}}]}` + "\n\n"))
	}))
	defer server.Close()

	events, err := client(t, server, Chat).Chat(context.Background(), Request{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	sawError := false
	for event := range events {
		if event.Kind == EventError {
			sawError = true
		}
	}
	if !sawError {
		t.Fatal("an unterminated stream was reported as complete")
	}
}

// A provider error inside a 200 stream is shown as an error, not as text and
// not as a clean ending.
func TestAnErrorInsideTheStreamIsReported(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: " + `{"error":{"message":"quota exhausted"}}` + "\n\n"))
	}))
	defer server.Close()

	events, err := client(t, server, Chat).Chat(context.Background(), Request{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var seen string
	for event := range events {
		if event.Kind == EventError {
			seen = event.Text
		}
	}
	if seen != "quota exhausted" {
		t.Fatalf("error text = %q", seen)
	}
}
