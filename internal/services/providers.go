package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/laincat/veneracat-core/service"
)

const (
	// ServiceIDTrakt is the unified collection identifier for the Trakt service package.
	ServiceIDTrakt = "service.trakt"
	// ServiceIDTMDB is the unified collection identifier for the TMDB service package.
	ServiceIDTMDB = "service.tmdb"
	// ServiceIDBangumi is the unified collection identifier for the Bangumi service package.
	ServiceIDBangumi = "service.bangumi"
)

const (
	// SecretKeyAccessToken is the isolated account key used by Trakt and Bangumi.
	SecretKeyAccessToken = "access_token"
	// SecretKeyAPIReadToken is the isolated account key used by TMDB.
	SecretKeyAPIReadToken = "api_read_token"
)

// ProviderSpec describes the known network origins and auth secret keys of a
// first-class VeneraCat service package.
type ProviderSpec struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	DefaultHosts    []string `json:"defaultHosts"`
	RequiredSecrets []string `json:"requiredSecrets"`
}

var knownSpecs = map[string]ProviderSpec{
	ServiceIDTrakt: {
		ID:              ServiceIDTrakt,
		Name:            "Trakt",
		DefaultHosts:    []string{"api.trakt.tv", "trakt.tv"},
		RequiredSecrets: []string{SecretKeyAccessToken},
	},
	ServiceIDTMDB: {
		ID:              ServiceIDTMDB,
		Name:            "TMDB",
		DefaultHosts:    []string{"api.themoviedb.org", "image.tmdb.org", "www.themoviedb.org"},
		RequiredSecrets: []string{SecretKeyAPIReadToken},
	},
	ServiceIDBangumi: {
		ID:              ServiceIDBangumi,
		Name:            "Bangumi",
		DefaultHosts:    []string{"api.bgm.tv", "bgm.tv", "next.bgm.tv"},
		RequiredSecrets: []string{SecretKeyAccessToken},
	},
}

// SpecFor returns the built-in specification for Trakt, TMDB or Bangumi.
func SpecFor(id string) (ProviderSpec, bool) {
	spec, ok := knownSpecs[id]
	if !ok {
		return ProviderSpec{}, false
	}
	spec.DefaultHosts = slices.Clone(spec.DefaultHosts)
	spec.RequiredSecrets = slices.Clone(spec.RequiredSecrets)
	return spec, true
}

// DefaultHosts returns the official allowed network hosts for a known service.
func DefaultHosts(id string) []string {
	spec, ok := SpecFor(id)
	if !ok {
		return nil
	}
	return spec.DefaultHosts
}

// AccountProfile is the normalized response of `account.me`.
type AccountProfile struct {
	ID          string          `json:"id"`
	Username    string          `json:"username"`
	DisplayName string          `json:"displayName,omitempty"`
	AvatarURL   string          `json:"avatarUrl,omitempty"`
	Provider    string          `json:"provider,omitempty"`
	Raw         json.RawMessage `json:"raw,omitempty"`
}

// MetadataQuery is the request payload for `metadata.search`.
type MetadataQuery struct {
	Query     string `json:"query"`
	MediaType string `json:"mediaType,omitempty"`
	Language  string `json:"language,omitempty"`
	Page      int    `json:"page,omitempty"`
}

// MetadataItem is one search hit or base metadata record.
type MetadataItem struct {
	ID            string            `json:"id"`
	MediaType     string            `json:"mediaType,omitempty"`
	Title         string            `json:"title"`
	OriginalTitle string            `json:"originalTitle,omitempty"`
	Year          int               `json:"year,omitempty"`
	Overview      string            `json:"overview,omitempty"`
	PosterURL     string            `json:"posterUrl,omitempty"`
	Rating        float64           `json:"rating,omitempty"`
	ExternalIDs   map[string]string `json:"externalIds,omitempty"`
}

// MetadataSearchResult is the response of `metadata.search`.
type MetadataSearchResult struct {
	Page         int            `json:"page,omitempty"`
	TotalPages   int            `json:"totalPages,omitempty"`
	TotalResults int            `json:"totalResults,omitempty"`
	Items        []MetadataItem `json:"items"`
}

