package probesite

import (
	"bytes"
	"image/png"
	"io"
	"net/http"
	"testing"
	"time"
)

// The served page is what makes an offline reader check possible, so it has to be
// a real PNG and not just any bytes: the reader decodes it, and a wrong
// content-type or a truncated body would read as a client that cannot render.
func TestServedPageIsAPNG(t *testing.T) {
	stop, err := ServeOnePage(18799)
	if err != nil {
		t.Fatalf("serve: %v", err)
	}
	defer stop()

	client := &http.Client{Timeout: 5 * time.Second}
	var resp *http.Response
	// The listener is started in a goroutine, so the first attempt may lose the race.
	for i := 0; i < 20; i++ {
		resp, err = client.Get("http://127.0.0.1:18799/page.png")
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "image/png" {
		t.Errorf("content-type = %q, want image/png", got)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(body) < 8 || string(body[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("body is not a PNG: %d bytes, starts %q", len(body), string(body[:min(8, len(body))]))
	}
	// The page has to be taller than a window, or a reading-position check can only
	// ever report zero.
	cfg, err := png.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if cfg.Height <= 800 || cfg.Width < 200 {
		t.Fatalf("page is %dx%d, want a scrollable page", cfg.Width, cfg.Height)
	}
}
