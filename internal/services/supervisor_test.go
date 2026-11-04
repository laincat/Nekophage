package services

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/laincat/veneracat-core/service"
)

// installLoopingService installs a service whose method never returns.
func installLoopingService(t *testing.T, data string) {
	t.Helper()
	manifest := `{"id":"loop","name":"Loop","version":"1","kind":"service.js","entry":"index.mjs","permissions":[]}`
	entry := "export function handle() { while (true) {} }"
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, body := range map[string]string{"service.json": manifest, "index.mjs": entry} {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(context.Background(), data, "test", "", buf.Bytes()); err != nil {
		t.Fatal(err)
	}
}

// Cancelling through the desktop supervisor has to stop the script, not just
// the wait: the window relies on this to leave a stuck method.
func TestSupervisorCancelsAStuckMethod(t *testing.T) {
	data := t.TempDir()
	installLoopingService(t, data)
	s, err := NewSupervisor(data, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := s.Call(ctx, "loop", "go", nil)
		done <- err
	}()
	time.Sleep(150 * time.Millisecond)
	start := time.Now()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v", err)
		}
		if elapsed := time.Since(start); elapsed > 3*time.Second {
			t.Fatalf("took %s to stop", elapsed)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the cancelled call never returned")
	}
}

// A supervisor call returns the service's JSON, and a describe lists methods.
func TestSupervisorRunsAndDescribes(t *testing.T) {
	data := t.TempDir()
	manifest := `{"id":"demo","name":"Demo","version":"1","kind":"service.js","entry":"index.mjs","permissions":[]}`
	entry := `export function describe() { return { methods: [{ name: "echo", title: "回显" }] }; }
export function handle() { return { ok: true }; }`
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, body := range map[string]string{"service.json": manifest, "index.mjs": entry} {
		f, _ := w.Create(name)
		f.Write([]byte(body))
	}
	w.Close()
	if _, err := Install(context.Background(), data, "test", "", buf.Bytes()); err != nil {
		t.Fatal(err)
	}
	s, err := NewSupervisor(data, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	methods, err := s.Describe(context.Background(), "demo")
	if err != nil || len(methods) != 1 || methods[0].Name != "echo" {
		t.Fatalf("describe = %+v (%v)", methods, err)
	}
	out, err := s.Call(context.Background(), "demo", "echo", json.RawMessage(`{"a":1}`))
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if !strings.Contains(string(out), `"ok":true`) {
		t.Fatalf("result = %s", out)
	}
	// The same runtime served both, so the service was compiled once.
	if got := s.Loaded(); len(got) != 1 || got[0] != "demo" {
		t.Fatalf("loaded = %v", got)
	}
	var _ = service.PermissionNetwork
}
