package ui

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/library"
	"github.com/laincat/veneracat-core/plugin"
	"github.com/laincat/veneracat-core/subscription"
)

func TestClosingSourceManagerStopsInstallHTTP(t *testing.T) {
	for _, action := range []string{"url", "entry"} {
		t.Run(action, func(t *testing.T) {
			started, stopped, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				close(started)
				select {
				case <-r.Context().Done():
					close(stopped)
				case <-release:
					http.Error(w, "held", http.StatusOK)
				}
			}))
			defer server.Close()
			defer close(release)
			c, err := core.Open(core.Options{DataDir: t.TempDir(), AppVersion: "1.6.3"})
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			a := &App{core: c, lib: library.New(c)}
			a.manage.lifetime, a.manage.cancel = context.WithCancel(context.Background())
			go func() {
				ctx, cancel := context.WithTimeout(a.manage.lifetime, 30*time.Second)
				defer cancel()
				if action == "url" {
					_, _ = a.lib.InstallTypedURL(ctx, plugin.TypeComic, server.URL)
					return
				}
				_, _ = a.lib.InstallEntry(ctx, subscription.Entry{Key: "held", URL: server.URL}, "https://index.invalid/index.json")
			}()
			select {
			case <-started:
			case <-time.After(3 * time.Second):
				t.Fatal("the source request never reached the server")
			}
			a.manage.mu.Lock()
			a.manage.cancel()
			a.manage.mu.Unlock()
			select {
			case <-stopped:
			case <-time.After(2 * time.Second):
				t.Fatal("closing source management kept its HTTP request running")
			}
		})
	}
}

func TestClosingSourceManagerStopsAccountHTTP(t *testing.T) {
	started, stopped, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
			close(stopped)
		case <-release:
			http.Error(w, "held", http.StatusOK)
		}
	}))
	defer server.Close()
	defer close(release)
	c, err := core.Open(core.Options{DataDir: t.TempDir(), AppVersion: "1.6.3"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	script := fmt.Sprintf(`class AccountCancel extends ComicSource {
 name = "Account cancellation"; key = "account_cancel"; version = "1.0.0"
 account = { login: async () => { await Network.get(%q, {}); return "ok" } }
 search = {load: async () => ({comics: [], maxPage: 1})}
}`, server.URL)
	if _, err = c.Install(context.Background(), script, "file", ""); err != nil {
		t.Fatal(err)
	}
	a := &App{core: c, lib: library.New(c)}
	a.manage.lifetime, a.manage.cancel = context.WithCancel(context.Background())
	go func() {
		ctx, cancel := context.WithTimeout(a.manage.lifetime, 30*time.Second)
		defer cancel()
		_, _ = a.core.LoginAccount(ctx, "account_cancel", "user", "secret")
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("the account request never reached the server")
	}
	a.manage.mu.Lock()
	a.manage.cancel()
	a.manage.mu.Unlock()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("closing source management kept its account HTTP request running")
	}
}
