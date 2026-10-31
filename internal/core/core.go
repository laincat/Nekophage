// Package core links VeneraCat Core into the desktop app: one JavaScript
// runtime and one plugin registry, with the network, storage, cookies,
// WebView verification and dialogs the client provides.
package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/laincat/nekophage/internal/cookies"
	"github.com/laincat/nekophage/internal/store"
	coreimage "github.com/laincat/veneracat-core/comic/image"
	"github.com/laincat/veneracat-core/comic/js"
	"github.com/laincat/veneracat-core/host"
	"github.com/laincat/veneracat-core/plugin"
)

// Options configure Open.
type Options struct {
	// DataDir holds sources, settings and plugin records.
	DataDir string
	Locale  string
	// AppVersion is what sources see and what index entries are compared
	// with (minAppVersion).
	AppVersion string
	// UserAgent is the default request agent.
	UserAgent   string
	MemoryLimit int64
	// UI, Clipboard and Verifier are the client's services. A nil Verifier
	// means challenges fail instead of opening a browser.
	UI        js.UIHandler
	Clipboard js.Clipboard
	Verifier  host.Verifier
	// VerifierFactory builds the verifier once the cookie jar exists; it is
	// used when Verifier is nil.
	VerifierFactory func(jar *cookies.Jar) host.Verifier
}

// Core is one profile: its runtime, registry and stores.
type Core struct {
	rt    *js.Runtime
	reg   *plugin.Registry
	store *store.Store
	jar   *cookies.Jar
	tr    *Transport
	opts  Options

	// cursors holds the paging cursor that led to each page of a source's
	// search, for a source that pages by cursor rather than by page number.
	mu      sync.Mutex
	cursors map[string]map[int]string
}

// Open assembles Core. Close it when the app quits.
func Open(opts Options) (*Core, error) {
	if opts.DataDir == "" {
		return nil, errors.New("core: DataDir is required")
	}
	if opts.AppVersion == "" {
		opts.AppVersion = "0.1.0"
	}
	if opts.Locale == "" {
		opts.Locale = "zh_CN"
	}
	if opts.UserAgent == "" {
		opts.UserAgent = DefaultUserAgent
	}
	if opts.MemoryLimit <= 0 {
		opts.MemoryLimit = 256 << 20
	}
	st, err := store.Open(opts.DataDir)
	if err != nil {
		return nil, err
	}
	jar, err := cookies.Open(filepath.Join(opts.DataDir, "cookies.json"))
	if err != nil {
		return nil, err
	}
	tr := NewTransport(jar, opts.UserAgent, 40*time.Second)
	verifier := opts.Verifier
	if verifier == nil && opts.VerifierFactory != nil {
		verifier = opts.VerifierFactory(jar)
	}
	rt, err := js.New(js.Options{MemoryLimit: opts.MemoryLimit}, js.Services{
		Storage:   st,
		Env:       js.Environment{Locale: opts.Locale, Platform: platformName(), AppVersion: opts.AppVersion, DefaultUA: opts.UserAgent},
		Log:       logLine,
		Transport: tr,
		Verifier:  verifier,
		Cookies:   jar,
		UI:        opts.UI,
		Clipboard: opts.Clipboard,
	})
	if err != nil {
		return nil, err
	}
	return &Core{rt: rt, reg: plugin.New(rt, tr), store: st, jar: jar, tr: tr, opts: opts,
		cursors: map[string]map[int]string{}}, nil
}

// Close releases the runtime.
func (c *Core) Close() error { return c.rt.Close() }

// DataDir is the directory the app keeps its data in.
func (c *Core) DataDir() string { return c.opts.DataDir }

// Store, Jar and Transport are the app's own handles to the same services.
func (c *Core) Store() *store.Store { return c.store }
func (c *Core) Jar() *cookies.Jar   { return c.jar }

// ComicPartition is the cookie partition a comic source uses.
//
// Every request and every challenge names it, so anything else that writes
// cookies — a backup restore, a test — has to name it the same way. A restore
// that used the bare key stored cookies nothing would ever send.
func ComicPartition(key string) string { return "comic:" + key }
func (c *Core) Transport() *Transport  { return c.tr }
func (c *Core) Runtime() *js.Runtime   { return c.rt }

