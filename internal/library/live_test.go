package library

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"encoding/base64"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/reader"
	"github.com/laincat/nekophage/internal/shelf"
)

// TestLiveSource runs the reading path against a real source and the real
// network. It is skipped unless NK_LIVE_JS names a source script and
// NK_LIVE_QUERY gives a search keyword:
//
//	NK_LIVE_JS=/path/to/wnacg.js NK_LIVE_QUERY=test go test -run TestLiveSource -v ./internal/library
//
// Nothing here is asserted about the content: a community source changes, and
// the point of the test is to see where the path stops, so every stage logs
// what it got and its error.
func TestLiveSource(t *testing.T) {
	script := os.Getenv("NK_LIVE_JS")
	query := os.Getenv("NK_LIVE_QUERY")
	if script == "" || query == "" {
		t.Skip("set NK_LIVE_JS and NK_LIVE_QUERY to run against a real source")
	}
	body, err := os.ReadFile(script)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	c, err := core.Open(core.Options{
		DataDir:    filepath.Join(t.TempDir(), "data"),
		AppVersion: "1.6.0",
		Locale:     "zh_CN",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	lib := New(c)

	rec, err := lib.InstallText(ctx, string(body), script)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	t.Logf("installed %s (%s) v%s", rec.Name, rec.Key, rec.Version)

	options, err := c.DefaultOptions(ctx, rec.Key)
	if err != nil {
		t.Fatalf("search options: %v", err)
	}
	t.Logf("search options: %v", options)
	page, err := c.Search(ctx, rec.Key, query, options, 1)
	if err != nil {
		t.Fatalf("search %q: %v", query, err)
	}
	t.Logf("search %q: %d comics, maxPage %d", query, len(page.Comics), page.MaxPage)
	if len(page.Comics) == 0 {
		t.Fatal("search returned nothing, the rest of the path cannot run")
	}
	for i, comic := range page.Comics {
		if i >= 3 {
			break
		}
		t.Logf("  [%d] id=%q title=%q cover=%q", i, comic.ID, comic.Title, comic.Cover)
	}

	pick := page.Comics[0]
	d, err := c.Details(ctx, rec.Key, pick.ID)
	if err != nil {
		t.Fatalf("details %q: %v", pick.ID, err)
	}
	t.Logf("details: title=%q chapters=%d tags=%d", d.Title, len(d.Chapters), len(d.Tags))
	if len(d.Chapters) > 0 {
		t.Logf("  first chapter: id=%q title=%q group=%q", d.Chapters[0].ID, d.Chapters[0].Title, d.Chapters[0].Group)
	}

	// Some chapters of a source carry no pages (another language's volume,
	// a taken-down chapter), so try a few as a reader would.
	chapterID := ""
	var images []string
	for i, ch := range d.Chapters {
		if i >= 10 {
			break
		}
		got, err := c.ChapterImages(ctx, rec.Key, pick.ID, ch.ID)
		if err != nil {
			t.Logf("chapter %q: %v", ch.Title, err)
			continue
		}
		if len(got) == 0 {
			t.Logf("chapter %q: no pages", ch.Title)
			continue
		}
		chapterID, images = ch.ID, got
		t.Logf("chapter %q: %d images", ch.Title, len(got))
		break
	}
	if len(images) == 0 {
		t.Fatal("no chapter of the first comic had pages")
	}
	for i, u := range images {
		if i >= 2 {
			break
		}
		t.Logf("  image %d: %s", i, shorten(u, 110))
	}

	img, ctype, err := c.FetchImage(ctx, rec.Key, images[0], pick.ID, chapterID)
	if err != nil {
		t.Fatalf("fetch image: %v", err)
	}
	t.Logf("image bytes: %d, content type %q", len(img), ctype)
	// What matters is that the body is image data, not how the server
	// labelled it: sources answer with octet-stream or with a type that
	// does not match the bytes.
	if kind := imageKind(img); kind == "" {
		t.Errorf("not an image: %d bytes, %q", len(img), ctype)
	}

	// The reader page's own requests, exactly as the WebView makes them: it
	// asks /chapter for the list, then /img per page.
	srv := reader.New(c)
	chapterURL := reader.Scheme + "://localhost/chapter?source=" + rec.Key + "&comic=" + pick.ID + "&chapter=" + chapterID
	rec2 := httptest.NewRecorder()
	srv.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, chapterURL, nil))
	if rec2.Code != http.StatusOK {
		t.Fatalf("reader /chapter: %d %s", rec2.Code, rec2.Body.String())
	}
	var listed struct {
		Images []string `json:"images"`
		Count  int      `json:"count"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &listed); err != nil {
		t.Fatalf("reader /chapter json: %v", err)
	}
	t.Logf("reader /chapter: %d images", len(listed.Images))
	if len(listed.Images) != len(images) {
		t.Errorf("reader /chapter has %d images, the source gave %d", len(listed.Images), len(images))
	}

	imgURL := reader.ImageURL(rec.Key, pick.ID, chapterID, images[0])
	rec3 := httptest.NewRecorder()
	srv.ServeHTTP(rec3, httptest.NewRequest(http.MethodGet, imgURL, nil))
	if rec3.Code != http.StatusOK {
		t.Fatalf("reader /img: %d %s", rec3.Code, rec3.Body.String())
	}
	ctype2 := rec3.Header().Get("Content-Type")
	t.Logf("reader /img: %d bytes, %s", rec3.Body.Len(), ctype2)
	if imageKind(rec3.Body.Bytes()) == "" {
		t.Errorf("reader /img is not an image: %d bytes, %q", rec3.Body.Len(), ctype2)
	}

	// Reading further: the shelf remembers the position, the reader keeps a
	// page, and asking for the next chapter answers with its address.
	sh, err := shelf.Open(filepath.Join(t.TempDir(), "shelf"))
	if err != nil {
		t.Fatal(err)
	}
	chapterTitle, chapterIndex := "", 0
	for i, ch := range d.Chapters {
		if ch.ID == chapterID {
			chapterTitle, chapterIndex = ch.Title, i
			break
		}
	}
	if err := sh.Record(shelf.History{
		Comic:   shelf.Comic{Source: rec.Key, ID: pick.ID, Title: d.Title, Cover: pick.Cover},
		Chapter: chapterTitle, ChapterIndex: chapterIndex,
	}); err != nil {
		t.Fatalf("shelf record: %v", err)
	}

	srv.OnProgress(func(p reader.Progress) {
		_, err := sh.UpdateProgress(p.Source, p.Comic, chapterTitle, chapterIndex, p.Page, p.Percent)
		if err != nil {
			t.Errorf("shelf progress: %v", err)
		}
	})
	progressURL := reader.Scheme + "://localhost/progress?source=" + rec.Key + "&comic=" + pick.ID +
		"&chapter=" + chapterID + "&page=3&pages=" + itoa(len(images)) + "&percent=42"
	if rec := httptest.NewRecorder(); true {
		srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, progressURL, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("reader /progress: %d", rec.Code)
		}
	}
	entry := sh.History(1)[0]
	t.Logf("shelf after progress: page %d, %d%%", entry.Page, entry.Percent)
	if entry.Page != 3 || entry.Percent != 42 {
		t.Errorf("progress not stored: %+v", entry)
	}

	srv.OnKeep(func(k reader.Keep) bool {
		return sh.KeepPage(shelf.ImageFavourite{Source: k.Source, ComicID: k.Comic,
			ComicTitle: d.Title, ChapterID: k.Chapter, ChapterIndex: chapterIndex,
			ChapterName: chapterTitle, Page: k.Page, URL: k.URL}) == nil
	})
	keepURL := reader.Scheme + "://localhost/keep?source=" + rec.Key + "&comic=" + pick.ID +
		"&chapter=" + chapterID + "&page=1&u=" + base64.RawURLEncoding.EncodeToString([]byte(images[0]))
	if rec := httptest.NewRecorder(); true {
		srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, keepURL, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("reader /keep: %d", rec.Code)
		}
	}
	if pages := sh.Pages(1); len(pages) != 1 || pages[0].URL != images[0] {
		t.Errorf("kept page not stored: %+v", pages)
	} else {
		t.Logf("kept page: %q page %d", shorten(pages[0].URL, 80), pages[0].Page)
	}

	srv.OnSwitch(func(sw reader.Switch) (string, bool) {
		return reader.ReaderURL(sw.Source, sw.Comic, sw.Chapter, d.Title), true
	})
	switchURL := reader.Scheme + "://localhost/switch?source=" + rec.Key + "&comic=" + pick.ID + "&chapter=" + chapterID
	rec6 := httptest.NewRecorder()
	srv.ServeHTTP(rec6, httptest.NewRequest(http.MethodGet, switchURL, nil))
	if rec6.Code != http.StatusOK || !strings.Contains(rec6.Body.String(), "chapter="+chapterID) {
		t.Errorf("reader /switch: %d %s", rec6.Code, rec6.Body.String())
	}

	// Reopening the profile reads the same shelf back.
	again, err := shelf.Open(sh.Root())
	if err != nil {
		t.Fatal(err)
	}
	if h, pages := len(again.History(0)), len(again.Pages(0)); h != 1 || pages != 1 {
		t.Errorf("shelf after reopen: %d history, %d pages", h, pages)
	}

	// A cover goes through the source's thumbnail hook.
	if pick.Cover != "" {
		cover, ctype, err := c.FetchThumbnail(ctx, rec.Key, pick.Cover, pick.ID)
		if err != nil {
			t.Errorf("cover: %v", err)
		} else if imageKind(cover) == "" {
			t.Errorf("cover is not an image: %d bytes, %q", len(cover), ctype)
		} else {
			t.Logf("cover bytes: %d, content type %q", len(cover), ctype)
		}
	}
}

// imageKind names the image container by its magic bytes, whatever the
// server said in Content-Type. Sources do mislabel: Komiic answers with
// image/jpeg for a WebP body, and Go's standard decoders cannot read WebP.
// Returns "" for bytes that are not a known container.
func imageKind(data []byte) string {
	png := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	switch {
	case len(data) > 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return "jpeg"
	case len(data) >= len(png) && string(data[:len(png)]) == string(png):
		return "png"
	case len(data) > 6 && (string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a"):
		return "gif"
	case len(data) > 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return "webp"
	case len(data) > 12 && string(data[4:8]) == "ftyp":
		return "heif/avif"
	case len(data) > 2 && string(data[:2]) == "BM":
		return "bmp"
	}
	return ""
}

// itoa keeps the URL building above readable.
func itoa(n int) string { return strconv.Itoa(n) }

func shorten(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
