package ui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/library"
	"github.com/laincat/veneracat-core/plugin"
)

func TestClosingSourceManagerStopsInstallHTTP(t *testing.T) {
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
		_, _ = a.lib.InstallTypedURL(ctx, plugin.TypeComic, server.URL)
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("the source install never reached the server")
	}
	a.manage.mu.Lock()
	a.manage.cancel()
	a.manage.mu.Unlock()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("closing source management kept its install HTTP request running")
	}
}