// logLine writes script and Core log lines to stderr, prefixed so they are
// easy to filter in a terminal.
func logLine(level, title, content string) {
	if content == "" {
		return
	}
	fmt.Fprintf(os.Stderr, "[%s] %s: %s\n", level, title, content)
}

// --- plugins ---

// Install installs a comic source script.
func (c *Core) Install(ctx context.Context, body, origin, indexURL string) (*plugin.Record, error) {
	return c.reg.Install(ctx, plugin.TypeComic, body, origin, indexURL, "")
}

// Replace updates an installed source.
func (c *Core) Replace(ctx context.Context, key, body, origin, indexURL string) (*plugin.Record, error) {
	return c.reg.Replace(ctx, key, plugin.TypeComic, body, origin, indexURL, "")
}

// Records lists installed plugins with bodies, for persistence.
func (c *Core) Records() []plugin.Record { return c.reg.List() }

// Plugins lists installed plugins without bodies.
func (c *Core) Plugins() []plugin.Record {
	list := c.reg.List()
	for i := range list {
		list[i].Body = ""
	}
	return list
}

// Restore re-activates the saved records and returns the ones that failed.
func (c *Core) Restore(ctx context.Context, recs []plugin.Record) map[string]error {
	return c.reg.Restore(ctx, recs)
}

// Remove uninstalls a plugin and forgets what it stored.
func (c *Core) Remove(ctx context.Context, key string) error {
	if err := c.reg.Remove(ctx, key); err != nil {
		return err
	}
	c.jar.Clear(ComicPartition(key))
	return c.store.RemoveSource(key)
}

// SetEnabled turns a plugin on or off.
func (c *Core) SetEnabled(ctx context.Context, key string, on bool) error {
	return c.reg.SetEnabled(ctx, key, on)
}

// Has reports whether a comic source defines path, such as "search.load".
func (c *Core) Has(ctx context.Context, key, path string) bool {
	rt, err := c.reg.Comic(key)
	if err != nil {
		return false
	}
	ok, err := rt.Has(ctx, key, path)
	return err == nil && ok
}

// Call invokes ComicSource.<path> and returns the JSON result.
func (c *Core) Call(ctx context.Context, key, path string, args ...any) (json.RawMessage, error) {
	rt, err := c.reg.Comic(key)
	if err != nil {
		return nil, err
	}
	return rt.Call(ctx, key, path, args...)
}

// --- comic data ---

// Comic is one comic in a list.
type Comic struct {
	ID          string
	Title       string
	Subtitle    string
	Cover       string
	Tags        []string
	Description string
}

// Page is one page of search results.
type Page struct {
	Comics  []Comic
	MaxPage int
	Next    string
}

// ExplorePage is one browse a source offers: a tab in the interface.
type ExplorePage struct {
	Title string `json:"title"`
	// Type is "singlePageWithMultiPart", "multiPartPage",
	// "multiPageComicList" or "mixed".
	Type string `json:"type"`
}

// ExplorePages lists a source's browses. A source without any returns nil.
func (c *Core) ExplorePages(ctx context.Context, key string) ([]ExplorePage, error) {
	rt, err := c.reg.Comic(key)
	if err != nil {
		return nil, err
	}
	pages, err := rt.ExplorePages(ctx, key)
	if err != nil {
		return nil, err
	}
	out := make([]ExplorePage, 0, len(pages))
	for _, p := range pages {
		out = append(out, ExplorePage{Title: p.Title, Type: p.Type})
	}
	return out, nil
}

// ExploreSection is a titled group of comics in one browse.
type ExploreSection struct {
	Title  string
	Comics []Comic
}

// ExploreResult is what one browse returned.
type ExploreResult struct {
	Sections []ExploreSection
	// MaxPage is for "multiPageComicList" browses; 0 when unknown.
	MaxPage int
}

