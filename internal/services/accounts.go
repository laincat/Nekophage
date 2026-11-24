package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// Accounts stores the values a user enters for one service.
//
// The file lives next to the service, is readable only by its owner (0600) and
// is not encrypted: this host has no platform secret store yet, and pretending
// otherwise would be worse than saying so. A client with a keychain swaps this
// for its own implementation; the interface is deliberately small for that.
type Accounts struct {
	mu   sync.Mutex
	path string
	id   string
}

// OpenAccounts opens the account values of one installed service.
func OpenAccounts(dataDir, id string) (*Accounts, error) {
	dir, err := InstalledDir(dataDir, id)
	if err != nil {
		return nil, err
	}
	return &Accounts{path: filepath.Join(dir, "accounts.json"), id: id}, nil
}

// Set stores one value.
func (a *Accounts) Set(ctx context.Context, key, value string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if key == "" {
		return errors.New("账号键不能为空")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	values, err := a.readLocked()
	if err != nil {
		return err
	}
	values[key] = value
	return a.writeLocked(values)
}

// Get returns one value.
func (a *Accounts) Get(ctx context.Context, key string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	values, err := a.readLocked()
	if err != nil {
		return "", err
	}
	value, ok := values[key]
	if !ok {
		return "", fmt.Errorf("没有为服务 %s 保存账号项 %q", a.id, key)
	}
	return value, nil
}

// Keys lists the stored keys, never the values. It is what a settings screen
// shows: which account fields exist, not what they hold.
func (a *Accounts) Keys() ([]string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	values, err := a.readLocked()
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out, nil
}

// Missing returns the subset of requiredKeys not present (or empty) in this
// service's isolated account store. It never exposes secret values.
func (a *Accounts) Missing(requiredKeys ...string) ([]string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	values, err := a.readLocked()
	if err != nil {
		return nil, err
	}
	var missing []string
	for _, key := range requiredKeys {
		if key == "" {
			continue
		}
		if v, ok := values[key]; !ok || v == "" {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	return missing, nil
}

// Count reports how many values are stored.
func (a *Accounts) Count() (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	values, err := a.readLocked()
	if err != nil {
		return 0, err
	}
	return len(values), nil
}

// Remove deletes one value.
func (a *Accounts) Remove(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	values, err := a.readLocked()
	if err != nil {
		return err
	}
	if _, ok := values[key]; !ok {
		return fmt.Errorf("没有为服务 %s 保存账号项 %q", a.id, key)
	}
	delete(values, key)
	return a.writeLocked(values)
}

// snapshot returns a copy of stored values for internal upgrade preservation
// and secret-redaction checks. It is intentionally unexported.
func (a *Accounts) snapshot() (map[string]string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.readLocked()
}

// restore writes a snapshot back atomically (used across package upgrades).
func (a *Accounts) restore(values map[string]string) error {
	if len(values) == 0 {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.writeLocked(values)
}

// readLocked loads the stored values.
func (a *Accounts) readLocked() (map[string]string, error) {
	values := map[string]string{}
	data, err := os.ReadFile(a.path)
	if err != nil {
		if os.IsNotExist(err) {
			return values, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return values, nil
	}
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("服务账号文件 %s 损坏: %w", a.path, err)
	}
	return values, nil
}

// writeLocked atomically replaces the file, keeping it private to its owner.
func (a *Accounts) writeLocked(values map[string]string) error {
	data, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(a.path)
	tmp, err := os.CreateTemp(dir, ".accounts-*")
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
	if err := tmp.Chmod(0o600); err != nil {
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
	if err := os.Rename(tmpName, a.path); err != nil {
		return err
	}
	cleanup = false
	return nil
}
