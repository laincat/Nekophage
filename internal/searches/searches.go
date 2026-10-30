// Package searches remembers what the reader typed, so a keyword can be
// tried again without retyping it.
//
// The official client keeps the same list in appdata.json's searchHistory
// and a `.venera` backup carries it, so an import can fill this in.
package searches

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// limit is how many keywords are kept; the official client does the same.
const limit = 30

// Store is the profile's search history.
type Store struct {
	mu    sync.Mutex
	path  string
	items []string
}

// Open reads the history; a profile without one starts empty.
func Open(dir string) (*Store, error) {
	if dir == "" {
		return nil, errors.New("searches: directory is required")
	}
	s := &Store{path: filepath.Join(dir, "searches.json")}
	b, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return s, nil
		}
		return nil, err
	}
	var doc struct {
		Queries []string `json:"queries"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	s.items = doc.Queries
	return s, nil
}

// Add puts a keyword at the front. Case-insensitive repeats replace the old
// entry, so searching "One Piece" after "one piece" keeps one.
func (s *Store) Add(q string) error {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []string{q}
	for _, it := range s.items {
		if strings.EqualFold(it, q) {
			continue
		}
		out = append(out, it)
	}
	if len(out) > limit {
		out = out[:limit]
	}
	s.items = out
	return s.save()
}

// Items returns the keywords, newest first.
func (s *Store) Items(max int) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if max <= 0 || max > len(s.items) {
		max = len(s.items)
	}
	return append([]string(nil), s.items[:max]...)
}

// Merge adds keywords from an import, keeping local ones ahead of them.
func (s *Store) Merge(items []string) (int, error) {
	s.mu.Lock()
	out := append([]string(nil), s.items...)
	seen := map[string]bool{}
	for _, it := range out {
		seen[strings.ToLower(it)] = true
	}
	added := 0
	// The backup lists newest first, which is the order kept here.
	for _, raw := range items {
		q := strings.TrimSpace(raw)
		if q == "" || seen[strings.ToLower(q)] {
			continue
		}
		seen[strings.ToLower(q)] = true
		out = append(out, q)
		added++
	}
	if len(out) > limit {
		out = out[:limit]
	}
	s.items = out
	s.mu.Unlock()
	return added, s.save()
}

// Remove drops one keyword.
func (s *Store) Remove(q string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.items[:0:0]
	for _, it := range s.items {
		if strings.EqualFold(it, q) {
			continue
		}
		out = append(out, it)
	}
	s.items = out
	return s.save()
}

// Clear drops every keyword.
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = nil
	return s.save()
}

// save writes the file through a temporary name, so a crash cannot leave a
// half-written list.
func (s *Store) save() error {
	items := s.items
	if items == nil {
		items = []string{}
	}
	b, err := json.MarshalIndent(struct {
		Queries []string `json:"queries"`
	}{Queries: items}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