// Explore loads one browse. page is the page number for
// "multiPageComicList" browses and the index for "mixed" ones.
func (c *Core) Explore(ctx context.Context, key string, index, page int) (ExploreResult, error) {
	if page < 1 {
		page = 1
	}
	path := fmt.Sprintf("explore[%d].load", index)
	switch c.exploreType(ctx, key, index) {
	case "multiPageComicList":
		raw, err := c.Call(ctx, key, path, page)
		if err != nil {
			return ExploreResult{}, err
		}
		p, err := parsePage(raw)
		if err != nil {
			return ExploreResult{}, err
		}
		return ExploreResult{Sections: []ExploreSection{{Comics: p.Comics}}, MaxPage: p.MaxPage}, nil
	case "multiPartPage":
		raw, err := c.Call(ctx, key, path)
		if err != nil {
			return ExploreResult{}, err
		}
		return parseParts(raw)
	case "mixed":
		raw, err := c.Call(ctx, key, path, page)
		if err != nil {
			return ExploreResult{}, err
		}
		res, err := parseParts(raw, "data")
		if err != nil {
			return ExploreResult{}, err
		}
		var env struct {
			MaxPage json.Number `json:"maxPage"`
		}
		if json.Unmarshal(raw, &env) == nil {
			if n, err := env.MaxPage.Int64(); err == nil {
				res.MaxPage = int(n)
			}
		}
		return res, nil
	default: // singlePageWithMultiPart: {partTitle: [comic, ...]}
		raw, err := c.Call(ctx, key, path)
		if err != nil {
			return ExploreResult{}, err
		}
		return parseOrderedParts(raw), nil
	}
}

// parseOrderedParts reads {partTitle: [comic, ...]}, the shape of a
// singlePageWithMultiPart browse, keeping the source's part order.
func parseOrderedParts(raw json.RawMessage) ExploreResult {
	var out ExploreResult
	for _, f := range objectFields(raw) {
		var items []json.RawMessage
		if json.Unmarshal(f.raw, &items) != nil {
			continue
		}
		// The part's title is the field's name: its value is the comics.
		section := ExploreSection{Title: f.name}
		for _, item := range items {
			if comic := parseComic(item); comic.ID != "" {
				section.Comics = append(section.Comics, comic)
			}
		}
		if len(section.Comics) > 0 {
			out.Sections = append(out.Sections, section)
		}
	}
	return out
}

// exploreType is the declared type of browse index.
func (c *Core) exploreType(ctx context.Context, key string, index int) string {
	pages, err := c.ExplorePages(ctx, key)
	if err != nil || index < 0 || index >= len(pages) {
		return ""
	}
	return pages[index].Type
}

// parseParts reads a browse whose value is a list of parts
// ([{title, comics}] or, with field set, {field: [...]}); a plain list of
// comics becomes one untitled part.
func parseParts(raw json.RawMessage, field ...string) (ExploreResult, error) {
	var out ExploreResult
	body := raw
	for _, name := range field {
		found := false
		for _, f := range objectFields(raw) {
			if f.name == name {
				body, found = f.raw, true
				break
			}
		}
		if !found {
			return out, nil
		}
	}
	var list []json.RawMessage
	if json.Unmarshal(body, &list) != nil {
		return out, nil
	}
	for _, item := range list {
		var part struct {
			Title  flexString        `json:"title"`
			Comics []json.RawMessage `json:"comics"`
		}
		if json.Unmarshal(item, &part) == nil && part.Comics != nil {
			section := ExploreSection{Title: string(part.Title)}
			for _, c := range part.Comics {
				if comic := parseComic(c); comic.ID != "" {
					section.Comics = append(section.Comics, comic)
				}
			}
			if len(section.Comics) > 0 {
				out.Sections = append(out.Sections, section)
			}
			continue
		}
		// A bare list of comics: a part without a title.
		var comics []json.RawMessage
		if json.Unmarshal(item, &comics) == nil {
			section := ExploreSection{}
			for _, c := range comics {
				if comic := parseComic(c); comic.ID != "" {
					section.Comics = append(section.Comics, comic)
				}
			}
			if len(section.Comics) > 0 {
				out.Sections = append(out.Sections, section)
			}
		}
	}
	return out, nil
}

// SearchOptionGroup is one group of a source's search.optionList.
type SearchOptionGroup struct {
	Label   string   `json:"label"`
	Type    string   `json:"type"`
	Values  []string `json:"values"`
	Labels  []string `json:"labels"`
	Default string   `json:"default"`
	// DefaultValues is what a multi-select group starts with; the source
	// receives it as a JSON array string.
	DefaultValues []string `json:"defaultValues"`
}

// IsMultiSelect reports whether the group takes several values at once.
func (g SearchOptionGroup) IsMultiSelect() bool { return g.Type == "multi-select" }

