package shelf

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProgressWritesOnlyChangesAndRollsBackOnFailure(t *testing.T) {
	root := t.TempDir()
	shelf, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	position := History{Comic: Comic{Source: "src", ID: "comic"}, Chapter: "One", ChapterIndex: 0, Page: 4, Percent: 12, Time: 10}
	if err := shelf.Record(position); err != nil {
		t.Fatal(err)
	}
	marker := time.Unix(1_600_000_000, 0)
	if err := os.Chtimes(shelf.path(), marker, marker); err != nil {
		t.Fatal(err)
	}
	if err := shelf.BeginChapter(History{Comic: position.Comic, Chapter: position.Chapter, ChapterIndex: position.ChapterIndex}); err != nil {
		t.Fatal(err)
	}
	if changed, err := shelf.UpdateProgress("src", "comic", "One", 0, 4, 12); err != nil || changed {
		t.Fatalf("same progress changed: %v %v", changed, err)
	}
	file, err := os.Stat(shelf.path())
	if err != nil {
		t.Fatal(err)
	}
	if !file.ModTime().Equal(marker) {
		t.Fatal("same progress rewrote the shelf")
	}
	blocked := shelf.path()
	saved := filepath.Join(root, "previous.json")
	if err := os.Rename(blocked, saved); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(blocked, 0700); err != nil {
		t.Fatal(err)
	}
	if changed, err := shelf.UpdateProgress("src", "comic", "One", 0, 5, 15); err == nil || changed {
		t.Fatalf("write failure: %v %v", changed, err)
	}
	if got := shelf.History(1)[0]; got != position {
		t.Fatalf("failed write changed memory: %+v", got)
	}
	if err := os.Remove(blocked); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(saved, blocked); err != nil {
		t.Fatal(err)
	}
	if changed, err := shelf.UpdateProgress("src", "comic", "One", 0, 5, 15); err != nil || !changed {
		t.Fatalf("retry: %v %v", changed, err)
	}
	loaded, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.History(1)[0]; got.Page != 5 || got.Percent != 15 {
		t.Fatalf("last progress was lost: %+v", got)
	}
}
