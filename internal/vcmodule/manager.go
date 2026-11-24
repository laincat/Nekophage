package vcmodule

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	indexFileName    = "modules.json"
	modulesSubdir    = "modules"
	indexSchemaVer   = 1
	maxModuleIDBytes = 128
)

// idPattern restricts module IDs to safe, lowercase dot/dash-separated tokens
// that cannot traverse directories or collide across case-insensitive filesystems.
var idPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)*$`)

// Record describes one installed `.vcmodule` overlay in the profile.
type Record struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Desc      string    `json:"desc,omitempty"`
	Version   string    `json:"version"`
	Format    string    `json:"format"`
	Requires  []string  `json:"requires,omitempty"`
	Origin    string    `json:"origin,omitempty"`
	SHA256    string    `json:"sha256"`
	Enabled   bool      `json:"enabled"`
	Keys      []string  `json:"keys"`
	Installed time.Time `json:"installed"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type indexDisk struct {
	Version int      `json:"version"`
	Modules []Record `json:"modules"`
}

// Manager persists installed `.vcmodule` files under `<dataDir>/modules/` and
// tracks enablement and stacking order in `<dataDir>/modules.json`.
//
// It never writes or modifies base user settings.
type Manager struct {
	mu      sync.Mutex
	root    string
	records []Record
	parsed  map[string]*Module
}

// ValidateID checks that id is safe to use as a module identifier and file stem.
func ValidateID(id string) error {
	if id == "" || len(id) > maxModuleIDBytes || !idPattern.MatchString(id) {
		return fmt.Errorf("%w: %q", ErrInvalidID, id)
	}
	if strings.ContainsAny(id, `/\:`) || strings.Contains(id, "..") {
		return fmt.Errorf("%w: %q", ErrInvalidID, id)
	}
	return nil
}

// Open loads the module index and validates all installed `.vcmodule` files on
// disk. Corrupt index or tampered module files return an error instead of
// silently resetting state.
func Open(dataDir string) (*Manager, error) {
	if strings.TrimSpace(dataDir) == "" {
		return nil, errors.New("vcmodule: data directory is required")
	}
	absRoot, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, err
	}
	m := &Manager{
		root:   absRoot,
		parsed: map[string]*Module{},
	}
	data, err := os.ReadFile(m.indexPath())
	if err != nil {
		if os.IsNotExist(err) {
			return m, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("%w: empty module index %s", ErrInvalidFormat, m.indexPath())
	}
	var disk indexDisk
	if err := json.Unmarshal(data, &disk); err != nil {
		return nil, fmt.Errorf("vcmodule: corrupt module index: %w", err)
	}
	if disk.Version != indexSchemaVer {
		return nil, fmt.Errorf("vcmodule: unsupported module index version %d", disk.Version)
	}
	seen := map[string]bool{}
	for _, rec := range disk.Modules {
		if err := ValidateID(rec.ID); err != nil {
			return nil, err
		}
		if seen[rec.ID] {
			return nil, fmt.Errorf("vcmodule: duplicate module id %q in index", rec.ID)
		}
		seen[rec.ID] = true
		modPath, err := m.modulePath(rec.ID)
		if err != nil {
			return nil, err
		}
		raw, err := readBoundedFile(modPath, MaxModuleBytes)
		if err != nil {
			return nil, fmt.Errorf("vcmodule: read installed module %q: %w", rec.ID, err)
		}
		mod, err := Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("vcmodule: installed module %q is invalid: %w", rec.ID, err)
		}
		sum := sha256.Sum256(raw)
		if rec.SHA256 != "" && !strings.EqualFold(hex.EncodeToString(sum[:]), rec.SHA256) {
			return nil, fmt.Errorf("vcmodule: installed module %q failed digest verification", rec.ID)
		}
		m.records = append(m.records, cloneRecord(rec))
		m.parsed[rec.ID] = mod
	}
	return m, nil
}