// MetadataDetailRequest is the request payload for `metadata.detail`.
type MetadataDetailRequest struct {
	ID        string `json:"id"`
	MediaType string `json:"mediaType,omitempty"`
	Language  string `json:"language,omitempty"`
}

// MetadataDetail is the response of `metadata.detail`.
type MetadataDetail struct {
	MetadataItem
	Genres   []string        `json:"genres,omitempty"`
	Status   string          `json:"status,omitempty"`
	Episodes int             `json:"episodes,omitempty"`
	Seasons  int             `json:"seasons,omitempty"`
	AirDate  string          `json:"airDate,omitempty"`
	Raw      json.RawMessage `json:"raw,omitempty"`
}

// CollectionEntry is one tracked entry in a user's collection (e.g. Bangumi).
type CollectionEntry struct {
	ID        string  `json:"id"`
	MediaType string  `json:"mediaType,omitempty"`
	Title     string  `json:"title"`
	Status    string  `json:"status"`
	Progress  int     `json:"progress,omitempty"`
	Total     int     `json:"total,omitempty"`
	Score     float64 `json:"score,omitempty"`
	UpdatedAt string  `json:"updatedAt,omitempty"`
}

// CollectionListRequest is the request payload for `collection.list`.
type CollectionListRequest struct {
	MediaType string `json:"mediaType,omitempty"`
	Status    string `json:"status,omitempty"`
	Page      int    `json:"page,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

// CollectionListResult is the response of `collection.list`.
type CollectionListResult struct {
	Items []CollectionEntry `json:"items"`
	Total int               `json:"total,omitempty"`
}

// CollectionUpdateRequest is the request payload for `collection.update`.
type CollectionUpdateRequest struct {
	ID        string  `json:"id"`
	MediaType string  `json:"mediaType,omitempty"`
	Status    string  `json:"status"`
	Progress  int     `json:"progress,omitempty"`
	Score     float64 `json:"score,omitempty"`
}

// CollectionUpdateResult is the response of `collection.update`.
type CollectionUpdateResult struct {
	OK       bool   `json:"ok"`
	ID       string `json:"id"`
	Status   string `json:"status,omitempty"`
	Progress int    `json:"progress,omitempty"`
}

// WatchlistItem is one item in a user's watchlist (e.g. Trakt / TMDB).
type WatchlistItem struct {
	ID        string `json:"id"`
	MediaType string `json:"mediaType,omitempty"`
	Title     string `json:"title"`
	AddedAt   string `json:"addedAt,omitempty"`
}

// WatchlistRequest is the request payload for `watchlist.list`.
type WatchlistRequest struct {
	MediaType string `json:"mediaType,omitempty"`
	Page      int    `json:"page,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

// WatchlistResult is the response of `watchlist.list`.
type WatchlistResult struct {
	Items []WatchlistItem `json:"items"`
	Total int             `json:"total,omitempty"`
}

// WatchlistMutationRequest is the request payload for `watchlist.update`.
type WatchlistMutationRequest struct {
	ID        string `json:"id"`
	MediaType string `json:"mediaType,omitempty"`
	Action    string `json:"action"` // "add" or "remove"
}

// WatchlistMutationResult is the response of `watchlist.update`.
type WatchlistMutationResult struct {
	OK     bool   `json:"ok"`
	ID     string `json:"id"`
	Action string `json:"action,omitempty"`
}

// ScrobbleRequest is the request payload for `scrobble`.
type ScrobbleRequest struct {
	ID        string  `json:"id"`
	MediaType string  `json:"mediaType,omitempty"`
	Action    string  `json:"action"` // "start", "pause", or "stop"
	Season    int     `json:"season,omitempty"`
	Episode   int     `json:"episode,omitempty"`
	Progress  float64 `json:"progress"`
}

// ScrobbleResult is the response of `scrobble`.
type ScrobbleResult struct {
	OK         bool    `json:"ok"`
	ID         string  `json:"id"`
	Action     string  `json:"action,omitempty"`
	Progress   float64 `json:"progress,omitempty"`
	RecordedAt string  `json:"recordedAt,omitempty"`
}

// Client is a typed, supervised client bound to one installed service package.
//
// It enforces permission checks, isolated account secret checks, secret
// redaction, and request cancellation/timeouts through Core's Supervisor.
type Client struct {
	dataDir    string
	serviceID  string
	opts       HostOptions
	supervisor *service.Supervisor
}

// NewClient constructs a supervised client for an installed service ID.
func NewClient(dataDir, serviceID string, opts HostOptions) (*Client, error) {
	if err := validServiceID(serviceID); err != nil {
		return nil, err
	}
	if _, err := InstalledDir(dataDir, serviceID); err != nil {
		return nil, err
	}
	sup, err := NewSupervisorWithOptions(dataDir, opts)
	if err != nil {
		return nil, err
	}
	return &Client{
		dataDir:    dataDir,
		serviceID:  serviceID,
		opts:       opts,
		supervisor: sup,
	}, nil
}

// Close stops and unloads the underlying service supervisor.
func (c *Client) Close() {
	if c.supervisor != nil {
		c.supervisor.Close()
	}
}

// ServiceID returns the bound service identifier.
func (c *Client) ServiceID() string { return c.serviceID }

// Manifest reads the installed service manifest from disk.
func (c *Client) Manifest() (*service.Manifest, error) {
	m, _, err := service.ReadInstalledPackage(c.dataDir, c.serviceID)
	return m, err
}

// Permissions inspects the service's requested, granted and denied permissions
// under this client's HostOptions.
func (c *Client) Permissions() (PermissionReport, error) {
	m, err := c.Manifest()
	if err != nil {
		return PermissionReport{}, err
	}
	return EvaluatePermissions(m, c.effectiveHostAllowed())
}

func (c *Client) effectiveHostAllowed() []service.Permission {
	if c.opts.AllowedPermissions != nil {
		return c.opts.AllowedPermissions
	}
	return []service.Permission{service.PermissionNetwork, service.PermissionStorage, service.PermissionSecrets}
}

// RequirePermissions ensures every permission in required is both declared by
// the service manifest and granted by the host. Any missing permission is
// returned in *PermissionDeniedError.
func (c *Client) RequirePermissions(required ...service.Permission) (PermissionReport, error) {
	m, err := c.Manifest()
	if err != nil {
		return PermissionReport{}, err
	}
	allowed := c.effectiveHostAllowed()
	grants, err := service.Grant(m, allowed)
	if err != nil {
		return PermissionReport{}, err
	}
	report, err := EvaluatePermissions(m, allowed)
	if err != nil {
		return PermissionReport{}, err
	}
	var denied []service.Permission
	for _, p := range required {
		if !grants.Allows(p) {
			denied = append(denied, p)
		}
	}
	if len(denied) > 0 {
		sort.Slice(denied, func(i, j int) bool { return denied[i] < denied[j] })
		report.Denied = mergeUniquePermissions(report.Denied, denied)
		return report, &PermissionDeniedError{ServiceID: c.serviceID, Denied: denied}
	}
	return report, nil
}

// RequireSecrets verifies that every key in requiredKeys is stored in the
// service's isolated account store. Missing keys are reported in
// *MissingSecretsError without ever exposing values.
func (c *Client) RequireSecrets(requiredKeys ...string) error {
	if len(requiredKeys) == 0 {
		return nil
	}
	accounts, err := OpenAccounts(c.dataDir, c.serviceID)
	if err != nil {
		return err
	}
	missing, err := accounts.Missing(requiredKeys...)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return &MissingSecretsError{ServiceID: c.serviceID, Keys: missing}
	}
	return nil
}

