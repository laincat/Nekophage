package ui

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/veneracat-core/backup"
)

// The export is only useful if the file reads back: this writes one through the
// same code path the button uses, then opens it with Core's reader.
func TestWriteBackupRoundTrips(t *testing.T) {
	c, err := core.Open(core.Options{DataDir: filepath.Join(t.TempDir(), "d"), AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	script := `class Ex extends ComicSource {
  name = 'Ex'; key = 'ex'; version = '1'
  search = {load: async () => ({comics: [], maxPage: 1})}
}`
	ctx := context.Background()
	if _, err := c.Install(ctx, script, "file", ""); err != nil {
		t.Fatal(err)
	}
	if err := c.Store().SaveSetting("ex", "domain", []byte(`"a.example"`)); err != nil {
		t.Fatal(err)
	}

	a := New(c, nil, nil)
	path := filepath.Join(t.TempDir(), "export.venera")
	if err := a.writeBackup(path); err != nil {
		t.Fatalf("write: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("the export is empty")
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	archive, err := backup.Open(f, info.Size(), backup.Limits{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	content, err := archive.Read(ctx, backup.Options{})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(content.Sources) != 1 {
		t.Fatalf("sources = %+v", content.Sources)
	}
	if content.Sources[0].Key != "ex" {
		t.Errorf("key = %q", content.Sources[0].Key)
	}
	if content.Sources[0].Script == "" {
		t.Error("the script did not travel")
	}
	// Cookies are the one thing that must never be in an export.
	if archive.Has(backup.MemberCookies) {
		t.Error("the export contains cookies")
	}
}
