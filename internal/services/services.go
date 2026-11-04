// Package services installs and loads service packages on disk.
//
// A service package is a zip holding service.json and the entry it names. It is
// installed under <data dir>/services/<id>/ and runs from there, so a service
// survives a restart and an upgrade replaces it as a whole.
package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/laincat/veneracat-core/service"
)

// Record is what an installed service says about where it came from.
type Record struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	Origin    string    `json:"origin"`
	SHA256    string    `json:"sha256,omitempty"`
	Installed time.Time `json:"installed"`
	Dir       string    `json:"dir"`
	// Permissions is what the manifest asked for, kept so a client can show it
	// without unpacking the package again.
	Permissions []string `json:"permissions,omitempty"`
	// Accounts is how many account values are stored for this service.
	Accounts int `json:"accounts,omitempty"`
}

// Install verifies and installs a service package.
//
// The package is parsed and compiled before anything is written: a service that
// does not load is refused rather than installed and then failing at first use.
// An existing service of the same id is replaced as a whole, so an upgrade
// cannot leave files from the previous version behind.
func Install(ctx context.Context, dataDir, origin, digest string, archive []byte) (*Record, error) {
	manifest, source, files, err := service.ReadPackageZip(archive)
	if err != nil {
		return nil, err
	}
	if err := checkCompiles(ctx, manifest, source); err != nil {
		return nil, err
	}
	dir := filepath.Join(dataDir, "services", manifest.ID)
	tmp := dir + ".new"
	if err := os.RemoveAll(tmp); err != nil {
		return nil, err
	}
	for name, content := range files {
		if err := service.ValidFilePath(name); err != nil {
			return nil, err
		}
		path := filepath.Join(tmp, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, content, 0o600); err != nil {
			return nil, err
		}
	}
	if err := os.RemoveAll(dir); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp, dir); err != nil {
		return nil, err
	}
	rec := &Record{
		ID: manifest.ID, Name: manifest.Name, Version: manifest.Version,
		Origin: origin, SHA256: digest, Installed: time.Now().UTC(), Dir: dir,
		Permissions: permissionNames(manifest.Permissions),
	}
	if err := writeRecord(dataDir, rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// permissionNames turns the manifest permissions into plain strings for the
// install record.
func permissionNames(list []service.Permission) []string {
	out := make([]string, 0, len(list))
	for _, p := range list {
		out = append(out, string(p))
	}
	return out
}

// checkCompiles loads the service in a throwaway runtime: a package whose
// manifest is fine but whose JavaScript does not parse is refused at install
// time rather than at the first call.
func checkCompiles(ctx context.Context, manifest *service.Manifest, source string) error {
	rt := service.NewRuntime(service.Host{})
	defer rt.Close()
	return rt.Load(ctx, manifest, source)
}

// Load reads an installed service.
func Load(ctx context.Context, dataDir, id string) (*service.Manifest, string, error) {
	dir, err := InstalledDir(dataDir, id)
	if err != nil {
		return nil, "", err
	}
	return service.ReadPackage(dir)
}

// InstalledDir returns the directory of an installed service.
func InstalledDir(dataDir, id string) (string, error) {
	if id == "" || strings.ContainsAny(id, `/\`) || strings.HasPrefix(id, ".") {
		return "", fmt.Errorf("服务 ID %q 不合法", id)
	}
	dir := filepath.Join(dataDir, "services", id)
	info, err := os.Stat(filepath.Join(dir, service.ManifestFileName))
	if err != nil || info.IsDir() {
		if err == nil {
			err = errors.New("清单是个目录")
		}
		return "", fmt.Errorf("未安装服务 %s: %w", id, err)
	}
	return dir, nil
}

// List returns the installed services, ordered by id.
func List(dataDir string) ([]Record, error) {
	all, err := readRecords(dataDir)
	if err != nil {
		return nil, err
	}
	out := make([]Record, 0, len(all))
	for _, rec := range all {
		if accounts, err := OpenAccounts(dataDir, rec.ID); err == nil {
			if count, err := accounts.Count(); err == nil {
				rec.Accounts = count
			}
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// Remove deletes an installed service.
func Remove(dataDir, id string) error {
	dir, err := InstalledDir(dataDir, id)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	all, err := readRecords(dataDir)
	if err != nil {
		return err
	}
	delete(all, id)
	return writeRecords(dataDir, all)
}

// indexPath is where the install records live.
func indexPath(dataDir string) string {
	return filepath.Join(dataDir, "services.json")
}

// readRecords reads the install index, keyed by id.
func readRecords(dataDir string) (map[string]Record, error) {
	out := map[string]Record{}
	data, err := os.ReadFile(indexPath(dataDir))
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("服务索引 %s 损坏: %w", indexPath(dataDir), err)
	}
	return out, nil
}

// writeRecord adds or replaces one record.
func writeRecord(dataDir string, rec *Record) error {
	all, err := readRecords(dataDir)
	if err != nil {
		return err
	}
	all[rec.ID] = *rec
	return writeRecords(dataDir, all)
}

// writeRecords writes the whole index.
func writeRecords(dataDir string, all map[string]Record) error {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	path := indexPath(dataDir)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
