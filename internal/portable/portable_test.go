package portable

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Everything the app writes lands in the four named folders: the program's own
// folder must keep only what was shipped.
func TestPortableLayoutKeepsTheRootClean(t *testing.T) {
	root := t.TempDir()
	shipped := filepath.Join(root, "Nekophage.exe")
	if err := os.WriteFile(shipped, []byte("binary"), 0o600); err != nil {
		t.Fatal(err)
	}
	dirs, err := Prepare(root)
	if err != nil {
		t.Fatalf("preparePortable: %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, e := range entries {
		got[e.Name()] = e.IsDir()
	}
	want := map[string]bool{"Nekophage.exe": false, "data": true, "cache": true, "logs": true, "webview": true}
	if len(got) != len(want) {
		t.Fatalf("root holds %v, want exactly %v", got, want)
	}
	for name, isDir := range want {
		if dir, ok := got[name]; !ok || dir != isDir {
			t.Errorf("root entry %q: present=%v dir=%v, want dir=%v", name, ok, dir, isDir)
		}
	}
	// The write check must not leave its probe behind.
	left, err := os.ReadDir(dirs.Data)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		t.Errorf("data/ holds %d leftover entries after the write check", len(left))
	}
}

// A folder the user cannot write to is reported, so the caller can fall back to
// the per-user profile instead of failing later with lost data.
func TestPortableRefusesAFolderItCannotWrite(t *testing.T) {
	root := t.TempDir()
	// A file where data/ should be makes MkdirAll fail on every platform, which
	// is the same refusal a read-only install folder produces.
	if err := os.WriteFile(filepath.Join(root, portableData), []byte("not a folder"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Prepare(root)
	if !errors.Is(err, ErrNotWritable) {
		t.Fatalf("preparePortable = %v, want ErrNotWritable", err)
	}
}

// The WebView2 profile follows the portable folder, but a folder the user set
// explicitly wins.
func TestWebViewProfileFollowsThePortableFolder(t *testing.T) {
	env := map[string]string{}
	getenv := func(k string) string { return env[k] }
	setenv := func(k, v string) error { env[k] = v; return nil }

	if err := PointWebViewAt(`D:\Nekophage\webview`, getenv, setenv); err != nil {
		t.Fatal(err)
	}
	if env[WebViewEnv] != `D:\Nekophage\webview` {
		t.Fatalf("%s = %q, want the portable folder", WebViewEnv, env[WebViewEnv])
	}

	env[WebViewEnv] = `E:\chosen`
	if err := PointWebViewAt(`D:\Nekophage\webview`, getenv, setenv); err != nil {
		t.Fatal(err)
	}
	if env[WebViewEnv] != `E:\chosen` {
		t.Fatalf("%s = %q, want the user's own choice kept", WebViewEnv, env[WebViewEnv])
	}
}

// The first portable start brings an installed build's profile along, copying
// so the old profile is untouched.
func TestMigrationCopiesAnEarlierProfileOnce(t *testing.T) {
	old := t.TempDir()
	for name, body := range map[string]string{
		"plugins.json":        `[{"key":"a"}]`,
		"shelf.json":          `{"history":[]}`,
		"sources/a.json":      `{"data":{}}`,
		"services/x/pkg.json": `{}`,
	} {
		path := filepath.Join(old, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	dirs, err := Prepare(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	n, err := MigrateFrom(old, dirs.Data)
	if err != nil || n != 4 {
		t.Fatalf("MigrateFrom = %d, %v; want 4 files", n, err)
	}
	got, err := os.ReadFile(filepath.Join(dirs.Data, "sources", "a.json"))
	if err != nil || string(got) != `{"data":{}}` {
		t.Fatalf("copied source = %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(old, "plugins.json")); err != nil {
		t.Fatalf("the old profile must stay where it was: %v", err)
	}
	// No temporary names survive a successful copy.
	_ = filepath.WalkDir(dirs.Data, func(path string, d os.DirEntry, err error) error {
		if filepath.Ext(path) == ".migrating" {
			t.Errorf("leftover %s", path)
		}
		return nil
	})

	// A second start finds data of its own and copies nothing, so a newer
	// portable state is never overwritten by the older profile.
	if err := os.WriteFile(filepath.Join(old, "plugins.json"), []byte(`["older"]`), 0o600); err != nil {
		t.Fatal(err)
	}
	n, err = MigrateFrom(old, dirs.Data)
	if err != nil || n != 0 {
		t.Fatalf("second MigrateFrom = %d, %v; want 0", n, err)
	}
	got, _ = os.ReadFile(filepath.Join(dirs.Data, "plugins.json"))
	if string(got) != `[{"key":"a"}]` {
		t.Fatalf("portable plugins.json = %q, overwritten by the older profile", got)
	}
}

// Nothing to migrate is not an error.
func TestMigrationWithoutAnEarlierProfile(t *testing.T) {
	dirs, err := Prepare(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, old := range []string{"", filepath.Join(t.TempDir(), "missing"), dirs.Data} {
		if n, err := MigrateFrom(old, dirs.Data); err != nil || n != 0 {
			t.Errorf("MigrateFrom(%q) = %d, %v; want 0, nil", old, n, err)
		}
	}
}

// On Windows a read-only attribute is not enough to stop a write to a folder,
// so this case only runs where permission bits are enforced.
func TestPortableRefusesAReadOnlyFolder(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("folder permission bits are not enforced on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores permission bits")
	}
	root := t.TempDir()
	if err := os.Chmod(root, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o700) })
	if _, err := Prepare(root); !errors.Is(err, ErrNotWritable) {
		t.Fatalf("preparePortable = %v, want ErrNotWritable", err)
	}
}
