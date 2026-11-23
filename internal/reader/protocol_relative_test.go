package reader

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/laincat/nekophage/internal/core"
)

func TestReaderImageInheritsSchemeAndRejectsNonHTTP(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		fmt.Fprint(w, "image-fixture")
	}))
	defer upstream.Close()
	c, err := core.Open(core.Options{DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	body := fmt.Sprintf("class Reader extends ComicSource {name='Reader'; key='reader'; version='1'; comic={loadEp:async()=>{await Network.get(%q);return {images:[%q]}}}}", upstream.URL, strings.TrimPrefix(upstream.URL, "http:"))
	ctx := context.Background()
	if _, err := c.Install(ctx, body, "fixture", ""); err != nil {
		t.Fatal(err)
	}
	urls, err := c.ChapterImages(ctx, "reader", "comic", "ep")
	if err != nil || len(urls) != 1 {
		t.Fatalf("chapter: %v %v", urls, err)
	}
	reader := New(c)
	for _, test := range []struct {
		url    string
		status int
	}{{urls[0], http.StatusOK}, {"file:///test", http.StatusBadGateway}, {"http:///missing-host", http.StatusBadGateway}} {
		response := httptest.NewRecorder()
		reader.ServeHTTP(response, httptest.NewRequest(http.MethodGet, ImageURL("reader", "comic", "ep", test.url), nil))
		if response.Code != test.status {
			t.Fatalf("%s HTTP=%d; expected %d (%s)", test.url, response.Code, test.status, response.Body.String())
		}
		if test.status == http.StatusOK && response.Body.String() != "image-fixture" {
			t.Fatal("reader did not fetch the fixture")
		}
	}
}