// SetAccount stores one secret key/value in the service's isolated account store.
func (c *Client) SetAccount(ctx context.Context, key, value string) error {
	accounts, err := OpenAccounts(c.dataDir, c.serviceID)
	if err != nil {
		return err
	}
	return accounts.Set(ctx, key, value)
}

// AccountKeys returns stored account key names (never values).
func (c *Client) AccountKeys() ([]string, error) {
	accounts, err := OpenAccounts(c.dataDir, c.serviceID)
	if err != nil {
		return nil, err
	}
	return accounts.Keys()
}

// RemoveAccount removes one secret key from the service's isolated account store.
func (c *Client) RemoveAccount(ctx context.Context, key string) error {
	accounts, err := OpenAccounts(c.dataDir, c.serviceID)
	if err != nil {
		return err
	}
	return accounts.Remove(ctx, key)
}

// Describe calls the service's exported `describe()` function.
func (c *Client) Describe(ctx context.Context) ([]service.Method, error) {
	if c.opts.StrictPermissions {
		m, err := c.Manifest()
		if err != nil {
			return nil, err
		}
		if _, err := RequirePermissions(m, c.effectiveHostAllowed()); err != nil {
			return nil, err
		}
	}
	methods, err := c.supervisor.Describe(ctx, c.serviceID)
	if err != nil {
		return nil, c.sanitize(err)
	}
	return methods, nil
}

