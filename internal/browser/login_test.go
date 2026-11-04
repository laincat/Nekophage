package browser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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

// Closing the login window is the user giving up, and must end the wait rather
// than leave the panel spinning while a destroyed page is polled for "logged in".
func TestLoginWindowClosedByUserEndsTheWait(t *testing.T) {
	l := &LoginWindow{key: "nhentai", partition: "comic:nhentai"}
	state := &loginState{done: make(chan struct{}), closedCh: make(chan struct{})}
	state.closed.Store(true)
	state.once.Do(func() { close(state.done) })

	stored, err := l.watch(context.Background(), core.WebviewLogin{}, state)
	if err == nil {
		t.Fatalf("closing the window reported success with %d cookie(s)", stored)
	}
	if stored != 0 {
		t.Errorf("stored %d cookie(s) after the user closed the window", stored)
	}
}

// The source manager's window owns the login wait: closing it cancels the
// context the wait polls with. The evidence that the wait really stopped is
// the server seeing the request connection end, not the panel clearing.
func TestCancellingTheLoginWaitStopsTheSourceRequest(t *testing.T) {
	var once sync.Once
	hit := make(chan struct{})
	gone := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(hit) })
		<-r.Context().Done()
		close(gone)
	}))
	defer server.Close()

	c, err := core.Open(core.Options{DataDir: filepath.Join(t.TempDir(), "d"), AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	script := `class Poller extends ComicSource {
  name = 'Poller'; key = 'poller'; version = '1'
  account = {
    loginWithWebview: {
      url: 'https://site.test/login',
      checkStatus: async (url, title) => { await Network.get('` + server.URL + `/poll'); return false },
    },
  }
  search = {load: async () => ({comics: [], maxPage: 1})}
}`
	if _, err := c.Install(context.Background(), script, "file", ""); err != nil {
		t.Fatal(err)
	}
	decl, err := c.AccountWebviewLogin(context.Background(), "poller")
	if err != nil {
		t.Fatalf("declaration: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan error, 1)
	go func() {
		_, err := c.WebviewStatus(ctx, "poller", decl.Handle, "https://site.test/home", "title")
		finished <- err
	}()

	select {
	case <-hit:
	case <-time.After(20 * time.Second):
		t.Fatal("the source's status request never reached the server")
	}
	cancel()
	select {
	case err := <-finished:
		if err == nil {
			t.Fatal("the status check reported success after the manager closed")
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the status check kept running after the manager closed")
	}
	select {
	case <-gone:
	case <-time.After(10 * time.Second):
		t.Fatal("the server never saw the request connection stop")
	}
}

// Pressing "我已登录" and then closing the window must still stop the source's
// onLoginSuccess: it runs after done is closed, so a close-only-once signal
// would let it keep working for a window the user already dismissed. The server
// seeing the connection stop is the evidence; dropping the UI is not.
func TestClosingTheLoginWindowStopsOnLoginSuccess(t *testing.T) {
	var once sync.Once
	hit := make(chan struct{})
	gone := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(hit) })
		<-r.Context().Done()
		close(gone)
	}))
	defer server.Close()

	c, err := core.Open(core.Options{DataDir: filepath.Join(t.TempDir(), "d"), AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	script := `class Slow extends ComicSource {
  name = 'Slow'; key = 'slow'; version = '1'
  account = {
    loginWithWebview: {
      url: 'https://site.test/login',
      checkStatus: (url, title) => true,
      onLoginSuccess: async () => { await Network.get('` + server.URL + `/settle') },
    },
  }
  search = {load: async () => ({comics: [], maxPage: 1})}
}`
	if _, err := c.Install(context.Background(), script, "file", ""); err != nil {
		t.Fatal(err)
	}
	decl, err := c.AccountWebviewLogin(context.Background(), "slow")
	if err != nil {
		t.Fatalf("declaration: %v", err)
	}

	l := NewLoginWindow(c, nil, "slow", "comic:slow")
	// The user already pressed "我已登录": done has fired, the window has not.
	state := &loginState{done: make(chan struct{}), closedCh: make(chan struct{})}
	close(state.done)

	done := make(chan error, 1)
	go func() { done <- l.runLoginHook(context.Background(), decl, state) }()
	select {
	case <-hit:
	case <-time.After(20 * time.Second):
		t.Fatal("onLoginSuccess never reached the server")
	}

	state.closed.Store(true)
	close(state.closedCh)
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("the hook reported success after the window closed")
		}
		if !strings.Contains(err.Error(), "closed") {
			t.Errorf("error %q does not say the window closed", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("onLoginSuccess kept running after the window closed")
	}
	select {
	case <-gone:
	case <-time.After(10 * time.Second):
		t.Fatal("the server never saw the request connection stop")
	}
}
