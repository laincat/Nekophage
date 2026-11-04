package collections

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/store"
)

func document(id, version string) string {
	return fmt.Sprintf(`{"$schema":"veneracat-source/1-draft","id":%q,"name":"合集","version":%q,"entries":[{"id":"video","kind":"video-source","name":"视频","version":"1","status":"experimental","supports":{"platforms":["windows","macos"],"requiresCapabilities":["video-source.js"]},"artifact":{"url":"packages/source.js","sha256":"%s","bytes":0}},{"id":"future","kind":"settings-module","name":"设置","version":"1","status":"planned","supports":{"platforms":["ios"],"requiresCapabilities":["settings-module.v1"]}}]}`, id, version, strings.Repeat("0", 64))
}

func openTest(t *testing.T, root string) *Manager {
	t.Helper()
	s, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Open(s, core.NewTransport(nil, "collection-test", 5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Close)
	return m
}

func versionOf(t *testing.T, m *Manager) string {
	t.Helper()
	rows, err := m.List("windows", []string{"video-source.js"})
	if err != nil || len(rows) != 1 {
		t.Fatalf("list=%+v, err=%v", rows, err)
	}
	return rows[0].Version
}

func wait(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout: " + what)
	}
}

func TestFollowRestoreCompatibilityAndUnfollow(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		fmt.Fprint(w, document("publisher", "1"))
	}))
	defer server.Close()
	root := t.TempDir()
	m := openTest(t, root)
	// Subscribing neither installs plugins nor replaces existing plugin records.
	pluginFile := filepath.Join(root, "plugins.json")
	sentinel := []byte(`[{"key":"keep"}]`)
	if err := os.WriteFile(pluginFile, sentinel, 0600); err != nil {
		t.Fatal(err)
	}
	if err := m.Follow(context.Background(), server.URL+"/source.json#ignored"); err != nil {
		t.Fatal(err)
	}
	rows, err := m.List("windows", []string{"video-source.js"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || len(rows[0].Entries) != 2 || !rows[0].Entries[0].Compatible || rows[0].Entries[1].Compatible {
		t.Fatalf("rows=%+v", rows)
	}
	if got := rows[0].Entries[0].Artifact.URL; got != server.URL+"/packages/source.js" {
		t.Fatal(got)
	}
	if len(rows[0].Entries[1].Missing) != 2 {
		t.Fatalf("missing=%v", rows[0].Entries[1].Missing)
	}
	rows[0].Entries[0].Supports.Platforms[0] = "wrong"
	if again, _ := m.List("windows", []string{"video-source.js"}); !again[0].Entries[0].Compatible {
		t.Fatal("caller mutated stored snapshot")
	}
	m.Close()
	restored := openTest(t, root)
	if versionOf(t, restored) != "1" || requests.Load() != 1 {
		t.Fatal("restore fetched remotely or lost data")
	}
	other := openTest(t, t.TempDir())
	if rows, _ := other.List("windows", nil); len(rows) != 0 {
		t.Fatal("profile leaked")
	}
	if err := restored.Unfollow("publisher"); err != nil {
		t.Fatal(err)
	}
	reopened := openTest(t, root)
	if rows, _ := reopened.List("windows", nil); len(rows) != 0 {
		t.Fatal("unfollow not persisted")
	}
	if actual, _ := os.ReadFile(pluginFile); !bytes.Equal(actual, sentinel) {
		t.Fatal("subscription mutated installed plugins")
	}
}

func TestRefreshErrorsKeepPreviousSnapshot(t *testing.T) {
	var mode atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch mode.Load() {
		case 1:
			http.Error(w, "unavailable", 503)
		case 2:
			fmt.Fprint(w, `{"bad":true}`)
		case 3:
			fmt.Fprint(w, document("different-id", "2"))
		case 4:
			fmt.Fprint(w, strings.Repeat("x", MaxDocumentBytes+1))
		default:
			fmt.Fprint(w, document("publisher", "1"))
		}
	}))
	defer server.Close()
	root := t.TempDir()
	m := openTest(t, root)
	if err := m.Follow(context.Background(), server.URL); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(filepath.Join(root, "collections.json"))
	for _, value := range []int32{1, 2, 3, 4} {
		mode.Store(value)
		if err := m.Refresh(context.Background(), "publisher"); err == nil {
			t.Fatalf("accepted failure mode %d", value)
		}
		if versionOf(t, m) != "1" {
			t.Fatal("failed refresh changed memory")
		}
		after, _ := os.ReadFile(filepath.Join(root, "collections.json"))
		if !bytes.Equal(before, after) {
			t.Fatal("failed refresh changed disk")
		}
	}
	mode.Store(0)
	if err := m.Follow(context.Background(), server.URL+"/another-url"); err == nil {
		t.Fatal("second URL took over publisher identity")
	}
}

func TestWriteFailurePreservesMemoryAndDisk(t *testing.T) {
	var version atomic.Int32
	version.Store(1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, document("publisher", fmt.Sprint(version.Load())))
	}))
	defer server.Close()
	root := t.TempDir()
	m := openTest(t, root)
	if err := m.Follow(context.Background(), server.URL); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(filepath.Join(root, "collections.json"))
	version.Store(2)
	blocked := filepath.Join(root, "collections.json.tmp")
	if err := os.Mkdir(blocked, 0700); err != nil {
		t.Fatal(err)
	}
	if err := m.Refresh(context.Background(), "publisher"); err == nil {
		t.Fatal("ignored write failure")
	}
	if err := m.Unfollow("publisher"); err == nil {
		t.Fatal("ignored unfollow write failure")
	}
	if versionOf(t, m) != "1" {
		t.Fatal("failed persistence changed memory")
	}
	after, _ := os.ReadFile(filepath.Join(root, "collections.json"))
	if !bytes.Equal(before, after) {
		t.Fatal("failed persistence changed disk")
	}
	if err := os.Remove(blocked); err != nil {
		t.Fatal(err)
	}
	if err := m.Refresh(context.Background(), "publisher"); err != nil {
		t.Fatal(err)
	}
	if versionOf(t, m) != "2" {
		t.Fatal("could not retry after write failure")
	}
}