// Call invokes an arbitrary method on the service with raw JSON payload.
func (c *Client) Call(ctx context.Context, method string, payload json.RawMessage) (json.RawMessage, error) {
	if strings.TrimSpace(method) == "" {
		return nil, errors.New("service: method name is required")
	}
	if c.opts.StrictPermissions {
		m, err := c.Manifest()
		if err != nil {
			return nil, err
		}
		if _, err := RequirePermissions(m, c.effectiveHostAllowed()); err != nil {
			return nil, err
		}
	}
	raw, err := c.supervisor.Call(ctx, c.serviceID, method, payload)
	if err != nil {
		return nil, c.sanitize(err)
	}
	return raw, nil
}

// CallJSON enforces required permissions and isolated account keys, marshals
// req to JSON, calls method, and decodes the JSON response into resp.
func (c *Client) CallJSON(ctx context.Context, method string, req any, resp any, requiredPerms []service.Permission, requiredSecrets []string) error {
	if len(requiredPerms) > 0 {
		if _, err := c.RequirePermissions(requiredPerms...); err != nil {
			return err
		}
	}
	if err := c.RequireSecrets(requiredSecrets...); err != nil {
		return err
	}
	var payload json.RawMessage
	if req != nil {
		b, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("service: encode %s payload: %w", method, err)
		}
		payload = b
	}
	out, err := c.Call(ctx, method, payload)
	if err != nil {
		return err
	}
	if resp == nil {
		return nil
	}
	if err := json.Unmarshal(out, resp); err != nil {
		return c.sanitize(fmt.Errorf("service: decode %s response: %w", method, err))
	}
	return nil
}

func (c *Client) sanitize(err error) error {
	accounts, openErr := OpenAccounts(c.dataDir, c.serviceID)
	if openErr != nil {
		return err
	}
	return sanitizeError(accounts, err)
}

