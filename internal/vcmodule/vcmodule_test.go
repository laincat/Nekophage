package vcmodule

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleTemplate = `#!id=reader-preferences
#!name=阅读偏好示例
#!desc=声明式配置覆盖示例；当前客户端尚未实现此模块格式
#!version=0.1.0
#!format=veneracat-module/1-draft
#!requires=settings-module.v1

[Reader]
page-gap = 8
keep-screen-on = true

; Only declared settings are permitted. This is not a Surge-compatible module.
; Disabling a module should remove its overlay rather than rewrite base settings.
`

func TestParseOfficialTemplateAndAllAllowlistedKeys(t *testing.T) {
	mod, err := Parse([]byte("\xef\xbb\xbf" + strings.ReplaceAll(sampleTemplate, "\n", "\r\n")))
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}
	if mod.Meta.ID != "reader-preferences" || mod.Meta.Name != "阅读偏好示例" || mod.Meta.Version != "0.1.0" || mod.Meta.Format != Format {
		t.Fatalf("meta = %+v", mod.Meta)
	}
	if len(mod.Meta.Requires) != 1 || mod.Meta.Requires[0] != Capability {
		t.Fatalf("requires = %v, want [%s]", mod.Meta.Requires, Capability)
	}
	if mod.Reader.PageGap == nil || *mod.Reader.PageGap != 8 {
		t.Fatalf("page-gap = %v", mod.Reader.PageGap)
	}
	if mod.Reader.KeepScreenOn == nil || !*mod.Reader.KeepScreenOn {
		t.Fatalf("keep-screen-on = %v", mod.Reader.KeepScreenOn)
	}
	if len(mod.Keys) != 2 || mod.Keys[0] != "Reader.keep-screen-on" || mod.Keys[1] != "Reader.page-gap" {
		t.Fatalf("keys = %v", mod.Keys)
	}

	full := `#!name=Full Reader
#!version=1.2.0
#!format=veneracat-module/1-draft
[Reader]
page-gap = 12 ; pixels
keep-screen-on = false
fit-width = 1200
spread = true
direction = rtl
auto-next = true
theme = dark
`
	m2, err := Parse([]byte(full))
	if err != nil {
		t.Fatalf("parse full: %v", err)
	}
	if *m2.Reader.PageGap != 12 || *m2.Reader.KeepScreenOn != false ||
		*m2.Reader.FitWidth != 1200 || !*m2.Reader.Spread ||
		*m2.Reader.Direction != "rtl" || !*m2.Reader.AutoNext ||
		*m2.Reader.Theme != "dark" {
		t.Fatalf("full reader = %+v", m2.Reader)
	}
	if len(m2.Keys) != len(AllowlistedKeys()) {
		t.Fatalf("keys = %v, want %v", m2.Keys, AllowlistedKeys())
	}
}

