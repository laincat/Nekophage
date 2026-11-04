package ui

import (
	"strings"
	"testing"

	"github.com/laincat/veneracat-core/service"

	"github.com/laincat/nekophage/internal/services"
)

func TestServiceLineStatesWhatARowShows(t *testing.T) {
	rec := services.Record{Accounts: 2}
	rec.ID, rec.Name, rec.Version, rec.Origin = "trakt", "Trakt", "1", "collection:x/trakt"
	rec.Permissions = []service.Permission{service.PermissionNetwork, service.PermissionSecrets}
	line := serviceLine(rec)
	for _, want := range []string{"trakt", "v1", "network,secrets", "账号 2", "collection:x/trakt"} {
		if !strings.Contains(line, want) {
			t.Errorf("line %q does not mention %q", line, want)
		}
	}

	bare := serviceLine(services.Record{Installed: service.Installed{ID: "demo", Version: "2"}})
	if !strings.Contains(bare, "无权限") {
		t.Errorf("line %q does not say there are no permissions", bare)
	}
	if strings.Contains(bare, "账号") {
		t.Errorf("line %q mentions accounts when there are none", bare)
	}
}

func TestBaseNameHandlesBothSeparators(t *testing.T) {
	for in, want := range map[string]string{
		`C:\Temp\demo.zip`: "demo.zip",
		"/tmp/demo.zip":    "demo.zip",
		"demo.zip":         "demo.zip",
		"":                 "",
	} {
		if got := baseName(in); got != want {
			t.Errorf("baseName(%q) = %q, want %q", in, got, want)
		}
	}
}

// The panel's own status handling: a failure replaces the notice, a success
// clears the error.
func TestServicePanelStatus(t *testing.T) {
	var p servicePanel
	p.start("正在安装…")
	if p.busy == "" {
		t.Fatal("start did not record what is running")
	}
	p.done(nil, "已安装 Demo")
	if p.busy != "" || p.errText != "" || p.notice != "已安装 Demo" {
		t.Fatalf("after success: %+v", p)
	}
	p.done(errFake{}, "")
	if p.errText == "" || p.notice != "" {
		t.Fatalf("after failure: %+v", p)
	}
}

type errFake struct{}

func (errFake) Error() string { return "boom" }

// A method button says its title when the service gave one.
func TestMethodLabelAndShortResult(t *testing.T) {
	if got := methodLabel(service.Method{Name: "sync", Title: "同步"}); got != "同步" {
		t.Errorf("label = %q", got)
	}
	if got := methodLabel(service.Method{Name: "sync"}); got != "sync" {
		t.Errorf("label = %q", got)
	}
	long := strings.Repeat("x", 400)
	if got := shortResult(long); len(got) > 260 || !strings.HasSuffix(got, "…") {
		t.Errorf("shortResult kept %d bytes", len(got))
	}
	if got := shortResult("  {\"ok\":\n true}  "); got != `{"ok": true}` {
		t.Errorf("shortResult = %q", got)
	}
}
