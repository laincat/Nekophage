package services

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// packageZip builds a service package holding one working service.
func packageZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, body := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func manifest(id string) string {
	return `{"id":"` + id + `","name":"Demo","version":"1","kind":"service.js","entry":"index.mjs","permissions":["storage"]}`
}

func TestInstallWritesThePackageAndRunsIt(t *testing.T) {
	data := t.TempDir()
	archive := packageZip(t, map[string]string{
		"service.json": manifest("demo"),
		"index.mjs":    "export function handle(method) { return { method: method }; }",
		"lib/a.mjs":    "export const a = 1;",
	})
	rec, err := Install(context.Background(), data, "collection:x/y", "abc", archive)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if rec.ID != "demo" || rec.Version != "1" {
		t.Fatalf("record = %+v", rec)
	}
	for _, name := range []string{"service.json", "index.mjs", filepath.Join("lib", "a.mjs")} {
		if _, err := os.Stat(filepath.Join(rec.Dir, name)); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	m, source, err := Load(context.Background(), data, "demo")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if m.ID != "demo" || source == "" {
		t.Fatalf("m=%+v source=%q", m, source)
	}
	all, err := List(data)
	if err != nil || len(all) != 1 || all[0].ID != "demo" {
		t.Fatalf("list = %+v (%v)", all, err)
	}
}

// A package that does not compile is refused before anything is written.
func TestInstallRefusesAPackageThatDoesNotCompile(t *testing.T) {
	data := t.TempDir()
	archive := packageZip(t, map[string]string{
		"service.json": manifest("broken"),
		"index.mjs":    "export function handle( {",
	})
	if _, err := Install(context.Background(), data, "x", "", archive); err == nil {
		t.Fatal("a package that does not parse was installed")
	}
	if _, err := os.Stat(filepath.Join(data, "services", "broken")); err == nil {
		t.Fatal("a refused package left files behind")
	}
}

// Upgrading replaces the whole directory, so files of the previous version do
// not survive.
func TestInstallReplacesTheWholeDirectory(t *testing.T) {
	data := t.TempDir()
	first := packageZip(t, map[string]string{
		"service.json": manifest("demo"),
		"index.mjs":    "export function handle() { return 1; }",
		"old.mjs":      "export const old = 1;",
	})
	if _, err := Install(context.Background(), data, "a", "", first); err != nil {
		t.Fatal(err)
	}
	second := packageZip(t, map[string]string{
		"service.json": manifest("demo"),
		"index.mjs":    "export function handle() { return 2; }",
	})
	rec, err := Install(context.Background(), data, "b", "", second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(rec.Dir, "old.mjs")); err == nil {
		t.Fatal("a file of the previous version survived the upgrade")
	}
}

func TestRemove(t *testing.T) {
	data := t.TempDir()
	archive := packageZip(t, map[string]string{"service.json": manifest("demo"), "index.mjs": "export function handle() {}"})
	if _, err := Install(context.Background(), data, "a", "", archive); err != nil {
		t.Fatal(err)
	}
	if err := Remove(data, "demo"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := os.Stat(filepath.Join(data, "services", "demo")); err == nil {
		t.Fatal("the directory survived removal")
	}
	all, err := List(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 0 {
		t.Fatalf("list = %+v", all)
	}
	data2, err := os.ReadFile(filepath.Join(data, "services.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index map[string]json.RawMessage
	if err := json.Unmarshal(data2, &index); err != nil {
		t.Fatal(err)
	}
	if _, ok := index["demo"]; ok {
		t.Fatal("the record survived removal")
	}
}

// An id becomes a directory name, so it may not walk out of the data directory.
func TestInstalledDirRefusesABadID(t *testing.T) {
	data := t.TempDir()
	for _, id := range []string{"", "..", "a/b", `a\b`, ".hidden"} {
		if _, err := InstalledDir(data, id); err == nil {
			t.Errorf("%q was accepted", id)
		}
	}
}

func TestAccountsKeepKeysApartFromValues(t *testing.T) {
	data := t.TempDir()
	archive := packageZip(t, map[string]string{"service.json": manifest("demo"), "index.mjs": "export function handle() {}"})
	if _, err := Install(context.Background(), data, "a", "", archive); err != nil {
		t.Fatal(err)
	}
	accounts, err := OpenAccounts(data, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if err := accounts.Set(context.Background(), "token", "abc"); err != nil {
		t.Fatal(err)
	}
	if err := accounts.Set(context.Background(), "user", "alice"); err != nil {
		t.Fatal(err)
	}
	keys, err := accounts.Keys()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 || keys[0] != "token" || keys[1] != "user" {
		t.Fatalf("keys = %v", keys)
	}
	value, err := accounts.Get(context.Background(), "token")
	if err != nil || value != "abc" {
		t.Fatalf("get = %q (%v)", value, err)
	}
	if _, err := accounts.Get(context.Background(), "missing"); err == nil {
		t.Fatal("a missing key was returned")
	}
	if err := accounts.Remove(context.Background(), "token"); err != nil {
		t.Fatal(err)
	}
	if _, err := accounts.Get(context.Background(), "token"); err == nil {
		t.Fatal("a removed key was still readable")
	}
	if err := accounts.Remove(context.Background(), "token"); err == nil {
		t.Fatal("removing a missing key was accepted")
	}
}

// Accounts belong to one service: another service of the same id cannot read
// them, and the file sits inside the service directory.
func TestAccountsArePerService(t *testing.T) {
	data := t.TempDir()
	for _, id := range []string{"a", "b"} {
		archive := packageZip(t, map[string]string{"service.json": manifest(id), "index.mjs": "export function handle() {}"})
		if _, err := Install(context.Background(), data, "x", "", archive); err != nil {
			t.Fatal(err)
		}
	}
	a, err := OpenAccounts(data, "a")
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Set(context.Background(), "token", "for-a"); err != nil {
		t.Fatal(err)
	}
	b, err := OpenAccounts(data, "b")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Get(context.Background(), "token"); err == nil {
		t.Fatal("one service read another's account value")
	}
	if _, err := os.Stat(filepath.Join(data, "services", "a", "accounts.json")); err != nil {
		t.Fatalf("account file: %v", err)
	}
}

// A service that is not installed has nowhere to keep accounts.
func TestAccountsNeedAnInstalledService(t *testing.T) {
	if _, err := OpenAccounts(t.TempDir(), "missing"); err == nil {
		t.Fatal("accounts opened for a service that is not installed")
	}
}