func TestParseRejectsUnknownAndSurgeDirectives(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want error
	}{
		{
			name: "missing format",
			raw:  "#!name=A\n#!version=1\n[Reader]\npage-gap = 4\n",
			want: ErrInvalidFormat,
		},
		{
			name: "wrong format",
			raw:  "#!name=A\n#!version=1\n#!format=surge-module/1\n[Reader]\npage-gap = 4\n",
			want: ErrInvalidFormat,
		},
		{
			name: "surge system meta",
			raw:  "#!name=A\n#!version=1\n#!format=veneracat-module/1-draft\n#!system=ios\n[Reader]\npage-gap = 4\n",
			want: ErrForbiddenDirective,
		},
		{
			name: "surge arguments meta",
			raw:  "#!name=A\n#!version=1\n#!format=veneracat-module/1-draft\n#!arguments=foo:bar\n[Reader]\npage-gap = 4\n",
			want: ErrForbiddenDirective,
		},
		{
			name: "mitm section",
			raw:  "#!name=A\n#!version=1\n#!format=veneracat-module/1-draft\n[MITM]\nhostname = %APPEND% api.example.com\n",
			want: ErrForbiddenDirective,
		},
		{
			name: "script section",
			raw:  "#!name=A\n#!version=1\n#!format=veneracat-module/1-draft\n[Script]\nfoo = type=http-response,script-path=evil.js\n",
			want: ErrForbiddenDirective,
		},
		{
			name: "url rewrite section",
			raw:  "#!name=A\n#!version=1\n#!format=veneracat-module/1-draft\n[URL Rewrite]\n^https://a https://b 302\n",
			want: ErrForbiddenDirective,
		},
		{
			name: "proxy section",
			raw:  "#!name=A\n#!version=1\n#!format=veneracat-module/1-draft\n[Proxy]\ndirect = direct\n",
			want: ErrForbiddenDirective,
		},
		{
			name: "unknown section",
			raw:  "#!name=A\n#!version=1\n#!format=veneracat-module/1-draft\n[Experimental]\nfoo = bar\n",
			want: ErrUnknownSection,
		},
		{
			name: "unknown key in allowlisted section",
			raw:  "#!name=A\n#!version=1\n#!format=veneracat-module/1-draft\n[Reader]\nsecret-key = 1\n",
			want: ErrUnknownKey,
		},
		{
			name: "duplicate key",
			raw:  "#!name=A\n#!version=1\n#!format=veneracat-module/1-draft\n[Reader]\npage-gap = 4\npage-gap = 8\n",
			want: ErrInvalidFormat,
		},
		{
			name: "out of range page-gap",
			raw:  "#!name=A\n#!version=1\n#!format=veneracat-module/1-draft\n[Reader]\npage-gap = 999\n",
			want: ErrInvalidValue,
		},
		{
			name: "invalid bool",
			raw:  "#!name=A\n#!version=1\n#!format=veneracat-module/1-draft\n[Reader]\nkeep-screen-on = 1\n",
			want: ErrInvalidValue,
		},
		{
			name: "invalid direction",
			raw:  "#!name=A\n#!version=1\n#!format=veneracat-module/1-draft\n[Reader]\ndirection = vertical\n",
			want: ErrInvalidValue,
		},
		{
			name: "empty settings",
			raw:  "#!name=A\n#!version=1\n#!format=veneracat-module/1-draft\n[Reader]\n; comment only\n",
			want: ErrInvalidFormat,
		},
		{
			name: "meta after section",
			raw:  "#!name=A\n#!version=1\n[Reader]\n#!format=veneracat-module/1-draft\npage-gap = 4\n",
			want: ErrInvalidFormat,
		},
		{
			name: "invalid id header",
			raw:  "#!id=../escape\n#!name=A\n#!version=1\n#!format=veneracat-module/1-draft\n[Reader]\npage-gap = 4\n",
			want: ErrInvalidID,
		},
		{
			name: "unsupported requires capability",
			raw:  "#!name=A\n#!version=1\n#!format=veneracat-module/1-draft\n#!requires=future-cap.v2\n[Reader]\npage-gap = 4\n",
			want: ErrUnsupportedCapability,
		},
		{
			name: "forbidden requires capability",
			raw:  "#!name=A\n#!version=1\n#!format=veneracat-module/1-draft\n#!requires=mitm.v1\n[Reader]\npage-gap = 4\n",
			want: ErrForbiddenDirective,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Parse([]byte(tc.raw)); !errors.Is(err, tc.want) {
				t.Fatalf("Parse() error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestManagerLifecycleStackingAndCleanWithdrawal(t *testing.T) {
	dir := t.TempDir()
	mgr, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	base := DefaultSettings()
	base.Reader.PageGap = 4
	base.Reader.FitWidth = 880
	base.Reader.Theme = "light"
	baseSnapshot := base

	sum := sha256.Sum256([]byte(sampleTemplate))
	// Empty install ID derives from #!id=reader-preferences.
	rec1, err := mgr.Install("", "collection:veneracat.official/reader-preferences", hex.EncodeToString(sum[:]), []byte(sampleTemplate))
	if err != nil {
		t.Fatalf("install reader-preferences: %v", err)
	}
	if rec1.ID != "reader-preferences" || rec1.Enabled || len(rec1.Requires) != 1 || rec1.Requires[0] != Capability {
		t.Fatalf("unexpected record: %+v", rec1)
	}

	// Disabled module does not alter effective settings.
	r0 := mgr.Resolve(base)
	if r0.Effective != base || len(r0.Applied) != 0 {
		t.Fatalf("disabled module altered settings: %+v", r0)
	}

	// Enable first module.
	if err := mgr.Enable("reader-preferences"); err != nil {
		t.Fatalf("enable: %v", err)
	}
	r1 := mgr.Resolve(base)
	if r1.Effective.Reader.PageGap != 8 || !r1.Effective.Reader.KeepScreenOn || r1.Effective.Reader.FitWidth != 880 || r1.Effective.Reader.Theme != "light" {
		t.Fatalf("r1 = %+v", r1)
	}
	if r1.Sources["Reader.page-gap"] != "reader-preferences" {
		t.Fatalf("sources = %+v", r1.Sources)
	}

	// Install and enable second module overriding page-gap and setting direction/theme.
	mod2 := `#!name=右开双页夜间
#!version=0.2.0
#!format=veneracat-module/1-draft
[Reader]
page-gap = 0
spread = true
direction = rtl
theme = dark
`
	if _, err := mgr.Install("reader.manga-night", "file:manga.vcmodule", "", []byte(mod2)); err != nil {
		t.Fatalf("install reader.manga-night: %v", err)
	}
	if err := mgr.Enable("reader.manga-night"); err != nil {
		t.Fatalf("enable mod2: %v", err)
	}
	r2 := mgr.Resolve(base)
	if r2.Effective.Reader.PageGap != 0 || !r2.Effective.Reader.KeepScreenOn || !r2.Effective.Reader.Spread || r2.Effective.Reader.Direction != "rtl" || r2.Effective.Reader.Theme != "dark" {
		t.Fatalf("stacked r2 = %+v", r2)
	}
	if r2.Sources["Reader.page-gap"] != "reader.manga-night" || r2.Sources["Reader.keep-screen-on"] != "reader-preferences" {
		t.Fatalf("stacked sources = %+v", r2.Sources)
	}

	// Reorder so reader-preferences overlays reader.manga-night: page-gap becomes 8 again.
	if err := mgr.Reorder([]string{"reader.manga-night", "reader-preferences"}); err != nil {
		t.Fatalf("reorder: %v", err)
	}
	r3 := mgr.Resolve(base)
	if r3.Effective.Reader.PageGap != 8 || r3.Sources["Reader.page-gap"] != "reader-preferences" || r3.Effective.Reader.Theme != "dark" {
		t.Fatalf("reordered r3 = %+v", r3)
	}

	// Reopen from disk to verify atomic persistence & order preservation.
	reopened, err := Open(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	list := reopened.List()
	if len(list) != 2 || list[0].ID != "reader.manga-night" || list[1].ID != "reader-preferences" {
		t.Fatalf("reopened list = %+v", list)
	}

	// Disabling reader-preferences cleanly restores page-gap=0 (from reader.manga-night) and keep-screen-on=false (from base).
	if err := reopened.Disable("reader-preferences"); err != nil {
		t.Fatalf("disable: %v", err)
	}
	r4 := reopened.Resolve(base)
	if r4.Effective.Reader.PageGap != 0 || r4.Effective.Reader.KeepScreenOn != false || r4.Effective.Reader.Theme != "dark" {
		t.Fatalf("after disable r4 = %+v", r4)
	}

	// Removing reader.manga-night cleanly restores base completely.
	if err := reopened.Remove("reader.manga-night"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "modules", "reader.manga-night.vcmodule")); !os.IsNotExist(err) {
		t.Fatalf("removed module file still exists: %v", err)
	}
	r5 := reopened.Resolve(base)
	if r5.Effective != baseSnapshot || len(r5.Applied) != 0 || len(r5.Sources) != 0 {
		t.Fatalf("after remove r5 = %+v, want base %+v", r5, baseSnapshot)
	}
	// Base itself was never mutated.
	if base != baseSnapshot {
		t.Fatalf("base mutated: %+v", base)
	}
}

func TestManagerPathTraversalAndIntegrityGuards(t *testing.T) {
	dir := t.TempDir()
	mgr, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	noIDTemplate := "#!name=NoID\n#!version=0.1.0\n#!format=veneracat-module/1-draft\n[Reader]\npage-gap = 8\n"
	for _, bad := range []string{"", "..", "../escape", "a/b", `a\b`, "C:foo", ".hidden", "UPPER", "a..b"} {
		if _, err := mgr.Install(bad, "test", "", []byte(noIDTemplate)); !errors.Is(err, ErrInvalidID) {
			t.Errorf("Install(%q) error = %v, want ErrInvalidID", bad, err)
		}
	}

	// Wrong digest refused.
	if _, err := mgr.Install("reader-preferences", "test", strings.Repeat("0", 64), []byte(sampleTemplate)); err == nil {
		t.Fatal("wrong digest accepted")
	}

	// Mismatched #!id vs install ID refused.
	if _, err := mgr.Install("different-id", "test", "", []byte(sampleTemplate)); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("mismatched #!id error = %v, want ErrInvalidID", err)
	}

	// Upgrade preserves enabled state and order.
	if _, err := mgr.Install("reader-preferences", "test", "", []byte(sampleTemplate)); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Enable("reader-preferences"); err != nil {
		t.Fatal(err)
	}
	v2 := strings.Replace(sampleTemplate, "page-gap = 8", "page-gap = 16", 1)
	v2 = strings.Replace(v2, "0.1.0", "0.2.0", 1)
	rec, err := mgr.Install("reader-preferences", "test", "", []byte(v2))
	if err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	if !rec.Enabled || rec.Version != "0.2.0" {
		t.Fatalf("upgraded rec = %+v", rec)
	}
	if got := mgr.Resolve(DefaultSettings()).Effective.Reader.PageGap; got != 16 {
		t.Fatalf("upgraded page-gap = %d, want 16", got)
	}

	// Tampered file on disk is rejected on Open.
	if err := os.WriteFile(filepath.Join(dir, "modules", "reader-preferences.vcmodule"), []byte(sampleTemplate), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(dir); err == nil {
		t.Fatal("tampered module file was accepted on Open")
	}
}
