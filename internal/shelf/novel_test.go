package shelf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNovelShelfAndProgressSurviveReopen(t *testing.T) {
	root := t.TempDir()
	legacy := `{"history":[{"source":"shared","id":"book","title":"Comic","chapterIndex":0}],"folders":[]}`
	if err := os.WriteFile(filepath.Join(root, "shelf.json"), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(root)
	if err != nil || len(s.Novels()) != 0 {
		t.Fatalf("open legacy shelf: %v", err)
	}
	book := Novel{Source: "shared", ID: "book", Title: "小说"}
	if err := s.KeepNovel(book, true); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordNovelProgress(book, "chapter-1", 2); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordNovelProgress(book, "chapter-2", 0); err != nil {
		t.Fatal(err)
	}
	other := Novel{Source: "other", ID: book.ID, Title: "Another edition"}
	if err := s.KeepNovel(other, true); err != nil {
		t.Fatal(err)
	}
	if err := s.KeepNovel(other, false); err != nil || len(s.Novels()) != 1 {
		t.Fatalf("remove untouched bookmark: %v", err)
	}
	if err := s.RecordNovelProgress(other, "other-chapter", 6); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	entries := reopened.Novels()
	if len(entries) != 2 || entries[0] != (NovelEntry{Novel: other, ChapterID: "other-chapter", Page: 6}) ||
		entries[1] != (NovelEntry{Novel: book, Kept: true, ChapterID: "chapter-2"}) {
		t.Fatalf("entries after reopen: %+v", entries)
	}
	if err := reopened.KeepNovel(book, false); err != nil {
		t.Fatal(err)
	}
	if got := reopened.Novels()[0]; got.Kept || got.ChapterID != "chapter-2" {
		t.Fatalf("unkeeping removed progress: %+v", got)
	}
	if got := reopened.History(1); len(got) != 1 || got[0].Title != "Comic" {
		t.Fatalf("novel operations changed comic history: %+v", got)
	}
}

func TestNovelProgressRollsBackOnWriteFailure(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	book := Novel{Source: "source", ID: "novel"}
	if err := s.RecordNovelProgress(book, "first", 3); err != nil {
		t.Fatal(err)
	}
	want := s.Novels()[0]
	previous := filepath.Join(s.Root(), "previous.json")
	if err := os.Rename(s.path(), previous); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(s.path(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordNovelProgress(book, "second", 0); err == nil {
		t.Fatal("failed write reported success")
	}
	if got := s.Novels()[0]; got != want {
		t.Fatalf("failed write changed memory: %+v", got)
	}
	if err := os.Remove(s.path()); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(previous, s.path()); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(s.Root())
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Novels()[0]; got != want {
		t.Fatalf("failed write changed disk: %+v", got)
	}
}
