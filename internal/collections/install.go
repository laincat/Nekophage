package collections

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/laincat/veneracat-core/subscription"
)

// MaxArtifactBytes bounds one collection artifact: a source script is text, and
// an entry that declares something larger is not something this client runs.
const MaxArtifactBytes = 8 << 20

// Entry returns one collection entry with the platform/capability verdict, so a
// caller can say why an entry is not installable here.
func (m *Manager) Entry(id, entryID, platform string, capabilities []string) (Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return Entry{}, ErrClosed
	}
	rec, ok := m.records[id]
	if !ok {
		return Entry{}, fmt.Errorf("collections: unknown collection %q", id)
	}
	c, err := subscription.ParseCollection(rec.Document, rec.URL)
	if err != nil {
		return Entry{}, err
	}
	for _, e := range c.Entries {
		if e.ID != entryID {
			continue
		}
		compatible, missing := e.Compatible(platform, capabilities)
		return Entry{CollectionEntry: e, Compatible: compatible, Missing: missing}, nil
	}
	return Entry{}, fmt.Errorf("collections: collection %q has no entry %q", id, entryID)
}

// Artifact downloads an entry's script and verifies it before anything runs:
// the declared size and SHA-256 are the only identity this client trusts.
func (m *Manager) Artifact(ctx context.Context, entry Entry) ([]byte, error) {
	a := entry.Artifact
	if a == nil {
		return nil, fmt.Errorf("collections: entry %q declares no artifact", entry.ID)
	}
	if a.Bytes <= 0 || a.Bytes > MaxArtifactBytes {
		return nil, fmt.Errorf("collections: artifact size %d is not acceptable", a.Bytes)
	}
	if len(a.SHA256) != 64 {
		return nil, errors.New("collections: artifact digest is malformed")
	}
	// Fetch with the protocol's own ceiling, not the declared size: a snapshot
	// whose artifact grew would otherwise fail as "response larger than the
	// limit", which says nothing about the artifact being wrong.
	body, _, err := m.transport.Fetch(ctx, a.URL, MaxArtifactBytes)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(body)
	if int64(len(body)) != a.Bytes || hex.EncodeToString(sum[:]) != strings.ToLower(a.SHA256) {
		return nil, fmt.Errorf("collections: artifact %q does not match its declared size or digest", entry.ID)
	}
	return body, nil
}
