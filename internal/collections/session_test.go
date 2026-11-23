package collections

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestUnfollowDuringRefreshPreservesAnotherCollection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var refreshing atomic.Bool
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[1:]
		version := "1"
		if refreshing.Load() {
			version = "2"
			if id == "first" {
				close(started)
				<-release
			}
		}
		fmt.Fprint(w, document(id, version))
	}))
	defer server.Close()
	defer close(release)
	data := t.TempDir()
	manager := openTest(t, data)
	for _, id := range []string{"first", "second"} {
		if err := manager.Follow(ctx, server.URL+"/"+id); err != nil {
			t.Fatal(err)
		}
	}
	refreshing.Store(true)
	result := make(chan error, 1)
	go func() { result <- manager.Refresh(ctx, "first") }()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if err := manager.Refresh(ctx, "second"); err != nil {
		t.Fatal(err)
	}
	if err := manager.Unfollow("first"); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("late refresh succeeded")
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	for _, m := range []*Manager{manager, openTest(t, data)} {
		views, err := m.List(Platform(), Capabilities())
		if err != nil || len(views) != 1 || views[0].ID != "second" || views[0].Version != "2" {
			t.Fatalf("snapshot: %+v, %v", views, err)
		}
	}
}
