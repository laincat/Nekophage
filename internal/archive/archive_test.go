package archive

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// venera writes a small archive shaped like the official export: a source
// script, its data, and the app data the settings screen writes.
func venera(t *testing.T, extra ...[2]string) string {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	entries := [][2]string{
		{"appdata.json", `{"settings":{"theme":"dark"},"searchHistory":["海贼王","one piece"]}`},
		{"comic_source/wnacg.js", `class W extends ComicSource { name = 'W'; key = 'wnacg'; version = '1' }`},
		{"comic_source/wnacg.data", `{"token":"t","n":12345678901234567890,"z":1}`},
		{"comic_source/copy_manga.data", `{"account":["u","p"]}`},
	}
	entries = append(entries, extra...)
	for _, e := range entries {
		fw, err := w.Create(e[0])
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write([]byte(e[1])); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "backup.venera")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadSummarizeAndSplitData(t *testing.T) {
	path := venera(t)
	b, err := Read(context.Background(), path, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := b.Summarize()
	if s.Sources != 1 {
		t.Errorf("sources = %d, want 1 (only wnacg carries a script)", s.Sources)
	}
	if s.WithData != 2 {
		t.Errorf("sources with data = %d, want 2", s.WithData)
	}
	if s.SearchHistory != 2 || s.Settings != 1 {
		t.Errorf("app data = %d search, %d settings", s.SearchHistory, s.Settings)
	}
	if s.Importable != 1 || s.RestoredData != 2 {
		t.Errorf("importable = %d, data = %d", s.Importable, s.RestoredData)
	}

	data, err := b.SourceData()
	if err != nil {
		t.Fatal(err)
	}
	byKey := map[string]Data{}
	for _, d := range data {
		byKey[d.Key] = d
	}
	wnacg, ok := byKey["wnacg"]
	if !ok {
		t.Fatalf("no wnacg data: %+v", data)
	}
	// The keys keep the archive's order and the value keeps its digits.
	want := []string{"token", "n", "z"}
	if len(wnacg.Fields) != len(want) {
		t.Fatalf("fields = %+v", wnacg.Fields)
	}
	for i, name := range want {
		if wnacg.Fields[i].Name != name {
			t.Errorf("field %d = %q, want %q", i, wnacg.Fields[i].Name, name)
		}
	}
	if string(wnacg.Fields[1].Value) != "12345678901234567890" {
		t.Errorf("n = %s", wnacg.Fields[1].Value)
	}
	// A source with data but no script is still planned, for its data.
	if _, ok := byKey["copy_manga"]; !ok {
		t.Error("copy_manga data missing")
	}
}

func TestReadRejectsGarbage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-backup.venera")
	if err := os.WriteFile(path, []byte("this is not a zip"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(context.Background(), path, nil); err == nil {
		t.Fatal("garbage accepted")
	}
	if _, err := Read(context.Background(), filepath.Join(t.TempDir(), "missing.venera"), nil); err == nil {
		t.Fatal("missing file accepted")
	}
}

func TestFieldsKeepsOrderAndRejectsNonObjects(t *testing.T) {
	got, err := fields(json.RawMessage(`{"b":1,"a":{"x":2},"c":[1,2]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Name != "b" || got[1].Name != "a" || got[2].Name != "c" {
		t.Fatalf("fields = %+v", got)
	}
	if _, err := fields(json.RawMessage(`[1,2]`)); err == nil {
		t.Error("array accepted")
	}
	if _, err := fields(json.RawMessage(`null`)); err == nil {
		t.Error("null accepted")
	}
}