// SourceSetting is one entry of a source's settings declaration: what the
// source wants drawn in its settings panel.
type SourceSetting struct {
	Key   string `json:"key"`
	Title string `json:"title"`
	// Type is "select", "switch", "input" or "callback".
	Type string `json:"type"`
	// Default is the value the source wants when nothing is stored, kept as
	// JSON because a source may declare a number or a boolean.
	Default json.RawMessage `json:"default,omitempty"`
	// Options are the choices of a select, in the source's order.
	Options []SettingOption `json:"options,omitempty"`
	// Validator is a regular expression an input's text must match.
	Validator string `json:"validator,omitempty"`
}

// SettingOption is one choice of a select.
type SettingOption struct {
	Value json.RawMessage `json:"value"`
	Text  string          `json:"text,omitempty"`
}

// Settings reads a source's settings declaration.
func (c *Core) Settings(ctx context.Context, key string) ([]SourceSetting, error) {
	rt, err := c.reg.Comic(key)
	if err != nil {
		return nil, err
	}
	list, err := rt.Settings(ctx, key)
	if err != nil {
		return nil, err
	}
	out := make([]SourceSetting, 0, len(list))
	for _, s := range list {
		setting := SourceSetting{Key: s.Key, Title: s.Title, Type: s.Type,
			Default: s.Default, Validator: s.Validator}
		for _, o := range s.Options {
			setting.Options = append(setting.Options, SettingOption{Value: o.Value, Text: o.Text})
		}
		out = append(out, setting)
	}
	return out, nil
}

// SearchOptions reads a source's search filters and their defaults.
func (c *Core) SearchOptions(ctx context.Context, key string) ([]SearchOptionGroup, error) {
	rt, err := c.reg.Comic(key)
	if err != nil {
		return nil, err
	}
	groups, err := rt.SearchOptions(ctx, key)
	if err != nil {
		return nil, err
	}
	out := make([]SearchOptionGroup, 0, len(groups))
	for _, g := range groups {
		out = append(out, SearchOptionGroup{Label: g.Label, Type: g.Type, Values: g.Values,
			Labels: g.Labels, Default: g.Default, DefaultValues: g.DefaultValues})
	}
	return out, nil
}

// DefaultOptions is one value per search group: the group's default, which
// search.load expects.
func (c *Core) DefaultOptions(ctx context.Context, key string) ([]string, error) {
	groups, err := c.SearchOptions(ctx, key)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(groups))
	for _, g := range groups {
		if g.IsMultiSelect() {
			// A multi-select group's value is a JSON array of the chosen keys,
			// which is what the source parses out of options[i].
			chosen := g.DefaultValues
			if len(chosen) == 0 {
				chosen = g.Values
			}
			encoded, err := json.Marshal(chosen)
			if err != nil {
				return nil, err
			}
			out = append(out, string(encoded))
			continue
		}
		out = append(out, g.Default)
	}
	return out, nil
}

// Search cursors, per source and page. A source that pages by cursor hands one
// back with each page; the interface asks for a page number, so the cursor that
// led to each page is kept and replayed. A cursor belongs to one search, so a
// new search (page 1) clears what the previous one left.
func (c *Core) searchCursor(key string, page int) any {
	c.mu.Lock()
	defer c.mu.Unlock()
	if page <= 1 {
		delete(c.cursors, key)
		return nil
	}
	if v, ok := c.cursors[key][page]; ok {
		return v
	}
	return nil
}

func (c *Core) rememberSearchCursor(key string, page int, next string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if page <= 1 || c.cursors[key] == nil {
		c.cursors[key] = map[int]string{}
	}
	c.cursors[key][page+1] = next
}

