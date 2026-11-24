package collections

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/laincat/veneracat-core/subscription"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/services"
	"github.com/laincat/nekophage/internal/vcmodule"
	"github.com/laincat/veneracat-core/gohost/library"
	"github.com/laincat/veneracat-core/plugin"
)

const probeScript = `class Probe extends ComicSource {
  name = 'Probe'; key = 'probe'; version = '1'
  search = {load: async () => ({comics: [], maxPage: 1})}
}
`

func probeCollection(t *testing.T, base string, body []byte, declared []byte, key string) *Manager {
	t.Helper()
	sum := sha256.Sum256(declared)
	doc := map[string]any{
		"$schema": "veneracat-source/1-draft",
		"id":      "probe-collection",
		"name":    "Probe",
		"version": "1",
		"entries": []map[string]any{{
			"id": "probe", "kind": "comic-source.js", "name": "Probe", "version": "1",
			"status": "active", "key": key,
			"supports": map[string]any{"platforms": []string{"windows", "macos", "linux"}, "requiresCapabilities": []string{"comic-source.js"}},
			"artifact": map[string]any{"url": base + "/probe.js", "sha256": hex.EncodeToString(sum[:]), "bytes": len(declared)},
		}},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Open(t.TempDir(), core.NewTransport(nil, "Nekophage/test", 0))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Close)
	if err := m.installForTest(raw, base+"/source.json"); err != nil {
		t.Fatal(err)
	}
	return m
}

// A digest that does not describe the served bytes is refused before anything
// is written: the artifact identity is the only thing this client can check.
func TestArtifactRefusesBytesThatDoNotMatchTheDigest(t *testing.T) {
	served := []byte(probeScript + "// extra\n")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(served)
	}))
	defer server.Close()

	m := probeCollection(t, server.URL, served, []byte(probeScript), "probe")
	entry, err := m.Entry("probe-collection", "probe", "windows", []string{"comic-source.js"})
	if err != nil {
		t.Fatalf("entry: %v", err)
	}
	if _, err := m.Artifact(context.Background(), entry); err == nil {
		t.Fatal("bytes that do not match the declared digest were accepted")
	}
}

// The declared key travels with the entry so a caller can compare it with what
// the script actually registers.
func TestEntryCarriesTheDeclaredKeyAndCompatibility(t *testing.T) {
	served := []byte(probeScript)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(served)
	}))
	defer server.Close()

	m := probeCollection(t, server.URL, served, served, "probe")
	entry, err := m.Entry("probe-collection", "probe", "windows", []string{"comic-source.js"})
	if err != nil {
		t.Fatalf("entry: %v", err)
	}
	if entry.Key != "probe" || !entry.Compatible {
		t.Fatalf("entry = %+v", entry)
	}
	ios, err := m.Entry("probe-collection", "probe", "ios", []string{"comic-source.js"})
	if err != nil {
		t.Fatalf("ios entry: %v", err)
	}
	if ios.Compatible {
		t.Fatalf("an entry limited to windows reported compatible on ios: %+v", ios)
	}
	if _, err := m.Entry("probe-collection", "missing", "windows", nil); err == nil {
		t.Fatal("an unknown entry id was accepted")
	}
	if _, err := m.Entry("no-such-collection", "probe", "windows", nil); err == nil {
		t.Fatal("an unknown collection id was accepted")
	}
}

func TestCollectionInstallChecksIdentityBeforePublishing(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	c, err := core.Open(core.Options{DataDir: root})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	lib := library.New(c)
	if _, err := lib.InstallText(ctx, probeScript, "original"); err != nil {
		t.Fatal(err)
	}
	if err := c.Store().SaveData("probe", "account", []byte("123")); err != nil {
		t.Fatal(err)
	}
	served := []byte(strings.Replace(probeScript, "version = '1'", "version = '2'", 1))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(served) }))
	defer server.Close()
	manager := probeCollection(t, server.URL, served, served, "other")
	file := filepath.Join(root, "plugins.json")
	before, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := InstallEntry(ctx, manager, "probe-collection", "probe", root, lib); !errors.Is(err, plugin.ErrKey) {
		t.Fatalf("mismatch: %v", err)
	}
	after, err := os.ReadFile(file)
	if err != nil || string(after) != string(before) {
		t.Fatalf("records changed on mismatch: %v", err)
	}
	if raw, _, err := c.Store().LoadData("probe", "account"); err != nil || string(raw) != "123" {
		t.Fatalf("source data changed: %s %v", raw, err)
	}
	valid := probeCollection(t, server.URL, served, served, "probe")
	result, err := InstallEntry(ctx, valid, "probe-collection", "probe", root, lib)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.(*plugin.Record); got.Version != "2" || got.Origin != "collection:probe-collection/probe" {
		t.Fatalf("upgrade: %+v", got)
	}
}

