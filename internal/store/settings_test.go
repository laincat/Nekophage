package store

import (
	"bytes"
	"os"
	"testing"

	"github.com/laincat/nekophage/internal/vcmodule"
)

func TestSettingsStoreNeverMutatesBaseWithModuleOverlays(t *testing.T) {
	dir := t.TempDir()
	st, err := OpenSettings(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	base := vcmodule.DefaultSettings()
	base.Reader.PageGap = 6
	base.Reader.FitWidth = 960
	base.Reader.Theme = "light"
	if err := st.WriteBase(base); err != nil {
		t.Fatalf("write base: %v", err)
	}
	rawBefore, err := os.ReadFile(st.Path())
	if err != nil {
		t.Fatal(err)
	}

	mgr, err := st.Modules()
	if err != nil {
		t.Fatal(err)
	}
	mod := `#!name=阅读偏好示例
#!version=0.1.0
#!format=veneracat-module/1-draft
[Reader]
page-gap = 14
keep-screen-on = true
theme = dark
`
	if _, err := mgr.Install("reader.pref", "test", "", []byte(mod)); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Enable("reader.pref"); err != nil {
		t.Fatal(err)
	}

	eff, err := st.EffectiveReader()
	if err != nil {
		t.Fatalf("effective reader: %v", err)
	}
	if eff.PageGap != 14 || !eff.KeepScreenOn || eff.FitWidth != 960 || eff.Theme != "dark" {
		t.Fatalf("effective = %+v", eff)
	}

	// Base settings file must be byte-for-byte untouched.
	rawDuring, err := os.ReadFile(st.Path())
	if err != nil || !bytes.Equal(rawBefore, rawDuring) {
		t.Fatalf("base settings file changed under overlay: %v", err)
	}

	// Disabling module cleanly reverts effective reader config to base.
	if err := mgr.Disable("reader.pref"); err != nil {
		t.Fatal(err)
	}
	effAfter, err := st.EffectiveReader()
	if err != nil {
		t.Fatalf("effective after disable: %v", err)
	}
	if effAfter != base.Reader {
		t.Fatalf("after disable = %+v, want %+v", effAfter, base.Reader)
	}
}

func TestSettingsStoreRejectsCorruptFile(t *testing.T) {
	dir := t.TempDir()
	st, err := OpenSettings(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(st.Path(), []byte("{not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenSettings(dir); err == nil {
		t.Fatal("corrupt settings file was accepted")
	}
}