func mergeUniquePermissions(a, b []service.Permission) []service.Permission {
	seen := map[service.Permission]bool{}
	out := make([]service.Permission, 0, len(a)+len(b))
	for _, p := range append(append([]service.Permission(nil), a...), b...) {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// TraktClient provides typed operations against an installed `service.trakt` package.
type TraktClient struct {
	*Client
}

// NewTraktClient binds a typed adapter to `service.trakt`.
func NewTraktClient(dataDir string, opts HostOptions) (*TraktClient, error) {
	c, err := NewClient(dataDir, ServiceIDTrakt, opts)
	if err != nil {
		return nil, err
	}
	return &TraktClient{Client: c}, nil
}

func (t *TraktClient) AccountMe(ctx context.Context) (*AccountProfile, error) {
	var out AccountProfile
	err := t.CallJSON(ctx, "account.me", nil, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		[]string{SecretKeyAccessToken})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *TraktClient) MetadataSearch(ctx context.Context, q MetadataQuery) (*MetadataSearchResult, error) {
	var out MetadataSearchResult
	err := t.CallJSON(ctx, "metadata.search", q, &out,
		[]service.Permission{service.PermissionNetwork}, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *TraktClient) MetadataDetail(ctx context.Context, req MetadataDetailRequest) (*MetadataDetail, error) {
	var out MetadataDetail
	err := t.CallJSON(ctx, "metadata.detail", req, &out,
		[]service.Permission{service.PermissionNetwork}, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *TraktClient) Watchlist(ctx context.Context, req WatchlistRequest) (*WatchlistResult, error) {
	var out WatchlistResult
	err := t.CallJSON(ctx, "watchlist.list", req, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		[]string{SecretKeyAccessToken})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *TraktClient) UpdateWatchlist(ctx context.Context, req WatchlistMutationRequest) (*WatchlistMutationResult, error) {
	var out WatchlistMutationResult
	err := t.CallJSON(ctx, "watchlist.update", req, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		[]string{SecretKeyAccessToken})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *TraktClient) Scrobble(ctx context.Context, req ScrobbleRequest) (*ScrobbleResult, error) {
	var out ScrobbleResult
	err := t.CallJSON(ctx, "scrobble", req, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		[]string{SecretKeyAccessToken})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// TMDBClient provides typed operations against an installed `service.tmdb` package.
type TMDBClient struct {
	*Client
}

// NewTMDBClient binds a typed adapter to `service.tmdb`.
func NewTMDBClient(dataDir string, opts HostOptions) (*TMDBClient, error) {
	c, err := NewClient(dataDir, ServiceIDTMDB, opts)
	if err != nil {
		return nil, err
	}
	return &TMDBClient{Client: c}, nil
}

func (t *TMDBClient) AccountMe(ctx context.Context) (*AccountProfile, error) {
	var out AccountProfile
	err := t.CallJSON(ctx, "account.me", nil, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		[]string{SecretKeyAPIReadToken})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *TMDBClient) MetadataSearch(ctx context.Context, q MetadataQuery) (*MetadataSearchResult, error) {
	var out MetadataSearchResult
	err := t.CallJSON(ctx, "metadata.search", q, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		[]string{SecretKeyAPIReadToken})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *TMDBClient) MetadataDetail(ctx context.Context, req MetadataDetailRequest) (*MetadataDetail, error) {
	var out MetadataDetail
	err := t.CallJSON(ctx, "metadata.detail", req, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		[]string{SecretKeyAPIReadToken})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *TMDBClient) Watchlist(ctx context.Context, req WatchlistRequest) (*WatchlistResult, error) {
	var out WatchlistResult
	err := t.CallJSON(ctx, "watchlist.list", req, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		[]string{SecretKeyAPIReadToken})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// BangumiClient provides typed operations against an installed `service.bangumi` package.
type BangumiClient struct {
	*Client
}

// NewBangumiClient binds a typed adapter to `service.bangumi`.
func NewBangumiClient(dataDir string, opts HostOptions) (*BangumiClient, error) {
	c, err := NewClient(dataDir, ServiceIDBangumi, opts)
	if err != nil {
		return nil, err
	}
	return &BangumiClient{Client: c}, nil
}

func (b *BangumiClient) AccountMe(ctx context.Context) (*AccountProfile, error) {
	var out AccountProfile
	err := b.CallJSON(ctx, "account.me", nil, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		[]string{SecretKeyAccessToken})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (b *BangumiClient) MetadataSearch(ctx context.Context, q MetadataQuery) (*MetadataSearchResult, error) {
	var out MetadataSearchResult
	err := b.CallJSON(ctx, "metadata.search", q, &out,
		[]service.Permission{service.PermissionNetwork}, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (b *BangumiClient) MetadataDetail(ctx context.Context, req MetadataDetailRequest) (*MetadataDetail, error) {
	var out MetadataDetail
	err := b.CallJSON(ctx, "metadata.detail", req, &out,
		[]service.Permission{service.PermissionNetwork}, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (b *BangumiClient) Collection(ctx context.Context, req CollectionListRequest) (*CollectionListResult, error) {
	var out CollectionListResult
	err := b.CallJSON(ctx, "collection.list", req, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		[]string{SecretKeyAccessToken})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (b *BangumiClient) UpdateCollection(ctx context.Context, req CollectionUpdateRequest) (*CollectionUpdateResult, error) {
	var out CollectionUpdateResult
	err := b.CallJSON(ctx, "collection.update", req, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		[]string{SecretKeyAccessToken})
	if err != nil {
		return nil, err
	}
	return &out, nil
}
