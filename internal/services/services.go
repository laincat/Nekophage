// Package services keeps the account values of service plugins, enforces
// host permission/secret boundaries, and provides typed client adapters on
// top of Core's `service` runtime and `service.Store`.
//
// The package itself — verifying the zip, loading it once before writing,
// replacing a service of the same id as a whole — belongs to Core's
// `service.Store`. This package adds what a desktop client owns: isolated
// per-service account secrets, upgrade account preservation, permission
// inspection/trimming, secret-leak guards on KV storage and logs, and typed
// clients for Trakt / TMDB / Bangumi.
package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/laincat/veneracat-core/service"
)

// MaxPackageZipBytes bounds a local `.zip` service package read from disk.
const MaxPackageZipBytes = 16 << 20

var (
	// ErrPermissionDenied reports that one or more requested/required permissions
	// were not granted by the host or not declared by the service manifest.
	ErrPermissionDenied = errors.New("service: permission denied")
	// ErrMissingSecrets reports that one or more required account keys have not
	// been configured in the isolated account store.
	ErrMissingSecrets = errors.New("service: missing required account secrets")
	// ErrSecretLeak reports an attempt to persist an account secret into
	// ordinary KV storage.
	ErrSecretLeak = errors.New("service: refusing to write account secret into ordinary storage")
)

var validIDPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)*$`)

func validServiceID(id string) error {
	if !validIDPattern.MatchString(id) || strings.ContainsAny(id, `/\:`) || strings.Contains(id, "..") {
		return fmt.Errorf("service: %q is not a valid service id", id)
	}
	return nil
}

// PermissionDeniedError names the exact permissions that were denied.
type PermissionDeniedError struct {
	ServiceID string
	Denied    []service.Permission
}

func (e *PermissionDeniedError) Error() string {
	names := make([]string, len(e.Denied))
	for i, p := range e.Denied {
		names[i] = string(p)
	}
	return fmt.Sprintf("服务 %s 缺少所需权限授权，已拒绝项: [%s]", e.ServiceID, strings.Join(names, ", "))
}

func (e *PermissionDeniedError) Unwrap() error { return ErrPermissionDenied }

// MissingSecretsError names the missing account keys (never values).
type MissingSecretsError struct {
	ServiceID string
	Keys      []string
}

func (e *MissingSecretsError) Error() string {
	return fmt.Sprintf("服务 %s 缺少必需账号密钥项: [%s]", e.ServiceID, strings.Join(e.Keys, ", "))
}

func (e *MissingSecretsError) Unwrap() error { return ErrMissingSecrets }

// PermissionReport summarizes what a service requested versus what the host
// granted or denied.
type PermissionReport struct {
	ServiceID      string               `json:"serviceId"`
	Requested      []service.Permission `json:"requested"`
	Granted        []service.Permission `json:"granted"`
	Denied         []service.Permission `json:"denied"`
	MissingSecrets []string             `json:"missingSecrets,omitempty"`
}

// Record is one installed service as a client shows it: Core's install record
// plus how many account values exist.
type Record struct {
	service.Installed
	Accounts int `json:"accounts,omitempty"`
}

// HostOptions configures how a service runtime is hosted.
type HostOptions struct {
	UserAgent string
	// AllowedPermissions restricts the permissions granted by the host.
	// Nil means desktop defaults (network + storage, plus secrets when
	// accounts exist or for Supervisor calls). A non-nil slice (including an
	// empty slice) explicitly trims permissions to only those listed.
	AllowedPermissions []service.Permission
	// StrictPermissions causes OpenWithOptions to fail upfront with
	// *PermissionDeniedError if any manifest-requested permission is denied.
	StrictPermissions bool
	// BridgePolicy controls HTTP host allowlist, response ceiling, timeout and cookies.
	BridgePolicy service.BridgePolicy
	// Log receives sanitized service log lines (account secret values are
	// redacted before forwarding).
	Log func(level, message string)
	// SupervisorOptions configures resident limits, idle sweep and call timeout.
	SupervisorOptions service.SupervisorOptions
}

// InspectZip parses and validates a service package zip's manifest without
// writing or running any code.
func InspectZip(archive []byte) (*service.Manifest, error) {
	m, _, _, err := service.ReadPackageZip(archive)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// InspectZipFile reads a local `.zip` package (bounded by MaxPackageZipBytes),
// computes its SHA-256 digest and validates its manifest.
func InspectZipFile(zipPath string) (*service.Manifest, string, []byte, error) {
	f, err := os.Open(zipPath)
	if err != nil {
		return nil, "", nil, err
	}
	defer f.Close()
	archive, err := io.ReadAll(io.LimitReader(f, MaxPackageZipBytes+1))
	if err != nil {
		return nil, "", nil, err
	}
	if int64(len(archive)) > MaxPackageZipBytes {
		return nil, "", nil, fmt.Errorf("service: package archive exceeds %d bytes", MaxPackageZipBytes)
	}
	m, err := InspectZip(archive)
	if err != nil {
		return nil, "", nil, err
	}
	sum := sha256.Sum256(archive)
	return m, hex.EncodeToString(sum[:]), archive, nil
}

// EvaluatePermissions compares a service manifest's requested permissions
// against the host's allowed permissions.
func EvaluatePermissions(m *service.Manifest, allowed []service.Permission) (PermissionReport, error) {
	grants, err := service.Grant(m, allowed)
	if err != nil {
		return PermissionReport{}, err
	}
	requested := append([]service.Permission(nil), m.Permissions...)
	sort.Slice(requested, func(i, j int) bool { return requested[i] < requested[j] })
	return PermissionReport{
		ServiceID: m.ID,
		Requested: requested,
		Granted:   grants.Effective(),
		Denied:    grants.Denied(),
	}, nil
}

// RequirePermissions evaluates manifest permissions against allowed and
// returns *PermissionDeniedError if any requested permission is denied.
func RequirePermissions(m *service.Manifest, allowed []service.Permission) (PermissionReport, error) {
	report, err := EvaluatePermissions(m, allowed)
	if err != nil {
		return PermissionReport{}, err
	}
	if len(report.Denied) > 0 {
		return report, &PermissionDeniedError{ServiceID: m.ID, Denied: report.Denied}
	}
	return report, nil
}

// InspectInstalledPermissions evaluates the effective and denied permissions
// of an installed service. When allowed is nil, desktop defaults are used
// (network + storage, plus secrets only when at least one account value exists).
func InspectInstalledPermissions(dataDir, id string, allowed []service.Permission) (PermissionReport, error) {
	m, _, err := service.ReadInstalledPackage(dataDir, id)
	if err != nil {
		return PermissionReport{}, err
	}
	effAllowed := allowed
	if effAllowed == nil {
		effAllowed = defaultAllowedFor(dataDir, id)
	}
	return EvaluatePermissions(m, effAllowed)
}

func defaultAllowedFor(dataDir, id string) []service.Permission {
	granted := []service.Permission{service.PermissionNetwork, service.PermissionStorage}
	if accounts, err := OpenAccounts(dataDir, id); err == nil {
		if count, err := accounts.Count(); err == nil && count > 0 {
			granted = append(granted, service.PermissionSecrets)
		}
	}
	return granted
}

// Install verifies and installs a service package.
func Install(ctx context.Context, dataDir, origin, digest string, archive []byte) (*Record, error) {
	return InstallExpecting(ctx, dataDir, "", origin, digest, archive)
}

// InstallExpecting verifies and installs (or upgrades) a service package.
// When upgrading an existing service of the same id, previously saved isolated
// account values are preserved rather than lost when the package directory is
// replaced.
func InstallExpecting(ctx context.Context, dataDir, expectedID, origin, digest string, archive []byte) (*Record, error) {
	// Snapshot existing isolated accounts before replacing the package directory.
	var preserved map[string]string
	if manifest, err := InspectZip(archive); err == nil {
		if expectedID == "" || manifest.ID == expectedID {
			if existing, err := OpenAccounts(dataDir, manifest.ID); err == nil {
				if snap, err := existing.snapshot(); err == nil && len(snap) > 0 {
					preserved = snap
				}
			}
		}
	}

	installed, err := service.InstallPackageExpecting(ctx, dataDir, expectedID, origin, digest, archive)
	if err != nil {
		return nil, err
	}
	rec := &Record{Installed: *installed}
	if len(preserved) > 0 {
		accounts, err := OpenAccounts(dataDir, installed.ID)
		if err != nil {
			return nil, fmt.Errorf("service: restore accounts after upgrade: %w", err)
		}
		if err := accounts.restore(preserved); err != nil {
			return nil, fmt.Errorf("service: restore accounts after upgrade: %w", err)
		}
		rec.Accounts = len(preserved)
	}
	return rec, nil
}

// InstallZipFile verifies and installs (or upgrades) a local `.zip` package
// and returns both the install record and its permission report under desktop
// defaults.
func InstallZipFile(ctx context.Context, dataDir, zipPath, expectedID string) (*Record, PermissionReport, error) {
	_, digest, archive, err := InspectZipFile(zipPath)
	if err != nil {
		return nil, PermissionReport{}, err
	}
	rec, err := InstallExpecting(ctx, dataDir, expectedID, "file:"+filepath.Base(zipPath), digest, archive)
	if err != nil {
		return nil, PermissionReport{}, err
	}
	report, err := InspectInstalledPermissions(dataDir, rec.ID, nil)
	if err != nil {
		return rec, PermissionReport{}, err
	}
	return rec, report, nil
}

// Load reads an installed service.
func Load(ctx context.Context, dataDir, id string) (*service.Manifest, string, error) {
	return service.ReadInstalledPackage(dataDir, id)
}

// InstalledDir returns the directory of an installed service.
func InstalledDir(dataDir, id string) (string, error) {
	if err := validServiceID(id); err != nil {
		return "", err
	}
	dir := service.PackageDir(dataDir, id)
	info, err := os.Stat(filepath.Join(dir, service.ManifestFileName))
	if err != nil || info.IsDir() {
		if err == nil {
			err = errors.New("清单是个目录")
		}
		return "", err
	}
	return dir, nil
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
	if err := validServiceID(id); err != nil {
		return err
	}
	return service.RemovePackage(dataDir, id)
}

// Open loads an installed service with the host a desktop client gives it:
// the network through Core's bridge, storage in a file beside the package, and
// the account values the user stored. The caller closes the runtime.
//
// It is the single place a desktop build decides what a service may reach, so
// the command line and the window cannot drift apart.
func Open(ctx context.Context, dataDir, id, userAgent string) (*service.Runtime, *service.Manifest, error) {
	rt, manifest, _, err := OpenWithOptions(ctx, dataDir, id, HostOptions{UserAgent: userAgent})
	return rt, manifest, err
}

// OpenWithOptions loads an installed service with explicit permission,
// bridge-policy and secret-protection options.
func OpenWithOptions(ctx context.Context, dataDir, id string, opts HostOptions) (*service.Runtime, *service.Manifest, PermissionReport, error) {
	manifest, source, err := service.ReadInstalledPackage(dataDir, id)
	if err != nil {
		return nil, nil, PermissionReport{}, err
	}
	accounts, err := OpenAccounts(dataDir, id)
	if err != nil {
		return nil, nil, PermissionReport{}, err
	}
	stored, err := service.OpenStorage(dataDir, id)
	if err != nil {
		return nil, nil, PermissionReport{}, err
	}

	granted := opts.AllowedPermissions
	if granted == nil {
		granted = []service.Permission{service.PermissionNetwork, service.PermissionStorage}
		// Secrets are granted only once the user has stored something: a service
		// that asks for them but was never given an account has nothing to read.
		if count, err := accounts.Count(); err == nil && count > 0 {
			granted = append(granted, service.PermissionSecrets)
		}
	}

	report, err := EvaluatePermissions(manifest, granted)
	if err != nil {
		return nil, nil, PermissionReport{}, err
	}
	if opts.StrictPermissions && len(report.Denied) > 0 {
		return nil, manifest, report, &PermissionDeniedError{ServiceID: manifest.ID, Denied: report.Denied}
	}

	host := buildHost(stored, accounts, opts)
	rt := service.NewRuntime(host, granted...)
	if err := rt.Load(ctx, manifest, source); err != nil {
		rt.Close()
		return nil, nil, report, sanitizeError(accounts, err)
	}
	return rt, manifest, report, nil
}

// HostPackage hosts a package that is not installed yet, for the command line:
// a directory the user pointed at. It uses the same host the installed path
// uses, with storage kept under the service's id.
func HostPackage(ctx context.Context, dataDir string, manifest *service.Manifest, source string) (*service.Runtime, error) {
	return HostPackageWithOptions(ctx, dataDir, manifest, source, HostOptions{})
}

// HostPackageWithOptions hosts an uninstalled package with explicit options.
func HostPackageWithOptions(ctx context.Context, dataDir string, manifest *service.Manifest, source string, opts HostOptions) (*service.Runtime, error) {
	stored, err := service.OpenStorage(dataDir, manifest.ID)
	if err != nil {
		return nil, err
	}
	accounts, err := OpenAccounts(dataDir, manifest.ID)
	if err != nil {
		// A package that is not installed has no account values; that is not a
		// reason to refuse running it, only to withhold secrets.
		accounts = nil
	}
	granted := opts.AllowedPermissions
	if granted == nil {
		granted = []service.Permission{service.PermissionNetwork, service.PermissionStorage}
		if accounts != nil {
			if count, err := accounts.Count(); err == nil && count > 0 {
				granted = append(granted, service.PermissionSecrets)
			}
		}
	}
	if opts.StrictPermissions {
		if _, err := RequirePermissions(manifest, granted); err != nil {
			return nil, err
		}
	}
	host := buildHost(stored, accounts, opts)
	rt := service.NewRuntime(host, granted...)
	if err := rt.Load(ctx, manifest, source); err != nil {
		rt.Close()
		return nil, sanitizeError(accounts, err)
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
	return NewSupervisorWithOptions(dataDir, HostOptions{UserAgent: userAgent})
}

// NewSupervisorWithOptions builds a supervisor with explicit permission
// trimming, bridge policy, log redaction and timeout/idle options.
func NewSupervisorWithOptions(dataDir string, opts HostOptions) (*service.Supervisor, error) {
	hostFor := func(id string) service.Host {
		stored, _ := service.OpenStorage(dataDir, id)
		accounts, _ := OpenAccounts(dataDir, id)
		return buildHost(stored, accounts, opts)
	}
	allowed := opts.AllowedPermissions
	if allowed == nil {
		allowed = []service.Permission{service.PermissionNetwork, service.PermissionStorage, service.PermissionSecrets}
	}
	return service.NewSupervisorFor(hostFor, allowed, opts.SupervisorOptions, func(ctx context.Context, id string) (*service.Manifest, string, error) {
		return service.ReadInstalledPackage(dataDir, id)
	}), nil
}

func buildHost(stored *service.Storage, accounts *Accounts, opts HostOptions) service.Host {
	host := service.Host{
		Fetch: buildFetch(opts.BridgePolicy, opts.UserAgent, accounts),
	}
	if stored != nil {
		host.StorageGet = stored.Get
		host.StorageSet = guardedStorageSet(stored, accounts)
	}
	if accounts != nil {
		host.Secret = accounts.Get
	}
	if opts.Log != nil {
		host.Log = redactedLog(accounts, opts.Log)
	}
	return host
}

// buildFetch wraps Core's Bridge policy (scheme, host allowlist, redirect
// validation, cookie stripping, response size ceiling and timeout) while also
// setting User-Agent and redacting any stored account secrets from errors.
func buildFetch(policy service.BridgePolicy, userAgent string, accounts *Accounts) func(context.Context, string) (string, error) {
	if policy.MaxBytes <= 0 {
		policy.MaxBytes = service.DefaultBridgeMaxBytes
	}
	if policy.Timeout <= 0 {
		policy.Timeout = service.DefaultBridgeTimeout
	}
	bridge := service.NewBridge(policy)
	client := &http.Client{
		Timeout: policy.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("service: too many redirects")
			}
			return bridge.Allows(req.URL.String())
		},
	}
	return func(ctx context.Context, rawURL string) (string, error) {
		if err := bridge.Allows(rawURL); err != nil {
			return "", sanitizeError(accounts, err)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return "", sanitizeError(accounts, err)
		}
		if strings.TrimSpace(userAgent) != "" {
			req.Header.Set("User-Agent", userAgent)
		}
		if !policy.AllowCookies {
			req.Header.Del("Cookie")
		}
		resp, err := client.Do(req)
		if err != nil {
			return "", sanitizeError(accounts, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return "", fmt.Errorf("service: %s returned HTTP %s", req.URL.Host, resp.Status)
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, policy.MaxBytes+1))
		if err != nil {
			return "", sanitizeError(accounts, err)
		}
		if int64(len(body)) > policy.MaxBytes {
			return "", fmt.Errorf("service: response is larger than %d bytes", policy.MaxBytes)
		}
		return string(body), nil
	}
}

// guardedStorageSet prevents a service from copying any secret from the
// isolated accounts store into ordinary KV storage (`storageSet`).
func guardedStorageSet(stored *service.Storage, accounts *Accounts) func(context.Context, string, string) error {
	return func(ctx context.Context, key, value string) error {
		if accounts != nil && value != "" {
			if snap, err := accounts.snapshot(); err == nil {
				for _, secretVal := range snap {
					if secretVal != "" && strings.Contains(value, secretVal) {
						return fmt.Errorf("%w (key=%q)", ErrSecretLeak, key)
					}
				}
			}
		}
		return stored.Set(ctx, key, value)
	}
}

// redactedLog scrubs any stored secret value before handing log text to the
// host logger.
func redactedLog(accounts *Accounts, sink func(level, message string)) func(level, message string) {
	return func(level, message string) {
		sink(level, redactString(accounts, message))
	}
}

func redactString(accounts *Accounts, text string) string {
	if accounts == nil || text == "" {
		return text
	}
	snap, err := accounts.snapshot()
	if err != nil || len(snap) == 0 {
		return text
	}
	vals := make([]string, 0, len(snap))
	for _, v := range snap {
		if v != "" {
			vals = append(vals, v)
		}
	}
	// Replace longest secrets first to avoid partial substring residue.
	sort.Slice(vals, func(i, j int) bool { return len(vals[i]) > len(vals[j]) })
	for _, v := range vals {
		text = strings.ReplaceAll(text, v, "[REDACTED]")
	}
	return text
}

func sanitizeError(accounts *Accounts, err error) error {
	if err == nil || accounts == nil {
		return err
	}
	msg := err.Error()
	redacted := redactString(accounts, msg)
	if redacted == msg {
		return err
	}
	return errors.New(redacted)
}
