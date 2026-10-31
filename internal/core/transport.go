package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/laincat/nekophage/internal/cookies"
	"github.com/laincat/veneracat-core/host"
)

// Transport sends the requests Core asks for, with net/http.
//
// Cookies never come from a shared jar: Core names a partition, and the jar
// of that partition supplies the Cookie header and receives every Set-Cookie,
// including those on redirect hops. Responses larger than maxBody fail
// instead of exhausting memory.
type Transport struct {
	client *http.Client
	jar    *cookies.Jar
	ua     string
	// The scheme each partition last used, so a source that serves its pages
	// over http keeps its images over http too. A scheme-relative image
	// address inherits this, as RFC 3986 says it inherits the page's.
	mu      sync.Mutex
	schemes map[string]string
}

// ErrTooLarge is returned for a response over the size limit.
var ErrTooLarge = errors.New("core: response larger than the limit")

// NewTransport builds a Transport.
func NewTransport(jar *cookies.Jar, userAgent string, timeout time.Duration) *Transport {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	t := &Transport{jar: jar, ua: userAgent, schemes: map[string]string{}}
	t.client = &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("core: too many redirects")
			}
			// Provider keys are not all recognised by net/http's default
			// sensitive-header policy. A change of origin must drop them.
			if len(via) > 0 {
				previous := via[0].URL
				if !strings.EqualFold(previous.Scheme, req.URL.Scheme) || !strings.EqualFold(previous.Host, req.URL.Host) {
					for _, name := range []string{"Authorization", "X-Api-Key", "Proxy-Authorization", "Cookie"} {
						req.Header.Del(name)
					}
				}
			}
			// Store what the previous hop set, then send this partition's
			// cookies to the next URL, which may be another host.
			//
			// The redirect response is `req.Response`, not the last entry of
			// `via`: Go fills Response only on the request it creates for the
			// next hop, so reading it from `via` finds nil and quietly stores
			// nothing.
			part := partitionOf(req.Context())
			if part != "" {
				if prev := req.Response; prev != nil && prev.Request != nil && prev.Request.URL != nil {
					t.jar.SetURL(part, prev.Request.URL, prev.Cookies())
				}
				// Drop cookies copied by net/http before selecting for the next
				// host/path; absence in this jar must clear the previous header.
				req.Header.Del("Cookie")
				if h := t.jar.Header(part, req.URL.String()); h != "" {
					req.Header.Set("Cookie", h)
				}
			}
			return nil
		},
	}
	return t
}

// escapeInvalid percent-encodes the bytes a request line cannot carry.
//
// Sources build URLs by pasting text into them: a keyword with a space
// ("?title=one piece") is common. Dart's Uri escapes it, and Go's client
// instead writes the raw byte, which servers answer with a protocol error.
// Existing percent escapes are left alone, and the characters that structure
// a URL stay as they are.
func escapeInvalid(u *url.URL) {
	// The path is written from RawPath when it is set, so the escaped form
	// goes there and Path keeps the plain text: setting Path to an escaped
	// string would escape the percent signs again.
	if escaped := escapeKeeping(u.Path, "/:@!$&'()*+,;=-._~%"); escaped != u.Path {
		u.RawPath = escaped
	}
	u.RawQuery = escapeKeeping(u.RawQuery, "/?:@!$&'()*+,;=-._~%[]")
}

func escapeKeeping(s, allowed string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 0x20 || c == 0x7f || strings.IndexByte(" \"<>^`{}|\\", c) >= 0 {
			fmt.Fprintf(&b, "%%%02X", c)
			continue
		}
		if strings.IndexByte(allowed, c) >= 0 {
			b.WriteByte(c)
			continue
		}
		// Bytes of UTF-8 text are escaped; ASCII punctuation is kept.
		if c >= 0x80 {
			fmt.Fprintf(&b, "%%%02X", c)
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// maxBody bounds one response body.
const maxBody = 64 << 20

type partitionKey struct{}

// partitionOf returns the partition recorded on the request context.
func partitionOf(ctx context.Context) string {
	if v, ok := ctx.Value(partitionKey{}).(string); ok {
		return v
	}
	return ""
}

// SchemeOf returns the scheme a partition last used, or "" when it has not
// asked for anything yet.
func (t *Transport) SchemeOf(partition string) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.schemes[partition]
}

// Do implements host.Transport.
func (t *Transport) Do(ctx context.Context, r *host.Request) (*host.Response, error) {
	u, err := url.Parse(r.URL)
	if err != nil {
		return nil, err
	}
	escapeInvalid(u)
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, errors.New("core: only http and https requests")
	}
	method := r.Method
	if method == "" {
		method = http.MethodGet
	}
	var body io.Reader
	if len(r.Body) > 0 {
		body = bytes.NewReader(r.Body)
	}
	// Only cookie-enabled requests carry a jar partition into redirect hops.
	partition := ""
	if r.Cookies {
		partition = r.Partition
	}
	req, err := http.NewRequestWithContext(context.WithValue(ctx, partitionKey{}, partition), method, u.String(), body)
	if err != nil {
		return nil, err
	}
	for k, vs := range r.Header {
		if strings.EqualFold(k, "Cookie") {
			continue
		}
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", t.ua)
	}
	var cookie string
	for k, vs := range r.Header {
		if strings.EqualFold(k, "Cookie") {
			cookie = strings.Join(vs, "; ")
		}
	}
	if r.Cookies {
		if stored := t.jar.Header(r.Partition, u.String()); stored != "" {
			if cookie != "" {
				cookie += "; "
			}
			cookie += stored
		}
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// The partition's scheme is remembered, so an image address without one
	// can inherit it.
	if partition != "" {
		t.mu.Lock()
		t.schemes[partition] = u.Scheme
		t.mu.Unlock()
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxBody {
		return nil, ErrTooLarge
	}
	if r.Cookies {
		// The last hop's cookies; earlier hops were stored on redirect.
		t.jar.SetURL(r.Partition, resp.Request.URL, resp.Cookies())
	}
	final := r.URL
	if resp.Request != nil && resp.Request.URL != nil {
		final = resp.Request.URL.String()
	}
	return &host.Response{Status: resp.StatusCode, Header: resp.Header, Body: data, FinalURL: final}, nil
}

// Fetch downloads a URL with no cookies, for indexes and scripts.
func (t *Transport) Fetch(ctx context.Context, rawURL string, limit int64) ([]byte, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, "", err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, "", errors.New("core: only http and https URLs")
	}
	escapeInvalid(u)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", t.ua)
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, "", errors.New("core: download failed: HTTP " + resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(data)) > limit {
		return nil, "", ErrTooLarge
	}
	ctype := resp.Header.Get("Content-Type")
	if i := strings.IndexByte(ctype, ';'); i >= 0 {
		ctype = ctype[:i]
	}
	return data, strings.TrimSpace(ctype), nil
}