// Search runs a search against one source.
//
// A source pages its results in one of two ways, and the official client
// supports both:
//
//   - search.load(keyword, options, page) takes a page number;
//   - search.loadNext(keyword, options, next) takes whatever cursor the
//     previous page returned in `next`.
//
// A source that declares both is asked for search.load, because a page number
// is what the interface offers. A source that declares only search.loadNext is
// asked for that instead: calling search.load there would report "source does
// not define this function" for a source that works in the official client.
// options holds one chosen value per group of SearchOptions, usually
// DefaultOptions.
func (c *Core) Search(ctx context.Context, key, keyword string, options []string, page int) (Page, error) {
	if options == nil {
		options = []string{}
	}
	if c.Has(ctx, key, "search.load") {
		raw, err := c.Call(ctx, key, "search.load", keyword, options, page)
		if err != nil {
			return Page{}, err
		}
		return parsePage(raw)
	}
	if c.Has(ctx, key, "search.loadNext") {
		// The cursor is the source's own; the first page is when it has none.
		var cursor any
		if page > 1 {
			cursor = c.searchCursor(key, page)
		}
		raw, err := c.Call(ctx, key, "search.loadNext", keyword, options, cursor)
		if err != nil {
			return Page{}, err
		}
		out, err := parsePage(raw)
		if err != nil {
			return Page{}, err
		}
		c.rememberSearchCursor(key, page, out.Next)
		return out, nil
	}
	return Page{}, fmt.Errorf("core: %s offers no search entry point", key)
}

// Chapter is one chapter of a comic.
type Chapter struct {
	ID    string
	Title string
	// Group is the chapter group's name, empty for a flat chapter list.
	Group string
}

// TagGroup is one tag group, such as 作者 → [..].
type TagGroup struct {
	Name   string
	Values []string
}

// Details is comic.loadInfo's result.
type Details struct {
	Title       string
	Subtitle    string
	Cover       string
	Description string
	Tags        []TagGroup
	// Chapters keep the source's order, which is reading order.
	Chapters []Chapter
}

// Details runs comic.loadInfo(id).
func (c *Core) Details(ctx context.Context, key, comicID string) (Details, error) {
	raw, err := c.Call(ctx, key, "comic.loadInfo", comicID)
	if err != nil {
		return Details{}, err
	}
	return parseDetails(raw)
}

