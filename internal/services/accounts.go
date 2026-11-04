package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Accounts stores the values a user enters for one service.
//
// The file lives next to the service, is readable only by its owner (0600) and
// is not encrypted: this host has no platform secret store yet, and pretending
// otherwise would be worse than saying so. A client with a keychain swaps this
// for its own implementation; the interface is deliberately small for that.
type Accounts struct {
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
	if key == "" {
		return errors.New("账号键不能为空")
	}
	values, err := a.read()
	if err != nil {
		return err
	}
	values[key] = value
	return a.write(values)
}

// Get returns one value.
func (a *Accounts) Get(ctx context.Context, key string) (string, error) {
	values, err := a.read()
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
	values, err := a.read()
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

// Count reports how many values are stored.
func (a *Accounts) Count() (int, error) {
	values, err := a.read()
	if err != nil {
		return 0, err
	}
	return len(values), nil
}

// Remove deletes one value.
func (a *Accounts) Remove(ctx context.Context, key string) error {
	values, err := a.read()
	if err != nil {
		return err
	}
	if _, ok := values[key]; !ok {
		return fmt.Errorf("没有为服务 %s 保存账号项 %q", a.id, key)
	}
	delete(values, key)
	return a.write(values)
}

// read loads the stored values.
func (a *Accounts) read() (map[string]string, error) {
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

// write replaces the file, keeping it private to its owner.
func (a *Accounts) write(values map[string]string) error {
	data, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return err
	}
	tmp := a.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, a.path)
}
