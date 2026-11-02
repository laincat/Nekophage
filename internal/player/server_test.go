package player

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/laincat/veneracat-core/video"
)

// httpSource is the smallest honest MediaSource: the server tests are about
// routing and page delivery, not about a provider's parsing.
type httpSource struct{ client *http.Client }

func (s httpSource) OpenVideoMedia(ctx context.Context, _ string, rawURL string, header http.Header) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	for name, values := range header {
		req.Header[name] = values
	}
	return s.client.Do(req)
}

func testPlay(t *testing.T, target string, format video.Format) *video.Play {
	t.Helper()
	play := &video.Play{URL: target, Format: format}
	if err := play.Validate(); err != nil {
		t.Fatal(err)
	}
	return play
}

func TestServerServesPageScriptAndSortedProgress(t *testing.T) {
	media := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		fmt.Fprint(w, "media-bytes")
	}))
	defer media.Close()
	server, err := NewServer()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	var reports []Progress
	var metas []Meta
	server.OnProgress = func(meta Meta, progress Progress) {
		metas = append(metas, meta)
		reports = append(reports, progress)
	}
	session, err := server.Open(httpSource{client: http.DefaultClient}, "probe", testPlay(t, media.URL, video.FormatMP4), Meta{
		Source: "probe", ItemID: "clip", Title: "Probe", EpisodeID: "ep1",
	})
	if err != nil {
		t.Fatal(err)
	}
	pageURL := session.URL("Probe · 第 1 集", 42.7)
	base := strings.SplitN(strings.TrimPrefix(pageURL, "http://"), "/s/", 2)[0]
	prefix := pageURL[:strings.Index(pageURL, "/play")]

	page, err := http.Get(pageURL)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(page.Body)
	page.Body.Close()
	if page.StatusCode != http.StatusOK || !bytes.Contains(body, []byte("<video")) || !bytes.Contains(body, []byte("hls.js")) {
		t.Fatalf("player page = %d, %d bytes", page.StatusCode, len(body))
	}
	if ct := page.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("player page content type = %q", ct)
	}
	if !strings.Contains(pageURL, "start=42.7") {
		t.Fatalf("page URL lost the resume position: %s", pageURL)
	}

	script, err := http.Get("http://" + base + "/s/" + session.token + "/hls.js")
	if err != nil {
		t.Fatal(err)
	}
	scriptBody, _ := io.ReadAll(script.Body)
	script.Body.Close()
	if script.StatusCode != http.StatusOK || len(scriptBody) != len(hlsScript) || !strings.Contains(script.Header.Get("Content-Type"), "javascript") {
		t.Fatalf("hls.js = %d, %d bytes, %q", script.StatusCode, len(scriptBody), script.Header.Get("Content-Type"))
	}

	resource, err := http.Get("http://" + base + session.proxy.RootPath())
	if err != nil {
		t.Fatal(err)
	}
	mediaBody, _ := io.ReadAll(resource.Body)
	resource.Body.Close()
	if string(mediaBody) != "media-bytes" {
		t.Fatalf("proxied media = %q", mediaBody)
	}

	post := func(payload string) int {
		t.Helper()
		response, err := http.Post(prefix+"/progress", "application/json", strings.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		return response.StatusCode
	}
	if code := post(`{"time":12.5,"duration":60,"paused":false}`); code != http.StatusNoContent {
		t.Fatalf("progress POST = %d", code)
	}
	if len(reports) != 1 || reports[0].Time != 12.5 || reports[0].Duration != 60 || reports[0].Paused {
		t.Fatalf("progress reports = %+v", reports)
	}
	if metas[0].ItemID != "clip" || metas[0].EpisodeID != "ep1" || metas[0].Source != "probe" {
		t.Fatalf("progress was filed under %+v", metas[0])
	}
	if code := post(`{"time":-1}`); code != http.StatusBadRequest {
		t.Fatalf("negative progress = %d", code)
	}
	if code := post(`not json`); code != http.StatusBadRequest {
		t.Fatalf("malformed progress = %d", code)
	}
	if code := post(strings.Repeat("x", maxProgressBody+10)); code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized progress = %d", code)
	}
	session.SetMeta(Meta{Source: "probe", ItemID: "clip", EpisodeID: "ep2"})
	if code := post(`{"time":5,"duration":60}`); code != http.StatusNoContent || metas[len(metas)-1].EpisodeID != "ep2" {
		t.Fatalf("updated session meta = %d, %+v", code, metas[len(metas)-1])
	}

	session.Close()
	for _, target := range []string{pageURL, "http://" + base + session.proxy.RootPath(), prefix + "/progress"} {
		response, err := http.Get(target)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusNotFound {
			t.Fatalf("closed session still served %s (%d)", target, response.StatusCode)
		}
	}
	if _, err := server.Open(httpSource{client: http.DefaultClient}, "probe", nil, Meta{}); err == nil {
		t.Fatal("a session opened without a play declaration")
	}
}