// ChapterImages runs comic.loadEp(comicID, chapterID); chapterID may be "".
func (c *Core) ChapterImages(ctx context.Context, key, comicID, chapterID string) ([]string, error) {
	var arg any
	if chapterID != "" {
		arg = chapterID
	}
	raw, err := c.Call(ctx, key, "comic.loadEp", comicID, arg)
	if err != nil {
		return nil, err
	}
	var out struct {
		Images []string `json:"images"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("core: loadEp: %w", err)
	}
	return out.Images, nil
}

// ImageRequest is how to fetch one image, from onImageLoad/onThumbnailLoad.
type ImageRequest struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    []byte
	// Handle names the config's callbacks inside the source's runtime, and
	// Callbacks says which of onResponse/onLoadFailed it carries. A source
	// that only names an address leaves both empty.
	Handle    string
	Callbacks []string
	// ModifyImage is the source's pixel-buffer script, when it supplied one.
	// It is a string of JavaScript, run by Core over the decoded image.
	ModifyImage string
}

// HasCallback reports whether the config carries name.
func (r ImageRequest) HasCallback(name string) bool {
	for _, c := range r.Callbacks {
		if c == name {
			return true
		}
	}
	return false
}

// ImageRequest asks the source how to fetch an image. thumbnail asks
// onThumbnailLoad instead of onImageLoad, for covers. Without either hook the
// URL is fetched as it is.
func (c *Core) ImageRequest(ctx context.Context, key, url, comicID, chapterID string, thumbnail bool) (ImageRequest, error) {
	req := ImageRequest{URL: url, Method: "GET"}
	rt, err := c.reg.Comic(key)
	if err != nil {
		return req, err
	}
	// Core keeps the config's callbacks and reports a handle for them; a
	// source without a hook reports ErrNoImageCallback and the address is
	// fetched as it is.
	cfg, err := rt.ImageConfig(ctx, key, url, comicID, chapterID, thumbnail)
	if errors.Is(err, js.ErrNoImageCallback) {
		return req, nil
	}
	if err != nil {
		return req, err
	}
	if cfg.URL == "" && cfg.Headers == nil && cfg.Data == "" && cfg.Handle == "" {
		return req, nil // the source returned nothing useful
	}
	if cfg.URL != "" {
		req.URL = cfg.URL
	}
	if cfg.Method != "" {
		req.Method = strings.ToUpper(cfg.Method)
	}
	req.Headers = cfg.Headers
	if cfg.Data != "" {
		req.Body = []byte(cfg.Data)
	}
	req.Handle = cfg.Handle
	req.Callbacks = cfg.Callbacks
	req.ModifyImage = cfg.ModifyImage
	return req, nil
}

// AppVersion is what sources see and what index entries are compared with.
func (c *Core) AppVersion() string { return c.opts.AppVersion }

// FetchThumbnail is FetchImage through a source's cover hook.
func (c *Core) FetchThumbnail(ctx context.Context, key, rawURL, comicID string) ([]byte, string, error) {
	req, err := c.ImageRequest(ctx, key, rawURL, comicID, "", true)
	if err != nil {
		return nil, "", err
	}
	return c.doImage(ctx, key, req)
}

// MaxImageBytes bounds one image.
const MaxImageBytes = 32 << 20

// FetchImage downloads one image through the source's rules and returns its
// bytes and content type.
func (c *Core) FetchImage(ctx context.Context, key, rawURL, comicID, chapterID string) ([]byte, string, error) {
	req, err := c.ImageRequest(ctx, key, rawURL, comicID, chapterID, false)
	if err != nil {
		return nil, "", err
	}
	return c.doImage(ctx, key, req)
}

// DoImageRequest fetches an image a source has already described, the way
// FetchImage would after asking the source. A caller that needs to inspect the
// resolved address first — the reader page, which must not fetch anything but
// http(s) — resolves with ImageRequest and then calls this.
func (c *Core) DoImageRequest(ctx context.Context, key string, req ImageRequest) ([]byte, string, error) {
	return c.doImage(ctx, key, req)
}

func (c *Core) doImage(ctx context.Context, key string, req ImageRequest) ([]byte, string, error) {
	// A config may name several mirrors and a body transform. Both live in the
	// source's runtime, so a failure asks for the next mirror and a success
	// hands the body to onResponse. The hop count bounds a source that keeps
	// answering with another failing address.
	const maxHops = 4
	var lastErr error
	for hop := 0; hop < maxHops; hop++ {
		data, ctype, err := c.fetchImageOnce(ctx, key, req)
		if err == nil && looksLikeHTML(data) && req.HasCallback("onLoadFailed") {
			// A mirror can answer 200 with an error page. The official client
			// treats that as a failed load because decoding it fails, and asks
			// the source for the next mirror; here the fetch itself succeeded,
			// so the body is what has to say so.
			err = errors.New("core: image response is an HTML page")
		}
		if err == nil {
			if req.Handle != "" && req.HasCallback("onResponse") {
				rt, rerr := c.reg.Comic(key)
				if rerr != nil {
					return nil, "", rerr
				}
				got, aerr := rt.ApplyImageResponse(ctx, key, req.Handle, data)
				if aerr != nil && !errors.Is(aerr, js.ErrNoImageCallback) {
					return nil, "", aerr
				}
				if aerr == nil {
					// The bytes are whatever the source produced now, so the
					// server's label no longer describes them.
					data, ctype = got, ""
				}
			}
			if req.ModifyImage != "" {
				// A source may ask for a pixel pass: jm reorders the blocks of a
				// long strip, and several sources rotate a page. The result is
				// re-encoded, so the server's label no longer applies.
				fixed, merr := c.modifyImage(ctx, key, req.ModifyImage, data)
				if errors.Is(merr, errUndecodable) {
					// The source asked for a pixel pass but this build cannot
					// read the format it sent (avif, for one). Showing the image
					// as it arrived beats failing the page over an ordering the
					// reader may not even notice.
					return data, ctype, nil
				}
				if merr != nil {
					return nil, "", merr
				}
				return fixed, "image/png", nil
			}
			return data, ctype, nil
		}
		lastErr = err
		if req.Handle == "" || !req.HasCallback("onLoadFailed") {
			return nil, "", err
		}
		rt, rerr := c.reg.Comic(key)
		if rerr != nil {
			return nil, "", err
		}
		next, ferr := rt.ImageFallback(ctx, key, req.Handle)
		if ferr != nil || next.URL == "" {
			return nil, "", err
		}
		req.URL, req.Headers, req.Handle, req.Callbacks = next.URL, next.Headers, next.Handle, next.Callbacks
		if next.Method != "" {
			req.Method = strings.ToUpper(next.Method)
		}
		if next.Data != "" {
			req.Body = []byte(next.Data)
		}
	}
	return nil, "", lastErr
}

// looksLikeHTML reports whether a body is a web page rather than an image. A
// mirror that is out of quota answers 200 with an error page, and the source is
// the only one that knows another mirror.
func looksLikeHTML(data []byte) bool {
	head := data
	if len(head) > 512 {
		head = head[:512]
	}
	head = bytes.TrimLeft(head, " \t\r\n\ufeff")
	lower := bytes.ToLower(head)
	for _, prefix := range []string{"<!doctype html", "<html", "<?xml", "<head", "<body"} {
		if bytes.HasPrefix(lower, []byte(prefix)) {
			return true
		}
	}
	return false
}

// fetchImageOnce performs one hop of a config: one address, one request.
func (c *Core) fetchImageOnce(ctx context.Context, key string, req ImageRequest) ([]byte, string, error) {
	// A source may hand back an image address without its scheme, such as
	// "//images.example.com/1.jpg". RFC 3986 says such an address inherits
	// the scheme of the page, so the scheme this source last used is the one
	// to use. The result still has to be an http(s) URL: nothing else gets
	// through.
	req.URL = host.NormalizeURL(req.URL, c.tr.SchemeOf(ComicPartition(key))+"://")
	u, err := url.Parse(req.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, "", errors.New("core: image URL is not http(s)")
	}
	header := http.Header{}
	for k, v := range req.Headers {
		header.Set(k, v)
	}
	resp, err := c.tr.Do(ctx, &host.Request{Method: req.Method, URL: req.URL, Header: header, Body: req.Body,
		Cookies: true, Partition: ComicPartition(key)})
	if err != nil {
		return nil, "", err
	}
	if resp.Status < 200 || resp.Status > 299 {
		return nil, "", fmt.Errorf("core: image HTTP %d", resp.Status)
	}
	if len(resp.Body) > MaxImageBytes {
		return nil, "", ErrTooLarge
	}
	ctype := resp.Header.Get("Content-Type")
	if i := strings.IndexByte(ctype, ';'); i >= 0 {
		ctype = ctype[:i]
	}
	return resp.Body, strings.TrimSpace(ctype), nil
}

// --- parsing ---

func parsePage(raw json.RawMessage) (Page, error) {
	var envelope struct {
		Comics  json.RawMessage `json:"comics"`
		MaxPage json.Number     `json:"maxPage"`
		Next    string          `json:"next"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return Page{}, fmt.Errorf("core: search.load: %w", err)
	}
	var items []json.RawMessage
	if len(envelope.Comics) > 0 {
		// One malformed comic must not hide the rest of the page.
		_ = json.Unmarshal(mustArray(envelope.Comics), &items)
	}
	p := Page{Next: envelope.Next}
	if n, err := envelope.MaxPage.Int64(); err == nil {
		p.MaxPage = int(n)
	}
	seen := map[string]bool{}
	for _, item := range items {
		c := parseComic(item)
		if c.ID == "" || seen[c.ID] {
			continue
		}
		seen[c.ID] = true
		p.Comics = append(p.Comics, c)
	}
	return p, nil
}