// Install parses, validates and persists one `.vcmodule` under `<dataDir>/modules/<id>.vcmodule`.
//
// If id is empty and the module declares `#!id=...`, the declared ID is used.
// If both are present and differ, Install rejects the mismatch.
// Newly installed modules start disabled so installing never alters effective
// settings without explicit user enablement. Upgrading an existing module
// preserves its enablement and overlay stacking order.
func (m *Manager) Install(id, origin, declaredSHA256 string, raw []byte) (*Record, error) {
	if id != "" {
		if err := ValidateID(id); err != nil {
			return nil, err
		}
	}
	mod, err := Parse(raw)
	if err != nil {
		return nil, err
	}
	if id == "" {
		id = mod.Meta.ID
	}
	if err := ValidateID(id); err != nil {
		return nil, err
	}
	if mod.Meta.ID != "" && mod.Meta.ID != id {
		return nil, fmt.Errorf("%w: declared #!id %q does not match install id %q", ErrInvalidID, mod.Meta.ID, id)
	}
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	if declaredSHA256 != "" && !strings.EqualFold(strings.TrimSpace(declaredSHA256), digest) {
		return nil, fmt.Errorf("vcmodule: digest mismatch for %q", id)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	modPath, err := m.modulePath(id)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(modPath), 0o700); err != nil {
		return nil, err
	}
	if err := writeFileAtomic(modPath, raw, 0o600); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	next := make([]Record, len(m.records))
	for i, r := range m.records {
		next[i] = cloneRecord(r)
	}

	idx := -1
	for i, r := range next {
		if r.ID == id {
			idx = i
			break
		}
	}

	rec := Record{
		ID:        id,
		Name:      mod.Meta.Name,
		Desc:      mod.Meta.Desc,
		Version:   mod.Meta.Version,
		Format:    mod.Meta.Format,
		Requires:  slices.Clone(mod.Meta.Requires),
		Origin:    origin,
		SHA256:    digest,
		Enabled:   false,
		Keys:      append([]string(nil), mod.Keys...),
		Installed: now,
		UpdatedAt: now,
	}
	if idx >= 0 {
		rec.Enabled = next[idx].Enabled
		rec.Installed = next[idx].Installed
		next[idx] = rec
	} else {
		next = append(next, rec)
	}

	if err := m.writeIndexLocked(next); err != nil {
		if idx < 0 {
			_ = os.Remove(modPath)
		}
		return nil, err
	}
	m.records = next
	m.parsed[id] = mod
	out := cloneRecord(rec)
	return &out, nil
}

// InstallFile reads a local `.vcmodule` file (bounded by MaxModuleBytes) and
// installs it under id.
func (m *Manager) InstallFile(id, filePath string) (*Record, error) {
	raw, err := readBoundedFile(filePath, MaxModuleBytes)
	if err != nil {
		return nil, err
	}
	return m.Install(id, "file:"+filepath.Base(filePath), "", raw)
}

