// Package reader serves the reading view: a web page in a WebView of native
// UI, with the pages streamed from Go.
//
// The page asks for a chapter's images and draws them; Go fetches each one
// through the source's own rules (onImageLoad, headers, cookies), so a
// source that needs a Referer or a login still shows its pages. Serving the
// images through the nkreader:// scheme keeps them out of the page's memory
// as base64 and lets the web engine decode as the reader scrolls.
package reader

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"time"

	"github.com/laincat/nekophage/internal/core"
)

// Scheme is the custom scheme the page and its images are served from.
const Scheme = "nkreader"

// readerHTML is the reader page. It is embedded: the app ships one binary,
// and the page needs no build step.
//
//go:embed assets/reader.html
var readerHTML []byte

// Switch is a request from the page to open another chapter of the same
// comic. The window that loaded the chapter answers it, so its own state
// stays right.
type Switch struct {
	Source, Comic, Chapter string
}

// Keep is a request to store one page of the chapter being read.
type Keep struct {
	Source, Comic, Chapter string
	Page                   int
	// URL is the image's address in the source.
	URL string
}

// Progress is how far the reader has scrolled in a chapter.
type Progress struct {
	Source  string
	Comic   string
	Chapter string
	// Page is the 1-based page the reader is looking at; Pages is how many
	// the chapter has.
	Page, Pages int
	// Percent is the distance through the chapter, 0-100.
	Percent int
}

// Server answers the reader's requests.
type Server struct {
	core *core.Core
	// Timeout bounds one image fetch.
	Timeout time.Duration

	onProgress func(Progress)
	onSwitch   func(Switch) (string, bool)
	onKeep     func(Keep) bool
}

// OnProgress sets what receives the reader's position as it scrolls. The
// page reports it with a request, so no messages cross the WebView boundary.
func (s *Server) OnProgress(fn func(Progress)) { s.onProgress = fn }

// OnSwitch sets what opens a chapter the page asked for. It returns the
// chapter's own reader URL, which the page then loads, and whether the
// chapter was found at all.
func (s *Server) OnSwitch(fn func(Switch) (string, bool)) { s.onSwitch = fn }

// OnKeep sets what stores a page the reader kept. It reports whether the page
// was taken.
func (s *Server) OnKeep(fn func(Keep) bool) { s.onKeep = fn }

// New builds the server.
func New(c *core.Core) *Server { return &Server{core: c, Timeout: 60 * time.Second} }

// ServeHTTP routes the scheme's requests.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/", "/reader":
		s.page(w, r)
	case "/chapter":
		s.chapter(w, r)
	case "/img":
		s.image(w, r)
	case "/progress":
		s.progress(w, r)
	case "/switch":
		s.openChapter(w, r)
	case "/keep":
		s.keepPage(w, r)
	default:
		http.NotFound(w, r)
	}
}

// progress records where the reader is. The page sends it as it scrolls;
// an out-of-range value is clamped rather than trusted.
func (s *Server) progress(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	p := Progress{
		Source:  q.Get("source"),
		Comic:   q.Get("comic"),
		Chapter: q.Get("chapter"),
		Page:    atoi(q.Get("page")),
		Pages:   atoi(q.Get("pages")),
		Percent: atoi(q.Get("percent")),
	}
	if p.Percent < 0 {
		p.Percent = 0
	}
	if p.Percent > 100 {
		p.Percent = 100
	}
	if p.Page < 0 {
		p.Page = 0
	}
	if p.Pages < 0 {
		p.Pages = 0
	}
	if p.Page > p.Pages && p.Pages > 0 {
		p.Page = p.Pages
	}
	if s.onProgress != nil && p.Source != "" && p.Comic != "" {
		s.onProgress(p)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// openChapter hands a chapter change to the window and answers with the
// chapter's reader URL. The page loads it itself: loading the WebView from
// inside its own request would leave the two waiting on each other.
func (s *Server) openChapter(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	sw := Switch{Source: q.Get("source"), Comic: q.Get("comic"), Chapter: q.Get("chapter")}
	url, ok := "", false
	if s.onSwitch != nil && sw.Source != "" && sw.Comic != "" && sw.Chapter != "" {
		url, ok = s.onSwitch(sw)
	}
	w.Header().Set("Content-Type", "application/json")
	if !ok || url == "" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"ok":false}`))
		return
	}
	body, err := json.Marshal(map[string]any{"ok": true, "url": url})
	if err != nil {
		http.Error(w, `{"ok":false}`, http.StatusInternalServerError)
		return
	}
	w.Write(body)
}

// keepPage hands one page to the window, which knows the comic and chapter
// it belongs to.
func (s *Server) keepPage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	raw, err := base64.RawURLEncoding.DecodeString(q.Get("u"))
	keep := Keep{Source: q.Get("source"), Comic: q.Get("comic"), Chapter: q.Get("chapter"),
		Page: atoi(q.Get("page")), URL: string(raw)}
	if err != nil {
		keep.URL = ""
	}
	ok := false
	if s.onKeep != nil && keep.URL != "" && keep.Source != "" && keep.Comic != "" {
		ok = s.onKeep(keep)
	}
	w.Header().Set("Content-Type", "application/json")
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"ok":false}`))
		return
	}
	w.Write([]byte(`{"ok":true}`))
}

