// Package store persists user base configuration inside the profile/portable
// data directory and resolves effective settings with enabled `.vcmodule`
// overlays without ever overwriting the user's base configuration.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/laincat/nekophage/internal/vcmodule"
)

const (
	settingsFileName  = "settings.json"
	settingsSchemaVer = 1
)

type settingsDisk struct {
	Version  int               `json:"version"`
	Settings vcmodule.Settings `json:"settings"`
}

// SettingsStore owns `<dataDir>/settings.json` for base user settings and
// combines them with `<dataDir>/modules.json` when computing effective config.
type SettingsStore struct {
	mu   sync.Mutex
	root string
}

// OpenSettings opens the settings store rooted at dataDir.
func OpenSettings(dataDir string) (*SettingsStore, error) {
	if strings.TrimSpace(dataDir) == "" {
		return nil, errors.New("store: data directory is required")
	}
	abs, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, err
	}
	s := &SettingsStore{root: abs}
	if _, err := s.ReadBase(); err != nil {
		return nil, err
	}
	return s, nil
}

// Path returns the absolute path of the base settings file.
func (s *SettingsStore) Path() string {
	return filepath.Join(s.root, settingsFileName)
}

// ReadBase loads base user settings without applying any `.vcmodule` overlays.
// If no settings file exists yet, it returns DefaultSettings.
func (s *SettingsStore) ReadBase() (vcmodule.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readBaseLocked()
}

func (s *SettingsStore) readBaseLocked() (vcmodule.Settings, error) {
	raw, err := os.ReadFile(s.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return vcmodule.DefaultSettings(), nil
		}
		return vcmodule.Settings{}, err
	}
	if len(raw) == 0 {
		return vcmodule.Settings{}, fmt.Errorf("store: empty settings file %s", s.Path())
	}
	disk := settingsDisk{Settings: vcmodule.DefaultSettings()}
	if err := json.Unmarshal(raw, &disk); err != nil {
		return vcmodule.Settings{}, fmt.Errorf("store: corrupt settings file: %w", err)
	}
	if disk.Version != settingsSchemaVer {
		return vcmodule.Settings{}, fmt.Errorf("store: unsupported settings version %d", disk.Version)
	}
	if err := disk.Settings.Validate(); err != nil {
		return vcmodule.Settings{}, fmt.Errorf("store: invalid base settings: %w", err)
	}
	return disk.Settings, nil
}

// WriteBase validates and atomically persists base user settings.
// Modules never call WriteBase; only explicit user changes write here.
func (s *SettingsStore) WriteBase(settings vcmodule.Settings) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(settingsDisk{
		Version:  settingsSchemaVer,
		Settings: settings,
	}, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(s.Path(), raw, 0o600)
}

// Modules opens the `.vcmodule` manager rooted at the same data directory.
func (s *SettingsStore) Modules() (*vcmodule.Manager, error) {
	return vcmodule.Open(s.root)
}

// Effective computes the live settings by layering enabled `.vcmodule`
// overlays on top of base settings. Base settings on disk are never modified.
func (s *SettingsStore) Effective() (vcmodule.Resolved, error) {
	base, err := s.ReadBase()
	if err != nil {
		return vcmodule.Resolved{}, err
	}
	mgr, err := vcmodule.Open(s.root)
	if err != nil {
		return vcmodule.Resolved{}, err
	}
	return mgr.Resolve(base), nil
}

// EffectiveReader returns the resolved effective `[Reader]` settings.
func (s *SettingsStore) EffectiveReader() (vcmodule.ReaderSettings, error) {
	res, err := s.Effective()
	if err != nil {
		return vcmodule.DefaultReaderSettings(), err
	}
	return res.Effective.Reader, nil
}

func writeAtomic(path string, data []byte, perm os.FileMode) error {
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
