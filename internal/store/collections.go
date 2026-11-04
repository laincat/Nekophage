package store

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

// MaxCollectionsBytes bounds the combined cached subscription documents.
const MaxCollectionsBytes = 32 << 20

// Collections returns the profile's shared-source subscriptions, separately
// from installed plugins. An absent file represents an empty subscription set.
func (s *Store) Collections() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.Open(filepath.Join(s.root, "collections.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxCollectionsBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxCollectionsBytes {
		return nil, errors.New("store: collection snapshots exceed 32 MiB")
	}
	return data, nil
}

// SaveCollections publishes a complete subscription snapshot atomically.
func (s *Store) SaveCollections(data []byte) error {
	if len(data) > MaxCollectionsBytes {
		return errors.New("store: collection snapshots exceed 32 MiB")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return writeFile(filepath.Join(s.root, "collections.json"), data)
}
