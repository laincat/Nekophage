package ui

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/laincat/nekophage/internal/core"
)

func TestReplacingSearchStopsFilterFollowup(t *testing.T) {
	received := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		select {
		case received <- struct{}{}:
		default:
		}
	}))
	defer server.Close()
	c, err := core.Open(core.Options{DataDir: t.TempDir(), AppVersion: "1.6.3"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	script := fmt.Sprintf(`class FilterCancel extends ComicSource {
 name = "Filter cancellation"; key = "filter_cancel"; version = "1.0.0"
 search = { optionList: [{label: "Sort", options: ["new"], default: "new"}], load: async () => { await Network.get(%q, {}); return {comics: [], maxPage: 1} } }
}`, server.URL)
	if _, err = c.Install(context.Background(), script, "file", ""); err != nil {
		t.Fatal(err)
	}
	a := &App{core: c, sourceKey: "filter_cancel", query: "probe", dev: DevOptions{Filter: "0=new"}}
	a.loadSearchOptions("filter_cancel")
	a.mu.Lock()
	a.searchGen.next()
	a.mu.Unlock()
	select {
	case <-received:
		t.Fatal("a replaced search still ran the filter follow-up request")
	case <-time.After(300 * time.Millisecond):
	}
	if a.searchKey == "filter_cancel" {
		t.Fatal("the replaced search accepted its late filter result")
	}
}