// installForTest seeds the manager with one collection document, standing in for
// a followed subscription so the tests do not need the network twice.
func (m *Manager) installForTest(document []byte, url string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	next := m.copyRecords()
	var c struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(document, &c); err != nil {
		return err
	}
	next[c.ID] = record{ID: c.ID, URL: url, Document: document}
	return m.publish(next)
}

// An artifact that no longer matches its declared size must be reported as a
// mismatch, not as a transport limit: the first points at the collection, the
// second at the network.
func TestArtifactReportsAGrownArtifactAsAMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("x", 2048)))
	}))
	defer server.Close()

	m, err := Open(t.TempDir(), core.NewTransport(nil, "test", 0))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	// A snapshot that declares 512 bytes while the server now serves 2048.
	entry := Entry{}
	entry.ID = "grown"
	entry.Artifact = &subscription.CollectionArtifact{URL: server.URL, Bytes: 512, SHA256: strings.Repeat("0", 64)}
	_, err = m.Artifact(context.Background(), entry)
	if err == nil {
		t.Fatal("a grown artifact was accepted")
	}
	if !strings.Contains(err.Error(), "does not match its declared size or digest") {
		t.Fatalf("error = %v", err)
	}
}

const novelProbeScript = `class NovelCollectionProbe extends NovelSource {
  name = "小说测试源"; key = "novel_probe"; version = "1.0.0"
  async popular(page) {
    return {items: [{id: "n1", title: "热门小说 " + page, author: "作者"}], maxPage: 2}
  }
  async search(keyword, page) {
    return {items: [{id: "n2", title: "搜索:" + keyword}], maxPage: 1}
  }
  async detail(id) {
    return {id: id, title: "小说详情", author: "作者", summary: "简介", chapters: [{id: "ch1", title: "第一章"}]}
  }
  async chapter(novelId, chapId) {
    return {html: "<p>正文 " + novelId + "/" + chapId + "</p>"}
  }
}
`