// SetEnabled enables or disables an installed module.
// Disabling removes the module's overlay on subsequent Resolve calls without
// touching base user settings.
func (m *Manager) SetEnabled(id string, enabled bool) error {
	if err := ValidateID(id); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	idx := m.indexOfLocked(id)
	if idx < 0 {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	if m.records[idx].Enabled == enabled {
		return nil
	}
	next := m.cloneRecordsLocked()
	next[idx].Enabled = enabled
	next[idx].UpdatedAt = time.Now().UTC()
	if err := m.writeIndexLocked(next); err != nil {
		return err
	}
	m.records = next
	return nil
}

// Enable enables an installed module.
func (m *Manager) Enable(id string) error { return m.SetEnabled(id, true) }

// Disable disables an installed module, cleanly withdrawing its overlay.
func (m *Manager) Disable(id string) error { return m.SetEnabled(id, false) }

// Reorder updates the deterministic overlay stacking order. Later IDs take
// precedence over earlier IDs. ids must contain every installed module ID
// exactly once.
func (m *Manager) Reorder(ids []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(ids) != len(m.records) {
		return fmt.Errorf("vcmodule: reorder expects %d ids, got %d", len(m.records), len(ids))
	}
	byID := make(map[string]Record, len(m.records))
	for _, r := range m.records {
		byID[r.ID] = r
	}
	seen := make(map[string]bool, len(ids))
	next := make([]Record, 0, len(ids))
	for _, id := range ids {
		if seen[id] {
			return fmt.Errorf("vcmodule: duplicate id %q in reorder", id)
		}
		rec, ok := byID[id]
		if !ok {
			return fmt.Errorf("%w: %s", ErrNotFound, id)
		}
		seen[id] = true
		next = append(next, cloneRecord(rec))
	}
	if err := m.writeIndexLocked(next); err != nil {
		return err
	}
	m.records = next
	return nil
}

// Remove deletes an installed module from disk and from the index, cleanly
// withdrawing its overlay without touching base user settings.
func (m *Manager) Remove(id string) error {
	if err := ValidateID(id); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	idx := m.indexOfLocked(id)
	if idx < 0 {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	modPath, err := m.modulePath(id)
	if err != nil {
		return err
	}
	next := make([]Record, 0, len(m.records)-1)
	for i, r := range m.records {
		if i == idx {
			continue
		}
		next = append(next, cloneRecord(r))
	}
	if err := m.writeIndexLocked(next); err != nil {
		return err
	}
	if err := os.Remove(modPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	m.records = next
	delete(m.parsed, id)
	return nil
}

// List returns copies of all installed module records in stacking order.
func (m *Manager) List() []Record {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cloneRecordsLocked()
}

// Get returns a copy of the record and parsed module for id.
func (m *Manager) Get(id string) (*Record, *Module, error) {
	if err := ValidateID(id); err != nil {
		return nil, nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	idx := m.indexOfLocked(id)
	if idx < 0 {
		return nil, nil, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	rec := cloneRecord(m.records[idx])
	mod := cloneModule(m.parsed[id])
	return &rec, mod, nil
}

// Resolve evaluates all enabled modules in stacking order on top of base.
// Base is never modified.
func (m *Manager) Resolve(base Settings) Resolved {
	m.mu.Lock()
	layers := make([]Layer, 0, len(m.records))
	for _, rec := range m.records {
		if !rec.Enabled {
			continue
		}
		if mod := m.parsed[rec.ID]; mod != nil {
			layers = append(layers, Layer{ID: rec.ID, Module: mod})
		}
	}
	m.mu.Unlock()
	return Apply(base, layers)
}

func (m *Manager) indexPath() string {
	return filepath.Join(m.root, indexFileName)
}

func (m *Manager) modulePath(id string) (string, error) {
	if err := ValidateID(id); err != nil {
		return "", err
	}
	dir := filepath.Join(m.root, modulesSubdir)
	target := filepath.Join(dir, id+".vcmodule")
	rel, err := filepath.Rel(dir, target)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") || strings.ContainsAny(rel, `/\`) {
		return "", fmt.Errorf("%w: path escapes modules directory", ErrInvalidID)
	}
	return target, nil
}

func (m *Manager) indexOfLocked(id string) int {
	for i, r := range m.records {
		if r.ID == id {
			return i
		}
	}
	return -1
}

func (m *Manager) cloneRecordsLocked() []Record {
	out := make([]Record, len(m.records))
	for i, r := range m.records {
		out[i] = cloneRecord(r)
	}
	return out
}

func (m *Manager) writeIndexLocked(records []Record) error {
	if err := os.MkdirAll(m.root, 0o700); err != nil {
		return err
	}
	payload := indexDisk{
		Version: indexSchemaVer,
		Modules: records,
	}
	if payload.Modules == nil {
		payload.Modules = []Record{}
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(m.indexPath(), raw, 0o600)
}

func cloneRecord(r Record) Record {
	r.Requires = slices.Clone(r.Requires)
	r.Keys = slices.Clone(r.Keys)
	return r
}

func cloneModule(m *Module) *Module {
	if m == nil {
		return nil
	}
	cp := *m
	cp.Meta.Requires = slices.Clone(m.Meta.Requires)
	cp.Keys = slices.Clone(m.Keys)
	if m.Reader.PageGap != nil {
		v := *m.Reader.PageGap
		cp.Reader.PageGap = &v
	}
	if m.Reader.KeepScreenOn != nil {
		v := *m.Reader.KeepScreenOn
		cp.Reader.KeepScreenOn = &v
	}
	if m.Reader.FitWidth != nil {
		v := *m.Reader.FitWidth
		cp.Reader.FitWidth = &v
	}
	if m.Reader.Spread != nil {
		v := *m.Reader.Spread
		cp.Reader.Spread = &v
	}
	if m.Reader.Direction != nil {
		v := *m.Reader.Direction
		cp.Reader.Direction = &v
	}
	if m.Reader.AutoNext != nil {
		v := *m.Reader.AutoNext
		cp.Reader.AutoNext = &v
	}
	if m.Reader.Theme != nil {
		v := *m.Reader.Theme
		cp.Reader.Theme = &v
	}
	return &cp
}

func readBoundedFile(path string, maxBytes int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("%w: file exceeds %d bytes", ErrInvalidFormat, maxBytes)
	}
	return data, nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}