func TestServerSessionTokensAreNotGuessed(t *testing.T) {
	media := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "media")
	}))
	defer media.Close()
	server, err := NewServer()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	session, err := server.Open(httpSource{client: http.DefaultClient}, "probe", testPlay(t, media.URL, video.FormatMP4), Meta{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	base := "http://" + server.ln.Addr().String()
	for _, target := range []string{"/s/00000000000000000000000000000000/play", "/s//play", "/media/x", "/", "/s/" + session.token + "/unknown"} {
		response, err := http.Get(base + target)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusNotFound {
			t.Fatalf("%s = %d", target, response.StatusCode)
		}
	}
	// A second session's token must not reach the first session's media.
	other, err := server.Open(httpSource{client: http.DefaultClient}, "probe", testPlay(t, media.URL, video.FormatMP4), Meta{})
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	response, err := http.Get(base + "/s/" + other.token + session.proxy.RootPath())
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("another session's token reached the first session's media (%d)", response.StatusCode)
	}
}

func TestSessionPreservesMediaFormatAndFractionalResume(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	for _, tc := range []struct {
		url    string
		format video.Format
		want   video.Format
	}{
		{"https://fixture.invalid/clip.mp4", video.FormatMP4, video.FormatMP4},
		{"https://fixture.invalid/play?id=1", video.FormatHLS, video.FormatHLS},
		{"https://fixture.invalid/main.m3u8?signature=private", video.FormatAuto, video.FormatHLS},
	} {
		session, err := server.Open(httpSource{client: http.DefaultClient}, "fixture", testPlay(t, tc.url, tc.format), Meta{})
		if err != nil {
			t.Fatal(err)
		}
		page, err := url.Parse(session.URL("Clip", 2.375))
		if err != nil {
			t.Fatal(err)
		}
		if got := page.Query().Get("format"); got != string(tc.want) {
			t.Fatalf("%s playback is routed as %s, want %s", tc.url, got, tc.want)
		}
		if got := page.Query().Get("start"); got != "2.375" {
			t.Fatalf("resume position was rounded: %s", got)
		}
		if strings.Contains(page.String(), "fixture.invalid") || strings.Contains(page.String(), "private") {
			t.Fatal("the page URL exposed an upstream URL or signature")
		}
		session.Close()
	}
}

func TestServerStopsOnClose(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatal(err)
	}
	address := server.ln.Addr().String()
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: 500 * time.Millisecond}
	if response, err := client.Get("http://" + address + "/"); err == nil {
		response.Body.Close()
		t.Fatal("the loopback server still answered after Close")
	}
}

func TestSubtitleChoiceRoundTripAndValidation(t *testing.T) {
	media := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "media-bytes")
	}))
	defer media.Close()
	server, err := NewServer()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	var saved []Subtitle
	server.OnSubtitle = func(meta Meta, choice Subtitle) { saved = append(saved, choice) }
	// The first session knows a choice; the second does not. Same episode,
	// so the page can be reopened in either direction.
	var current *Subtitle
	server.Subtitle = func(meta Meta) (Subtitle, bool) {
		if current == nil {
			return Subtitle{}, false
		}
		return *current, true
	}
	open := func() *Session {
		session, err := server.Open(httpSource{client: http.DefaultClient}, "probe",
			testPlay(t, media.URL, video.FormatMP4), Meta{Source: "probe", ItemID: "clip", EpisodeID: "ep1"})
		if err != nil {
			t.Fatal(err)
		}
		return session
	}
	session := open()
	prefix := session.URL("Probe", 0)
	prefix = prefix[:strings.Index(prefix, "/play")]
	defer session.Close()

	get := func() map[string]any {
		resp, err := http.Get(prefix + "/subtitle")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET subtitle = %d", resp.StatusCode)
		}
		var body map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if cc := resp.Header.Get("Cache-Control"); cc != "no-store" {
			t.Fatalf("subtitle cache control = %q", cc)
		}
		return body
	}
	post := func(body string) int {
		resp, err := http.Post(prefix+"/subtitle", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}

	if body := get(); body["chosen"] != false {
		t.Fatalf("no choice yet = %v", body)
	}
	if code := post(`{"name":"English","language":"en"}`); code != http.StatusNoContent {
		t.Fatalf("POST subtitle = %d", code)
	}
	if len(saved) != 1 || saved[0].Name != "English" || saved[0].Language != "en" || saved[0].Off {
		t.Fatalf("saved = %+v", saved)
	}
	// What the app stored is what the next page load sees.
	current = &Subtitle{Name: "English", Language: "en"}
	if body := get(); body["chosen"] != true || body["name"] != "English" || body["language"] != "en" {
		t.Fatalf("saved choice = %v", body)
	}
	if code := post(`{"off":true}`); code != http.StatusNoContent {
		t.Fatalf("POST off = %d", code)
	}
	current = &Subtitle{Off: true}
	if body := get(); body["chosen"] != true || body["off"] != true {
		t.Fatalf("off choice = %v", body)
	}
	for _, bad := range []string{``, `not json`, `{}`, `{"name":""}`} {
		if code := post(bad); code != http.StatusBadRequest {
			t.Fatalf("POST %q = %d, want 400", bad, code)
		}
	}
	if code := post(strings.Repeat("x", maxSubtitleBody+1)); code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized POST = %d, want 413", code)
	}
	resp, err := http.Head(prefix + "/subtitle")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("HEAD subtitle = %d", resp.StatusCode)
	}
}
