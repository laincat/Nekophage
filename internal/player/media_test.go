package player

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/veneracat-core/plugin"
	"github.com/laincat/veneracat-core/video"
	"github.com/laincat/veneracat-core/video/hls"
)

func declaredMedia(t *testing.T, target string, format video.Format) (*core.Core, *video.Play) {
	t.Helper()
	c, err := core.Open(core.Options{DataDir: t.TempDir(), AppVersion: "1.6.3"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	script := fmt.Sprintf(`class MediaProbe extends VideoSource {
 name = "Media probe"; key = "media_probe"; version = "1.0.0"
 video = {loadPlay: async () => ({url: %q, format: %q, headers: {Referer: "https://media.example.test/"}})}
}`, target, format)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := c.InstallTyped(ctx, plugin.TypeVideoJS, script, "file", ""); err != nil {
		t.Fatal(err)
	}
	play, err := c.VideoPlay(ctx, "media_probe", "clip", "line", "episode")
	if err != nil {
		t.Fatal(err)
	}
	return c, play
}

func TestMediaProxyRewritesNestedHLSAndPreservesRangeAndHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Referer") != "https://media.example.test/" {
			http.Error(w, "missing source referer", http.StatusForbidden)
			return
		}
		switch r.URL.Path {
		case "/entry.m3u8":
			http.Redirect(w, r, "/vod/master.m3u8", http.StatusFound)
		case "/vod/master.m3u8":
			w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
			fmt.Fprint(w, "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=1000\nlow/media.m3u8\n")
		case "/vod/low/media.m3u8":
			w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
			fmt.Fprint(w, "#EXTM3U\n#EXT-X-TARGETDURATION:6\n#EXT-X-KEY:METHOD=AES-128,URI=\"../key\"\n#EXT-X-MAP:URI=\"init.mp4\"\n#EXTINF:6,\nseg.ts\n#EXT-X-ENDLIST\n")
		case "/vod/key":
			fmt.Fprint(w, "0123456789abcdef")
		case "/vod/low/init.mp4":
			fmt.Fprint(w, "init")
		case "/vod/low/seg.ts":
			w.Header().Set("Content-Type", "video/mp2t")
			http.ServeContent(w, r, "seg.ts", time.Time{}, bytes.NewReader([]byte("abcdef")))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()
	c, play := declaredMedia(t, upstream.URL+"/entry.m3u8", video.FormatHLS)
	proxy, err := NewMediaProxy(c, "media_probe", play)
	if err != nil {
		t.Fatal(err)
	}
	defer proxy.Close()
	local := httptest.NewServer(proxy)
	defer local.Close()
	client := &http.Client{Timeout: 3 * time.Second}
	fetch := func(path string) []byte {
		t.Helper()
		if !strings.HasPrefix(path, "/media/") {
			t.Fatalf("upstream URL escaped the proxy: %s", path)
		}
		response, err := client.Get(local.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil || response.StatusCode != http.StatusOK {
			t.Fatalf("proxy fetch %s = %d %q, %v", path, response.StatusCode, body, err)
		}
		return body
	}
	masterBody := fetch(proxy.RootPath())
	if bytes.Contains(masterBody, []byte(upstream.URL)) {
		t.Fatal("the rewritten playlist exposed an upstream fetch URL")
	}
	master, err := hls.Parse(masterBody)
	if err != nil || len(master.Variants) != 1 {
		t.Fatalf("master playlist = %+v, %v", master, err)
	}
	mediaBody := fetch(master.Variants[0].URI)
	media, err := hls.Parse(mediaBody)
	if err != nil || len(media.Segments) != 1 || media.Segments[0].Key == nil || media.Segments[0].Map == nil {
		t.Fatalf("media playlist = %+v, %v", media, err)
	}
	segment := media.Segments[0]
	if got := string(fetch(segment.Key.URI)); got != "0123456789abcdef" {
		t.Fatalf("key body = %q", got)
	}
	if got := string(fetch(segment.Map.URI)); got != "init" {
		t.Fatalf("initialization body = %q", got)
	}
	req, _ := http.NewRequest(http.MethodGet, local.URL+segment.URI, nil)
	req.Header.Set("Range", "bytes=1-3")
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || response.StatusCode != http.StatusPartialContent || string(body) != "bcd" || response.Header.Get("Content-Range") != "bytes 1-3/6" {
		t.Fatalf("segment range = %d %q %q, %v", response.StatusCode, body, response.Header.Get("Content-Range"), err)
	}
	if play.Header.Get("Range") != "" {
		t.Fatal("a range request changed the source declaration")
	}
	proxy.Close()
	response, err = client.Get(local.URL + proxy.RootPath())
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusGone {
		t.Fatal("a closed player continued serving media")
	}
}

// A CDN can drop a chunked response without a Content-Length. Completing the
// downstream chunk stream would turn that broken response into a clean EOF, so
// the player must see the interrupted transfer as an error too.
func TestMediaProxyPreservesTruncatedStreamErrors(t *testing.T) {
	for _, length := range []string{"", "10"} {
		t.Run("content-length="+length, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Referer") != "https://media.example.test/" {
					t.Error("the source Referer did not reach the media server")
				}
				w.Header().Set("Content-Type", "video/mp4")
				if length != "" {
					w.Header().Set("Content-Length", length)
				}
				fmt.Fprint(w, "first")
				w.(http.Flusher).Flush()
				// No final chunk (or the rest of the declared body): the upstream
				// connection ends abnormally after delivering real media bytes.
				panic(http.ErrAbortHandler)
			}))
			defer upstream.Close()
			c, play := declaredMedia(t, upstream.URL+"/clip.mp4", video.FormatMP4)
			proxy, err := NewMediaProxy(c, "media_probe", play)
			if err != nil {
				t.Fatal(err)
			}
			defer proxy.Close()
			local := httptest.NewServer(proxy)
			defer local.Close()
			client := &http.Client{Timeout: 3 * time.Second}
			response, err := client.Get(local.URL + proxy.RootPath())
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if string(body) != "first" || !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("truncated upstream became body=%q err=%v; want first and unexpected EOF", body, err)
			}
		})
	}
}

func TestClosingPlayerStopsAStreamingResponse(t *testing.T) {
	stopped := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		fmt.Fprint(w, "first")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		close(stopped)
	}))
	defer upstream.Close()
	c, play := declaredMedia(t, upstream.URL+"/clip.mp4", video.FormatMP4)
	proxy, err := NewMediaProxy(c, "media_probe", play)
	if err != nil {
		t.Fatal(err)
	}
	defer proxy.Close()
	local := httptest.NewServer(proxy)
	defer local.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, local.URL+proxy.RootPath(), nil)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	first := make([]byte, 5)
	if _, err := io.ReadFull(response.Body, first); err != nil || string(first) != "first" {
		t.Fatalf("first proxied chunk = %q, %v", first, err)
	}
	proxy.Close()
	select {
	case <-stopped:
	case <-ctx.Done():
		t.Fatal("closing the player did not stop upstream media")
	}
}

func TestUnsupportedPlaybackIsRejectedBeforeNetworking(t *testing.T) {
	for _, play := range []*video.Play{
		{URL: "https://example.test/drm.mp4", Encrypted: true},
		{Sniff: &video.Sniff{Page: "https://example.test/watch"}},
		{URL: "https://example.test/dash.mpd", Format: video.FormatDASH},
	} {
		if _, err := NewMediaProxy(nil, "probe", play); err == nil {
			t.Fatal("unsupported playback was accepted")
		}
	}
}