func TestUnifiedCollectionNovelSourceInstallAndToggleLifecycle(t *testing.T) {
	ctx := context.Background()
	served := []byte(novelProbeScript)
	sum := sha256.Sum256(served)
	digest := hex.EncodeToString(sum[:])

	const artifact = "packages/novel_probe/novel_probe.js"
	probe := subscription.CollectionEntry{
		ID: "novel_probe", Kind: "novel-source", Name: "小说测试源",
		Version: "1.0.0", Status: "experimental", Key: "novel_probe",
		Supports: subscription.CollectionSupport{
			Platforms: []string{"ios", "macos", "windows", "linux"}, RequiresCapabilities: []string{subscription.CapabilityNovelJS},
		},
		Artifact: &subscription.CollectionArtifact{URL: artifact, SHA256: digest, Bytes: int64(len(served))},
	}
	badDigest, wrongKey, broken := probe, probe, probe
	badDigest.ID = "novel_bad_digest"
	badDigest.Artifact = &subscription.CollectionArtifact{URL: artifact, SHA256: strings.Repeat("a", 64), Bytes: int64(len(served))}
	wrongKey.ID, wrongKey.Key = "novel_wrong_key", "different_key"
	broken.ID, broken.Status, broken.Artifact = "novel_broken", "broken", nil
	doc := subscription.Collection{
		Schema: subscription.CollectionSchema, ID: "veneracat.official", Name: "VeneraCat 官方合集", Version: "0.1.0",
		Entries: []subscription.CollectionEntry{probe, badDigest, wrongKey, broken},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/source.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(doc)
	})
	mux.HandleFunc("/"+artifact, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(served) })
	server := httptest.NewServer(mux)
	defer server.Close()

	root := t.TempDir()
	m, err := Open(root, core.NewTransport(nil, "Nekophage/test", 0))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	if err := m.Follow(ctx, server.URL+"/source.json"); err != nil {
		t.Fatalf("follow: %v", err)
	}
	views, err := m.List(Platform(), Capabilities())
	if err != nil || len(views) != 1 || len(views[0].Entries) != 4 {
		t.Fatalf("views = %+v, %v", views, err)
	}
	if !views[0].Entries[0].Compatible || !CanInstall(views[0].Entries[0]) {
		t.Fatalf("valid novel entry should be installable: %+v", views[0].Entries[0])
	}
	if CanInstall(views[0].Entries[3]) {
		t.Fatalf("broken novel entry without artifact must not be installable: %+v", views[0].Entries[3])
	}

	c, err := core.Open(core.Options{DataDir: root})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if c != nil {
			c.Close()
		}
	})
	lib := library.New(c)

	if _, err := InstallEntry(ctx, m, "veneracat.official", "novel_bad_digest", root, lib); err == nil || !strings.Contains(err.Error(), "size or digest") {
		t.Fatalf("digest mismatch should fail: %v", err)
	}
	if _, err := InstallEntry(ctx, m, "veneracat.official", "novel_wrong_key", root, lib); !errors.Is(err, plugin.ErrKey) {
		t.Fatalf("key mismatch = %v, want ErrKey", err)
	}
	if _, err := InstallEntry(ctx, m, "veneracat.official", "novel_broken", root, lib); err == nil {
		t.Fatal("broken entry without artifact was installed")
	}

	res, err := InstallEntry(ctx, m, "veneracat.official", "novel_probe", root, lib)
	if err != nil {
		t.Fatalf("install novel entry: %v", err)
	}
	rec := res.(*plugin.Record)
	if rec.Type != plugin.TypeNovelJS || rec.Key != "novel_probe" || !rec.Enabled || rec.Origin != "collection:veneracat.official/novel_probe" {
		t.Fatalf("installed record = %+v", rec)
	}

	page, err := c.NovelPopular(ctx, rec.Key, 1)
	if err != nil || page == nil || len(page.Items) != 1 || page.Items[0].Title != "热门小说 1" {
		t.Fatalf("popular = %+v, %v", page, err)
	}
	if err := lib.SetEnabled(ctx, rec.Key, false); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if _, err := c.NovelPopular(ctx, rec.Key, 1); !errors.Is(err, plugin.ErrDisabled) {
		t.Fatalf("disabled source answered: %v", err)
	}
	c.Close()
	c = nil

	// Reopen Core and verify disabled novel source persists and can be re-enabled.
	reopened, err := core.Open(core.Options{DataDir: root})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	restoredLib := library.New(reopened)
	if failed, err := restoredLib.Restore(ctx); err != nil || len(failed) != 0 {
		t.Fatalf("restore = %v, %v", failed, err)
	}
	if _, err := reopened.NovelSearch(ctx, rec.Key, "星海", 1); !errors.Is(err, plugin.ErrDisabled) {
		t.Fatalf("restored disabled source answered: %v", err)
	}
	if err := restoredLib.SetEnabled(ctx, rec.Key, true); err != nil {
		t.Fatalf("re-enable: %v", err)
	}
	found, err := reopened.NovelSearch(ctx, rec.Key, "星海", 1)
	if err != nil || found == nil || len(found.Items) != 1 || found.Items[0].Title != "搜索:星海" {
		t.Fatalf("re-enabled search = %+v, %v", found, err)
	}
}

