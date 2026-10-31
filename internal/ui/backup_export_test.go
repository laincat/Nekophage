package ui

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/laincat/nekophage/internal/archive"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/library"
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

// The export is only worth the button if the import can restore it. This writes
// one from a client that has a source, then imports it into a second, empty
// client through the same steps applyBackup takes.
func TestExportRestoresIntoAClientWithoutIt(t *testing.T) {
	source := `class Cycle extends ComicSource {
  name = 'Cycle'; key = 'cycle'; version = '1'
  search = {load: async () => ({comics: [], maxPage: 1})}
}`
	from, err := core.Open(core.Options{DataDir: filepath.Join(t.TempDir(), "a"), AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { from.Close() })
	if _, err := from.Install(context.Background(), source, "file", ""); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "cycle.venera")
	if err := New(from, library.New(from), nil).writeBackup(path); err != nil {
		t.Fatalf("write: %v", err)
	}

	// A second client, with nothing installed.
	into, err := core.Open(core.Options{DataDir: filepath.Join(t.TempDir(), "b"), AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { into.Close() })
	lib := library.New(into)
	if len(lib.Sources()) != 0 {
		t.Fatalf("the second client starts with %d sources", len(lib.Sources()))
	}

	ctx := context.Background()
	known := []string{}
	backupFile, err := archive.Read(ctx, path, known, archive.Options{})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	installed := 0
	for _, src := range backupFile.Plan.Sources {
		if src.Script == "" {
			continue
		}
		if _, err := lib.InstallText(ctx, src.Script, "backup:"+src.Key); err != nil {
			t.Fatalf("install %s: %v", src.Key, err)
		}
		installed++
	}
	if installed != 1 {
		t.Fatalf("installed %d sources", installed)
	}
	keys := []string{}
	for _, s := range lib.Sources() {
		keys = append(keys, s.Key)
	}
	if len(keys) != 1 || keys[0] != "cycle" {
		t.Errorf("after import: %v", keys)
	}
}

// A source that stored something must come back with it: a restore that only
// brings back code leaves the source logged out.
func TestExportCarriesSourceData(t *testing.T) {
	source := `class Cycle extends ComicSource {
  name = 'Cycle'; key = 'cycle'; version = '1'
  search = {load: async () => ({comics: [], maxPage: 1})}
}`
	from, err := core.Open(core.Options{DataDir: filepath.Join(t.TempDir(), "a"), AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { from.Close() })
	ctx := context.Background()
	if _, err := from.Install(ctx, source, "file", ""); err != nil {
		t.Fatal(err)
	}
	if err := from.Store().SaveData("cycle", "token", json.RawMessage(`"abc"`)); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "cycle.venera")
	if err := New(from, library.New(from), nil).writeBackup(path); err != nil {
		t.Fatalf("write: %v", err)
	}

	into, err := core.Open(core.Options{DataDir: filepath.Join(t.TempDir(), "b"), AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { into.Close() })
	lib := library.New(into)
	backupFile, err := archive.Read(ctx, path, nil, archive.Options{})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	for _, src := range backupFile.Plan.Sources {
		if src.Script == "" {
			continue
		}
		if _, err := lib.InstallText(ctx, src.Script, "backup:"+src.Key); err != nil {
			t.Fatal(err)
		}
	}
	for _, d := range mustSourceData(t, backupFile) {
		values := map[string]json.RawMessage{}
		for _, f := range d.Fields {
			values[f.Name] = f.Value
		}
		if _, err := lib.ImportSourceData(d.Key, values); err != nil {
			t.Fatalf("import data: %v", err)
		}
	}
	got, ok, err := into.Store().LoadData("cycle", "token")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || string(got) != `"abc"` {
		t.Errorf("token = %s (present %v)", got, ok)
	}
}

func mustSourceData(t *testing.T, b *archive.Backup) []archive.Data {
	t.Helper()
	data, err := b.SourceData()
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// This client keeps a source's data and settings in two buckets; the archive has
// one object per source. Both have to come back, in the buckets they started in.
func TestExportAndImportKeepSettingsApart(t *testing.T) {
	source := `class Cycle extends ComicSource {
  name = 'Cycle'; key = 'cycle'; version = '1'
  settings = {domain: {title: '域名', type: 'input', default: 'a.example'}}
  search = {load: async () => ({comics: [], maxPage: 1})}
}`
	from, err := core.Open(core.Options{DataDir: filepath.Join(t.TempDir(), "a"), AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { from.Close() })
	ctx := context.Background()
	if _, err := from.Install(ctx, source, "file", ""); err != nil {
		t.Fatal(err)
	}
	if err := from.Store().SaveData("cycle", "token", json.RawMessage(`"abc"`)); err != nil {
		t.Fatal(err)
	}
	if err := from.Store().SaveSetting("cycle", "domain", json.RawMessage(`"b.example"`)); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "cycle.venera")
	if err := New(from, library.New(from), nil).writeBackup(path); err != nil {
		t.Fatalf("write: %v", err)
	}

	into, err := core.Open(core.Options{DataDir: filepath.Join(t.TempDir(), "b"), AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { into.Close() })
	lib := library.New(into)
	backupFile, err := archive.Read(ctx, path, nil, archive.Options{})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	for _, src := range backupFile.Plan.Sources {
		if src.Script != "" {
			if _, err := lib.InstallText(ctx, src.Script, "backup:"+src.Key); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, d := range mustSourceData(t, backupFile) {
		values := map[string]json.RawMessage{}
		for _, f := range d.Fields {
			values[f.Name] = f.Value
		}
		if _, _, err := lib.ImportSourceValues(ctx, d.Key, values); err != nil {
			t.Fatalf("import: %v", err)
		}
	}
	if got, ok, _ := into.Store().LoadData("cycle", "token"); !ok || string(got) != `"abc"` {
		t.Errorf("token came back as %s (present %v)", got, ok)
	}
	// The setting has to land in the settings bucket, where the settings panel and
	// the source's own getSetting() look for it.
	if got, ok, _ := into.Store().LoadSetting("cycle", "domain"); !ok || string(got) != `"b.example"` {
		t.Errorf("domain came back as %s (present %v)", got, ok)
	}
	if _, ok, _ := into.Store().LoadData("cycle", "domain"); ok {
		t.Error("the setting was filed as data")
	}
}

// The refusal has to read as a sentence, not as Core's English.
func TestTooMuchSourceDataIsExplained(t *testing.T) {
	err := errors.New(`backup: source "cat.js" has more than 16777216 bytes of data`)
	got := explain(err)
	if !strings.Contains(got, "数据太大") {
		t.Errorf("explain = %q", got)
	}
}
