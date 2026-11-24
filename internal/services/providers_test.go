package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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

func traktZipFiles(baseURL, version string) map[string]string {
	manifest := fmt.Sprintf(`{"id":"service.trakt","name":"Trakt","version":%q,"kind":"service.js","entry":"index.mjs","permissions":["network","storage","secrets"]}`, version)
	entry := fmt.Sprintf(`const BASE = %q;
export function describe() {
  return {
    methods: [
      { name: "account.me", title: "当前账号" },
      { name: "metadata.search", title: "搜索条目" },
      { name: "metadata.detail", title: "条目详情" },
      { name: "watchlist.list", title: "想看列表" },
      { name: "watchlist.update", title: "更新想看" },
      { name: "scrobble", title: "播放进度同步" }
    ]
  };
}
export function handle(method, rawPayload) {
  const payload = rawPayload ? JSON.parse(rawPayload) : {};
  switch (method) {
    case "account.me": {
      const tok = secret("access_token");
      log("info", "trakt auth check with " + tok);
      return JSON.parse(fetch(BASE + "/trakt/me?token=" + encodeURIComponent(tok)));
    }
    case "metadata.search":
      return JSON.parse(fetch(BASE + "/trakt/search?q=" + encodeURIComponent(payload.query || "")));
    case "metadata.detail":
      return JSON.parse(fetch(BASE + "/trakt/detail?id=" + encodeURIComponent(payload.id || "")));
    case "watchlist.list": {
      const tok = secret("access_token");
      return JSON.parse(fetch(BASE + "/trakt/watchlist?token=" + encodeURIComponent(tok)));
    }
    case "watchlist.update": {
      const tok = secret("access_token");
      return JSON.parse(fetch(BASE + "/trakt/watchlist/update?id=" + encodeURIComponent(payload.id || "") + "&action=" + encodeURIComponent(payload.action || "") + "&token=" + encodeURIComponent(tok)));
    }
    case "scrobble": {
      const tok = secret("access_token");
      return JSON.parse(fetch(BASE + "/trakt/scrobble?id=" + encodeURIComponent(payload.id || "") + "&action=" + encodeURIComponent(payload.action || "") + "&progress=" + encodeURIComponent(String(payload.progress || 0)) + "&token=" + encodeURIComponent(tok)));
    }
    case "leak.storage": {
      const tok = secret("access_token");
      storageSet("cached_tok", "bearer=" + tok);
      return { ok: true };
    }
    case "leak.error": {
      const tok = secret("access_token");
      throw new Error("upstream rejected token=" + tok);
    }
    case "slow":
      return JSON.parse(fetch(BASE + "/slow"));
    default:
      throw new Error("unknown method: " + method);
  }
}
`, baseURL)
	return map[string]string{"service.json": manifest, "index.mjs": entry}
}

func tmdbZipFiles(baseURL, version string) map[string]string {
	manifest := fmt.Sprintf(`{"id":"service.tmdb","name":"TMDB","version":%q,"kind":"service.js","entry":"index.mjs","permissions":["network","storage","secrets"]}`, version)
	entry := fmt.Sprintf(`const BASE = %q;
export function describe() {
  return {
    methods: [
      { name: "account.me", title: "账号信息" },
      { name: "metadata.search", title: "搜索影视" },
      { name: "metadata.detail", title: "影视详情" },
      { name: "watchlist.list", title: "待看片单" }
    ]
  };
}
export function handle(method, rawPayload) {
  const payload = rawPayload ? JSON.parse(rawPayload) : {};
  const tok = secret("api_read_token");
  switch (method) {
    case "account.me":
      return JSON.parse(fetch(BASE + "/tmdb/me?token=" + encodeURIComponent(tok)));
    case "metadata.search":
      return JSON.parse(fetch(BASE + "/tmdb/search?q=" + encodeURIComponent(payload.query || "") + "&token=" + encodeURIComponent(tok)));
    case "metadata.detail":
      return JSON.parse(fetch(BASE + "/tmdb/detail?id=" + encodeURIComponent(payload.id || "") + "&token=" + encodeURIComponent(tok)));
    case "watchlist.list":
      return JSON.parse(fetch(BASE + "/tmdb/watchlist?token=" + encodeURIComponent(tok)));
    default:
      throw new Error("unknown method: " + method);
  }
}
`, baseURL)
	return map[string]string{"service.json": manifest, "index.mjs": entry}
}

