package reader

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestReaderURLRoundTrip(t *testing.T) {
	u, err := url.Parse(ReaderURL("t", "c1", "ep 2", "标题 · 第一话"))
	if err != nil {
		t.Fatal(err)
	}
	// nkreader://localhost on macOS and Linux, http://nkreader.localhost on
	// Windows, where WebView2 serves custom schemes that way.
	if (u.Scheme != Scheme && !(u.Scheme == "http" && u.Host == Scheme+".localhost")) || u.Path != "/reader" {
		t.Fatalf("url = %s", u)
	}
	q := u.Query()
	if q.Get("source") != "t" || q.Get("comic") != "c1" || q.Get("chapter") != "ep 2" || q.Get("title") != "标题 · 第一话" {
		t.Fatalf("query = %v", q)
	}
}

func TestImageURLCarriesTheImageInBase64(t *testing.T) {
	original := "https://img.test/a/b.jpg?x=1&y=2"
	u, err := url.Parse(ImageURL("t", "c1", "ep", original))
	if err != nil {
		t.Fatal(err)
	}
	if u.Path != "/img" {
		t.Fatalf("path = %s", u.Path)
	}
	raw, err := base64.RawURLEncoding.DecodeString(u.Query().Get("u"))
	if err != nil || string(raw) != original {
		t.Fatalf("u = %q, %v", raw, err)
	}
	// The value is URL-safe: no characters that need escaping.
	if v := u.Query().Get("u"); strings.ContainsAny(v, "+/=") {
		t.Errorf("value needs escaping: %s", v)
	}
}