func parseComic(raw json.RawMessage) Comic {
	var c Comic
	var f struct {
		ID          flexString `json:"id"`
		Title       flexString `json:"title"`
		Subtitle    flexString `json:"subtitle"`
		SubTitle    flexString `json:"subTitle"`
		Cover       flexString `json:"cover"`
		Tags        []string   `json:"tags"`
		Description flexString `json:"description"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		return Comic{}
	}
	c.ID, c.Title, c.Cover, c.Description = string(f.ID), string(f.Title), string(f.Cover), string(f.Description)
	c.Subtitle = string(f.Subtitle)
	if c.Subtitle == "" {
		c.Subtitle = string(f.SubTitle)
	}
	c.Tags = f.Tags
	return c
}

func parseDetails(raw json.RawMessage) (Details, error) {
	var f struct {
		Title       flexString      `json:"title"`
		Subtitle    flexString      `json:"subtitle"`
		SubTitle    flexString      `json:"subTitle"`
		Cover       flexString      `json:"cover"`
		Description flexString      `json:"description"`
		Tags        json.RawMessage `json:"tags"`
		Chapters    json.RawMessage `json:"chapters"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		return Details{}, fmt.Errorf("core: comic.loadInfo: %w", err)
	}
	d := Details{Title: string(f.Title), Cover: string(f.Cover), Description: string(f.Description)}
	d.Subtitle = string(f.Subtitle)
	if d.Subtitle == "" {
		d.Subtitle = string(f.SubTitle)
	}
	for _, field := range objectFields(f.Tags) {
		var values []string
		if err := json.Unmarshal(field.raw, &values); err == nil {
			d.Tags = append(d.Tags, TagGroup{Name: field.name, Values: values})
		}
	}
	// Chapters are either a flat {id: title} or grouped
	// {group: {id: title}}; the source's key order is reading order.
	for _, field := range objectFields(f.Chapters) {
		var nested []rawField
		if len(field.raw) > 0 && field.raw[0] == '{' {
			nested = objectFields(field.raw)
		}
		if len(nested) == 0 {
			d.Chapters = append(d.Chapters, Chapter{ID: field.name, Title: field.text()})
			continue
		}
		for _, n := range nested {
			d.Chapters = append(d.Chapters, Chapter{ID: n.name, Title: n.text(), Group: field.name})
		}
	}
	return d, nil
}

