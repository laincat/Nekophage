// Package portable keeps a copy's data beside its executable.
package portable

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// The portable layout keeps everything the app writes beside the executable,
// in a few named folders, so the program's own folder stays readable and
// deleting it removes every trace.
//
//	Nekophage/
//	  Nekophage.exe, LICENSE, README...   shipped files only
//	  data/      sources, plugins, services, cookies, shelf, settings
//	  cache/     anything that can be rebuilt
//	  logs/      diagnostics
//	  webview/   the WebView2 profile (local storage, its own cache)
const (
	portableData    = "data"
	portableCache   = "cache"
	portableLogs    = "logs"
	portableWebView = "webview"
)

// Dirs are the folders of one portable layout.
type Dirs struct {
	Root, Data, Cache, Logs, WebView string
}

func Layout(root string) Dirs {
	return Dirs{
		Root:    root,
		Data:    filepath.Join(root, portableData),
		Cache:   filepath.Join(root, portableCache),
		Logs:    filepath.Join(root, portableLogs),
		WebView: filepath.Join(root, portableWebView),
	}
}

// ErrNotWritable says the program folder cannot hold the data, for example an
// installation under Program Files that a normal user cannot write to.
var ErrNotWritable = errors.New("program folder is not writable")

// Prepare creates the layout under root, or reports why it cannot.
//
// Writability is proven by writing, not by reading permission bits: on Windows
// an ACL, a read-only medium or virtualisation can each make a folder that
// looks writable refuse the write, and the app would then lose data later.
func Prepare(root string) (Dirs, error) {
	dirs := Layout(root)
	for _, dir := range []string{dirs.Data, dirs.Cache, dirs.Logs, dirs.WebView} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return dirs, fmt.Errorf("%w: %v", ErrNotWritable, err)
		}
	}
	probe, err := os.CreateTemp(dirs.Data, ".write-check-*")
	if err != nil {
		return dirs, fmt.Errorf("%w: %v", ErrNotWritable, err)
	}
	name := probe.Name()
	_, writeErr := probe.Write([]byte("ok"))
	closeErr := probe.Close()
	removeErr := os.Remove(name)
	if err := errors.Join(writeErr, closeErr, removeErr); err != nil {
		return dirs, fmt.Errorf("%w: %v", ErrNotWritable, err)
	}
	return dirs, nil
}

// MigrateFrom copies an earlier per-user profile into data when the portable
// copy has nothing of its own yet, and reports how many files it copied.
//
// It copies and never moves: the old profile stays where it was, so a user who
// goes back to an installed build loses nothing, and an interrupted copy can
// simply run again. A data folder that already holds anything is left alone,
// because that is this copy's own state and must win over an older one.
func MigrateFrom(old, data string) (int, error) {
	if old == "" || filepath.Clean(old) == filepath.Clean(data) {
		return 0, nil
	}
	if info, err := os.Stat(old); err != nil || !info.IsDir() {
		return 0, nil
	}
	if entries, err := os.ReadDir(data); err != nil {
		return 0, err
	} else if len(entries) > 0 {
		return 0, nil
	}
	copied := 0
	err := filepath.WalkDir(old, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(old, path)
		if err != nil {
			return err
		}
		target := filepath.Join(data, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if err := copyFile(path, target); err != nil {
			return err
		}
		copied++
		return nil
	})
	return copied, err
}

// copyFile writes through a temporary name, so a half-copied file is never
// mistaken for the real one on the next start.
func copyFile(from, to string) error {
	body, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	tmp := to + ".migrating"
	if err := os.WriteFile(tmp, body, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, to); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// WebViewEnv is the variable the WebView2 loader reads for its profile
// folder. MyGo honours it when it loads the runtime itself as well.
const WebViewEnv = "WEBVIEW2_USER_DATA_FOLDER"

// PointWebViewAt sends the WebView2 profile into dir unless the user already
// chose one, so a portable copy does not leave a profile in %LOCALAPPDATA%.
func PointWebViewAt(dir string, getenv func(string) string, setenv func(string, string) error) error {
	if getenv(WebViewEnv) != "" {
		return nil
	}
	return setenv(WebViewEnv, dir)
}
