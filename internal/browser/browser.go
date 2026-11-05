// Package browser does the parts of a source that need a real web engine: a
// human-verification page, and (later) finding a video's stream.
//
// The window is MyGo native UI holding a WebView, so a challenge page shows
// beside the app's own windows with the browser's own rendering and
// JavaScript. Go watches it and hands what it finds back to Core.
package browser

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/laincat/nekophage/internal/cookies"
	"github.com/laincat/veneracat-core/host"
)

// Verifier shows challenges in a WebView.
//
// Every window of the app shares one browser data store, so only one
// challenge runs at a time: the store is cleared before a page loads and
// after it closes, and pages are loaded one at a time, so a source never
// sees another source's login.
type Verifier struct {
	jar *cookies.Jar
}

// browserStoreMu serializes every use of the shared WebView cookie store.
var browserStoreMu sync.Mutex

// NewVerifier builds a Verifier over the app's cookie jars.
func NewVerifier(jar *cookies.Jar) *Verifier { return &Verifier{jar: jar} }

// Verify implements host.Verifier: it shows the challenge until the user
// passes it or closes the window.
func (v *Verifier) Verify(ctx context.Context, ch *host.Challenge) (*host.Clearance, error) {
	if ch == nil || ch.URL == "" {
		return nil, errors.New("browser: challenge without a URL")
	}
	browserStoreMu.Lock()
	defer browserStoreMu.Unlock()

	// A page must start from a clean store, or it inherits the last one's
	// clearance and the cookie it should set is never set.
	if err := mygo.Cookies.Clear(); err != nil {
		return nil, err
	}
	defer func() { _ = mygo.Cookies.Clear() }()

	s := &challenge{ch: ch, jar: v.jar, done: make(chan struct{})}
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:   "需要验证",
		Width:   520,
		Height:  780,
		Content: ui.View(s.draw),
	})
	view, err := win.NewWebView(mygo.WebViewOptions{URL: ch.URL})
	if err != nil {
		win.Close()
		return nil, err
	}
	s.win, s.web = win, view
	win.Show()
	win.Focus()

	clearance, err := s.watch(ctx)
	win.Close()
	return clearance, err
}

// challenge is one verification window.
type challenge struct {
	ch  *host.Challenge
	jar *cookies.Jar
	win *mygo.Window
	web *mygo.WebView

	once   sync.Once
	done   chan struct{}
	closed bool
}

// cancel is the window's 关闭 button: the user gives up.
func (c *challenge) cancel() {
	c.once.Do(func() { c.closed = true; close(c.done) })
}

// watch polls the page until Core's done rule holds.
func (c *challenge) watch(ctx context.Context) (*host.Clearance, error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	// Some pages pass by script without navigating, so the rule is checked
	// on a timer rather than on load events.
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-c.done:
			return nil, host.ErrChallengeDismissed
		case <-ticker.C:
			ua, cookies, params, ok := c.check()
			if !ok {
				debugState(c)
				continue
			}
			c.jar.Import(c.ch.Partition, cookies)
			if os.Getenv("NKDEBUG") != "" {
				log.Printf("verify: partition %q, %d cookie(s) imported, jar now has %d for %s",
					c.ch.Partition, len(cookies), len(c.jar.Get(c.ch.Partition, c.ch.URL)), c.ch.URL)
			}
			// A challenge whose clearance is in the URL hands back parameters
			// the retry must repeat; a cookie-based one hands back none.
			return &host.Clearance{UserAgent: ua, Cookies: cookies, Query: params}, nil
		}
	}
}

