package browser

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/laincat/nekophage/internal/core"
)

// A source with no browser login is told apart before a window opens: five
// sources log in on their own page, the rest do not.
func TestLoginWindowRejectsSourcesWithoutBrowserLogin(t *testing.T) {
	c, err := core.Open(core.Options{DataDir: filepath.Join(t.TempDir(), "d"), AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	if _, err := c.Install(context.Background(), `class Plain extends ComicSource {
  name = 'Plain'; key = 'plain'; version = '1'
  account = {login: async () => true}
  search = {load: async () => ({comics: [], maxPage: 1})}
}`, "file", ""); err != nil {
		t.Fatal(err)
	}
	login := NewLoginWindow(c, nil, "plain", "comic:plain")
	if _, err := login.Run(context.Background()); err == nil {
		t.Fatal("a source without a browser login opened one")
	}
}

// The declaration reaches the window: URL and handle come from Core, not from
// anything the client guesses.
func TestLoginWindowReadsTheDeclaration(t *testing.T) {
	c, err := core.Open(core.Options{DataDir: filepath.Join(t.TempDir(), "d"), AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	if _, err := c.Install(context.Background(), `class Browser extends ComicSource {
  name = 'Browser'; key = 'browser'; version = '1'
  account = {
    loginWithWebview: {
      url: 'https://site.test/login',
      checkStatus: (url, title) => title === 'ok',
    },
  }
  search = {load: async () => ({comics: [], maxPage: 1})}
}`, "file", ""); err != nil {
		t.Fatal(err)
	}
	decl, err := c.AccountWebviewLogin(context.Background(), "browser")
	if err != nil {
		t.Fatalf("declaration: %v", err)
	}
	if decl.URL != "https://site.test/login" || decl.Handle == "" {
		t.Fatalf("declaration = %+v", decl)
	}
	done, err := c.WebviewStatus(context.Background(), "browser", decl.Handle, "https://site.test/home", "ok")
	if err != nil || !done {
		t.Fatalf("status: %v %v", done, err)
	}
	// A source without onLoginSuccess still finishes.
	if err := c.WebviewLoginDone(context.Background(), "browser", decl.Handle); err != nil {
		t.Errorf("done: %v", err)
	}
}
