package ui

import (
	"context"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/library"
	"github.com/laincat/nekophage/internal/sourceapi"
	"github.com/laincat/veneracat-core/plugin"
)

const codeFixture = `// 中文注释：只读源码，不应执行搜索
class SourceCodeFixture extends ComicSource {
  name = "源码测试"; key = "source_code_fixture"; version = "1.0.0"
  search = {load: async () => { throw new Error("viewer must not search") }}
}
`

func TestSourceCodeInstalledSnapshotIsReadOnly(t *testing.T) {
	c, err := core.Open(core.Options{DataDir: t.TempDir(), AppVersion: sourceapi.Level})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	lib := library.New(c)
	if _, err := lib.InstallText(context.Background(), codeFixture, "fixture.js"); err != nil {
		t.Fatal(err)
	}
	before, err := c.Store().Records()
	if err != nil {
		t.Fatal(err)
	}
	if lib.Sources()[0].Body != "" {
		t.Fatal("listing unexpectedly contains source text")
	}
	code, name, err := sourceCodeState(c.Records(), "source_code_fixture")
	if err != nil {
		t.Fatal(err)
	}
	if name != "源码测试" || code.Buffer.String() != codeFixture || code.Buffer.Language().Name != "JavaScript" {
		t.Fatalf("wrong snapshot: name=%q language=%v", name, code.Buffer.Language())
	}
	p := &sourceCodePanel{code: code, name: name}
	tt := ui.NewTester(p.view, 900, 680)
	if err := tt.Click("源代码（只读）"); err != nil {
		t.Fatal(err)
	}
	// Real component input paths, including a simulated composition on a
	// selection. This is not an operating-system IME acceptance test.
	start := strings.Index(codeFixture, "中文注释")
	code.Select(start, start+len("中文注释"))
	tt.Frame()
	tt.Command("copy")
	if got := tt.Clipboard(); got != "中文注释" {
		t.Fatalf("read-only selection copy = %q", got)
	}
	tt.Compose("拼", 1)
	tt.Compose("", 0)
	tt.Command("cut")
	tt.Type("MUTATION")
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	tt.Compose("ni", 2)
	tt.Compose("", 0)
	if code.Buffer.String() != codeFixture {
		t.Fatal("read-only input changed the buffer")
	}
	after, err := c.Store().Records()
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) || c.Records()[0].Body != codeFixture {
		t.Fatal("viewing changed installed source")
	}
	if _, err := c.Replace(context.Background(), "source_code_fixture", strings.ReplaceAll(codeFixture, "1.0.0", "2.0.0"), "fixture.js", ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(code.Buffer.String(), "1.0.0") {
		t.Fatal("snapshot changed under an open viewer")
	}
	fresh, _, err := sourceCodeState(c.Records(), "source_code_fixture")
	if err != nil || !strings.Contains(fresh.Buffer.String(), "2.0.0") {
		t.Fatalf("reopen did not read updated source: %v", err)
	}
}

func TestSourceCodeKindsAndMissing(t *testing.T) {
	for _, typ := range []plugin.Type{plugin.TypeComic, plugin.TypeVideoJS, plugin.TypeKazumi} {
		st, _, err := sourceCodeState([]plugin.Record{{Key: "x", Type: typ, Body: "{}"}}, "x")
		if err != nil {
			t.Fatal(err)
		}
		want := "JavaScript"
		if typ == plugin.TypeKazumi {
			want = "JSON"
		}
		if !st.ReadOnly || st.Buffer.Language().Name != want {
			t.Fatalf("%s: wrong editor configuration", typ)
		}
	}
	if _, _, err := sourceCodeState(nil, "removed"); err == nil {
		t.Fatal("removed source must not display an empty success")
	}
	if _, _, err := sourceCodeState([]plugin.Record{{Key: "x", Type: plugin.TypeBuiltin}}, "x"); err == nil {
		t.Fatal("builtin source has no script")
	}
}

func TestSourceCodeCloseDropsStateAndRejectsLateLoad(t *testing.T) {
	st, name, err := sourceCodeState([]plugin.Record{{Key: "x", Name: "测试", Type: plugin.TypeComic, Body: "// 中文"}}, "x")
	p := &sourceCodePanel{key: "x"}
	if !p.loaded(st, name, err) {
		t.Fatal("initial load rejected")
	}
	p.release()
	if p.code != nil || p.win != nil || p.key != "" || p.name != "" || p.err != "" {
		t.Fatal("close retained panel state")
	}
	if p.loaded(st, name, err) || p.code != nil {
		t.Fatal("late completion resurrected a closed viewer")
	}
	fresh := &sourceCodePanel{key: "x"}
	if !fresh.loaded(st, name, err) {
		t.Fatal("new viewer rejected its own load")
	}
}
