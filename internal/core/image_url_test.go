package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// A source sometimes hands back an image address without its scheme, such as
// "//images.example.com/1.jpg": RFC 3986 reads that as "the scheme of the
// page", which for a reader is https. The client must be able to fetch such an
// address instead of refusing it.
//
// The source is a real one, so the whole path is exercised: install, ask the
// source for the image, fetch it.
func TestImageWithASchemeRelativeURLIsFetched(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/page/c1":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<html><body><img class="pic" src="//` + r.Host + `/img/1.jpg"></body></html>`))
		case "/img/1.jpg":
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0, 0, 0})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c, err := Open(Options{DataDir: filepath.Join(t.TempDir(), "data"), AppVersion: "1.6.0", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	script := `
class SchemeLess extends ComicSource {
    name = "SchemeLess"; key = "schemeless"; version = "1.0.0"
    comic = {
        loadEp: async (comicId, epId) => {
            const res = await Network.get("` + srv.URL + `/page/" + comicId, {})
            const doc = new HtmlDocument(res.body)
            const img = doc.querySelector("img.pic")
            return { images: [img.attributes["src"]] }
        }
    }
}
`
	if _, err := c.Install(context.Background(), script, "test", ""); err != nil {
		t.Fatalf("install: %v", err)
	}

	raw, err := c.Call(context.Background(), "schemeless", "comic.loadEp", "c1")
	if err != nil {
		t.Fatalf("loadEp: %v", err)
	}
	if !strings.Contains(string(raw), "//"+strings.TrimPrefix(srv.URL, "http://")) {
		t.Fatalf("the source did not return a scheme-relative address: %s", raw)
	}

	// The address without its scheme must still be fetchable.
	_, _, err = c.FetchImage(context.Background(), "schemeless", "//"+strings.TrimPrefix(srv.URL, "http://")+"/img/1.jpg", "c1", "e1")
	if err != nil {
		t.Fatalf("fetch image: %v", err)
	}
}

// The rule must not become a way through: a file or a script address is still
// refused, because it is not scheme-relative and nothing is rewritten.
func TestImageWithAnUnsafeSchemeIsRefused(t *testing.T) {
	c, err := Open(Options{DataDir: filepath.Join(t.TempDir(), "data"), AppVersion: "1.6.0", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	for _, raw := range []string{"file:///etc/passwd", "javascript:alert(1)", "ftp://x.test/a.jpg"} {
		if _, _, err := c.FetchImage(context.Background(), "mini", raw, "c1", "e1"); err == nil {
			t.Errorf("FetchImage(%q) was allowed", raw)
		}
	}
}
