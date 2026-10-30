package shelf

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func open(t *testing.T, root string) *Shelf {
	t.Helper()
	s, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestRecordKeepsOneEntryPerComic(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	a := Comic{Source: "copy_manga", ID: "1", Title: "A"}
	b := Comic{Source: "copy_manga", ID: "2", Title: "B"}
	if err := s.Record(History{Comic: a, Chapter: "第 1 话", ChapterIndex: 0, Time: 100}); err != nil {
		t.Fatal(err)
	}
	if err := s.Record(History{Comic: b, Chapter: "第 1 话", ChapterIndex: 0, Time: 200}); err != nil {
		t.Fatal(err)
	}
	// Reading A again replaces its entry and moves it to the front.
	if err := s.Record(History{Comic: a, Chapter: "第 9 话", ChapterIndex: 8, Time: 300}); err != nil {
		t.Fatal(err)
	}
	got := s.History(0)
	if len(got) != 2 {
		t.Fatalf("history = %+v", got)
	}
	if got[0].ID != "1" || got[0].Chapter != "第 9 话" || got[0].Time != 300 {
		t.Errorf("front = %+v", got[0])
	}
	if got[1].ID != "2" {
		t.Errorf("second = %+v", got[1])
	}
}

func TestRecordNeedsIdentityAndTime(t *testing.T) {
	s := open(t, t.TempDir())
	if err := s.Record(History{Comic: Comic{Source: "x"}}); err == nil {
		t.Error("entry without an id accepted")
	}
	if err := s.Record(History{Comic: Comic{Source: "x", ID: "1"}}); err != nil {
		t.Fatal(err)
	}
	if h := s.History(1); len(h) != 1 || h[0].Time == 0 {
		t.Errorf("time not filled: %+v", h)
	}
}

func TestHistoryIsCappedAndTrimmedNewestFirst(t *testing.T) {
	s := open(t, t.TempDir())
	for i := 0; i < historyLimit+20; i++ {
		c := Comic{Source: "s", ID: string(rune('a'+i%26)) + string(rune('a'+i/26))}
		if err := s.Record(History{Comic: c, Time: int64(i + 1)}); err != nil {
			t.Fatal(err)
		}
	}
	h := s.History(0)
	if len(h) != historyLimit {
		t.Fatalf("len = %d, want %d", len(h), historyLimit)
	}
	if h[0].Time != int64(historyLimit+20) {
		t.Errorf("front time = %d", h[0].Time)
	}
	if h[len(h)-1].Time == 1 {
		t.Error("oldest entries were kept")
	}
}

func TestAddRemoveAndHolds(t *testing.T) {
	s := open(t, t.TempDir())
	c := Comic{Source: "manga_dex", ID: "abc", Title: "One Piece"}
	added, err := s.Add("", c)
	if err != nil || !added {
		t.Fatalf("add = %v, %v", added, err)
	}
	if again, err := s.Add(DefaultFolder, c); err != nil || again {
		t.Errorf("second add = %v, %v; want false", again, err)
	}
	if folder, ok := s.Holds(c.Source, c.ID); !ok || folder != DefaultFolder {
		t.Errorf("holds = %q, %v", folder, ok)
	}
	if _, favourites := s.Counts(); favourites != 1 {
		t.Errorf("favourites = %d", favourites)
	}
	if err := s.Remove("", c.Source, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Holds(c.Source, c.ID); ok {
		t.Error("still held after remove")
	}
	// Removing something absent is not an error.
	if err := s.Remove(DefaultFolder, "nope", "nope"); err != nil {
		t.Error(err)
	}
	if err := s.Remove("missing folder", c.Source, c.ID); err != nil {
		t.Error(err)
	}
}

func TestFoldersAreCopies(t *testing.T) {
	s := open(t, t.TempDir())
	if _, err := s.Add("稍后读", Comic{Source: "s", ID: "1", Title: "t"}); err != nil {
		t.Fatal(err)
	}
	got := s.Folders()
	got[0].Items[0].Title = "changed"
	if s.Folders()[0].Items[0].Title != "t" {
		t.Error("caller could edit the shelf through a copied folder")
	}
}

func TestRecordsSurviveReopening(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	if err := s.Record(History{Comic: Comic{Source: "s", ID: "1", Title: "t"}, Chapter: "c", Time: 7}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Add("", Comic{Source: "s", ID: "2", Title: "u"}); err != nil {
		t.Fatal(err)
	}
	again := open(t, root)
	h := again.History(0)
	if len(h) != 1 || h[0].Chapter != "c" || h[0].Time != 7 {
		t.Fatalf("history after reopen = %+v", h)
	}
	if _, favourites := again.Counts(); favourites != 1 {
		t.Errorf("favourites after reopen = %d", favourites)
	}
	// The file is the only copy, so it must not be world readable. Windows
	// reports 0666 for any writable file, so the mode is only checked where
	// it means something.
	if runtime.GOOS != "windows" {
		st, err := os.Stat(filepath.Join(root, "shelf.json"))
		if err != nil {
			t.Fatal(err)
		}
		if perm := st.Mode().Perm(); perm != 0o600 {
			t.Errorf("shelf.json mode = %o", perm)
		}
	}
}

func TestOpenRejectsBrokenFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "shelf.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(root); err == nil {
		t.Fatal("broken shelf.json accepted")
	}
	if _, err := Open(""); err == nil {
		t.Fatal("empty root accepted")
	}
}

func TestForgetHistory(t *testing.T) {
	s := open(t, t.TempDir())
	for _, id := range []string{"1", "2"} {
		if err := s.Record(History{Comic: Comic{Source: "s", ID: id}, Time: 1}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.ForgetHistory("s", "1"); err != nil {
		t.Fatal(err)
	}
	if h := s.History(0); len(h) != 1 || h[0].ID != "2" {
		t.Errorf("history = %+v", h)
	}
}

func TestCreateRenameAndDeleteFolder(t *testing.T) {
	s := open(t, t.TempDir())
	if err := s.CreateFolder(" 稍后读 "); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateFolder("稍后读"); err != nil {
		t.Fatal(err)
	}
	folders := s.Folders()
	if len(folders) != 1 || folders[0].Name != "稍后读" {
		t.Fatalf("folders = %+v", folders)
	}
	if err := s.CreateFolder("  "); err == nil {
		t.Error("blank name accepted")
	}

	if _, err := s.Add("稍后读", Comic{Source: "s", ID: "1", Title: "t"}); err != nil {
		t.Fatal(err)
	}
	if err := s.RenameFolder("稍后读", "以后看"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Holds("s", "1"); !ok {
		t.Error("comic lost on rename")
	}
	if folder, _ := s.Holds("s", "1"); folder != "以后看" {
		t.Errorf("folder after rename = %q", folder)
	}
	if err := s.RenameFolder("以后看", "以后看"); err != nil {
		t.Errorf("renaming to itself: %v", err)
	}
	if err := s.CreateFolder("新夹"); err != nil {
		t.Fatal(err)
	}
	if err := s.RenameFolder("以后看", "新夹"); err == nil {
		t.Error("rename onto an existing folder accepted")
	}
	if err := s.RenameFolder("没有的夹", "x"); err == nil {
		t.Error("renaming a missing folder accepted")
	}

	if err := s.DeleteFolder("以后看"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Holds("s", "1"); ok {
		t.Error("comic still held after deleting its folder")
	}
	if err := s.DeleteFolder("以后看"); err == nil {
		t.Error("deleting a missing folder accepted")
	}
}

func TestMoveBetweenFolders(t *testing.T) {
	s := open(t, t.TempDir())
	c := Comic{Source: "s", ID: "1", Title: "t", Author: "a", Cover: "c"}
	if _, err := s.Add("甲", c); err != nil {
		t.Fatal(err)
	}
	moved, err := s.Move(c, "乙")
	if err != nil || !moved {
		t.Fatalf("move = %v, %v", moved, err)
	}
	if folder, _ := s.Holds("s", "1"); folder != "乙" {
		t.Errorf("folder = %q", folder)
	}
	kept := s.Folders()
	for _, f := range kept {
		if f.Name == "甲" && len(f.Items) != 0 {
			t.Errorf("甲 still holds %+v", f.Items)
		}
		if f.Name == "乙" && (len(f.Items) != 1 || f.Items[0].Author != "a") {
			t.Errorf("乙 = %+v", f.Items)
		}
	}
	// Moving to the folder it is already in changes nothing.
	if again, err := s.Move(c, "乙"); err != nil || again {
		t.Errorf("same-folder move = %v, %v", again, err)
	}
	// Moving something not on the shelf adds it.
	if added, err := s.Move(Comic{Source: "s", ID: "2", Title: "u"}, ""); err != nil || !added {
		t.Errorf("new move = %v, %v", added, err)
	}
	if _, err := s.Move(Comic{Source: "s"}, "甲"); err == nil {
		t.Error("move without an id accepted")
	}
}

func TestUpdateProgressKeepsTheEntryInPlace(t *testing.T) {
	s := open(t, t.TempDir())
	old := Comic{Source: "s", ID: "old", Title: "旧"}
	newer := Comic{Source: "s", ID: "new", Title: "新"}
	for _, c := range []Comic{old, newer} {
		if err := s.Record(History{Comic: c, Chapter: "第 1 话", Time: 10}); err != nil {
			t.Fatal(err)
		}
	}
	ok, err := s.UpdateProgress("s", "old", "第 5 话", 4, 12, 37)
	if err != nil || !ok {
		t.Fatalf("update = %v, %v", ok, err)
	}
	got := s.History(0)
	if len(got) != 2 || got[0].ID != "new" {
		t.Fatalf("order changed: %+v", got)
	}
	if e := got[1]; e.Chapter != "第 5 话" || e.ChapterIndex != 4 || e.Page != 12 || e.Percent != 37 {
		t.Errorf("entry = %+v", e)
	}
	if got[1].Time <= 10 {
		t.Errorf("time not refreshed: %d", got[1].Time)
	}
	// Values outside the range are clamped, not trusted.
	if _, err := s.UpdateProgress("s", "old", "", -1, -3, 250); err != nil {
		t.Fatal(err)
	}
	if e := s.History(0)[1]; e.Percent != 100 || e.Page != 0 {
		t.Errorf("clamped entry = %+v", e)
	}
	// A comic that is not on the shelf is not added.
	if ok, err := s.UpdateProgress("s", "absent", "", 0, 1, 1); err != nil || ok {
		t.Errorf("absent update = %v, %v", ok, err)
	}
	if h, _ := s.Counts(); h != 2 {
		t.Errorf("history = %d", h)
	}
	if _, err := s.UpdateProgress("", "x", "", 0, 0, 0); err == nil {
		t.Error("update without a source accepted")
	}
}

func TestKeptPagesNewestFirstAndUnique(t *testing.T) {
	s := open(t, t.TempDir())
	first := ImageFavourite{Source: "s", ComicID: "c", ChapterID: "ch1", ChapterIndex: 0,
		ChapterName: "第 1 话", Page: 3, URL: "https://img/1.jpg", Time: 10}
	if err := s.KeepPage(first); err != nil {
		t.Fatal(err)
	}
	second := ImageFavourite{Source: "s", ComicID: "c", ChapterID: "ch1", ChapterIndex: 0,
		Page: 9, URL: "https://img/2.jpg", Time: 20}
	if err := s.KeepPage(second); err != nil {
		t.Fatal(err)
	}
	// Keeping the same page again moves it to the front without duplicating.
	first.Time = 30
	if err := s.KeepPage(first); err != nil {
		t.Fatal(err)
	}
	pages := s.Pages(0)
	if len(pages) != 2 {
		t.Fatalf("pages = %+v", pages)
	}
	if pages[0].URL != "https://img/1.jpg" || pages[0].Time != 30 {
		t.Errorf("front = %+v", pages[0])
	}
	if err := s.ForgetPage("s", "c", "https://img/1.jpg"); err != nil {
		t.Fatal(err)
	}
	if got := s.Pages(0); len(got) != 1 || got[0].URL != "https://img/2.jpg" {
		t.Errorf("after forget = %+v", got)
	}
	if err := s.KeepPage(ImageFavourite{Source: "s"}); err == nil {
		t.Error("a page without a comic accepted")
	}
}

func TestKeptPagesSurviveReopening(t *testing.T) {
	root := t.TempDir()
	s := open(t, root)
	if err := s.KeepPage(ImageFavourite{Source: "s", ComicID: "c", ChapterID: "ch",
		Page: 2, URL: "https://img/x.jpg", Time: 5}); err != nil {
		t.Fatal(err)
	}
	again := open(t, root)
	if got := again.Pages(0); len(got) != 1 || got[0].URL != "https://img/x.jpg" {
		t.Fatalf("after reopen = %+v", got)
	}
	// A shelf without kept pages still reads back.
	other := open(t, t.TempDir())
	if got := other.Pages(0); len(got) != 0 {
		t.Errorf("empty shelf = %+v", got)
	}
}

func TestAnOlderCopyDoesNotReplaceANewerOne(t *testing.T) {
	s := open(t, t.TempDir())
	newer := ImageFavourite{Source: "s", ComicID: "c", ChapterID: "ch", Page: 1,
		URL: "https://img/x.jpg", Time: 50}
	if err := s.KeepPage(newer); err != nil {
		t.Fatal(err)
	}
	older := newer
	older.Time = 10
	if err := s.KeepPage(older); err != nil {
		t.Fatal(err)
	}
	if got := s.Pages(0); len(got) != 1 || got[0].Time != 50 {
		t.Errorf("after an older import = %+v", got)
	}
}

func TestHistoryEntryFindsOneComic(t *testing.T) {
	s := open(t, t.TempDir())
	a := Comic{Source: "copy_manga", ID: "1", Title: "A"}
	if err := s.Record(History{Comic: a, Chapter: "第 3 话", ChapterIndex: 2, Page: 7, Percent: 42, Time: 100}); err != nil {
		t.Fatal(err)
	}
	got, ok := s.HistoryEntry("copy_manga", "1")
	if !ok {
		t.Fatal("entry not found")
	}
	if got.Chapter != "第 3 话" || got.ChapterIndex != 2 || got.Page != 7 || got.Percent != 42 {
		t.Errorf("entry = %+v", got)
	}
	// Another source may use the same id, and an unknown comic has no entry.
	if _, ok := s.HistoryEntry("komiic", "1"); ok {
		t.Error("entry found under the wrong source")
	}
	if _, ok := s.HistoryEntry("copy_manga", "9"); ok {
		t.Error("entry found for an unknown comic")
	}
}
