package player

import (
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/laincat/veneracat-core/video"
)

//go:embed assets/player.html
var playerPage []byte

//go:embed assets/hls.min.js
var hlsScript []byte

// Meta identifies what a session is playing, so saved progress belongs to the
// right episode rather than to "the last video".
type Meta struct {
	Source      string
	ItemID      string
	Title       string
	LineID      string
	LineName    string
	EpisodeID   string
	EpisodeName string
}

// Progress is one report from the page. Time and Duration are seconds.
type Progress struct {
	Time     float64 `json:"time"`
	Duration float64 `json:"duration"`
	Paused   bool    `json:"paused"`
}

const maxProgressBody = 4096

// Server hosts player pages and media on loopback. Each session's URLs carry a
// random capability token; the port listens on 127.0.0.1 only.
type Server struct {
	ln         net.Listener
	httpServer *http.Server
	// OnProgress receives playback positions. It runs on the HTTP goroutine.
	OnProgress func(Meta, Progress)

	mu       sync.Mutex
	sessions map[string]*Session
}

func NewServer() (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &Server{ln: ln, sessions: map[string]*Session{}}
	s.httpServer = &http.Server{Handler: s}
	go func() {
		if err := s.httpServer.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			// The app keeps running without video; nothing else can use it.
			return
		}
	}()
	return s, nil
}

// Session is one opened player. Closing it stops its media requests and makes
// its URLs stop working.
type Session struct {
	server *Server
	token  string
	proxy  *MediaProxy
	format video.Format

	mu   sync.Mutex
	meta Meta
}

// Open starts a session for one play declaration. The returned page URL is
// what the WebView loads.
func (s *Server) Open(source MediaSource, key string, play *video.Play, meta Meta) (*Session, error) {
	var tokenBytes [16]byte
	if _, err := rand.Read(tokenBytes[:]); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(tokenBytes[:])
	proxy, err := NewMediaProxy(source, key, play, WithMount("/s/"+token))
	if err != nil {
		return nil, err
	}
	format := play.Format
	if format == "" || format == video.FormatAuto {
		if target, err := url.Parse(play.URL); err == nil && strings.HasSuffix(strings.ToLower(target.Path), ".m3u8") {
			format = video.FormatHLS
		}
	}
	session := &Session{server: s, token: token, proxy: proxy, meta: meta, format: format}
	s.mu.Lock()
	s.sessions[token] = session
	s.mu.Unlock()
	return session, nil
}

// SetMeta updates what progress reports are filed under.
func (s *Session) SetMeta(meta Meta) {
	s.mu.Lock()
	s.meta = meta
	s.mu.Unlock()
}

// URL is the page the WebView opens.
func (s *Session) URL(title string, start float64) string {
	query := url.Values{}
	query.Set("src", s.proxy.RootPath())
	query.Set("title", title)
	query.Set("token", s.token)
	query.Set("format", string(s.format))
	if start > 0 && !math.IsNaN(start) && !math.IsInf(start, 0) {
		query.Set("start", strconv.FormatFloat(start, 'f', -1, 64))
	}
	return "http://" + s.server.ln.Addr().String() + "/s/" + s.token + "/play?" + query.Encode()
}

// Close stops the session: media requests are cancelled and the URLs stop
// resolving.
func (s *Session) Close() {
	s.server.mu.Lock()
	delete(s.server.sessions, s.token)
	s.server.mu.Unlock()
	s.proxy.Close()
}

// Close shuts the server and every session down.
func (s *Server) Close() error {
	s.mu.Lock()
	sessions := s.sessions
	s.sessions = map[string]*Session{}
	s.mu.Unlock()
	for _, session := range sessions {
		session.proxy.Close()
	}
	return s.httpServer.Close()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/s/")
	token, rest, ok := strings.Cut(path, "/")
	if !ok || token == "" {
		http.NotFound(w, r)
		return
	}
	s.mu.Lock()
	session, known := s.sessions[token]
	s.mu.Unlock()
	if !known {
		http.NotFound(w, r)
		return
	}
	switch rest {
	case "play":
		servePlayerPage(w, r)
	case "hls.js":
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		_, _ = w.Write(hlsScript)
	case "progress":
		session.serveProgress(w, r)
	default:
		if !strings.HasPrefix(rest, "media/") {
			http.NotFound(w, r)
			return
		}
		// The page and the media share one prefix, so a rewritten playlist's
		// absolute resource paths already reach this server.
		session.proxy.ServeHTTP(w, r)
	}
}

func servePlayerPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(playerPage)
}

func (s *Session) serveProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxProgressBody+1))
	if err != nil || len(body) > maxProgressBody {
		http.Error(w, "progress body is too large", http.StatusRequestEntityTooLarge)
		return
	}
	var progress Progress
	if err := json.Unmarshal(body, &progress); err != nil {
		http.Error(w, "progress is not JSON", http.StatusBadRequest)
		return
	}
	if progress.Time < 0 || progress.Duration < 0 {
		http.Error(w, "progress is out of range", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	meta := s.meta
	s.mu.Unlock()
	if s.server.OnProgress != nil {
		s.server.OnProgress(meta, progress)
	}
	w.WriteHeader(http.StatusNoContent)
}
