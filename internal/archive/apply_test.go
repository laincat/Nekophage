package archive

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/searches"
	"github.com/laincat/nekophage/internal/shelf"
	"github.com/laincat/veneracat-core/backup"
	"github.com/laincat/veneracat-core/gohost/library"
)

func TestApplyPreservesSuccessesAndRestoresCookiesThroughHTTP(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	c, err := core.Open(core.Options{DataDir: root})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	lib := library.New(c)
	sh, err := shelf.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	history, err := searches.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, r.Header.Get("Cookie")) }))
	defer server.Close()
	source := fmt.Sprintf("class Restore extends ComicSource {name='restore';key='restored';version='1';search={load:async()=>({comics:[{id:(await Network.get(%q)).body,title:'cookie'}],maxPage:1})}}", server.URL)
	b := &Backup{Content: &backup.Content{SearchHistory: []string{"fixture"}}, Plan: &backup.Plan{
		Sources:       []backup.SourceRestore{{Key: "broken", Script: "invalid source"}, {Key: "restored", Script: source, Data: []byte("{\"number\":9007199254740993}")}},
		History:       []backup.HistoryItem{{SourceKey: "restored", ID: "comic", Title: "fixture", TimeMillis: 5_000, Episode: 1, Page: 4}},
		Cookies:       []backup.Cookie{{Name: "sid", Value: "restored", Domain: "127.0.0.1", Path: "/"}},
		CookieTargets: []string{"broken", "restored"},
	}}
	for index := range 2 {
		result, err := Apply(ctx, b, lib, sh, history)
		if err == nil || !strings.Contains(err.Error(), "broken") || result.Sources != 1 || result.Fields != 1 || result.Cookies != 1 {
			t.Fatalf("apply %d: %+v, %v", index, result, err)
		}
		if result.History != 1-index || result.Searches != 1-index {
			t.Fatalf("double apply repeated history: %+v", result)
		}
	}
	result, err := c.Search(ctx, "restored", "", nil, 1)
	if err != nil || len(result.Comics) != 1 || result.Comics[0].ID != "sid=restored" {
		t.Fatalf("source HTTP got %+v (%v)", result, err)
	}
	if got := c.Jar().Header(core.ComicPartition("broken"), server.URL); got != "" {
		t.Fatalf("failed script received cookies: %s", got)
	}
	if raw, _, err := c.Store().LoadData("restored", "number"); err != nil || string(raw) != "9007199254740993" {
		t.Fatalf("backup value = %s, %v", raw, err)
	}
}

func TestExportFailureDoesNotLeaveATemporaryArchive(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "profile")
	c, err := core.Open(core.Options{DataDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	destination := filepath.Join(root, "blocked.venera")
	if err := os.Mkdir(destination, 0700); err != nil {
		t.Fatal(err)
	}
	if err := Export(context.Background(), destination, c, nil); err == nil {
		t.Fatal("export wrote over an occupied directory")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("unexpected residue: %v", entries)
	}
}
