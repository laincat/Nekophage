package servicecli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeService writes a service package into a temporary directory.
func writeService(t *testing.T, manifest, entry string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "service.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.mjs"), []byte(entry), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func manifestFor(permissions string) string {
	return `{"id":"demo","name":"Demo","version":"1","kind":"service.js","entry":"index.mjs","permissions":[` + permissions + `]}`
}

// A service package on disk runs, and its result comes back as JSON.
func TestCallRunsTheService(t *testing.T) {
	dir := writeService(t, manifestFor(""),
		"export function handle(method, payload) { return { method: method, payload: payload }; }")
	var out bytes.Buffer
	err := RunCommand(context.Background(), "call", []string{dir, "ping", `{"a":1}`}, t.TempDir(), &out, "Nekophage/test")
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output %q: %v", out.String(), err)
	}
	if got["method"] != "ping" || got["payload"] != `{"a":1}` {
		t.Fatalf("result = %v", got)
	}
}

// The network permission is what lets a service reach the network at all.
func TestCallGatesTheNetwork(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("remote-body"))
	}))
	defer server.Close()

	entry := "export function handle() { return { body: fetch('" + server.URL + "/x') }; }"

	denied := writeService(t, manifestFor(""), entry)
	var out bytes.Buffer
	err := RunCommand(context.Background(), "call", []string{denied, "get"}, t.TempDir(), &out, "Nekophage/test")
	if err == nil {
		t.Fatal("the network was reachable without the permission")
	}
	if !strings.Contains(err.Error(), "network") {
		t.Fatalf("error = %v", err)
	}

	allowed := writeService(t, manifestFor(`"network"`), entry)
	out.Reset()
	err = RunCommand(context.Background(), "call", []string{allowed, "get"}, t.TempDir(), &out, "Nekophage/test")
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if !strings.Contains(out.String(), "remote-body") {
		t.Fatalf("output = %s", out.String())
	}
}

// Storage is kept per service, in the data directory the command was given.
func TestCallKeepsStoragePerService(t *testing.T) {
	entry := `
export function handle(method) {
  if (method === 'set') { storageSet('token', 'abc'); return { stored: true }; }
  return { value: storageGet('token') };
}
`
	dir := writeService(t, manifestFor(`"storage"`), entry)
	data := t.TempDir()
	var out bytes.Buffer
	if err := RunCommand(context.Background(), "call", []string{dir, "set"}, data, &out, "Nekophage/test"); err != nil {
		t.Fatalf("set: %v", err)
	}
	out.Reset()
	if err := RunCommand(context.Background(), "call", []string{dir, "get"}, data, &out, "Nekophage/test"); err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(out.String(), "abc") {
		t.Fatalf("output = %s", out.String())
	}
	stored := filepath.Join(data, "services", "demo.json")
	if _, err := os.Stat(stored); err != nil {
		t.Fatalf("storage file: %v", err)
	}
}

// The command line has no account store, so a service cannot read secrets.
func TestCallRefusesSecrets(t *testing.T) {
	dir := writeService(t, manifestFor(`"secrets"`),
		"export function handle() { return { value: secret('token') }; }")
	var out bytes.Buffer
	err := RunCommand(context.Background(), "call", []string{dir, "get"}, t.TempDir(), &out, "Nekophage/test")
	if err == nil {
		t.Fatal("a secret was readable from the command line")
	}
	if !strings.Contains(err.Error(), "secret") {
		t.Fatalf("error = %v", err)
	}
}

func TestInfoPrintsTheManifest(t *testing.T) {
	dir := writeService(t, manifestFor(`"network"`), "export function handle() { return null; }")
	var out bytes.Buffer
	if err := RunCommand(context.Background(), "info", []string{dir}, t.TempDir(), &out, "Nekophage/test"); err != nil {
		t.Fatalf("info: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output %q: %v", out.String(), err)
	}
	if got["id"] != "demo" || got["kind"] != "service.js" {
		t.Fatalf("info = %v", got)
	}
}

func TestUnknownActionAndArgumentsAreRefused(t *testing.T) {
	var out bytes.Buffer
	if err := RunCommand(context.Background(), "nope", nil, t.TempDir(), &out, "Nekophage/test"); err == nil {
		t.Fatal("an unknown action was accepted")
	}
	if err := RunCommand(context.Background(), "call", []string{"only-one"}, t.TempDir(), &out, "Nekophage/test"); err == nil {
		t.Fatal("a call without a method was accepted")
	}
}
