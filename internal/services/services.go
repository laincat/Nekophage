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
	"errors"
	"os"
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