func TestReplacementCancelsConnectionAndKeepsNewSnapshot(t *testing.T) {
	var count atomic.Int32
	started := make(chan struct{})
	peerClosed := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := count.Add(1)
		if n == 2 {
			close(started)
			select {
			case <-r.Context().Done():
				close(peerClosed)
			case <-time.After(5 * time.Second):
			}
			return
		}
		fmt.Fprint(w, document("publisher", fmt.Sprint(n)))
	}))
	defer server.Close()
	m := openTest(t, t.TempDir())
	defer m.Close()
	if err := m.Follow(context.Background(), server.URL); err != nil {
		t.Fatal(err)
	}
	oldDone := make(chan error, 1)
	go func() { oldDone <- m.Refresh(context.Background(), "publisher") }()
	wait(t, started, "old refresh arrival")
	if err := m.Refresh(context.Background(), "publisher"); err != nil {
		t.Fatal(err)
	}
	wait(t, peerClosed, "old connection cancellation")
	select {
	case err := <-oldDone:
		if err == nil {
			t.Fatal("old request succeeded")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("old refresh hung")
	}
	if versionOf(t, m) != "3" {
		t.Fatal("old result overwrote newer snapshot")
	}
}

func TestCancellationUnfollowAndCloseStopRefresh(t *testing.T) {
	for _, action := range []string{"cancel", "unfollow", "close"} {
		t.Run(action, func(t *testing.T) {
			var count atomic.Int32
			started := make(chan struct{})
			peerClosed := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if count.Add(1) == 1 {
					fmt.Fprint(w, document("publisher", "1"))
					return
				}
				close(started)
				select {
				case <-r.Context().Done():
					close(peerClosed)
				case <-time.After(5 * time.Second):
				}
			}))
			defer server.Close()
			root := t.TempDir()
			m := openTest(t, root)
			defer m.Close()
			if err := m.Follow(context.Background(), server.URL); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- m.Refresh(ctx, "publisher") }()
			wait(t, started, "refresh arrival")
			switch action {
			case "cancel":
				cancel()
			case "unfollow":
				if err := m.Unfollow("publisher"); err != nil {
					t.Fatal(err)
				}
			case "close":
				m.Close()
			}
			wait(t, peerClosed, "refresh connection cancellation")
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("cancelled refresh succeeded")
				}
			case <-time.After(3 * time.Second):
				t.Fatal("refresh hung")
			}
			restored := openTest(t, root)
			if action == "unfollow" {
				if rows, _ := restored.List("windows", nil); len(rows) != 0 {
					t.Fatal("late refresh resurrected subscription")
				}
			} else if versionOf(t, restored) != "1" {
				t.Fatal("cancellation changed snapshot")
			}
		})
	}
}

func TestPrecancelAndInvalidSavedDataNeverOverwrite(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		fmt.Fprint(w, document("publisher", "1"))
	}))
	defer server.Close()
	root := t.TempDir()
	m := openTest(t, root)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := m.Follow(ctx, server.URL); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
	if requests.Load() != 0 {
		t.Fatal("pre-cancelled request reached server")
	}
	m.Close()
	for _, data := range []string{"", `not json`, `{"version":2,"subscriptions":[]}`} {
		p := filepath.Join(root, "collections.json")
		if err := os.WriteFile(p, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		s, err := store.Open(root)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Open(s, core.NewTransport(nil, "test", 0)); err == nil {
			t.Fatal("accepted invalid saved data")
		}
		after, _ := os.ReadFile(p)
		if string(after) != data {
			t.Fatal("invalid data overwritten")
		}
	}
}

func TestSnapshotRestoresHTMLWithinDownloadLimit(t *testing.T) {
	// Do not expand literal HTML into escapes that exceed the download limit.
	body := strings.Replace(document("publisher", "1"), `"entries":`, `"description":"`+strings.Repeat("<", MaxDocumentBytes/3)+`","entries":`, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
	defer server.Close()
	root := t.TempDir()
	m := openTest(t, root)
	if err := m.Follow(context.Background(), server.URL); err != nil {
		t.Fatal(err)
	}
	m.Close()
	restored := openTest(t, root)
	if versionOf(t, restored) != "1" {
		t.Fatal("HTML document did not round trip")
	}
}

func TestRestoreRejectsConflictingSavedIdentities(t *testing.T) {
	root := t.TempDir()
	s, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	rec := record{ID: "publisher", URL: "https://example.test/source.json", Document: json.RawMessage(document("publisher", "1"))}
	data, _ := json.Marshal(disk{Version: 1, Subscriptions: []record{rec, rec}})
	if err = s.SaveCollections(data); err != nil {
		t.Fatal(err)
	}
	if _, err = Open(s, core.NewTransport(nil, "test", 0)); err == nil {
		t.Fatal("duplicate snapshot accepted")
	}
}
