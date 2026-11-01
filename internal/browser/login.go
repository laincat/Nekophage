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
	"github.com/laincat/nekophage/internal/core"
)

// Logging in on the site's own page.
//
// Five official sources log in this way and a client without it can only say
// "not supported". The source names the page and supplies two callbacks: one that
// says "this page means logged in", and one that runs afterwards. The window here
// is the same WebView the challenge flow uses, and the cookies it earns go into
// the same per-source partition every later request reads.

// LoginWindow logs a source in through its own page.
type LoginWindow struct {
	core *core.Core
	jar  *cookies.Jar
	key  string
	// Partition is the jar the cookies land in; the same one the source's
	// requests use.
	partition string
}

// NewLoginWindow builds the helper for one source.
func NewLoginWindow(c *core.Core, jar *cookies.Jar, key, partition string) *LoginWindow {
	return &LoginWindow{core: c, jar: jar, key: key, partition: partition}
}

// Run shows the login page and waits until the source says it is logged in, the
// user closes the window, or the context ends.
//
// The result is the number of cookies stored, so the caller can say what
// happened rather than "done".
func (l *LoginWindow) Run(ctx context.Context) (int, error) {
	if l.core == nil || l.key == "" {
		return 0, errors.New("browser: login without a source")
	}
	decl, err := l.core.AccountWebviewLogin(ctx, l.key)
	if err != nil {
		return 0, err
	}
	if decl.Handle == "" || decl.URL == "" {
		return 0, errors.New("browser: this source has no browser login")
	}

	// Login and verification share the WebView store across all sources.
	browserStoreMu.Lock()
	defer browserStoreMu.Unlock()

	// Refuse to open a page if the previous browser session cannot be cleared.
	if err := mygo.Cookies.Clear(); err != nil {
		return 0, err
	}
	defer func() { _ = mygo.Cookies.Clear() }()

	state := &loginState{done: make(chan struct{})}
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:   "登录 " + l.key,
		Width:   520,
		Height:  780,
		Content: ui.View(state.draw),
	})
	view, err := win.NewWebView(mygo.WebViewOptions{URL: decl.URL})
	if err != nil {
		win.Close()
		return 0, err
	}
	state.win, state.web = win, view
	// Closing the window is the user giving up. Without this the poll keeps
	// asking a destroyed page whether it is logged in, and the panel's busy
	// spinner never clears.
	win.OnClosed(func() { state.once.Do(func() { state.closed = true; close(state.done) }) })
	win.Show()
	win.Focus()

	stored, err := l.watch(ctx, decl, state)
	win.Close()
	return stored, err
}

// errNoCookiesYet reports a page that agreed but has not set its cookies yet.
var errNoCookiesYet = errors.New("browser: no cookies yet")

// loginState is the window's close button.
type loginState struct {
	win  *mygo.Window
	web  *mygo.WebView
	once sync.Once
	done chan struct{}
	// closed records that the user closed the window rather than saying they
	// finished, so the caller can report it instead of storing half a login.
	closed bool
}

func (s *loginState) draw(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(12).Gap(8).Children(func() {
		ui.Text(c, "在下面的页面里登录,完成后窗口会自动关闭。").FontSize(12).TextColor(t.TextMuted)
		if ui.Button(c, "我已登录").Clicked() {
			s.once.Do(func() { close(s.done) })
		}
	})
}

