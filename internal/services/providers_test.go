package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/laincat/nekophage/internal/vcmodule"
	"github.com/laincat/veneracat-core/service"
)

func writeZipFile(t *testing.T, dir, name string, files map[string]string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, packageZip(t, files), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// traktZipFiles builds a self-contained Trakt package matching the exact
// VeneraCat Sources (fa1efa4) contract: envelope {ok, service, method, data},
// client_id + access_token secrets, and the 6 official methods.
func traktZipFiles(baseURL, version string) map[string]string {
	manifest := fmt.Sprintf(`{"id":"service.trakt","name":"Trakt","version":%q,"kind":"service.js","entry":"index.mjs","permissions":["network","storage","secrets"]}`, version)
	entry := fmt.Sprintf(`const BASE = %q;
function ok(method, data) { return { ok: true, service: "service.trakt", method, data }; }
export function describe() {
  return {
    methods: [
      { name: "account.me", title: "账号信息 / Account Profile" },
      { name: "metadata.search", title: "条目搜索 / Search Media" },
      { name: "watchlist.list", title: "片单列表 / Watchlist" },
      { name: "history.list", title: "观看历史 / Watch History" },
      { name: "scrobble.start", title: "开始播放上报 / Start Scrobble" },
      { name: "scrobble.stop", title: "结束播放上报 / Stop Scrobble" }
    ]
  };
}
export function handle(method, rawPayload) {
  const payload = rawPayload ? JSON.parse(rawPayload) : {};
  switch (method) {
    case "account.me": {
      const cid = secret("client_id");
      const tok = secret("access_token");
      log("info", "trakt auth check cid=" + cid + " tok=" + tok);
      const body = JSON.parse(fetch(BASE + "/users/settings?cid=" + encodeURIComponent(cid) + "&token=" + encodeURIComponent(tok)));
      return ok("account.me", { username: body.user.username, slug: body.user.ids.slug, name: body.user.name, private: false, vip: true, vipEp: false });
    }
    case "metadata.search": {
      const cid = secret("client_id");
      const type = payload.type || "movie,show";
      const rows = JSON.parse(fetch(BASE + "/search/" + type + "?query=" + encodeURIComponent(payload.query || "") + "&cid=" + encodeURIComponent(cid)));
      return ok("metadata.search", {
        query: payload.query || "",
        type,
        page: payload.page || 1,
        limit: payload.limit || 20,
        items: rows.map(r => ({ type: r.type, score: r.score, title: r.show.title, year: r.show.year, ids: r.show.ids }))
      });
    }
    case "watchlist.list": {
      const cid = secret("client_id");
      const tok = secret("access_token");
      const rows = JSON.parse(fetch(BASE + "/sync/watchlist?cid=" + encodeURIComponent(cid) + "&token=" + encodeURIComponent(tok)));
      return ok("watchlist.list", {
        type: payload.type || "all",
        page: payload.page || 1,
        limit: payload.limit || 20,
        items: rows.map(r => ({ id: r.id, rank: r.rank, listedAt: r.listed_at, type: r.type, title: r.show.title, year: r.show.year, ids: r.show.ids }))
      });
    }
    case "history.list": {
      const cid = secret("client_id");
      const tok = secret("access_token");
      const rows = JSON.parse(fetch(BASE + "/sync/history?cid=" + encodeURIComponent(cid) + "&token=" + encodeURIComponent(tok)));
      return ok("history.list", {
        type: payload.type || "all",
        page: payload.page || 1,
        limit: payload.limit || 20,
        items: rows.map(r => ({ id: r.id, watchedAt: r.watched_at, action: r.action, type: r.type, title: r.show.title, year: r.show.year, ids: r.show.ids }))
      });
    }
    case "scrobble.start":
    case "scrobble.stop": {
      const cid = secret("client_id");
      const tok = secret("access_token");
      const action = method.split(".")[1];
      const res = JSON.parse(fetch(BASE + "/scrobble/" + action + "?progress=" + encodeURIComponent(String(payload.progress)) + "&cid=" + encodeURIComponent(cid) + "&token=" + encodeURIComponent(tok)));
      return ok(method, { id: res.id, action: res.action, progress: res.progress, movie: payload.movie || null, episode: payload.episode || null });
    }
    case "leak.storage": {
      const tok = secret("access_token");
      storageSet("cached_tok", "bearer=" + tok);
      return ok(method, { leaked: true });
    }
    case "leak.error": {
      const tok = secret("access_token");
      throw new Error("upstream rejected token=" + tok);
    }
    case "slow":
      return ok(method, JSON.parse(fetch(BASE + "/slow")));
    default:
      return { ok: false, service: "service.trakt", method, error: { code: "UNSUPPORTED_METHOD", message: "unsupported: " + method } };
  }
}
`, baseURL)
	return map[string]string{"service.json": manifest, "index.mjs": entry}
}

// tmdbZipFiles builds a self-contained TMDB package matching the exact
// VeneraCat Sources (fa1efa4) contract: api_read_token OR api_key, 3 official methods.
func tmdbZipFiles(baseURL, version string) map[string]string {
	manifest := fmt.Sprintf(`{"id":"service.tmdb","name":"TMDB","version":%q,"kind":"service.js","entry":"index.mjs","permissions":["network","storage","secrets"]}`, version)
	entry := fmt.Sprintf(`const BASE = %q;
function ok(method, data) { return { ok: true, service: "service.tmdb", method, data }; }
function readAuth() {
  try { const v = secret("api_read_token"); if (v) return v; } catch (_) {}
  return secret("api_key");
}
export function describe() {
  return {
    methods: [
      { name: "metadata.search", title: "多类型搜索 / Multi Search" },
      { name: "metadata.detail", title: "影视与季集详情 / Media & Season Detail" },
      { name: "artwork.list", title: "海报与背景图集 / Artwork Gallery" }
    ]
  };
}
export function handle(method, rawPayload) {
  const payload = rawPayload ? JSON.parse(rawPayload) : {};
  const tok = readAuth();
  switch (method) {
    case "metadata.search": {
      const mt = payload.mediaType || "multi";
      const body = JSON.parse(fetch(BASE + "/3/search/" + mt + "?query=" + encodeURIComponent(payload.query || "") + "&api_key=" + encodeURIComponent(tok)));
      return ok("metadata.search", {
        query: payload.query || "",
        mediaType: mt,
        language: payload.language || "zh-CN",
        page: body.page,
        totalPages: body.total_pages,
        totalResults: body.total_results,
        items: body.results.map(x => ({ id: x.id, mediaType: x.media_type || mt, title: x.title || x.name, originalTitle: x.original_name || "", overview: x.overview || "" }))
      });
    }
    case "metadata.detail": {
      const body = JSON.parse(fetch(BASE + "/3/" + payload.mediaType + "/" + payload.id + "?api_key=" + encodeURIComponent(tok)));
      return ok("metadata.detail", {
        id: body.id,
        mediaType: payload.mediaType,
        title: body.name || body.title,
        numberOfSeasons: body.number_of_seasons,
        numberOfEpisodes: body.number_of_episodes,
        genres: body.genres || []
      });
    }
    case "artwork.list": {
      const body = JSON.parse(fetch(BASE + "/3/" + payload.mediaType + "/" + payload.id + "/images?api_key=" + encodeURIComponent(tok)));
      return ok("artwork.list", {
        id: body.id,
        mediaType: payload.mediaType,
        baseUrl: "https://image.tmdb.org/t/p/",
        posters: body.posters.map(p => ({ filePath: p.file_path, url: "https://image.tmdb.org/t/p/original" + p.file_path, width: p.width, height: p.height })),
        backdrops: [],
        logos: [],
        stills: []
      });
    }
    default:
      return { ok: false, service: "service.tmdb", method, error: { code: "UNSUPPORTED_METHOD", message: "unsupported: " + method } };
  }
}
`, baseURL)
	return map[string]string{"service.json": manifest, "index.mjs": entry}
}

// bangumiZipFiles builds a self-contained Bangumi package matching the exact
// VeneraCat Sources (fa1efa4) contract: 6 official methods.
func bangumiZipFiles(baseURL, version string) map[string]string {
	manifest := fmt.Sprintf(`{"id":"service.bangumi","name":"Bangumi","version":%q,"kind":"service.js","entry":"index.mjs","permissions":["network","storage","secrets"]}`, version)
	entry := fmt.Sprintf(`const BASE = %q;
function ok(method, data) { return { ok: true, service: "service.bangumi", method, data }; }
export function describe() {
  return {
    methods: [
      { name: "account.me", title: "账号自检 / Account Profile" },
      { name: "metadata.search", title: "条目搜索 / Search Subjects" },
      { name: "metadata.detail", title: "条目详情 / Subject Detail" },
      { name: "collection.get", title: "读取收藏 / Get Collection" },
      { name: "collection.update", title: "更新收藏 / Update Collection" },
      { name: "episode.update", title: "更新章节进度 / Update Episode Progress" }
    ]
  };
}
export function handle(method, rawPayload) {
  const payload = rawPayload ? JSON.parse(rawPayload) : {};
  switch (method) {
    case "account.me": {
      const tok = secret("access_token");
      const u = JSON.parse(fetch(BASE + "/v0/me?token=" + encodeURIComponent(tok)));
      return ok("account.me", { id: u.id, username: u.username, nickname: u.nickname, url: "https://bgm.tv/user/" + u.username });
    }
    case "metadata.search": {
      const kw = payload.keyword || payload.query || "";
      const body = JSON.parse(fetch(BASE + "/v0/search/subjects?keyword=" + encodeURIComponent(kw)));
      return ok("metadata.search", {
        keyword: kw,
        items: body.data.map(s => ({ id: s.id, type: s.type, name: s.name, nameCn: s.name_cn, eps: s.eps })),
        total: body.total,
        limit: 20,
        offset: 0,
        fallbackUsed: false
      });
    }
    case "metadata.detail": {
      const id = payload.subjectId || payload.id;
      const s = JSON.parse(fetch(BASE + "/v0/subjects/" + id));
      return ok("metadata.detail", { id: s.id, type: s.type, name: s.name, nameCn: s.name_cn, eps: s.eps, totalEpisodes: s.total_episodes });
    }
    case "collection.get": {
      let username = payload.username || "";
      if (!username) {
        const tok = secret("access_token");
        const u = JSON.parse(fetch(BASE + "/v0/me?token=" + encodeURIComponent(tok)));
        username = u.username;
      }
      const body = JSON.parse(fetch(BASE + "/v0/users/" + username + "/collections"));
      return ok("collection.get", {
        username,
        items: body.data.map(x => ({ subjectId: x.subject_id, subjectType: x.subject_type, type: x.type, rate: x.rate, epStatus: x.ep_status })),
        total: body.total,
        limit: 30,
        offset: 0
      });
    }
    case "collection.update": {
      const tok = secret("access_token");
      fetch(BASE + "/v0/users/-/collections/" + payload.subjectId + "?token=" + encodeURIComponent(tok));
      return ok("collection.update", { subjectId: payload.subjectId, updated: true, payload: { type: payload.type, ep_status: payload.epStatus } });
    }
    case "episode.update": {
      const tok = secret("access_token");
      fetch(BASE + "/v0/users/-/collections/-/episodes/" + payload.episodeId + "?token=" + encodeURIComponent(tok));
      return ok("episode.update", { mode: "single", episodeId: payload.episodeId, type: 2, updated: true });
    }
    default:
      return { ok: false, service: "service.bangumi", method, error: { code: "UNSUPPORTED_METHOD", message: "unsupported: " + method } };
  }
}
`, baseURL)
	return map[string]string{"service.json": manifest, "index.mjs": entry}
}

func newStubProviderServer(t *testing.T) (*httptest.Server, chan struct{}, *string) {
	t.Helper()
	slowStarted := make(chan struct{}, 4)
	var mu sync.Mutex
	lastUA := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		lastUA = r.Header.Get("User-Agent")
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users/settings":
			if r.URL.Query().Get("cid") != "trakt-cid-123" || r.URL.Query().Get("token") != "trakt-secret-999" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"user":{"username":"lain","name":"Lain Iwakura","ids":{"slug":"lain-slug"}}}`))
		case "/search/show", "/search/movie,show":
			if r.URL.Query().Get("cid") != "trakt-cid-123" {
				http.Error(w, `{"error":"missing client_id"}`, http.StatusForbidden)
				return
			}
			_, _ = w.Write([]byte(`[{"type":"show","score":98.5,"show":{"title":"Serial Experiments Lain","year":1998,"ids":{"trakt":42,"slug":"serial-experiments-lain"}}}]`))
		case "/sync/watchlist":
			_, _ = w.Write([]byte(`[{"id":701,"rank":1,"listed_at":"2026-11-24T00:00:00Z","type":"show","show":{"title":"Serial Experiments Lain","year":1998,"ids":{"trakt":42}}}]`))
		case "/sync/history":
			_, _ = w.Write([]byte(`[{"id":801,"watched_at":"2026-11-24T12:00:00Z","action":"scrobble","type":"show","show":{"title":"Serial Experiments Lain","year":1998,"ids":{"trakt":42}}}]`))
		case "/scrobble/start":
			_, _ = w.Write([]byte(`{"id":901,"action":"start","progress":12.5}`))
		case "/scrobble/stop":
			_, _ = w.Write([]byte(`{"id":902,"action":"stop","progress":95.0}`))
		case "/3/search/tv", "/3/search/multi":
			key := r.URL.Query().Get("api_key")
			if key != "tmdb-read-secret-777" && key != "tmdb-apikey-888" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"page":1,"total_pages":1,"total_results":1,"results":[{"id":1396,"media_type":"tv","name":"绝命毒师","original_name":"Breaking Bad","overview":"chemistry"}]}`))
		case "/3/tv/1396":
			_, _ = w.Write([]byte(`{"id":1396,"name":"绝命毒师","number_of_seasons":5,"number_of_episodes":62,"genres":[{"id":18,"name":"剧情"}]}`))
		case "/3/tv/1396/images":
			_, _ = w.Write([]byte(`{"id":1396,"posters":[{"file_path":"/ggFHVNu6YYI5L9pCfOacjizRGt.jpg","width":1000,"height":1500}]}`))
		case "/v0/me":
			if r.URL.Query().Get("token") != "bgm-pat-secret-555" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"id":1001,"username":"nekocat","nickname":"猫猫"}`))
		case "/v0/search/subjects":
			_, _ = w.Write([]byte(`{"total":1,"data":[{"id":328609,"type":2,"name":"ぼっち・ざ・ろっく！","name_cn":"孤独摇滚！","eps":12}]}`))
		case "/v0/subjects/328609":
			_, _ = w.Write([]byte(`{"id":328609,"type":2,"name":"ぼっち・ざ・ろっく！","name_cn":"孤独摇滚！","eps":12,"total_episodes":12}`))
		case "/v0/users/nekocat/collections":
			_, _ = w.Write([]byte(`{"total":1,"data":[{"subject_id":328609,"subject_type":2,"type":3,"rate":9,"ep_status":8}]}`))
		case "/v0/users/-/collections/328609":
			_, _ = w.Write([]byte(`{}`))
		case "/v0/users/-/collections/-/episodes/110001":
			_, _ = w.Write([]byte(`{}`))
		case "/slow":
			select {
			case slowStarted <- struct{}{}:
			default:
			}
			<-r.Context().Done()
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, slowStarted, &lastUA
}

func TestTraktTMDBBangumiEndToEndTypedAdaptersAndUpgrade(t *testing.T) {
	ctx := context.Background()
	srv, _, lastUA := newStubProviderServer(t)
	dataDir := t.TempDir()
	zipDir := t.TempDir()

	traktZip := writeZipFile(t, zipDir, "trakt-v1.zip", traktZipFiles(srv.URL, "0.1.0"))
	tmdbZip := writeZipFile(t, zipDir, "tmdb-v1.zip", tmdbZipFiles(srv.URL, "0.1.0"))
	bgmZip := writeZipFile(t, zipDir, "bangumi-v1.zip", bangumiZipFiles(srv.URL, "0.1.0"))

	recT, reportT, err := InstallZipFile(ctx, dataDir, traktZip, ServiceIDTrakt)
	if err != nil {
		t.Fatalf("install trakt: %v", err)
	}
	if recT.ID != ServiceIDTrakt || len(reportT.Denied) != 1 || reportT.Denied[0] != service.PermissionSecrets {
		t.Fatalf("trakt install rec=%+v report=%+v", recT, reportT)
	}
	if _, _, err := InstallZipFile(ctx, dataDir, tmdbZip, ServiceIDTMDB); err != nil {
		t.Fatalf("install tmdb: %v", err)
	}
	if _, _, err := InstallZipFile(ctx, dataDir, bgmZip, ServiceIDBangumi); err != nil {
		t.Fatalf("install bangumi: %v", err)
	}

	var logMu sync.Mutex
	var capturedLogs []string
	opts := HostOptions{
		UserAgent: "Nekophage-Test/1.0",
		Log: func(level, message string) {
			logMu.Lock()
			capturedLogs = append(capturedLogs, level+":"+message)
			logMu.Unlock()
		},
	}

	trakt, err := NewTraktClient(dataDir, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer trakt.Close()

	// Trakt metadata.search requires client_id; missing must report client_id.
	if _, err := trakt.MetadataSearch(ctx, TraktSearchRequest{Query: "lain"}); !errors.Is(err, ErrMissingSecrets) || !strings.Contains(err.Error(), SecretKeyClientID) {
		t.Fatalf("missing client_id error = %v", err)
	}
	if err := trakt.SetAccount(ctx, SecretKeyClientID, "trakt-cid-123"); err != nil {
		t.Fatal(err)
	}
	// account.me still requires access_token in addition to client_id.
	if _, err := trakt.AccountMe(ctx); !errors.Is(err, ErrMissingSecrets) || !strings.Contains(err.Error(), SecretKeyAccessToken) {
		t.Fatalf("missing access_token error = %v", err)
	}
	if err := trakt.SetAccount(ctx, SecretKeyAccessToken, "trakt-secret-999"); err != nil {
		t.Fatal(err)
	}

	methods, err := trakt.Describe(ctx)
	if err != nil || len(methods) != 6 {
		t.Fatalf("trakt describe = %+v (%v)", methods, err)
	}
	me, err := trakt.AccountMe(ctx)
	if err != nil || me.Username != "lain" || me.Slug != "lain-slug" {
		t.Fatalf("trakt account.me = %+v (%v)", me, err)
	}
	if *lastUA != "Nekophage-Test/1.0" {
		t.Fatalf("User-Agent = %q, want Nekophage-Test/1.0", *lastUA)
	}
	search, err := trakt.MetadataSearch(ctx, TraktSearchRequest{Query: "lain", Type: "show"})
	if err != nil || len(search.Items) != 1 || search.Items[0].Title != "Serial Experiments Lain" {
		t.Fatalf("trakt search = %+v (%v)", search, err)
	}
	wl, err := trakt.Watchlist(ctx, TraktWatchlistRequest{Type: "shows"})
	if err != nil || len(wl.Items) != 1 || wl.Items[0].ID != 701 {
		t.Fatalf("trakt watchlist = %+v (%v)", wl, err)
	}
	hist, err := trakt.History(ctx, TraktHistoryRequest{Type: "shows"})
	if err != nil || len(hist.Items) != 1 || hist.Items[0].ID != 801 {
		t.Fatalf("trakt history = %+v (%v)", hist, err)
	}
	scStart, err := trakt.ScrobbleStart(ctx, TraktScrobbleRequest{Progress: 12.5, Episode: map[string]any{"ids": map[string]any{"trakt": 1}}})
	if err != nil || scStart.Action != "start" || scStart.Progress != 12.5 {
		t.Fatalf("trakt scrobble.start = %+v (%v)", scStart, err)
	}
	scStop, err := trakt.ScrobbleStop(ctx, TraktScrobbleRequest{Progress: 95.0, Episode: map[string]any{"ids": map[string]any{"trakt": 1}}})
	if err != nil || scStop.Action != "stop" || scStop.Progress != 95.0 {
		t.Fatalf("trakt scrobble.stop = %+v (%v)", scStop, err)
	}

	// Verify log redaction: neither client_id nor access_token may appear in host logs.
	logMu.Lock()
	joinedLogs := strings.Join(capturedLogs, "\n")
	logMu.Unlock()
	if strings.Contains(joinedLogs, "trakt-secret-999") || strings.Contains(joinedLogs, "trakt-cid-123") || !strings.Contains(joinedLogs, "[REDACTED]") {
		t.Fatalf("secret leaked or not redacted in logs: %q", joinedLogs)
	}

	// Upgrade Trakt zip to 0.2.0 and confirm both isolated secrets survive upgrade.
	traktZipV2 := writeZipFile(t, zipDir, "trakt-v2.zip", traktZipFiles(srv.URL, "0.2.0"))
	recV2, reportV2, err := InstallZipFile(ctx, dataDir, traktZipV2, ServiceIDTrakt)
	if err != nil {
		t.Fatalf("upgrade trakt: %v", err)
	}
	if recV2.Version != "0.2.0" || recV2.Accounts != 2 || len(reportV2.Denied) != 0 {
		t.Fatalf("upgraded trakt rec=%+v report=%+v", recV2, reportV2)
	}

	// TMDB typed workflow: verify api_read_token OR api_key fallback.
	tmdb, err := NewTMDBClient(dataDir, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer tmdb.Close()
	if _, err := tmdb.MetadataSearch(ctx, TMDBSearchRequest{Query: "毒师"}); !errors.Is(err, ErrMissingSecrets) || !strings.Contains(err.Error(), SecretKeyAPIReadToken) || !strings.Contains(err.Error(), SecretKeyAPIKey) {
		t.Fatalf("tmdb missing secret = %v", err)
	}
	// Configure api_key only (without api_read_token) and verify calls succeed.
	if err := tmdb.SetAccount(ctx, SecretKeyAPIKey, "tmdb-apikey-888"); err != nil {
		t.Fatal(err)
	}
	if m, err := tmdb.Describe(ctx); err != nil || len(m) != 3 {
		t.Fatalf("tmdb describe = %+v (%v)", m, err)
	}
	if res, err := tmdb.MetadataSearch(ctx, TMDBSearchRequest{Query: "毒师", MediaType: "tv"}); err != nil || len(res.Items) != 1 || res.Items[0].ID != 1396 || res.Items[0].Title != "绝命毒师" {
		t.Fatalf("tmdb search = %+v (%v)", res, err)
	}
	if d, err := tmdb.MetadataDetail(ctx, TMDBDetailRequest{ID: 1396, MediaType: "tv"}); err != nil || d.NumberOfSeasons == nil || *d.NumberOfSeasons != 5 || len(d.Genres) != 1 {
		t.Fatalf("tmdb detail = %+v (%v)", d, err)
	}
	if art, err := tmdb.ArtworkList(ctx, TMDBArtworkRequest{ID: 1396, MediaType: "tv"}); err != nil || len(art.Posters) != 1 || !strings.HasPrefix(*art.Posters[0].URL, "https://image.tmdb.org/") {
		t.Fatalf("tmdb artwork = %+v (%v)", art, err)
	}

	// Bangumi typed workflow.
	bgm, err := NewBangumiClient(dataDir, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer bgm.Close()
	if _, err := bgm.CollectionGet(ctx, BangumiCollectionGetRequest{}); !errors.Is(err, ErrMissingSecrets) {
		t.Fatalf("bgm collection.get without username or token = %v", err)
	}
	// Public collection.get with explicit username works without token.
	if col, err := bgm.CollectionGet(ctx, BangumiCollectionGetRequest{Username: "nekocat"}); err != nil || len(col.Items) != 1 || col.Items[0].SubjectID != 328609 {
		t.Fatalf("bgm public collection.get = %+v (%v)", col, err)
	}
	if err := bgm.SetAccount(ctx, SecretKeyAccessToken, "bgm-pat-secret-555"); err != nil {
		t.Fatal(err)
	}
	if m, err := bgm.Describe(ctx); err != nil || len(m) != 6 {
		t.Fatalf("bgm describe = %+v (%v)", m, err)
	}
	if me, err := bgm.AccountMe(ctx); err != nil || me.ID != 1001 || me.Username != "nekocat" {
		t.Fatalf("bgm me = %+v (%v)", me, err)
	}
	if res, err := bgm.MetadataSearch(ctx, BangumiSearchRequest{Keyword: "孤独摇滚"}); err != nil || len(res.Items) != 1 || res.Items[0].ID != 328609 {
		t.Fatalf("bgm search = %+v (%v)", res, err)
	}
	if d, err := bgm.MetadataDetail(ctx, BangumiDetailRequest{SubjectID: 328609}); err != nil || d.TotalEpisodes == nil || *d.TotalEpisodes != 12 {
		t.Fatalf("bgm detail = %+v (%v)", d, err)
	}
	epStat := 9
	if upd, err := bgm.CollectionUpdate(ctx, BangumiCollectionUpdateRequest{SubjectID: 328609, Type: "doing", EpStatus: &epStat}); err != nil || !upd.Updated {
		t.Fatalf("bgm collection.update = %+v (%v)", upd, err)
	}
	if epUpd, err := bgm.EpisodeUpdate(ctx, BangumiEpisodeUpdateRequest{EpisodeID: 110001, Type: "done"}); err != nil || !epUpd.Updated || epUpd.EpisodeID != 110001 {
		t.Fatalf("bgm episode.update = %+v (%v)", epUpd, err)
	}
}

func TestPermissionTrimmingAndStrictDenial(t *testing.T) {
	ctx := context.Background()
	srv, _, _ := newStubProviderServer(t)
	dataDir := t.TempDir()
	zipPath := writeZipFile(t, t.TempDir(), "bangumi.zip", bangumiZipFiles(srv.URL, "0.1.0"))
	if _, _, err := InstallZipFile(ctx, dataDir, zipPath, ServiceIDBangumi); err != nil {
		t.Fatal(err)
	}

	// Trim secrets permission while allowing network + storage.
	trimmed, err := NewBangumiClient(dataDir, HostOptions{
		AllowedPermissions: []service.Permission{service.PermissionNetwork, service.PermissionStorage},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer trimmed.Close()

	report, err := trimmed.Permissions()
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Denied) != 1 || report.Denied[0] != service.PermissionSecrets {
		t.Fatalf("report = %+v, want denied=[secrets]", report)
	}
	// Public metadata search still succeeds.
	if res, err := trimmed.MetadataSearch(ctx, BangumiSearchRequest{Keyword: "孤独摇滚"}); err != nil || len(res.Items) != 1 {
		t.Fatalf("search under trimmed secrets = %+v (%v)", res, err)
	}
	// Authenticated call must fail explicitly naming denied secrets permission.
	_, err = trimmed.AccountMe(ctx)
	var deniedErr *PermissionDeniedError
	if !errors.As(err, &deniedErr) || !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("AccountMe error = %v, want PermissionDeniedError", err)
	}
	if len(deniedErr.Denied) != 1 || deniedErr.Denied[0] != service.PermissionSecrets {
		t.Fatalf("denied items = %v", deniedErr.Denied)
	}

	// StrictPermissions refuses OpenWithOptions upfront when network/secrets are missing.
	_, _, rep, err := OpenWithOptions(ctx, dataDir, ServiceIDBangumi, HostOptions{
		AllowedPermissions: []service.Permission{service.PermissionStorage},
		StrictPermissions:  true,
	})
	if !errors.Is(err, ErrPermissionDenied) || len(rep.Denied) != 2 {
		t.Fatalf("strict open err=%v report=%+v", err, rep)
	}
}

func TestSecretLeakageBlockedInKVStorageAndErrors(t *testing.T) {
	ctx := context.Background()
	srv, _, _ := newStubProviderServer(t)
	dataDir := t.TempDir()
	zipPath := writeZipFile(t, t.TempDir(), "trakt.zip", traktZipFiles(srv.URL, "0.1.0"))
	if _, _, err := InstallZipFile(ctx, dataDir, zipPath, ServiceIDTrakt); err != nil {
		t.Fatal(err)
	}

	trakt, err := NewTraktClient(dataDir, HostOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer trakt.Close()
	if err := trakt.SetAccount(ctx, SecretKeyAccessToken, "trakt-secret-999"); err != nil {
		t.Fatal(err)
	}

	// Attempt to copy secret into ordinary KV storage must be refused.
	if _, err := trakt.Call(ctx, "leak.storage", nil); err == nil {
		t.Fatal("secret was written to ordinary KV storage")
	}
	storageBytes, _ := os.ReadFile(filepath.Join(dataDir, "services", ServiceIDTrakt+".json"))
	if strings.Contains(string(storageBytes), "trakt-secret-999") {
		t.Fatalf("ordinary KV storage contains secret: %s", storageBytes)
	}

	// Error thrown by JS containing secret must be redacted.
	_, err = trakt.Call(ctx, "leak.error", nil)
	if err == nil {
		t.Fatal("expected error from leak.error")
	}
	if strings.Contains(err.Error(), "trakt-secret-999") || !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("secret leaked in error message: %v", err)
	}
}

func TestProviderTimeoutAndCancellation(t *testing.T) {
	srv, slowStarted, _ := newStubProviderServer(t)
	dataDir := t.TempDir()
	zipPath := writeZipFile(t, t.TempDir(), "trakt.zip", traktZipFiles(srv.URL, "0.1.0"))
	if _, _, err := InstallZipFile(context.Background(), dataDir, zipPath, ServiceIDTrakt); err != nil {
		t.Fatal(err)
	}

	// Supervisor timeout aborts slow HTTP request.
	traktTimed, err := NewTraktClient(dataDir, HostOptions{
		SupervisorOptions: service.SupervisorOptions{Timeout: 180 * time.Millisecond},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer traktTimed.Close()
	if _, err := traktTimed.Call(context.Background(), "slow", nil); !errors.Is(err, service.ErrTimeout) {
		t.Fatalf("timeout error = %v, want service.ErrTimeout", err)
	}

	// Explicit context cancellation aborts slow HTTP request promptly and client recovers on next call.
	trakt, err := NewTraktClient(dataDir, HostOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer trakt.Close()
	if err := trakt.SetAccount(context.Background(), SecretKeyClientID, "trakt-cid-123"); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := trakt.Call(ctx, "slow", nil)
		done <- err
	}()
	select {
	case <-slowStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("slow request did not start")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel error = %v, want context.Canceled", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cancelled call did not return")
	}

	// After cancellation, subsequent call loads cleanly and succeeds.
	if res, err := trakt.MetadataSearch(context.Background(), TraktSearchRequest{Query: "lain", Type: "show"}); err != nil || len(res.Items) != 1 {
		t.Fatalf("post-cancel call = %+v (%v)", res, err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func stubJSONResp(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}

// TestFetchInitForwardsMethodHeadersBodyAndNon2xxStatus verifies fetch(url, init)
// forwarding of HTTP method (POST/PATCH), Authorization/trakt-api-key headers,
// request body, and non-2xx status codes through Core's service.Bridge and
// HostOptions.Transport without requiring sibling repositories.
func TestFetchInitForwardsMethodHeadersBodyAndNon2xxStatus(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	manifest := `{"id":"service.trakt","name":"Trakt","version":"0.1.0","kind":"service.js","entry":"index.mjs","permissions":["network","storage","secrets"]}`
	entry := `export function handle(method, rawPayload) {
  const payload = rawPayload ? JSON.parse(rawPayload) : {};
  const cid = secret("client_id");
  const tok = secret("access_token");
  if (method === "post.check") {
    const res = fetch("https://api.trakt.tv/scrobble/start", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Authorization": "Bearer " + tok,
        "trakt-api-key": cid,
        "trakt-api-version": "2"
      },
      body: JSON.stringify({ progress: payload.progress, movie: payload.movie })
    });
    return { status: res.status, ok: res.ok, body: res.body };
  }
  if (method === "patch.check") {
    const res = fetch("https://api.trakt.tv/sync/collection", {
      method: "PATCH",
      headers: {
        "Content-Type": "application/json",
        "Authorization": "Bearer " + tok,
        "trakt-api-key": cid
      },
      body: JSON.stringify({ episode_id: [1, 2], type: 2 })
    });
    return { status: res.status, ok: res.ok, body: res.body };
  }
  if (method === "status.401") {
    const res = fetch("https://api.trakt.tv/users/settings", {
      method: "GET",
      headers: { "Authorization": "Bearer expired", "trakt-api-key": cid }
    });
    return { status: res.status, ok: res.ok, body: res.body };
  }
  throw new Error("unknown: " + method);
}
`
	zipPath := writeZipFile(t, t.TempDir(), "trakt-fetch-init.zip", map[string]string{
		"service.json": manifest,
		"index.mjs":    entry,
	})
	if _, _, err := InstallZipFile(ctx, dataDir, zipPath, ServiceIDTrakt); err != nil {
		t.Fatal(err)
	}

	var gotMethods []string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotMethods = append(gotMethods, r.Method+" "+r.URL.Path)
		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, _ = io.ReadAll(r.Body)
		}
		switch r.URL.Path {
		case "/scrobble/start":
			if r.Method != http.MethodPost {
				t.Errorf("POST method = %s, want POST", r.Method)
			}
			if r.Header.Get("Authorization") != "Bearer init-tok" || r.Header.Get("trakt-api-key") != "init-cid" || r.Header.Get("trakt-api-version") != "2" {
				t.Errorf("unexpected POST headers: %v", r.Header)
			}
			if !strings.Contains(string(bodyBytes), `"progress":42.5`) {
				t.Errorf("unexpected POST body: %s", bodyBytes)
			}
			return stubJSONResp(201, `{"created":true}`), nil
		case "/sync/collection":
			if r.Method != http.MethodPatch {
				t.Errorf("PATCH method = %s, want PATCH", r.Method)
			}
			if r.Header.Get("Authorization") != "Bearer init-tok" || r.Header.Get("trakt-api-key") != "init-cid" {
				t.Errorf("unexpected PATCH headers: %v", r.Header)
			}
			if !strings.Contains(string(bodyBytes), `"episode_id":[1,2]`) {
				t.Errorf("unexpected PATCH body: %s", bodyBytes)
			}
			return stubJSONResp(200, `{"patched":true}`), nil
		case "/users/settings":
			return stubJSONResp(401, `{"error":"invalid_token"}`), nil
		default:
			return nil, fmt.Errorf("unexpected request %s %s", r.Method, r.URL.String())
		}
	})

	client, err := NewClient(dataDir, ServiceIDTrakt, HostOptions{
		BridgePolicy: service.BridgePolicy{Hosts: DefaultHosts(ServiceIDTrakt)},
		Transport:    transport,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.SetAccount(ctx, SecretKeyClientID, "init-cid"); err != nil {
		t.Fatal(err)
	}
	if err := client.SetAccount(ctx, SecretKeyAccessToken, "init-tok"); err != nil {
		t.Fatal(err)
	}

	var res struct {
		Status int    `json:"status"`
		OK     bool   `json:"ok"`
		Body   string `json:"body"`
	}
	if err := client.CallJSON(ctx, "post.check", map[string]any{"progress": 42.5, "movie": map[string]any{"ids": map[string]any{"trakt": 1}}}, &res, nil, nil); err != nil {
		t.Fatalf("post.check: %v", err)
	}
	if res.Status != 201 || !res.OK || !strings.Contains(res.Body, `"created":true`) {
		t.Fatalf("post.check res = %+v", res)
	}
	if err := client.CallJSON(ctx, "patch.check", nil, &res, nil, nil); err != nil {
		t.Fatalf("patch.check: %v", err)
	}
	if res.Status != 200 || !res.OK || !strings.Contains(res.Body, `"patched":true`) {
		t.Fatalf("patch.check res = %+v", res)
	}
	if err := client.CallJSON(ctx, "status.401", nil, &res, nil, nil); err != nil {
		t.Fatalf("status.401: %v", err)
	}
	if res.Status != 401 || res.OK || !strings.Contains(res.Body, `invalid_token`) {
		t.Fatalf("status.401 res = %+v", res)
	}
	if len(gotMethods) != 3 {
		t.Fatalf("methods = %v", gotMethods)
	}
}

// TestOfficialVeneraCatSourcesArtifactsContract verifies Nekophage against the
// actual built artifacts in the sibling `VeneraCat Sources` checkout when
// present on disk (skipped cleanly in standalone CI checkouts).
func TestOfficialVeneraCatSourcesArtifactsContract(t *testing.T) {
	sourcesRoot := filepath.Join("..", "..", "..", "VeneraCat Sources")
	modPath := filepath.Join(sourcesRoot, "templates", "modules", "reader-preferences.vcmodule")
	traktPath := filepath.Join(sourcesRoot, "build", "packages", "service.trakt", "service.trakt.zip")
	tmdbPath := filepath.Join(sourcesRoot, "build", "packages", "service.tmdb", "service.tmdb.zip")
	bgmPath := filepath.Join(sourcesRoot, "build", "packages", "service.bangumi", "service.bangumi.zip")
	for _, p := range []string{modPath, traktPath, tmdbPath, bgmPath} {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("sibling VeneraCat Sources artifact %s unavailable: %v", p, err)
		}
	}

	ctx := context.Background()
	dataDir := t.TempDir()

	// 1. Parse and install actual reader-preferences.vcmodule template directly.
	mgr, err := vcmodule.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	modRec, err := mgr.InstallFile("", modPath)
	if err != nil {
		t.Fatalf("install official reader-preferences.vcmodule: %v", err)
	}
	if modRec.ID != "reader-preferences" || len(modRec.Requires) != 1 || modRec.Requires[0] != vcmodule.Capability {
		t.Fatalf("official vcmodule record = %+v", modRec)
	}
	if err := mgr.Enable(modRec.ID); err != nil {
		t.Fatal(err)
	}
	if eff := mgr.Resolve(vcmodule.DefaultSettings()).Effective.Reader; eff.PageGap != 8 || !eff.KeepScreenOn {
		t.Fatalf("official vcmodule overlay = %+v", eff)
	}

	// 2. Install all three real built zip packages and verify describe(),
	// headers (Authorization / trakt-api-key), methods (GET/POST/PUT/PATCH),
	// JSON request bodies, and non-2xx HTTP status forwarding through
	// HostOptions.Transport while enforcing DefaultHosts(id) allowlists!
	if _, _, err := InstallZipFile(ctx, dataDir, traktPath, ServiceIDTrakt); err != nil {
		t.Fatalf("install official service.trakt.zip: %v", err)
	}
	if _, _, err := InstallZipFile(ctx, dataDir, tmdbPath, ServiceIDTMDB); err != nil {
		t.Fatalf("install official service.tmdb.zip: %v", err)
	}
	if _, _, err := InstallZipFile(ctx, dataDir, bgmPath, ServiceIDBangumi); err != nil {
		t.Fatalf("install official service.bangumi.zip: %v", err)
	}

	var mu sync.Mutex
	seen := map[string]int{}
	offlineTransport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, _ = io.ReadAll(r.Body)
		}
		key := r.Method + " " + r.URL.Host + r.URL.Path
		mu.Lock()
		seen[key]++
		mu.Unlock()

		switch r.URL.Host + r.URL.Path {
		case "api.trakt.tv/users/settings":
			if r.Method != http.MethodGet || r.Header.Get("trakt-api-key") != "official-cid" || r.Header.Get("trakt-api-version") != "2" {
				return stubJSONResp(400, `{"error":"bad trakt headers"}`), nil
			}
			if r.Header.Get("Authorization") != "Bearer official-tok" {
				return stubJSONResp(401, `{"error":"invalid_grant"}`), nil
			}
			return stubJSONResp(200, `{"user":{"username":"lain","name":"Lain","ids":{"slug":"lain"}}}`), nil
		case "api.trakt.tv/search/show":
			if r.Method != http.MethodGet || r.Header.Get("trakt-api-key") != "official-cid" {
				return stubJSONResp(403, `{"error":"missing trakt-api-key"}`), nil
			}
			return stubJSONResp(200, `[{"type":"show","score":99,"show":{"title":"Serial Experiments Lain","year":1998,"ids":{"trakt":42}}}]`), nil
		case "api.trakt.tv/sync/watchlist/shows":
			if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer official-tok" || r.Header.Get("trakt-api-key") != "official-cid" {
				return stubJSONResp(401, `{"error":"unauthorized"}`), nil
			}
			return stubJSONResp(200, `[{"id":11,"rank":1,"listed_at":"2026-11-24T00:00:00Z","type":"show","show":{"title":"Serial Experiments Lain","year":1998,"ids":{"trakt":42}}}]`), nil
		case "api.trakt.tv/sync/history/shows":
			if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer official-tok" || r.Header.Get("trakt-api-key") != "official-cid" {
				return stubJSONResp(401, `{"error":"unauthorized"}`), nil
			}
			return stubJSONResp(200, `[{"id":22,"watched_at":"2026-11-24T01:00:00Z","action":"scrobble","type":"show","show":{"title":"Serial Experiments Lain","year":1998,"ids":{"trakt":42}}}]`), nil
		case "api.trakt.tv/scrobble/start":
			if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer official-tok" || r.Header.Get("trakt-api-key") != "official-cid" {
				return stubJSONResp(401, `{"error":"unauthorized"}`), nil
			}
			var body map[string]any
			if err := json.Unmarshal(bodyBytes, &body); err != nil || body["progress"] != float64(15) || body["movie"] == nil {
				return stubJSONResp(400, `{"error":"bad scrobble start body"}`), nil
			}
			return stubJSONResp(201, `{"id":33,"action":"start","progress":15}`), nil
		case "api.trakt.tv/scrobble/stop":
			if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer official-tok" || r.Header.Get("trakt-api-key") != "official-cid" {
				return stubJSONResp(401, `{"error":"unauthorized"}`), nil
			}
			var body map[string]any
			if err := json.Unmarshal(bodyBytes, &body); err != nil || body["progress"] != float64(90) || body["movie"] == nil {
				return stubJSONResp(400, `{"error":"bad scrobble stop body"}`), nil
			}
			return stubJSONResp(201, `{"id":34,"action":"stop","progress":90}`), nil
		case "api.themoviedb.org/3/search/tv":
			if r.Method != http.MethodGet {
				return stubJSONResp(405, `{"error":"method not allowed"}`), nil
			}
			if r.Header.Get("Authorization") != "Bearer official-tmdb-bearer" {
				return stubJSONResp(401, `{"status_message":"Invalid API key"}`), nil
			}
			return stubJSONResp(200, `{"page":1,"total_pages":1,"total_results":1,"results":[{"id":1396,"name":"绝命毒师","original_name":"Breaking Bad"}]}`), nil
		case "api.themoviedb.org/3/tv/1396":
			if r.Header.Get("Authorization") != "Bearer official-tmdb-bearer" {
				return stubJSONResp(401, `{"status_message":"Invalid API key"}`), nil
			}
			return stubJSONResp(200, `{"id":1396,"name":"绝命毒师","number_of_seasons":5,"number_of_episodes":62,"genres":[{"id":18,"name":"剧情"}]}`), nil
		case "api.themoviedb.org/3/tv/1396/images":
			if r.Header.Get("Authorization") != "Bearer official-tmdb-bearer" {
				return stubJSONResp(401, `{"status_message":"Invalid API key"}`), nil
			}
			return stubJSONResp(200, `{"id":1396,"posters":[{"file_path":"/poster.jpg","width":1000,"height":1500}]}`), nil
		case "api.bgm.tv/v0/me":
			if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer official-bgm-pat" {
				return stubJSONResp(401, `{"title":"Unauthorized"}`), nil
			}
			return stubJSONResp(200, `{"id":1001,"username":"nekocat","nickname":"猫猫"}`), nil
		case "api.bgm.tv/v0/search/subjects":
			if r.Method != http.MethodPost {
				return stubJSONResp(405, `{"error":"expected POST"}`), nil
			}
			var body map[string]any
			if err := json.Unmarshal(bodyBytes, &body); err != nil || body["keyword"] != "孤独摇滚" {
				return stubJSONResp(400, `{"error":"missing keyword in POST body"}`), nil
			}
			return stubJSONResp(200, `{"total":1,"limit":20,"offset":0,"data":[{"id":328609,"type":2,"name":"ぼっち・ざ・ろっく！","name_cn":"孤独摇滚！","eps":12}]}`), nil
		case "api.bgm.tv/v0/subjects/328609":
			return stubJSONResp(200, `{"id":328609,"type":2,"name":"ぼっち・ざ・ろっく！","name_cn":"孤独摇滚！","eps":12,"total_episodes":12}`), nil
		case "api.bgm.tv/v0/users/nekocat/collections":
			if r.Header.Get("Authorization") != "Bearer official-bgm-pat" {
				return stubJSONResp(401, `{"title":"Unauthorized"}`), nil
			}
			return stubJSONResp(200, `{"total":1,"limit":30,"offset":0,"data":[{"subject_id":328609,"subject_type":2,"type":3,"rate":9,"ep_status":8}]}`), nil
		case "api.bgm.tv/v0/users/-/collections/328609":
			if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer official-bgm-pat" {
				return stubJSONResp(401, `{"title":"Unauthorized"}`), nil
			}
			var body map[string]any
			if err := json.Unmarshal(bodyBytes, &body); err != nil || body["type"] != float64(3) || body["ep_status"] != float64(9) {
				return stubJSONResp(400, `{"error":"bad collection update body"}`), nil
			}
			return stubJSONResp(202, ``), nil
		case "api.bgm.tv/v0/users/-/collections/-/episodes/110001":
			if r.Method != http.MethodPut || r.Header.Get("Authorization") != "Bearer official-bgm-pat" {
				return stubJSONResp(401, `{"title":"Unauthorized"}`), nil
			}
			var body map[string]any
			if err := json.Unmarshal(bodyBytes, &body); err != nil || body["type"] != float64(2) {
				return stubJSONResp(400, `{"error":"bad episode PUT body"}`), nil
			}
			return stubJSONResp(204, ``), nil
		case "api.bgm.tv/v0/users/-/collections/328609/episodes":
			if r.Method != http.MethodPatch || r.Header.Get("Authorization") != "Bearer official-bgm-pat" {
				return stubJSONResp(401, `{"title":"Unauthorized"}`), nil
			}
			var body map[string]any
			if err := json.Unmarshal(bodyBytes, &body); err != nil || body["type"] != float64(2) || body["episode_id"] == nil {
				return stubJSONResp(400, `{"error":"bad episode PATCH body"}`), nil
			}
			return stubJSONResp(204, ``), nil
		default:
			return nil, fmt.Errorf("unexpected offline request: %s %s", r.Method, r.URL.String())
		}
	})

	// Official Trakt zip: verify Authorization, trakt-api-key, POST scrobble bodies, and non-2xx 401 status forwarding.
	trakt, err := NewTraktClient(dataDir, HostOptions{
		BridgePolicy: service.BridgePolicy{Hosts: DefaultHosts(ServiceIDTrakt)},
		Transport:    offlineTransport,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer trakt.Close()
	if err := trakt.SetAccount(ctx, SecretKeyClientID, "official-cid"); err != nil {
		t.Fatal(err)
	}
	if err := trakt.SetAccount(ctx, SecretKeyAccessToken, "expired-tok"); err != nil {
		t.Fatal(err)
	}
	_, err = trakt.AccountMe(ctx)
	var svcErr *ServiceError
	if !errors.As(err, &svcErr) || svcErr.Code != "AUTH_ERROR" || svcErr.Status != 401 {
		t.Fatalf("official trakt 401 error = %v (want ServiceError AUTH_ERROR status 401)", err)
	}
	if err := trakt.SetAccount(ctx, SecretKeyAccessToken, "official-tok"); err != nil {
		t.Fatal(err)
	}
	if m, err := trakt.Describe(ctx); err != nil || len(m) != 6 {
		t.Fatalf("official trakt describe = %+v (%v)", m, err)
	}
	if me, err := trakt.AccountMe(ctx); err != nil || me.Username != "lain" {
		t.Fatalf("official trakt account.me = %+v (%v)", me, err)
	}
	if res, err := trakt.MetadataSearch(ctx, TraktSearchRequest{Query: "lain", Type: "show"}); err != nil || len(res.Items) != 1 {
		t.Fatalf("official trakt search = %+v (%v)", res, err)
	}
	if wl, err := trakt.Watchlist(ctx, TraktWatchlistRequest{Type: "shows"}); err != nil || len(wl.Items) != 1 {
		t.Fatalf("official trakt watchlist = %+v (%v)", wl, err)
	}
	if h, err := trakt.History(ctx, TraktHistoryRequest{Type: "shows"}); err != nil || len(h.Items) != 1 {
		t.Fatalf("official trakt history = %+v (%v)", h, err)
	}
	if sc, err := trakt.ScrobbleStart(ctx, TraktScrobbleRequest{Progress: 15, Movie: map[string]any{"ids": map[string]any{"trakt": 1}}}); err != nil || sc.Action != "start" {
		t.Fatalf("official trakt scrobble.start = %+v (%v)", sc, err)
	}
	if sc, err := trakt.ScrobbleStop(ctx, TraktScrobbleRequest{Progress: 90, Movie: map[string]any{"ids": map[string]any{"trakt": 1}}}); err != nil || sc.Action != "stop" {
		t.Fatalf("official trakt scrobble.stop = %+v (%v)", sc, err)
	}

	// Official TMDB zip: verify Bearer Authorization header forwarding and non-2xx 401 handling.
	tmdb, err := NewTMDBClient(dataDir, HostOptions{
		BridgePolicy: service.BridgePolicy{Hosts: DefaultHosts(ServiceIDTMDB)},
		Transport:    offlineTransport,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer tmdb.Close()
	if err := tmdb.SetAccount(ctx, SecretKeyAPIReadToken, "bad-tmdb-bearer"); err != nil {
		t.Fatal(err)
	}
	_, err = tmdb.MetadataSearch(ctx, TMDBSearchRequest{Query: "毒师", MediaType: "tv"})
	if !errors.As(err, &svcErr) || svcErr.Code != "AUTH_ERROR" || svcErr.Status != 401 {
		t.Fatalf("official tmdb 401 error = %v (want ServiceError AUTH_ERROR status 401)", err)
	}
	if err := tmdb.SetAccount(ctx, SecretKeyAPIReadToken, "official-tmdb-bearer"); err != nil {
		t.Fatal(err)
	}
	if m, err := tmdb.Describe(ctx); err != nil || len(m) != 3 {
		t.Fatalf("official tmdb describe = %+v (%v)", m, err)
	}
	if res, err := tmdb.MetadataSearch(ctx, TMDBSearchRequest{Query: "毒师", MediaType: "tv"}); err != nil || len(res.Items) != 1 || res.Items[0].ID != 1396 {
		t.Fatalf("official tmdb search = %+v (%v)", res, err)
	}
	if d, err := tmdb.MetadataDetail(ctx, TMDBDetailRequest{ID: 1396, MediaType: "tv"}); err != nil || d.NumberOfSeasons == nil || *d.NumberOfSeasons != 5 {
		t.Fatalf("official tmdb detail = %+v (%v)", d, err)
	}
	if art, err := tmdb.ArtworkList(ctx, TMDBArtworkRequest{ID: 1396, MediaType: "tv"}); err != nil || len(art.Posters) != 1 {
		t.Fatalf("official tmdb artwork = %+v (%v)", art, err)
	}

	// Official Bangumi zip: verify Authorization, POST/PUT/PATCH methods + JSON bodies.
	bgm, err := NewBangumiClient(dataDir, HostOptions{
		BridgePolicy: service.BridgePolicy{Hosts: DefaultHosts(ServiceIDBangumi)},
		Transport:    offlineTransport,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer bgm.Close()
	if err := bgm.SetAccount(ctx, SecretKeyAccessToken, "official-bgm-pat"); err != nil {
		t.Fatal(err)
	}
	if m, err := bgm.Describe(ctx); err != nil || len(m) != 6 {
		t.Fatalf("official bangumi describe = %+v (%v)", m, err)
	}
	if me, err := bgm.AccountMe(ctx); err != nil || me.Username != "nekocat" {
		t.Fatalf("official bangumi account.me = %+v (%v)", me, err)
	}
	if res, err := bgm.MetadataSearch(ctx, BangumiSearchRequest{Keyword: "孤独摇滚"}); err != nil || len(res.Items) != 1 || res.FallbackUsed {
		t.Fatalf("official bangumi search = %+v (%v)", res, err)
	}
	if d, err := bgm.MetadataDetail(ctx, BangumiDetailRequest{SubjectID: 328609}); err != nil || d.TotalEpisodes == nil || *d.TotalEpisodes != 12 {
		t.Fatalf("official bangumi detail = %+v (%v)", d, err)
	}
	if col, err := bgm.CollectionGet(ctx, BangumiCollectionGetRequest{}); err != nil || len(col.Items) != 1 || col.Items[0].SubjectID != 328609 {
		t.Fatalf("official bangumi collection.get = %+v (%v)", col, err)
	}
	epStat := 9
	if upd, err := bgm.CollectionUpdate(ctx, BangumiCollectionUpdateRequest{SubjectID: 328609, Type: "doing", EpStatus: &epStat}); err != nil || !upd.Updated {
		t.Fatalf("official bangumi collection.update = %+v (%v)", upd, err)
	}
	if epUpd, err := bgm.EpisodeUpdate(ctx, BangumiEpisodeUpdateRequest{EpisodeID: 110001, Type: "done"}); err != nil || !epUpd.Updated || epUpd.Mode != "single" {
		t.Fatalf("official bangumi episode.update single = %+v (%v)", epUpd, err)
	}
	if epBatch, err := bgm.EpisodeUpdate(ctx, BangumiEpisodeUpdateRequest{SubjectID: 328609, EpisodeIDs: []int64{110001, 110002}, Type: "done"}); err != nil || !epBatch.Updated || epBatch.Mode != "batch" {
		t.Fatalf("official bangumi episode.update batch = %+v (%v)", epBatch, err)
	}

	// Confirm every critical HTTP verb actually crossed HostOptions.Transport.
	for _, want := range []string{
		"POST api.trakt.tv/scrobble/start",
		"POST api.trakt.tv/scrobble/stop",
		"POST api.bgm.tv/v0/search/subjects",
		"POST api.bgm.tv/v0/users/-/collections/328609",
		"PUT api.bgm.tv/v0/users/-/collections/-/episodes/110001",
		"PATCH api.bgm.tv/v0/users/-/collections/328609/episodes",
	} {
		if seen[want] == 0 {
			t.Errorf("expected request %q was never observed; seen=%v", want, seen)
		}
	}
}