func TestPageIsSelfContained(t *testing.T) {
	s := New(nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, Scheme+"://localhost/reader?source=t", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("content type = %q", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{"正在加载", "chapter?", "img?", "location.href",
		// Chapter navigation is asked of the window, not done by the page.
		`id="next"`, `id="endnext"`, "switch?", "prevtitle",
		// Automatic next chapter: off unless asked for, and never before
		// the reader has moved.
		`id="auto"`, "nkreader.auto", "IntersectionObserver", "hasScrolled",
		// Keeping one page goes through the window too.
		`id="keep"`, "keep?",
		// The layout controls: two pages at a time and a right-first order.
		`id="spread"`, `id="dir"`, "#pages.spread", "#pages.spread.rtl",
		"figure.wide", "nkreader.spread", "nkreader.dir"} {
		if !strings.Contains(body, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	// One file: no external script or style.
	if strings.Contains(body, "<link") || strings.Contains(body, "src=\"http") {
		t.Error("page loads an external resource")
	}
}

func TestImageRejectsBadReferences(t *testing.T) {
	s := New(nil)
	for _, target := range []string{Scheme + "://localhost/img", Scheme + "://localhost/img?u=***",
		Scheme + "://localhost/img?u=" + base64.RawURLEncoding.EncodeToString([]byte("file:///etc/passwd"))} {
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		if rec.Code != http.StatusBadGateway {
			t.Errorf("%s: status = %d, want 502", target, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "error") {
			t.Errorf("%s: body = %s", target, rec.Body.String())
		}
	}
}

func TestUnknownPathIsNotFound(t *testing.T) {
	s := New(nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, Scheme+"://localhost/secret", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestProgressIsReadClampedAndDelivered(t *testing.T) {
	s := New(nil)
	var got []Progress
	s.OnProgress(func(p Progress) { got = append(got, p) })
	get := func(q string) {
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, Scheme+"://localhost/progress?"+q, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Errorf("content type = %q", ct)
		}
	}
	get("source=s&comic=c&chapter=ch1&page=12&pages=53&percent=37")
	if len(got) != 1 {
		t.Fatalf("reports = %+v", got)
	}
	if p := got[0]; p.Source != "s" || p.Comic != "c" || p.Chapter != "ch1" || p.Page != 12 || p.Pages != 53 || p.Percent != 37 {
		t.Errorf("progress = %+v", p)
	}
	// Values out of range are clamped, and junk reads as zero.
	get("source=s&comic=c&chapter=ch1&page=99&pages=53&percent=250")
	if p := got[1]; p.Page != 53 || p.Percent != 100 {
		t.Errorf("clamped = %+v", p)
	}
	get("source=s&comic=c&chapter=ch1&page=x&pages=y&percent=z")
	if p := got[2]; p.Page != 0 || p.Pages != 0 || p.Percent != 0 {
		t.Errorf("junk = %+v", p)
	}
	// A report without a comic is not passed on.
	before := len(got)
	get("source=s&percent=10")
	if len(got) != before {
		t.Errorf("report without a comic was delivered: %+v", got[before:])
	}
}

func TestProgressWithoutAHandlerIsFine(t *testing.T) {
	s := New(nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, Scheme+"://localhost/progress?source=s&comic=c", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestReaderURLNearCarriesNeighbours(t *testing.T) {
	u, err := url.Parse(ReaderURLNear("s", "c", "ch2", "标题",
		Neighbour{ID: "ch1", Title: "上一话"}, Neighbour{ID: "ch3", Title: "下一话"}))
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if q.Get("prev") != "ch1" || q.Get("prevtitle") != "上一话" {
		t.Errorf("prev = %q/%q", q.Get("prev"), q.Get("prevtitle"))
	}
	if q.Get("next") != "ch3" || q.Get("nexttitle") != "下一话" {
		t.Errorf("next = %q/%q", q.Get("next"), q.Get("nexttitle"))
	}
	// The ends of a comic carry no neighbour, and no empty parameters.
	plain, err := url.Parse(ReaderURL("s", "c", "ch1", "标题"))
	if err != nil {
		t.Fatal(err)
	}
	if plain.Query().Has("prev") || plain.Query().Has("next") {
		t.Errorf("ends carry neighbours: %s", plain.RawQuery)
	}
}

func TestSwitchChapterReachesTheWindow(t *testing.T) {
	s := New(nil)
	var got []Switch
	s.OnSwitch(func(sw Switch) (string, bool) {
		got = append(got, sw)
		if sw.Chapter != "known" {
			return "", false
		}
		return ReaderURL(sw.Source, sw.Comic, sw.Chapter, "下一话"), true
	})
	call := func(q string) int {
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, Scheme+"://localhost/switch?"+q, nil))
		return rec.Code
	}
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, Scheme+"://localhost/switch?source=s&comic=c&chapter=known", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("known chapter: status = %d", rec.Code)
	}
	var answer struct {
		OK  bool   `json:"ok"`
		URL string `json:"url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &answer); err != nil {
		t.Fatal(err)
	}
	if !answer.OK || !strings.Contains(answer.URL, "chapter=known") {
		t.Errorf("answer = %+v", answer)
	}
	if len(got) != 1 || got[0].Source != "s" || got[0].Comic != "c" || got[0].Chapter != "known" {
		t.Errorf("switch = %+v", got)
	}
	if code := call("source=s&comic=c&chapter=absent"); code != http.StatusNotFound {
		t.Errorf("unknown chapter: status = %d, want 404", code)
	}
	// A chapter id may be empty: a gallery source has no chapter list and
	// names its one reading unit with an empty id. The window decides, so the
	// request reaches it.
	before := len(got)
	if code := call("source=s&comic=c"); code != http.StatusNotFound {
		t.Errorf("empty chapter: status = %d, want 404 from the window", code)
	}
	if len(got) != before+1 {
		t.Errorf("empty chapter did not reach the window: %+v", got[before:])
	}
	// Without a comic there is nothing to look up, so the window is not
	// asked at all.
	before = len(got)
	if code := call("source=s"); code != http.StatusNotFound {
		t.Errorf("no comic: status = %d, want 404", code)
	}
	if len(got) != before {
		t.Errorf("a request without a comic reached the window: %+v", got[before:])
	}
	// Without a handler the page is told so, not left waiting.
	bare := New(nil)
	rec = httptest.NewRecorder()
	bare.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, Scheme+"://localhost/switch?source=s&comic=c&chapter=x", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("no handler: status = %d", rec.Code)
	}
}

func TestKeepPageReachesTheWindow(t *testing.T) {
	s := New(nil)
	var got []Keep
	s.OnKeep(func(k Keep) bool {
		got = append(got, k)
		return k.Page > 0
	})
	call := func(q string) int {
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, Scheme+"://localhost/keep?"+q, nil))
		return rec.Code
	}
	url := base64.RawURLEncoding.EncodeToString([]byte("https://img/1.jpg"))
	if code := call("source=s&comic=c&chapter=ch&page=3&u=" + url); code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if len(got) != 1 || got[0].URL != "https://img/1.jpg" || got[0].Page != 3 || got[0].Chapter != "ch" {
		t.Errorf("keep = %+v", got)
	}
	// A request without a decodable address never reaches the window.
	before := len(got)
	if code := call("source=s&comic=c&chapter=ch&page=1&u=%%%"); code != http.StatusNotFound {
		t.Errorf("bad url: status = %d", code)
	}
	if len(got) != before {
		t.Errorf("bad url reached the window: %+v", got[before:])
	}
	// The window may refuse, and the page is told so.
	if code := call("source=s&comic=c&chapter=ch&page=0&u=" + url); code != http.StatusNotFound {
		t.Errorf("refused keep: status = %d", code)
	}
}
