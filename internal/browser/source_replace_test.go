package browser

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/laincat/nekophage/internal/core"
)

// Updating the source while the login window is open leaves the window holding a
// handle whose callbacks belonged to the old runtime. Both Core calls the window
// makes must report that, so the wait can stop instead of polling a dead handle.
//
// The window itself needs a WebView, so this pins the signal the wait reads; the
// sentence it shows for it is covered by the same branch in watch.
func TestReplacingTheSourceReportsTheLoginHandleGone(t *testing.T) {
	c, err := core.Open(core.Options{DataDir: filepath.Join(t.TempDir(), "d"), AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	first := `class Shift extends ComicSource {
  name = 'Shift'; key = 'shift'; version = '1'
  account = {
    loginWithWebview: {
      url: 'https://site.test/login',
      checkStatus: (url, title) => true,
      onLoginSuccess: async () => {},
    },
  }
  search = {load: async () => ({comics: [], maxPage: 1})}
}`
	second := strings.Replace(first, "version = '1'", "version = '2'", 1)
	second = strings.Replace(second, "checkStatus: (url, title) => true", "checkStatus: (url, title) => false", 1)
	if _, err := c.Install(context.Background(), first, "file", ""); err != nil {
		t.Fatal(err)
	}
	decl, err := c.AccountWebviewLogin(context.Background(), "shift")
	if err != nil {
		t.Fatalf("declaration: %v", err)
	}
	if _, err := c.Replace(context.Background(), "shift", second, "file", ""); err != nil {
		t.Fatalf("replace: %v", err)
	}

	if _, err := c.WebviewStatus(context.Background(), "shift", decl.Handle, "https://site.test/home", "t"); !core.WebviewHandleGone(err) {
		t.Errorf("status error = %v; want the expired-handle error", err)
	}
	if err := c.WebviewLoginDone(context.Background(), "shift", decl.Handle); !core.WebviewHandleGone(err) {
		t.Errorf("done = %v; want the expired-handle error", err)
	}
}