func bangumiZipFiles(baseURL, version string) map[string]string {
	manifest := fmt.Sprintf(`{"id":"service.bangumi","name":"Bangumi","version":%q,"kind":"service.js","entry":"index.mjs","permissions":["network","storage","secrets"]}`, version)
	entry := fmt.Sprintf(`const BASE = %q;
export function describe() {
  return {
    methods: [
      { name: "account.me", title: "Bangumi 账号" },
      { name: "metadata.search", title: "搜索条目" },
      { name: "metadata.detail", title: "条目详情" },
      { name: "collection.list", title: "收藏列表" },
      { name: "collection.update", title: "更新收藏进度" }
    ]
  };
}
export function handle(method, rawPayload) {
  const payload = rawPayload ? JSON.parse(rawPayload) : {};
  switch (method) {
    case "account.me": {
      const tok = secret("access_token");
      return JSON.parse(fetch(BASE + "/bgm/me?token=" + encodeURIComponent(tok)));
    }
    case "metadata.search":
      return JSON.parse(fetch(BASE + "/bgm/search?q=" + encodeURIComponent(payload.query || "")));
    case "metadata.detail":
      return JSON.parse(fetch(BASE + "/bgm/detail?id=" + encodeURIComponent(payload.id || "")));
    case "collection.list": {
      const tok = secret("access_token");
      return JSON.parse(fetch(BASE + "/bgm/collection?token=" + encodeURIComponent(tok)));
    }
    case "collection.update": {
      const tok = secret("access_token");
      return JSON.parse(fetch(BASE + "/bgm/collection/update?id=" + encodeURIComponent(payload.id || "") + "&status=" + encodeURIComponent(payload.status || "") + "&progress=" + encodeURIComponent(String(payload.progress || 0)) + "&token=" + encodeURIComponent(tok)));
    }
    default:
      throw new Error("unknown method: " + method);
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
		case "/trakt/me":
			if r.URL.Query().Get("token") != "trakt-secret-999" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(AccountProfile{ID: "u-trakt", Username: "lain", DisplayName: "Lain", Provider: "trakt"})
		case "/trakt/search":
			_ = json.NewEncoder(w).Encode(MetadataSearchResult{
				Page: 1, TotalPages: 1, TotalResults: 1,
				Items: []MetadataItem{{ID: "trakt-42", MediaType: "show", Title: "Serial Experiments Lain", Year: 1998}},
			})
		case "/trakt/detail":
			_ = json.NewEncoder(w).Encode(MetadataDetail{
				MetadataItem: MetadataItem{ID: r.URL.Query().Get("id"), MediaType: "show", Title: "Serial Experiments Lain", Year: 1998},
				Episodes:     13, Status: "ended",
			})
		case "/trakt/watchlist":
			if r.URL.Query().Get("token") != "trakt-secret-999" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(WatchlistResult{
				Total: 1,
				Items: []WatchlistItem{{ID: "trakt-42", MediaType: "show", Title: "Serial Experiments Lain"}},
			})
		case "/trakt/watchlist/update":
			_ = json.NewEncoder(w).Encode(WatchlistMutationResult{OK: true, ID: r.URL.Query().Get("id"), Action: r.URL.Query().Get("action")})
		case "/trakt/scrobble":
			_ = json.NewEncoder(w).Encode(ScrobbleResult{OK: true, ID: r.URL.Query().Get("id"), Action: r.URL.Query().Get("action"), Progress: 85.5})
		case "/tmdb/me":
			if r.URL.Query().Get("token") != "tmdb-read-secret-777" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(AccountProfile{ID: "u-tmdb", Username: "tmdb_user", Provider: "tmdb"})
		case "/tmdb/search":
			if r.URL.Query().Get("token") != "tmdb-read-secret-777" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(MetadataSearchResult{
				Page: 1, TotalResults: 1,
				Items: []MetadataItem{{ID: "1396", MediaType: "tv", Title: "绝命毒师", Year: 2008}},
			})
		case "/tmdb/detail":
			_ = json.NewEncoder(w).Encode(MetadataDetail{
				MetadataItem: MetadataItem{ID: r.URL.Query().Get("id"), MediaType: "tv", Title: "绝命毒师", Year: 2008},
				Seasons:      5, Episodes: 62,
			})
		case "/tmdb/watchlist":
			_ = json.NewEncoder(w).Encode(WatchlistResult{Total: 1, Items: []WatchlistItem{{ID: "1396", MediaType: "tv", Title: "绝命毒师"}}})
		case "/bgm/me":
			if r.URL.Query().Get("token") != "bgm-pat-secret-555" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(AccountProfile{ID: "1001", Username: "nekocat", DisplayName: "猫猫", Provider: "bangumi"})
		case "/bgm/search":
			_ = json.NewEncoder(w).Encode(MetadataSearchResult{
				Page: 1, TotalResults: 1,
				Items: []MetadataItem{{ID: "328609", MediaType: "anime", Title: "孤独摇滚！", Year: 2022}},
			})
		case "/bgm/detail":
			_ = json.NewEncoder(w).Encode(MetadataDetail{
				MetadataItem: MetadataItem{ID: r.URL.Query().Get("id"), MediaType: "anime", Title: "孤独摇滚！", Year: 2022},
				Episodes:     12, Status: "finished",
			})
		case "/bgm/collection":
			if r.URL.Query().Get("token") != "bgm-pat-secret-555" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(CollectionListResult{
				Total: 1,
				Items: []CollectionEntry{{ID: "328609", MediaType: "anime", Title: "孤独摇滚！", Status: "watching", Progress: 8, Total: 12, Score: 9.5}},
			})
		case "/bgm/collection/update":
			_ = json.NewEncoder(w).Encode(CollectionUpdateResult{OK: true, ID: r.URL.Query().Get("id"), Status: r.URL.Query().Get("status"), Progress: 9})
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

	traktZip := writeZipFile(t, zipDir, "trakt-v1.zip", traktZipFiles(srv.URL, "1.0.0"))
	tmdbZip := writeZipFile(t, zipDir, "tmdb-v1.zip", tmdbZipFiles(srv.URL, "1.0.0"))
	bgmZip := writeZipFile(t, zipDir, "bangumi-v1.zip", bangumiZipFiles(srv.URL, "1.0.0"))

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

	// Missing secret must fail clearly naming access_token.
	if _, err := trakt.AccountMe(ctx); !errors.Is(err, ErrMissingSecrets) || !strings.Contains(err.Error(), SecretKeyAccessToken) {
		t.Fatalf("missing secret error = %v, want ErrMissingSecrets(access_token)", err)
	}

	// Store isolated Trakt token and run full typed workflow.
	if err := trakt.SetAccount(ctx, SecretKeyAccessToken, "trakt-secret-999"); err != nil {
		t.Fatal(err)
	}
	methods, err := trakt.Describe(ctx)
	if err != nil || len(methods) != 6 {
		t.Fatalf("trakt describe = %+v (%v)", methods, err)
	}
	me, err := trakt.AccountMe(ctx)
	if err != nil || me.Username != "lain" {
		t.Fatalf("trakt account.me = %+v (%v)", me, err)
	}
	if *lastUA != "Nekophage-Test/1.0" {
		t.Fatalf("User-Agent = %q, want Nekophage-Test/1.0", *lastUA)
	}
	search, err := trakt.MetadataSearch(ctx, MetadataQuery{Query: "lain", MediaType: "show"})
	if err != nil || len(search.Items) != 1 || search.Items[0].ID != "trakt-42" {
		t.Fatalf("trakt search = %+v (%v)", search, err)
	}
	detail, err := trakt.MetadataDetail(ctx, MetadataDetailRequest{ID: "trakt-42", MediaType: "show"})
	if err != nil || detail.Episodes != 13 {
		t.Fatalf("trakt detail = %+v (%v)", detail, err)
	}
	wl, err := trakt.Watchlist(ctx, WatchlistRequest{MediaType: "show"})
	if err != nil || len(wl.Items) != 1 {
		t.Fatalf("trakt watchlist = %+v (%v)", wl, err)
	}
	mut, err := trakt.UpdateWatchlist(ctx, WatchlistMutationRequest{ID: "trakt-42", MediaType: "show", Action: "add"})
	if err != nil || !mut.OK || mut.Action != "add" {
		t.Fatalf("trakt update watchlist = %+v (%v)", mut, err)
	}
	scrob, err := trakt.Scrobble(ctx, ScrobbleRequest{ID: "trakt-42", MediaType: "show", Action: "stop", Episode: 1, Progress: 85.5})
	if err != nil || !scrob.OK || scrob.Progress != 85.5 {
		t.Fatalf("trakt scrobble = %+v (%v)", scrob, err)
	}

	// Verify log redaction: token must NEVER appear in host logs.
	logMu.Lock()
	joinedLogs := strings.Join(capturedLogs, "\n")
	logMu.Unlock()
	if strings.Contains(joinedLogs, "trakt-secret-999") || !strings.Contains(joinedLogs, "[REDACTED]") {
		t.Fatalf("secret leaked or not redacted in logs: %q", joinedLogs)
	}

	// Upgrade Trakt zip to v2 and confirm isolated account secret is preserved.
	traktZipV2 := writeZipFile(t, zipDir, "trakt-v2.zip", traktZipFiles(srv.URL, "2.0.0"))
	recV2, reportV2, err := InstallZipFile(ctx, dataDir, traktZipV2, ServiceIDTrakt)
	if err != nil {
		t.Fatalf("upgrade trakt: %v", err)
	}
	if recV2.Version != "2.0.0" || recV2.Accounts != 1 || len(reportV2.Denied) != 0 {
		t.Fatalf("upgraded trakt rec=%+v report=%+v", recV2, reportV2)
	}

	// TMDB typed workflow.
	tmdb, err := NewTMDBClient(dataDir, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer tmdb.Close()
	if _, err := tmdb.MetadataSearch(ctx, MetadataQuery{Query: "breaking"}); !errors.Is(err, ErrMissingSecrets) || !strings.Contains(err.Error(), SecretKeyAPIReadToken) {
		t.Fatalf("tmdb missing secret = %v", err)
	}
	if err := tmdb.SetAccount(ctx, SecretKeyAPIReadToken, "tmdb-read-secret-777"); err != nil {
		t.Fatal(err)
	}
	if m, err := tmdb.Describe(ctx); err != nil || len(m) != 4 {
		t.Fatalf("tmdb describe = %+v (%v)", m, err)
	}
	if me, err := tmdb.AccountMe(ctx); err != nil || me.Username != "tmdb_user" {
		t.Fatalf("tmdb me = %+v (%v)", me, err)
	}
	if res, err := tmdb.MetadataSearch(ctx, MetadataQuery{Query: "毒师"}); err != nil || len(res.Items) != 1 || res.Items[0].Title != "绝命毒师" {
		t.Fatalf("tmdb search = %+v (%v)", res, err)
	}
	if d, err := tmdb.MetadataDetail(ctx, MetadataDetailRequest{ID: "1396"}); err != nil || d.Seasons != 5 {
		t.Fatalf("tmdb detail = %+v (%v)", d, err)
	}
	if wl, err := tmdb.Watchlist(ctx, WatchlistRequest{}); err != nil || len(wl.Items) != 1 {
		t.Fatalf("tmdb watchlist = %+v (%v)", wl, err)
	}

	// Bangumi typed workflow.
	bgm, err := NewBangumiClient(dataDir, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer bgm.Close()
	if _, err := bgm.Collection(ctx, CollectionListRequest{}); !errors.Is(err, ErrMissingSecrets) {
		t.Fatalf("bgm missing secret = %v", err)
	}
	if err := bgm.SetAccount(ctx, SecretKeyAccessToken, "bgm-pat-secret-555"); err != nil {
		t.Fatal(err)
	}
	if m, err := bgm.Describe(ctx); err != nil || len(m) != 5 {
		t.Fatalf("bgm describe = %+v (%v)", m, err)
	}
	if me, err := bgm.AccountMe(ctx); err != nil || me.Username != "nekocat" {
		t.Fatalf("bgm me = %+v (%v)", me, err)
	}
	if res, err := bgm.MetadataSearch(ctx, MetadataQuery{Query: "孤独摇滚"}); err != nil || len(res.Items) != 1 || res.Items[0].ID != "328609" {
		t.Fatalf("bgm search = %+v (%v)", res, err)
	}
	if d, err := bgm.MetadataDetail(ctx, MetadataDetailRequest{ID: "328609"}); err != nil || d.Episodes != 12 {
		t.Fatalf("bgm detail = %+v (%v)", d, err)
	}
	if col, err := bgm.Collection(ctx, CollectionListRequest{Status: "watching"}); err != nil || len(col.Items) != 1 || col.Items[0].Progress != 8 {
		t.Fatalf("bgm collection = %+v (%v)", col, err)
	}
	if upd, err := bgm.UpdateCollection(ctx, CollectionUpdateRequest{ID: "328609", Status: "watching", Progress: 9}); err != nil || !upd.OK || upd.Progress != 9 {
		t.Fatalf("bgm update = %+v (%v)", upd, err)
	}
}

func TestPermissionTrimmingAndStrictDenial(t *testing.T) {
	ctx := context.Background()
	srv, _, _ := newStubProviderServer(t)
	dataDir := t.TempDir()
	zipPath := writeZipFile(t, t.TempDir(), "trakt.zip", traktZipFiles(srv.URL, "1.0.0"))
	if _, _, err := InstallZipFile(ctx, dataDir, zipPath, ServiceIDTrakt); err != nil {
		t.Fatal(err)
	}

	// Trim secrets permission while allowing network + storage.
	trimmed, err := NewTraktClient(dataDir, HostOptions{
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
	if res, err := trimmed.MetadataSearch(ctx, MetadataQuery{Query: "lain"}); err != nil || len(res.Items) != 1 {
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
	_, _, rep, err := OpenWithOptions(ctx, dataDir, ServiceIDTrakt, HostOptions{
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
	zipPath := writeZipFile(t, t.TempDir(), "trakt.zip", traktZipFiles(srv.URL, "1.0.0"))
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
	zipPath := writeZipFile(t, t.TempDir(), "trakt.zip", traktZipFiles(srv.URL, "1.0.0"))
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
	if res, err := trakt.MetadataSearch(context.Background(), MetadataQuery{Query: "lain"}); err != nil || len(res.Items) != 1 {
		t.Fatalf("post-cancel call = %+v (%v)", res, err)
	}
}