type rawField struct {
	name string
	raw  json.RawMessage
}

// text is the field's value as text: a string, or a number as written.
func (f rawField) text() string {
	var s string
	if err := json.Unmarshal(f.raw, &s); err == nil {
		return s
	}
	return strings.Trim(string(f.raw), `"`)
}

// objectFields decodes a JSON object in document order, so key order
// survives (chapters depend on it).
func objectFields(raw json.RawMessage) []rawField {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	tok, err := dec.Token()
	if err != nil {
		return nil
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil
	}
	var out []rawField
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return out
		}
		key, _ := keyTok.(string)
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return out
		}
		out = append(out, rawField{name: key, raw: value})
	}
	return out
}

func mustArray(raw json.RawMessage) json.RawMessage {
	if len(raw) > 0 && raw[0] == '[' {
		return raw
	}
	return json.RawMessage("[]")
}

// flexString accepts a JSON string or number.
type flexString string

func (f *flexString) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = flexString(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		*f = flexString(n.String())
		return nil
	}
	*f = ""
	return nil
}

// SortedKeys is a helper for stable UI lists.
func SortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// platformName is what scripts see from getPlatform: the operating system,
// spelled as Dart's Platform.operatingSystem spells it.
func platformName() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos"
	default:
		return runtime.GOOS
	}
}

// DefaultUserAgent is a browser-like agent. Sites block unknown agents.
const DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"

// maxModifyPixels bounds a bitmap handed to a source's modifyImage script.
// Decoding a very large image just to reorder it costs far more memory than
// the reader will ever show, so a bigger one is passed through untouched.
const maxModifyPixels = 64 << 20

// errUndecodable marks a body this build cannot decode, which is not the
// source's mistake.
var errUndecodable = errors.New("core: image format not supported")

// modifyImage runs a source's pixel-buffer script and returns PNG bytes.
//
// The script works on RGBA and Core only moves pixels; decoding the fetched
// bytes and encoding the result happen here, which is what the comic/image
// package leaves to the client.
func (c *Core) modifyImage(ctx context.Context, key, script string, data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, errUndecodable
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 || int64(w)*int64(h) > maxModifyPixels {
		return nil, fmt.Errorf("core: modifyImage: %dx%d is out of range", w, h)
	}
	pix := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			i := (y*w + x) * 4
			pix[i], pix[i+1], pix[i+2], pix[i+3] = byte(r>>8), byte(g>>8), byte(b>>8), byte(a>>8)
		}
	}
	rt, err := c.reg.Comic(key)
	if err != nil {
		return nil, err
	}
	out, err := rt.ModifyImage(ctx, script, coreimage.Bitmap{W: w, H: h, Pix: pix})
	if err != nil {
		return nil, err
	}
	rgba := &image.RGBA{Pix: out.Pix, Stride: out.W * 4, Rect: image.Rect(0, 0, out.W, out.H)}
	var buf bytes.Buffer
	if err := png.Encode(&buf, rgba); err != nil {
		return nil, fmt.Errorf("core: modifyImage: %w", err)
	}
	return buf.Bytes(), nil
}