// atoi reads a number, treating anything else as zero.
func atoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

// page serves the reader itself. The query names what to read.
func (s *Server) page(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(readerHTML)
}

// chapter returns {"images": [...], "title": "..."} for the query's chapter.
func (s *Server) chapter(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	source, comic, chapter := q.Get("source"), q.Get("comic"), q.Get("chapter")
	ctx, cancel := context.WithTimeout(r.Context(), s.Timeout)
	defer cancel()
	images, err := s.core.ChapterImages(ctx, source, comic, chapter)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"images": images, "count": len(images)})
}

// image streams one page image, fetched through the source's rules.
func (s *Server) image(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	source, comic, chapter := q.Get("source"), q.Get("comic"), q.Get("chapter")
	raw, err := base64.RawURLEncoding.DecodeString(q.Get("u"))
	if err != nil || len(raw) == 0 {
		writeError(w, errors.New("reader: bad image reference"))
		return
	}
	// The page hands back what /chapter gave it, but nothing stops a page
	// from asking for another scheme, so only http(s) is ever fetched.
	target := string(raw)
	if u, err := url.Parse(target); err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		writeError(w, errors.New("reader: image URL is not http(s)"))
		return
	}
	if s.core == nil {
		writeError(w, errors.New("reader: no source is loaded"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.Timeout)
	defer cancel()
	data, ctype, err := s.core.FetchImage(ctx, source, target, comic, chapter)
	if err != nil {
		writeError(w, err)
		return
	}
	if ctype == "" {
		ctype = http.DetectContentType(data)
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Cache-Control", "private, max-age=600")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	_, _ = w.Write(data)
}

func writeError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusBadGateway)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

// Base is the origin a WebView loads the scheme from.
//
// WebView2 (Windows) serves custom schemes at http://<scheme>.localhost; the
// WKWebView and WebKitGTK backends serve them as <scheme>://localhost. Pages
// then use relative URLs, which resolve to the same origin either way.
func Base() string {
	if runtime.GOOS == "windows" {
		return "http://" + Scheme + ".localhost"
	}
	return Scheme + "://localhost"
}

// ReaderURL is the page's address for one chapter.
func ReaderURL(source, comicID, chapterID, title string) string {
	return ReaderURLNear(source, comicID, chapterID, title, Neighbour{}, Neighbour{})
}

// Neighbour is a chapter offered at the end of the current one.
type Neighbour struct {
	ID    string
	Title string
}

// ReaderURLNear is ReaderURL with the chapters on either side, so the page
// can offer them and ask Go to switch.
func ReaderURLNear(source, comicID, chapterID, title string, prev, next Neighbour) string {
	q := url.Values{}
	q.Set("source", source)
	q.Set("comic", comicID)
	q.Set("chapter", chapterID)
	q.Set("title", title)
	if prev.ID != "" {
		q.Set("prev", prev.ID)
		q.Set("prevtitle", prev.Title)
	}
	if next.ID != "" {
		q.Set("next", next.ID)
		q.Set("nexttitle", next.Title)
	}
	return Base() + "/reader?" + q.Encode()
}

// ImageURL is the scheme's address for one image of a chapter.
func ImageURL(source, comicID, chapterID, imageURL string) string {
	q := url.Values{}
	q.Set("source", source)
	q.Set("comic", comicID)
	q.Set("chapter", chapterID)
	q.Set("u", base64.RawURLEncoding.EncodeToString([]byte(imageURL)))
	return Base() + "/img?" + q.Encode()
}
