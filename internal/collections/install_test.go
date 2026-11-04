package collections

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/store"
)

const probeScript = `class Probe extends ComicSource {
  name = 'Probe'; key = 'probe'; version = '1'
  search = {load: async () => ({comics: [], maxPage: 1})}
}
`

func probeCollection(t *testing.T, base string, body []byte, declared []byte, key string) *Manager {
	t.Helper()
	sum := sha256.Sum256(declared)
	doc := map[string]any{
		"$schema": "veneracat-source/1-draft",
		"id":      "probe-collection",
		"name":    "Probe",
		"version": "1",
		"entries": []map[string]any{{
			"id": "probe", "kind": "comic-source.js", "name": "Probe", "version": "1",
			"status": "active", "key": key,
			"supports": map[string]any{"platforms": []string{"windows"}, "requiresCapabilities": []string{"comic-source.js"}},
			"artifact": map[string]any{"url": base + "/probe.js", "sha256": hex.EncodeToString(sum[:]), "bytes": len(declared)},
		}},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m, err := Open(s, core.NewTransport(nil, "Nekophage/test", 0))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Close)
	if err := m.installForTest(raw, base+"/source.json"); err != nil {
		t.Fatal(err)
	}
	return m
}

// A digest that does not describe the served bytes is refused before anything
// is written: the artifact identity is the only thing this client can check.
func TestArtifactRefusesBytesThatDoNotMatchTheDigest(t *testing.T) {
	served := []byte(probeScript + "// extra\n")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(served)
	}))
	defer server.Close()

	m := probeCollection(t, server.URL, served, []byte(probeScript), "probe")
	entry, err := m.Entry("probe-collection", "probe", "windows", []string{"comic-source.js"})
	if err != nil {
		t.Fatalf("entry: %v", err)
	}
	if _, err := m.Artifact(context.Background(), entry); err == nil {
		t.Fatal("bytes that do not match the declared digest were accepted")
	}
}

// The declared key travels with the entry so a caller can compare it with what
// the script actually registers.
func TestEntryCarriesTheDeclaredKeyAndCompatibility(t *testing.T) {
	served := []byte(probeScript)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(served)
	}))
	defer server.Close()

	m := probeCollection(t, server.URL, served, served, "probe")
	entry, err := m.Entry("probe-collection", "probe", "windows", []string{"comic-source.js"})
	if err != nil {
		t.Fatalf("entry: %v", err)
	}
	if entry.Key != "probe" || !entry.Compatible {
		t.Fatalf("entry = %+v", entry)
	}
	ios, err := m.Entry("probe-collection", "probe", "ios", []string{"comic-source.js"})
	if err != nil {
		t.Fatalf("ios entry: %v", err)
	}
	if ios.Compatible {
		t.Fatalf("an entry limited to windows reported compatible on ios: %+v", ios)
	}
	if _, err := m.Entry("probe-collection", "missing", "windows", nil); err == nil {
		t.Fatal("an unknown entry id was accepted")
	}
	if _, err := m.Entry("no-such-collection", "probe", "windows", nil); err == nil {
		t.Fatal("an unknown collection id was accepted")
	}
}

// installForTest seeds the manager with one collection document, standing in for
// a followed subscription so the tests do not need the network twice.
func (m *Manager) installForTest(document []byte, url string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	next := m.copyRecords()
	var c struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(document, &c); err != nil {
		return err
	}
	next[c.ID] = record{ID: c.ID, URL: url, Document: document}
	return m.publish(next)
}
