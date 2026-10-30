package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/laincat/nekophage/internal/core"
)

func TestFilterChapters(t *testing.T) {
	chapters := []core.Chapter{
		{ID: "a", Title: "1: Romance dawn", Group: "Volume 1"},
		{ID: "b", Title: "2: 冒險開始", Group: "Volume 1"},
		{ID: "c", Title: "Special", Group: "番外"},
	}
	// No query keeps every chapter, and the indices are the reader's.
	idx, found := filterChapters(chapters, "")
	if len(idx) != 3 || len(found) != 3 || idx[2] != 2 {
		t.Fatalf("all = %v %v", idx, found)
	}
	// A title matches, case-insensitively.
	idx, found = filterChapters(chapters, "ROMANCE")
	if len(idx) != 1 || idx[0] != 0 || found[0].ID != "a" {
		t.Fatalf("title = %v %v", idx, found)
	}
	// Spaces around the query do not matter, and a group matches too.
	idx, _ = filterChapters(chapters, "  番外 ")
	if len(idx) != 1 || idx[0] != 2 {
		t.Fatalf("group = %v", idx)
	}
	// Nothing matches: no indices, so the view can say so.
	if idx, found := filterChapters(chapters, "zzz"); len(idx) != 0 || len(found) != 0 {
		t.Fatalf("none = %v %v", idx, found)
	}
	// The indices point into the whole list, not the filtered one, so a
	// chapter opened from a filter is the one the row names.
	idx, _ = filterChapters(chapters, "冒險")
	if len(idx) != 1 || chapters[idx[0]].ID != "b" {
		t.Fatalf("index = %v", idx)
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ui.json")
	// Nothing saved yet: the window starts empty rather than failing.
	if _, ok := readSettings(path); ok {
		t.Fatal("a missing file read as saved settings")
	}
	writeSettings(path, settings{Source: "manga_dex", Query: "one piece"})
	got, ok := readSettings(path)
	if !ok || got.Source != "manga_dex" || got.Query != "one piece" {
		t.Fatalf("settings = %+v %v", got, ok)
	}
	// A damaged file is ignored, not fatal.
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := readSettings(path); ok {
		t.Fatal("a damaged file read as saved settings")
	}
}
