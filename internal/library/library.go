// Package library installs comic sources and keeps the installed set.
//
// Every change is written back through Core's records, so the next launch
// restores the same sources. Installing takes a script URL, a script's text,
// or an entry of a comic index (index.json).
package library

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/laincat/nekophage/internal/cookies"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/veneracat-core/plugin"
	"github.com/laincat/veneracat-core/subscription"
)

// MaxScriptBytes bounds a downloaded script or index.
const MaxScriptBytes = 4 << 20

// Library is the installed sources and their indexes.
type Library struct {
	core *core.Core

	mu      sync.Mutex
	records []plugin.Record
}

// New wraps a Core.
func New(c *core.Core) *Library { return &Library{core: c} }

// Restore loads the saved records into Core. It returns the sources that
// could not be restored, keyed by key.
func (l *Library) Restore(ctx context.Context) (map[string]error, error) {
	data, err := l.core.Store().Records()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	var recs []plugin.Record
	if err := json.Unmarshal(data, &recs); err != nil {
		return nil, fmt.Errorf("library: records: %w", err)
	}
	failed := l.core.Restore(ctx, recs)
	l.refresh()
	return failed, nil
}

// Sources lists installed plugins without script bodies.
//
// It reads Core every time: a source can also be installed through Core
// directly (a tool, a future import path), and the interface must not show a
// stale list.
func (l *Library) Sources() []plugin.Record { return l.core.Plugins() }

// Jar is where source cookies live, one partition per source. A backup import
// needs it to restore sign-ins without inventing a shared jar.
func (l *Library) Jar() *cookies.Jar { return l.core.Jar() }

// IsInstalled reports whether key is installed.
func (l *Library) IsInstalled(key string) bool {
	for _, r := range l.core.Plugins() {
		if r.Key == key {
			return true
		}
	}
	return false
}

// InstallText installs a script given as text (paste or file import).
func (l *Library) InstallText(ctx context.Context, body, origin string) (*plugin.Record, error) {
	return l.installOrReplace(ctx, body, origin, "")
}

// InstallURL downloads and installs a script, updating the source when its
// key is already installed.
func (l *Library) InstallURL(ctx context.Context, rawURL string) (*plugin.Record, error) {
	body, err := l.fetch(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	return l.installOrReplace(ctx, string(body), rawURL, "")
}

// InstallEntry installs or updates the source an index entry names.
func (l *Library) InstallEntry(ctx context.Context, entry subscription.Entry, indexURL string) (*plugin.Record, error) {
	body, err := l.fetch(ctx, entry.URL)
	if err != nil {
		return nil, err
	}
	if l.IsInstalled(entry.Key) {
		return l.replace(ctx, entry.Key, string(body), entry.URL, indexURL)
	}
	return l.installOrReplace(ctx, string(body), entry.URL, indexURL)
}

// Remove uninstalls a source and forgets what it stored.
func (l *Library) Remove(ctx context.Context, key string) error {
	if err := l.core.Remove(ctx, key); err != nil {
		return err
	}
	return l.persist()
}

// SetEnabled turns a source on or off.
func (l *Library) SetEnabled(ctx context.Context, key string, on bool) error {
	if err := l.core.SetEnabled(ctx, key, on); err != nil {
		return err
	}
	return l.persist()
}

// ImportSourceData writes the values of a source's data object, as a backup
// carried them. Values are stored exactly as written, so large integers and
// key order survive.
func (l *Library) ImportSourceData(key string, data map[string]json.RawMessage) (int, error) {
	n := 0
	for name, value := range data {
		if err := l.core.Store().SaveData(key, name, value); err != nil {
			return n, fmt.Errorf("library: %s/%s: %w", key, name, err)
		}
		n++
	}
	return n, nil
}

// Index downloads an index.json and compares it with the installed sources.
func (l *Library) Index(ctx context.Context, rawURL string) (*subscription.ComicIndex, []subscription.Row, error) {
	body, err := l.fetch(ctx, rawURL)
	if err != nil {
		return nil, nil, err
	}
	idx, err := subscription.ParseComic(body, rawURL)
	if err != nil {
		return nil, nil, err
	}
	return idx, idx.Compare(rawURL, l.core.AppVersion(), l.installed()), nil
}

// IndexFrom parses index text the app already has.
func (l *Library) IndexFrom(body []byte, rawURL string) (*subscription.ComicIndex, []subscription.Row, error) {
	idx, err := subscription.ParseComic(body, rawURL)
	if err != nil {
		return nil, nil, err
	}
	return idx, idx.Compare(rawURL, l.core.AppVersion(), l.installed()), nil
}

// --- internals ---

func (l *Library) installOrReplace(ctx context.Context, body, origin, indexURL string) (*plugin.Record, error) {
	rec, err := l.core.Install(ctx, body, origin, indexURL)
	if err != nil {
		// Core is the only one that knows a script's key, so a taken key is
		// only discovered here; update that source instead.
		if errors.Is(err, plugin.ErrExists) {
			if key := keyOf(err); key != "" && l.IsInstalled(key) {
				return l.replace(ctx, key, body, origin, indexURL)
			}
		}
		return nil, err
	}
	if err := l.persist(); err != nil {
		return rec, err
	}
	return rec, nil
}

func (l *Library) replace(ctx context.Context, key, body, origin, indexURL string) (*plugin.Record, error) {
	rec, err := l.core.Replace(ctx, key, body, origin, indexURL)
	if err != nil {
		return nil, err
	}
	if err := l.persist(); err != nil {
		return rec, err
	}
	return rec, nil
}

// keyOf reads the key from plugin.ErrExists's message ("...: key").
func keyOf(err error) string {
	msg := err.Error()
	if i := strings.LastIndex(msg, ": "); i >= 0 {
		return strings.TrimSpace(msg[i+2:])
	}
	return ""
}

func (l *Library) refresh() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = l.core.Records()
}

func (l *Library) persist() error {
	l.mu.Lock()
	recs := l.core.Records()
	l.records = recs
	l.mu.Unlock()
	data, err := json.Marshal(recs)
	if err != nil {
		return err
	}
	return l.core.Store().SaveRecords(data)
}

func (l *Library) installed() []subscription.Installed {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []subscription.Installed
	for _, r := range l.records {
		if r.Type == plugin.TypeComic {
			out = append(out, subscription.Installed{Key: r.Key, Version: r.Version, IndexURL: r.IndexURL})
		}
	}
	return out
}

func (l *Library) fetch(ctx context.Context, rawURL string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, errors.New("library: only http and https URLs")
	}
	body, _, err := l.core.Transport().Fetch(ctx, rawURL, MaxScriptBytes)
	return body, err
}
