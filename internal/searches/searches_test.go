package searches

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddKeepsNewestFirstAndDropsRepeats(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"海贼王", "one piece", "  海贼王  "} {
		if err := s.Add(q); err != nil {
			t.Fatal(err)
		}
	}
	got := s.Items(0)
	if len(got) != 2 {
		t.Fatalf("items = %v", got)
	}
	if got[0] != "海贼王" || got[1] != "one piece" {
		t.Errorf("items = %v", got)
	}
	// Case only differences are the same keyword.
	if err := s.Add("ONE PIECE"); err != nil {
		t.Fatal(err)
	}
	got = s.Items(0)
	if len(got) != 2 || got[0] != "ONE PIECE" {
		t.Errorf("items = %v", got)
	}
	if err := s.Add("   "); err != nil {
		t.Fatal(err)
	}
}

func TestItemsAreCapped(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"a", "b", "c", "d", "e"} {
		if err := s.Add(q); err != nil {
			t.Fatal(err)
		}
	}
	if got := s.Items(2); len(got) != 2 || got[0] != "e" || got[1] != "d" {
		t.Errorf("limited items = %v", got)
	}
	if got := s.Items(0); len(got) != 5 {
		t.Errorf("full items = %v", got)
	}
}

func TestMergeKeepsLocalFirst(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Add("本地"); err != nil {
		t.Fatal(err)
	}
	added, err := s.Merge([]string{"旧的", "更旧的", "本地"})
	if err != nil {
		t.Fatal(err)
	}
	if added != 2 {
		t.Errorf("added = %d", added)
	}
	got := s.Items(0)
	want := []string{"本地", "旧的", "更旧的"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("items = %v, want %v", got, want)
		}
	}
}

func TestRemoveClearAndReload(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"a", "b"} {
		if err := s.Add(q); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Remove("A"); err != nil {
		t.Fatal(err)
	}
	again, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := again.Items(0); len(got) != 1 || got[0] != "b" {
		t.Fatalf("after reload = %v", got)
	}
	if err := again.Clear(); err != nil {
		t.Fatal(err)
	}
	third, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := third.Items(0); len(got) != 0 {
		t.Errorf("after clear = %v", got)
	}
	// The file is still readable JSON with an empty list.
	b, err := os.ReadFile(filepath.Join(dir, "searches.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) == "" || string(b) == "null" {
		t.Errorf("file = %q", b)
	}
}

func TestOpenRejectsBrokenFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "searches.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(dir); err == nil {
		t.Fatal("broken file accepted")
	}
	if _, err := Open(""); err == nil {
		t.Fatal("empty dir accepted")
	}
}
