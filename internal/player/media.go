// Package player supplies desktop playback's local media proxy. Parsing HLS
// remains in the shared Core; this package owns HTTP and the player's lifetime.
package player

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/laincat/veneracat-core/host"
	"github.com/laincat/veneracat-core/video"
	"github.com/laincat/veneracat-core/video/hls"
)

var (
	ErrEncrypted = errors.New("player: this encrypted media requires an unsupported DRM path")
	ErrSniff     = errors.New("player: this source requires media sniffing, which is not connected yet")
)

const maxPlaylistBytes = 1 << 20

// MediaSource streams through the client's source-specific network policy.
type MediaSource interface {
	OpenVideoMedia(context.Context, string, string, http.Header) (*http.Response, error)
}

type resource struct {
	url      string
	playlist bool
}

// Option changes how a proxy addresses its resources.
type Option func(*MediaProxy)

// WithMount puts the resources under prefix, so one loopback server can host
// several players without their URLs colliding.
func WithMount(prefix string) Option {
	return func(p *MediaProxy) { p.prefix = strings.TrimSuffix(prefix, "/") }
}

// MediaProxy owns one play declaration. URLs exposed to the WebView contain
// opaque resource IDs, not upstream URLs, credentials or an arbitrary fetch URL.
// Close stops active requests when its player window goes away.
type MediaProxy struct {
	source  MediaSource
	key     string
	headers http.Header
	dropAds bool
	root    string
	prefix  string
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
	byID    map[string]resource
	byURL   map[resource]string
}

func NewMediaProxy(source MediaSource, key string, play *video.Play, opts ...Option) (*MediaProxy, error) {
	if play == nil {
		return nil, errors.New("player: missing playback declaration")
	}
	if source == nil {
		return nil, errors.New("player: missing media source")
	}
	if err := play.Validate(); err != nil {
		return nil, err
	}
	if play.Encrypted {
		return nil, ErrEncrypted
	}
	if play.Sniff != nil {
		return nil, ErrSniff
	}
	if play.Format == video.FormatDASH {
		return nil, errors.New("player: DASH playback is not connected yet")
	}
	ctx, cancel := context.WithCancel(context.Background())
	p := &MediaProxy{source: source, key: key, headers: play.Header.Clone(), ctx: ctx, cancel: cancel,
		byID: map[string]resource{}, byURL: map[resource]string{}}
	for _, opt := range opts {
		opt(p)
	}
	p.dropAds = play.HLS != nil && play.HLS.FilterAds
	root, err := p.add(play.URL, play.Format == video.FormatHLS)
	if err != nil {
		cancel()
		return nil, err
	}
	p.root = root
	return p, nil
}

func (p *MediaProxy) RootPath() string { return p.root }
func (p *MediaProxy) Close()           { p.cancel() }

func (p *MediaProxy) add(rawURL string, playlist bool) (string, error) {
	if err := host.ValidateURL(rawURL); err != nil {
		return "", err
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	r := resource{url: rawURL, playlist: playlist || strings.HasSuffix(strings.ToLower(u.Path), ".m3u8")}
	p.mu.Lock()
	defer p.mu.Unlock()
	if id, ok := p.byURL[r]; ok {
		return p.prefix + "/media/" + id, nil
	}
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", err
	}
	id := hex.EncodeToString(token[:])
	p.byID[id], p.byURL[r] = r, id
	return p.prefix + "/media/" + id, nil
}

func (p *MediaProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if p.ctx.Err() != nil {
		http.Error(w, "player closed", http.StatusGone)
		return
	}
	path := r.URL.Path
	if p.prefix != "" {
		if !strings.HasPrefix(path, p.prefix+"/") {
			http.NotFound(w, r)
			return
		}
		path = strings.TrimPrefix(path, p.prefix)
	}
	p.mu.Lock()
	item, ok := p.byID[strings.TrimPrefix(path, "/media/")]
	p.mu.Unlock()
	if !strings.HasPrefix(path, "/media/") || !ok {
		http.NotFound(w, r)
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	stop := context.AfterFunc(p.ctx, cancel)
	defer func() { stop(); cancel() }()
	headers := p.headers.Clone()
	if headers == nil {
		headers = make(http.Header)
	}
	if !item.playlist {
		for _, key := range []string{"Range", "If-Range"} {
			if value := r.Header.Get(key); value != "" {
				headers.Set(key, value)
			}
		}
	}
	upstream, err := p.source.OpenVideoMedia(ctx, p.key, item.url, headers)
	if err != nil {
		http.Error(w, "media request failed", http.StatusBadGateway)
		return
	}
	defer upstream.Body.Close()
	ctype := strings.ToLower(strings.SplitN(upstream.Header.Get("Content-Type"), ";", 2)[0])
	playlist := item.playlist || ctype == "application/vnd.apple.mpegurl" || ctype == "application/x-mpegurl"
	if playlist && upstream.StatusCode == http.StatusOK {
		body, err := io.ReadAll(io.LimitReader(upstream.Body, maxPlaylistBytes+1))
		if err != nil || len(body) > maxPlaylistBytes {
			http.Error(w, "playlist could not be read within its limit", http.StatusBadGateway)
			return
		}
		base := item.url
		if upstream.Request != nil && upstream.Request.URL != nil {
			base = upstream.Request.URL.String()
		}
		var mappingError error
		body, err = hls.Rewrite(body, hls.RewriteOptions{
			BaseURL: base, DropAds: p.dropAds, SelectVariant: hls.VariantChooser{Mode: hls.VariantAll},
			MapURL: func(raw string, kind hls.URIKind) string {
				mapped, err := p.add(raw, kind == hls.KindVariant || kind == hls.KindMedia || kind == hls.KindIFrame)
				if err != nil {
					mappingError = err
				}
				return mapped
			},
		})
		if err != nil || mappingError != nil {
			http.Error(w, "invalid media playlist", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		if r.Method != http.MethodHead {
			_, _ = w.Write(body)
		}
		return
	}
	for _, key := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges", "Content-Encoding", "Cache-Control", "ETag", "Last-Modified"} {
		if values := upstream.Header.Values(key); len(values) > 0 {
			w.Header()[key] = append([]string(nil), values...)
		}
	}
	w.WriteHeader(upstream.StatusCode)
	if r.Method == http.MethodHead {
		return
	}
	if _, err := io.Copy(flushWriter{w}, upstream.Body); err != nil {
		// Headers and some bytes have already reached the player. A normal
		// return would terminate a chunked response successfully and hide the
		// upstream failure. Abort this response so the player sees a truncated
		// transfer, as it would when fetching the broken stream directly.
		panic(http.ErrAbortHandler)
	}
}

type flushWriter struct{ http.ResponseWriter }

func (w flushWriter) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
	return n, err
}