// watch polls the page: the source's checkStatus decides when to stop, because
// only it knows which page means "logged in".
func (l *LoginWindow) watch(ctx context.Context, decl core.WebviewLogin, state *loginState) (int, error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-state.done:
			if state.closed {
				return 0, errors.New("browser: the login window was closed before the source agreed")
			}
			// The user says they are logged in even though checkStatus never
			// agreed; the cookies are still worth keeping.
			stored, err := l.store(decl, "")
			if errors.Is(err, errNoCookiesYet) {
				return 0, errors.New("browser: the page set no cookies to keep")
			}
			return stored, err
		case <-ticker.C:
			page := state.web.Page()
			rawURL, err := page.Eval("location.href")
			if err != nil {
				continue
			}
			pageURL, _ := rawURL.(string)
			rawTitle, err := page.Eval("document.title")
			if err != nil {
				continue
			}
			title, _ := rawTitle.(string)
			done, err := l.core.WebviewStatus(ctx, l.key, decl.Handle, pageURL, title)
			if err != nil {
				// A source that throws on a page it does not recognise is
				// ordinary: keep watching.
				continue
			}
			if os.Getenv("NKDEBUG") != "" {
				log.Printf("login %s: %q %q done=%v", l.key, title, pageURL, done)
			}
			if !done {
				continue
			}
			stored, err := l.store(decl, pageURL)
			if err == nil {
				return stored, nil
			}
			if !errors.Is(err, errNoCookiesYet) {
				// The cookies are in, so this is the source's own callback
				// failing. Reporting it now beats a deadline the user cannot
				// explain.
				return stored, err
			}
			// A source may say "logged in" a moment before the cookies settle;
			// keep polling rather than failing the login.
		}
	}
}

// store moves the browser's cookies into the source's partition and runs
// onLoginSuccess.
func (l *LoginWindow) store(decl core.WebviewLogin, pageURL string) (int, error) {
	listed, err := mygo.Cookies.List(mygo.CookieFilter{})
	if err != nil {
		return 0, err
	}
	all := make([]*http.Cookie, 0, len(listed))
	for _, mc := range listed {
		hc := &http.Cookie{Name: mc.Name, Value: mc.Value, Domain: mc.Domain, Path: mc.Path,
			Secure: mc.Secure, HttpOnly: mc.HTTPOnly}
		if !mc.Expires.IsZero() {
			hc.Expires = mc.Expires
		}
		all = append(all, hc)
	}
	// Only the login site's own cookies belong to this source. The page also
	// loads third parties, and storing their cookies in the source's partition
	// would make them part of a login the source never asked for.
	// The page a login finishes on may be another host than the one it started
	// on (manga_dex goes from forums.mangadex.org back to mangadex.org), so both
	// count as the login site.
	out := loginCookiesFor(all, decl.URL)
	if pageURL != "" {
		for _, c := range loginCookiesFor(all, pageURL) {
			if !hasCookie(out, c) {
				out = append(out, c)
			}
		}
	}
	if len(out) == 0 {
		// Not a failure yet: a page can say "logged in" a moment before its
		// cookies settle, so the caller keeps polling.
		return 0, errNoCookiesYet
	}
	l.jar.Import(l.partition, out)
	// The source may need to read its own cookie to finish: ehentai copies it to
	// another host, ccc refreshes its profile.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := l.core.WebviewLoginDone(ctx, l.key, decl.Handle); err != nil {
		return len(out), err
	}
	return len(out), nil
}

// loginCookiesFor keeps the cookies that belong to the page the user logged in
// on, and drops the ones third parties set while it loaded.
//
// The rule is the one a browser applies: a cookie without a leading dot is for
// that exact host, and one with a leading dot is for the host and its
// subdomains. A cookie with no domain at all is host-only for the page that set
// it, which here is the login page.
func loginCookiesFor(cs []*http.Cookie, rawURL string) []*http.Cookie {
	u, err := url.Parse(rawURL)
	if err != nil || u.Hostname() == "" {
		return nil
	}
	host := strings.ToLower(u.Hostname())
	out := make([]*http.Cookie, 0, len(cs))
	for _, c := range cs {
		domain := strings.ToLower(strings.TrimSpace(c.Domain))
		switch {
		case domain == "":
			out = append(out, c)
		case strings.HasPrefix(domain, "."):
			if host == domain[1:] || strings.HasSuffix(host, domain) {
				out = append(out, c)
			}
		default:
			if host == domain {
				out = append(out, c)
			}
		}
	}
	return out
}

// hasCookie reports whether cs already holds a cookie with this identity.
func hasCookie(cs []*http.Cookie, c *http.Cookie) bool {
	for _, have := range cs {
		if have.Name == c.Name && have.Domain == c.Domain && have.Path == c.Path {
			return true
		}
	}
	return false
}