// check evaluates the done rule and collects the browser's cookies.
func (c *challenge) check() (string, []*http.Cookie, map[string]string, bool) {
	page := c.web.Page()
	raw, err := page.Eval("document.documentElement.outerHTML")
	if err != nil {
		return "", nil, nil, false
	}
	html, _ := raw.(string)
	// A rule with nothing to check is never satisfied: an empty group means "no
	// condition", not "vacuously true", so a rule written for a cookie-based
	// challenge must not pass just because a URL-based one has no markers.
	if c.ch.Done.Empty() {
		return "", nil, nil, false
	}
	var params map[string]string
	if c.ch.Done.SameHost || len(c.ch.Done.RequiredQueryParams) > 0 {
		rawURL, err := page.Eval("location.href")
		if err != nil {
			return "", nil, nil, false
		}
		href, _ := rawURL.(string)
		u, err := url.Parse(href)
		if err != nil {
			return "", nil, nil, false
		}
		if c.ch.Done.SameHost {
			// The challenge page lives on another host than the address it
			// guards; being back on the guarded host is what says it finished.
			// A window that has not loaded anything yet has an empty host, so
			// this cannot pass before the first navigation.
			want, err := url.Parse(c.ch.URL)
			if err != nil || u.Host == "" || !strings.EqualFold(u.Host, want.Host) {
				return "", nil, nil, false
			}
		}
		q := u.Query()
		for _, want := range c.ch.Done.RequiredQueryParams {
			if want == "" || q.Get(want) == "" {
				return "", nil, nil, false
			}
		}
		params = map[string]string{}
		for _, want := range c.ch.Done.RequiredQueryParams {
			params[want] = q.Get(want)
		}
	}
	for _, marker := range c.ch.Done.AbsentMarkers {
		if marker != "" && strings.Contains(html, marker) {
			return "", nil, nil, false
		}
	}
	listed, err := mygo.Cookies.List(mygo.CookieFilter{URL: c.ch.URL})
	if err != nil {
		return "", nil, nil, false
	}
	names := make(map[string]*http.Cookie, len(listed))
	for _, mc := range listed {
		hc := &http.Cookie{Name: mc.Name, Value: mc.Value, Domain: mc.Domain, Path: mc.Path,
			Secure: mc.Secure, HttpOnly: mc.HTTPOnly}
		if !mc.Expires.IsZero() {
			hc.Expires = mc.Expires
		}
		names[mc.Name] = hc
	}
	var cookies []*http.Cookie
	for _, want := range c.ch.Done.RequiredCookies {
		hc, ok := names[want]
		if !ok {
			return "", nil, nil, false
		}
		cookies = append(cookies, hc)
	}
	// Cloudflare binds clearance to the exact User-Agent of the page that
	// earned it, so the requests that follow must send this one.
	rawUA, err := page.Eval("navigator.userAgent")
	if err != nil {
		return "", nil, nil, false
	}
	ua, _ := rawUA.(string)
	if ua == "" {
		return "", nil, nil, false
	}
	for _, hc := range names {
		if !containsCookie(cookies, hc.Name) {
			cookies = append(cookies, hc)
		}
	}
	return ua, cookies, params, true
}

// debugState logs what the page shows while a challenge is pending, when
// NKDEBUG is set: a challenge that never passes is otherwise invisible.
func debugState(c *challenge) {
	if os.Getenv("NKDEBUG") == "" {
		return
	}
	page := c.web.Page()
	title, _ := page.Eval("document.title")
	html, _ := page.Eval("document.documentElement.outerHTML")
	text := ""
	if s, ok := html.(string); ok {
		if len(s) > 400 {
			s = s[:400]
		}
		text = s
	}
	present := map[string]bool{}
	if s, ok := html.(string); ok {
		for _, m := range c.ch.Done.AbsentMarkers {
			present[m] = m != "" && strings.Contains(s, m)
		}
	}
	listed, _ := mygo.Cookies.List(mygo.CookieFilter{URL: c.ch.URL})
	var names []string
	for _, mc := range listed {
		names = append(names, mc.Name)
	}
	log.Printf("verify: title=%v markers=%v cookies=%v head=%q", title, present, names, text)
}

func containsCookie(cs []*http.Cookie, name string) bool {
	for _, c := range cs {
		if c.Name == name {
			return true
		}
	}
	return false
}

// draw is the verification window's view.
func (c *challenge) draw(ctx *ui.Context) {
	t := ctx.Theme()
	ui.Column(ctx).Fill().Children(func() {
		ui.Row(ctx).Padding(8, 12).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(ctx, "完成下面的验证后会自动继续").FontSize(12).TextColor(t.TextMuted).Grow(1)
			if ui.Button(ctx, "关闭").Clicked() {
				c.cancel()
			}
		})
		ui.Divider(ctx)
		if c.web != nil {
			ui.WebView(ctx, c.web).Grow(1)
		}
	})
}

// Sniffer finds a video's stream in a page. It is what a video source that
// answers with a page URL needs; nothing calls it until video sources are
// wired up, so it reports ErrUnsupported instead of pretending.
type Sniffer struct{}

// NewSniffer builds a Sniffer.
func NewSniffer() *Sniffer { return &Sniffer{} }

// Sniff implements host.Sniffer.
func (s *Sniffer) Sniff(ctx context.Context, req *host.SniffRequest) (*host.SniffResult, error) {
	return nil, host.ErrUnsupported
}
