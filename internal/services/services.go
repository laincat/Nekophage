// Package services keeps the account values of service plugins and adds them
// to what Core installed.
//
// The package itself — verifying the zip, loading it once before writing,
// replacing a service of the same id as a whole — belongs to Core's
// `service.Store`. This package only adds what a client owns: the values the
// user typed for one service, and their count for the list.
package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/laincat/veneracat-core/service"
)

// Record is one installed service as a client shows it: Core's install record
// plus how many account values exist.
type Record struct {
	service.Installed
	Accounts int `json:"accounts,omitempty"`
}

// Install verifies and installs a service package.
func Install(ctx context.Context, dataDir, origin, digest string, archive []byte) (*Record, error) {
	installed, err := service.InstallPackage(ctx, dataDir, origin, digest, archive)
	if err != nil {
		return nil, err
	}
	return &Record{Installed: *installed}, nil
}

// Load reads an installed service.
func Load(ctx context.Context, dataDir, id string) (*service.Manifest, string, error) {
	return service.ReadInstalledPackage(dataDir, id)
}

// InstalledDir returns the directory of an installed service.
func InstalledDir(dataDir, id string) (string, error) {
	info, err := os.Stat(service.PackageDir(dataDir, id) + string(os.PathSeparator) + service.ManifestFileName)
	if err != nil || info.IsDir() {
		if err == nil {
			err = errors.New("清单是个目录")
		}
		return "", err
	}
	return service.PackageDir(dataDir, id), nil
}

// List returns the installed services, ordered by id, with their account
// counts.
func List(dataDir string) ([]Record, error) {
	rows, err := service.InstalledPackages(dataDir)
	if err != nil {
		return nil, err
	}
	out := make([]Record, 0, len(rows))
	for _, rec := range rows {
		record := Record{Installed: rec}
		if accounts, err := OpenAccounts(dataDir, rec.ID); err == nil {
			if count, err := accounts.Count(); err == nil {
				record.Accounts = count
			}
		}
		out = append(out, record)
	}
	return out, nil
}

// Remove deletes an installed service and the account values beside it.
func Remove(dataDir, id string) error {
	return service.RemovePackage(dataDir, id)
}

// InstalledAt is the time a service was installed, or the zero time when the
// index has no record for it.
func InstalledAt(dataDir, id string) time.Time {
	rec, err := service.InstalledPackage(dataDir, id)
	if err != nil {
		return time.Time{}
	}
	return rec.Installed
}

// Open loads an installed service with the host a desktop client gives it:
// the network through Core's bridge, storage in a file beside the package, and
// the account values the user stored. The caller closes the runtime.
//
// It is the single place a desktop build decides what a service may reach, so
// the command line and the window cannot drift apart.
func Open(ctx context.Context, dataDir, id, userAgent string) (*service.Runtime, *service.Manifest, error) {
	manifest, source, err := service.ReadInstalledPackage(dataDir, id)
	if err != nil {
		return nil, nil, err
	}
	accounts, err := OpenAccounts(dataDir, id)
	if err != nil {
		return nil, nil, err
	}
	stored, err := openServiceData(dataDir, id)
	if err != nil {
		return nil, nil, err
	}
	bridge := service.NewBridge(service.BridgePolicy{})
	granted := []service.Permission{service.PermissionNetwork, service.PermissionStorage}
	// Secrets are granted only once the user has stored something: a service
	// that asks for them but was never given an account has nothing to read.
	if count, err := accounts.Count(); err == nil && count > 0 {
		granted = append(granted, service.PermissionSecrets)
	}
	host := service.Host{
		Fetch:      bridge.Fetch,
		StorageGet: stored.Get,
		StorageSet: stored.Set,
		Secret:     accounts.Get,
	}
	rt := service.NewRuntime(host, granted...)
	if err := rt.Load(ctx, manifest, source); err != nil {
		rt.Close()
		return nil, nil, err
	}
	return rt, manifest, nil
}

// serviceData keeps the storage values of one service in the data directory.
type serviceData struct {
	path string
}

// openServiceData opens the storage file of one service.
func openServiceData(dataDir, id string) (*serviceData, error) {
	dir := filepath.Join(dataDir, "services")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &serviceData{path: filepath.Join(dir, id+".json")}, nil
}

// Get returns one stored value.
func (s *serviceData) Get(ctx context.Context, key string) (string, bool, error) {
	values, err := s.read()
	if err != nil {
		return "", false, err
	}
	value, ok := values[key]
	return value, ok, nil
}

// Set writes one stored value.
func (s *serviceData) Set(ctx context.Context, key, value string) error {
	values, err := s.read()
	if err != nil {
		return err
	}
	values[key] = value
	data, err := json.Marshal(values)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

// read loads the stored values, treating a missing file as an empty set.
func (s *serviceData) read() (map[string]string, error) {
	values := map[string]string{}
	data, err := os.ReadFile(s.path)
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
		return nil, fmt.Errorf("服务存储 %s 损坏: %w", s.path, err)
	}
	return values, nil
}

// HostPackage hosts a package that is not installed yet, for the command line:
// a directory the user pointed at. It uses the same host the installed path
// uses, with storage kept under the service's id.
func HostPackage(ctx context.Context, dataDir string, manifest *service.Manifest, source string) (*service.Runtime, error) {
	stored, err := openServiceData(dataDir, manifest.ID)
	if err != nil {
		return nil, err
	}
	accounts, err := OpenAccounts(dataDir, manifest.ID)
	if err != nil {
		// A package that is not installed has no account values; that is not a
		// reason to refuse running it, only to withhold secrets.
		accounts = nil
	}
	bridge := service.NewBridge(service.BridgePolicy{})
	granted := []service.Permission{service.PermissionNetwork, service.PermissionStorage}
	host := service.Host{Fetch: bridge.Fetch, StorageGet: stored.Get, StorageSet: stored.Set}
	if accounts != nil {
		if count, err := accounts.Count(); err == nil && count > 0 {
			granted = append(granted, service.PermissionSecrets)
			host.Secret = accounts.Get
		}
	}
	rt := service.NewRuntime(host, granted...)
	if err := rt.Load(ctx, manifest, source); err != nil {
		rt.Close()
		return nil, err
	}
	return rt, nil
}

// NewSupervisor builds a supervisor that loads installed services and gives
// each one its own host: the network through Core's bridge, storage beside the
// package, and the account values the user stored for that service only.
//
// The window keeps one of these so a call can be cancelled and a service stays
// loaded between calls rather than being compiled again each time.
func NewSupervisor(dataDir, userAgent string) (*service.Supervisor, error) {
	hostFor := func(id string) service.Host {
		host := service.Host{}
		if stored, err := openServiceData(dataDir, id); err == nil {
			host.StorageGet = stored.Get
			host.StorageSet = stored.Set
		}
		if accounts, err := OpenAccounts(dataDir, id); err == nil {
			host.Secret = accounts.Get
		}
		bridge := service.NewBridge(service.BridgePolicy{})
		host.Fetch = bridge.Fetch
		return host
	}
	allowed := []service.Permission{service.PermissionNetwork, service.PermissionStorage, service.PermissionSecrets}
	return service.NewSupervisorFor(hostFor, allowed, service.SupervisorOptions{}, func(ctx context.Context, id string) (*service.Manifest, string, error) {
		return service.ReadInstalledPackage(dataDir, id)
	}), nil
}