func makeServiceZip(t *testing.T, id, version string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	files := map[string]string{
		"service.json": `{"id":"` + id + `","name":"` + id + `","version":"` + version + `","kind":"service.js","entry":"index.mjs","permissions":["network","storage","secrets"]}`,
		"index.mjs":    `export function describe() { return { methods: [{ name: "account.me", title: "账号" }] }; } export function handle(m) { return { id: "` + id + `", version: "` + version + `", method: m }; }`,
	}
	for name, body := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestUnifiedCollectionInstallsServicePackagesAndVCModule(t *testing.T) {
	ctx := context.Background()
	traktZip := makeServiceZip(t, services.ServiceIDTrakt, "1.0.0")
	tmdbZip := makeServiceZip(t, services.ServiceIDTMDB, "1.0.0")
	bgmZip := makeServiceZip(t, services.ServiceIDBangumi, "1.0.0")
	modBytes := []byte("#!id=reader-preferences\n#!name=阅读偏好示例\n#!desc=合集模块\n#!version=0.1.0\n#!format=veneracat-module/1-draft\n#!requires=settings-module.v1\n\n[Reader]\npage-gap = 8\nkeep-screen-on = true\n")

	sha := func(b []byte) string {
		s := sha256.Sum256(b)
		return hex.EncodeToString(s[:])
	}

	artifacts := map[string][]byte{
		"/packages/service.trakt.zip":          traktZip,
		"/packages/service.tmdb.zip":           tmdbZip,
		"/packages/service.bangumi.zip":        bgmZip,
		"/modules/reader-preferences.vcmodule": modBytes,
	}

	doc := subscription.Collection{
		Schema: subscription.CollectionSchema, ID: "veneracat.official", Name: "VeneraCat 官方合集", Version: "0.2.0",
		Entries: []subscription.CollectionEntry{
			{
				ID: services.ServiceIDTrakt, Kind: "service", Name: "Trakt", Version: "1.0.0", Status: "experimental",
				Supports: subscription.CollectionSupport{Platforms: []string{"ios", "macos", "windows", "linux"}, RequiresCapabilities: []string{"service.v1"}},
				Artifact: &subscription.CollectionArtifact{URL: "packages/service.trakt.zip", SHA256: sha(traktZip), Bytes: int64(len(traktZip))},
			},
			{
				ID: services.ServiceIDTMDB, Kind: "service", Name: "TMDB", Version: "1.0.0", Status: "experimental",
				Supports: subscription.CollectionSupport{Platforms: []string{"ios", "macos", "windows", "linux"}, RequiresCapabilities: []string{"service.v1"}},
				Artifact: &subscription.CollectionArtifact{URL: "packages/service.tmdb.zip", SHA256: sha(tmdbZip), Bytes: int64(len(tmdbZip))},
			},
			{
				ID: services.ServiceIDBangumi, Kind: "service", Name: "Bangumi", Version: "1.0.0", Status: "experimental",
				Supports: subscription.CollectionSupport{Platforms: []string{"ios", "macos", "windows", "linux"}, RequiresCapabilities: []string{"service.v1"}},
				Artifact: &subscription.CollectionArtifact{URL: "packages/service.bangumi.zip", SHA256: sha(bgmZip), Bytes: int64(len(bgmZip))},
			},
			{
				ID: "reader-preferences", Kind: "settings-module", Name: "阅读偏好示例", Version: "0.1.0", Status: "prototype",
				Supports: subscription.CollectionSupport{Platforms: []string{"ios", "macos", "windows", "linux"}, RequiresCapabilities: []string{vcmodule.Capability}},
				Artifact: &subscription.CollectionArtifact{URL: "modules/reader-preferences.vcmodule", SHA256: sha(modBytes), Bytes: int64(len(modBytes))},
			},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/source.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(doc)
	})
	for p, b := range artifacts {
		body := b
		mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(body) })
	}
	server := httptest.NewServer(mux)
	defer server.Close()

	root := t.TempDir()
	m, err := Open(root, core.NewTransport(nil, "Nekophage/test", 0))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	if err := m.Follow(ctx, server.URL+"/source.json"); err != nil {
		t.Fatalf("follow: %v", err)
	}

	c, err := core.Open(core.Options{DataDir: root})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	lib := library.New(c)

	for _, svcID := range []string{services.ServiceIDTrakt, services.ServiceIDTMDB, services.ServiceIDBangumi} {
		res, err := InstallEntry(ctx, m, "veneracat.official", svcID, root, lib)
		if err != nil {
			t.Fatalf("install %s: %v", svcID, err)
		}
		rec, ok := res.(*services.Record)
		if !ok || rec.ID != svcID {
			t.Fatalf("installed %s = %#v", svcID, res)
		}
	}

	modRes, err := InstallEntry(ctx, m, "veneracat.official", "reader-preferences", root, lib)
	if err != nil {
		t.Fatalf("install vcmodule: %v", err)
	}
	modRec, ok := modRes.(*vcmodule.Record)
	if !ok || modRec.ID != "reader-preferences" || modRec.Enabled {
		t.Fatalf("installed vcmodule = %#v", modRes)
	}

	mgr, err := vcmodule.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.Enable("reader-preferences"); err != nil {
		t.Fatal(err)
	}
	resolved := mgr.Resolve(vcmodule.DefaultSettings())
	if resolved.Effective.Reader.PageGap != 8 || !resolved.Effective.Reader.KeepScreenOn {
		t.Fatalf("resolved = %+v", resolved)
	}
}
