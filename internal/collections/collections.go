// Package collections tracks publisher collections without installing or
// executing their entries. Core owns parsing and capability matching; this
// package owns profile storage, cookie-free downloads and request lifetimes.
package collections

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/store"
	"github.com/laincat/veneracat-core/subscription"
)

const MaxDocumentBytes = 4 << 20

var (
	ErrNotFound   = errors.New("collections: subscription not found")
	ErrSuperseded = errors.New("collections: request superseded")
	ErrClosed     = errors.New("collections: manager closed")
)

type record struct {
	ID        string          `json:"id"`
	URL       string          `json:"url"`
	Document  json.RawMessage `json:"document"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type disk struct {
	Version       int      `json:"version"`
	Subscriptions []record `json:"subscriptions"`
}

type operation struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// Manager is owned by one application instance per profile. Multiple processes
// must not concurrently modify the same profile, as with other app stores.
type Manager struct {
	mu        sync.Mutex
	store     *store.Store
	transport *core.Transport
	records   map[string]record
	active    map[string]*operation // keyed by the subscribed URL
	closed    bool
}

// Open restores validated snapshots without network access. Corrupt or future
// formats return an error and are never silently replaced by an empty store.
func Open(s *store.Store, tr *core.Transport) (*Manager, error) {
	m := &Manager{store: s, transport: tr, records: map[string]record{}, active: map[string]*operation{}}
	data, err := s.Collections()
	if err != nil {
		return nil, err
	}
	if data == nil {
		return m, nil
	}
	var saved disk
	if err = json.Unmarshal(data, &saved); err != nil {
		return nil, fmt.Errorf("collections: saved snapshot: %w", err)
	}
	if saved.Version != 1 {
		return nil, fmt.Errorf("collections: unsupported snapshot version %d", saved.Version)
	}
	urls := map[string]bool{}
	for _, rec := range saved.Subscriptions {
		u, err := sourceURL(rec.URL)
		if err != nil || u != rec.URL || len(rec.Document) > MaxDocumentBytes {
			return nil, errors.New("collections: invalid saved subscription")
		}
		c, err := subscription.ParseCollection(rec.Document, rec.URL)
		if err != nil {
			return nil, err
		}
		if rec.ID != c.ID || urls[rec.URL] {
			return nil, errors.New("collections: conflicting saved identity")
		}
		if _, exists := m.records[rec.ID]; exists {
			return nil, errors.New("collections: duplicate saved identity")
		}
		urls[rec.URL] = true
		m.records[rec.ID] = rec
	}
	return m, nil
}

func sourceURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		return "", errors.New("collections: expected HTTP(S) URL without embedded credentials")
	}
	u.Fragment = ""
	return u.String(), nil
}

// Follow downloads and tracks one collection. Repeating a URL refreshes its
// existing identity; another URL cannot silently take over an existing ID.
func (m *Manager) Follow(ctx context.Context, raw string) error {
	u, err := sourceURL(raw)
	if err != nil {
		return err
	}
	m.mu.Lock()
	id := ""
	for _, rec := range m.records {
		if rec.URL == u {
			id = rec.ID
			break
		}
	}
	op, err := m.begin(ctx, u)
	m.mu.Unlock()
	if err != nil {
		return err
	}
	return m.fetch(u, id, op)
}

// Refresh replaces a valid snapshot only after a successful bounded fetch.
// A newer refresh cancels the older request and prevents late publication.
func (m *Manager) Refresh(ctx context.Context, id string) error {
	m.mu.Lock()
	rec, ok := m.records[id]
	if !ok {
		m.mu.Unlock()
		return ErrNotFound
	}
	op, err := m.begin(ctx, rec.URL)
	m.mu.Unlock()
	if err != nil {
		return err
	}
	return m.fetch(rec.URL, id, op)
}

// begin runs while holding mu. Cancelled calls do not replace a live request.
func (m *Manager) begin(ctx context.Context, u string) (*operation, error) {
	if m.closed {
		return nil, ErrClosed
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if old := m.active[u]; old != nil {
		old.cancel()
	}
	ctx, cancel := context.WithCancel(ctx)
	op := &operation{ctx: ctx, cancel: cancel}
	m.active[u] = op
	return op, nil
}

func (m *Manager) fetch(u, expectedID string, op *operation) error {
	defer op.cancel()
	defer func() {
		m.mu.Lock()
		if m.active[u] == op {
			delete(m.active, u)
		}
		m.mu.Unlock()
	}()
	body, _, err := m.transport.Fetch(op.ctx, u, MaxDocumentBytes)
	if err != nil {
		return err
	}
	c, err := subscription.ParseCollection(body, u)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return ErrClosed
	}
	if m.active[u] != op {
		return ErrSuperseded
	}
	if err := op.ctx.Err(); err != nil {
		return err
	}
	if expectedID != "" {
		rec, ok := m.records[expectedID]
		if !ok || rec.URL != u {
			return ErrSuperseded
		}
		if c.ID != expectedID {
			return errors.New("collections: refreshed collection changed its identity")
		}
	}
	if old, exists := m.records[c.ID]; exists && old.URL != u {
		return errors.New("collections: identity already belongs to another URL")
	}
	next := m.copyRecords()
	next[c.ID] = record{ID: c.ID, URL: u, Document: body, UpdatedAt: time.Now().UTC()}
	return m.publish(next)
}

// Unfollow forgets the subscription and cancels an in-flight refresh after
// persistence succeeds. It never removes previously installed plugins.
func (m *Manager) Unfollow(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return ErrClosed
	}
	rec, ok := m.records[id]
	if !ok {
		return ErrNotFound
	}
	next := m.copyRecords()
	delete(next, id)
	if err := m.publish(next); err != nil {
		return err
	}
	if op := m.active[rec.URL]; op != nil {
		op.cancel()
		delete(m.active, rec.URL)
	}
	return nil
}

func (m *Manager) copyRecords() map[string]record {
	next := make(map[string]record, len(m.records))
	for k, v := range m.records {
		next[k] = v
	}
	return next
}

func (m *Manager) publish(next map[string]record) error {
	saved := disk{Version: 1, Subscriptions: make([]record, 0, len(next))}
	for _, rec := range next {
		saved.Subscriptions = append(saved.Subscriptions, rec)
	}
	sort.Slice(saved.Subscriptions, func(i, j int) bool { return saved.Subscriptions[i].ID < saved.Subscriptions[j].ID })
	var data bytes.Buffer
	encoder := json.NewEncoder(&data)
	// This is a local JSON file, not inline HTML. Preserve literal description
	// text so a bounded download remains within the same bound after restart.
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(saved); err != nil {
		return err
	}
	if err := m.store.SaveCollections(data.Bytes()); err != nil {
		return err
	}
	m.records = next
	return nil
}

// Entry retains unsupported and planned content with explicit compatibility
// reasons. Compatibility is not permission or a promise of installation.
type Entry struct {
	subscription.CollectionEntry
	Compatible bool     `json:"compatible"`
	Missing    []string `json:"missing"`
}

type View struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	UpdatedAt time.Time `json:"updatedAt"`
	Entries   []Entry   `json:"entries"`
}

// List returns independent snapshots, with platform/capability checks delegated
// to Core. Callers may mutate the result without changing stored documents.
func (m *Manager) List(platform string, capabilities []string) ([]View, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, ErrClosed
	}
	views := make([]View, 0, len(m.records))
	for _, rec := range m.records {
		c, err := subscription.ParseCollection(rec.Document, rec.URL)
		if err != nil {
			return nil, err
		}
		v := View{ID: c.ID, URL: rec.URL, Name: c.Name, Version: c.Version, UpdatedAt: rec.UpdatedAt, Entries: make([]Entry, 0, len(c.Entries))}
		for _, e := range c.Entries {
			ok, missing := e.Compatible(platform, capabilities)
			v.Entries = append(v.Entries, Entry{CollectionEntry: e, Compatible: ok, Missing: missing})
		}
		views = append(views, v)
	}
	sort.Slice(views, func(i, j int) bool { return views[i].ID < views[j].ID })
	return views, nil
}

// Close cancels owned requests. Late responses cannot publish afterwards.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	for key, op := range m.active {
		op.cancel()
		delete(m.active, key)
	}
}
