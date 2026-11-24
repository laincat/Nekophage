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
	// SecretKeyClientID is Trakt's application client ID stored in isolated accounts.
	SecretKeyClientID = "client_id"
	// SecretKeyAccessToken is the OAuth/PAT token used by Trakt and Bangumi.
	SecretKeyAccessToken = "access_token"
	// SecretKeyAPIReadToken is TMDB's v4/v3 Bearer read access token.
	SecretKeyAPIReadToken = "api_read_token"
	// SecretKeyAPIKey is TMDB's v3 query-parameter API key alternative.
	SecretKeyAPIKey = "api_key"
)

// ProviderSpec describes the known network origins and auth secret keys of a
// first-class VeneraCat service package.
type ProviderSpec struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	DefaultHosts    []string `json:"defaultHosts"`
	RequiredSecrets []string `json:"requiredSecrets"`
	// AnyOfSecrets lists alternative secret keys where any single non-empty
	// key satisfies authentication (e.g. TMDB api_read_token or api_key).
	AnyOfSecrets []string `json:"anyOfSecrets,omitempty"`
}

var knownSpecs = map[string]ProviderSpec{
	ServiceIDTrakt: {
		ID:              ServiceIDTrakt,
		Name:            "Trakt",
		DefaultHosts:    []string{"api.trakt.tv", "trakt.tv"},
		RequiredSecrets: []string{SecretKeyClientID, SecretKeyAccessToken},
	},
	ServiceIDTMDB: {
		ID:           ServiceIDTMDB,
		Name:         "TMDB",
		DefaultHosts: []string{"api.themoviedb.org", "image.tmdb.org", "www.themoviedb.org"},
		AnyOfSecrets: []string{SecretKeyAPIReadToken, SecretKeyAPIKey},
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
	spec.AnyOfSecrets = slices.Clone(spec.AnyOfSecrets)
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

// ServiceError represents a structured `{ok:false, error:{...}}` envelope
// returned by a VeneraCat service plugin.
type ServiceError struct {
	Service        string   `json:"service,omitempty"`
	Method         string   `json:"method,omitempty"`
	Code           string   `json:"code"`
	Message        string   `json:"message"`
	Status         int      `json:"status,omitempty"`
	MissingSecrets []string `json:"missingSecrets,omitempty"`
}

func (e *ServiceError) Error() string {
	if e.Service != "" && e.Method != "" {
		return fmt.Sprintf("%s %s [%s]: %s", e.Service, e.Method, e.Code, e.Message)
	}
	return fmt.Sprintf("[%s]: %s", e.Code, e.Message)
}

func (e *ServiceError) Unwrap() error {
	switch e.Code {
	case "MISSING_SECRET":
		return ErrMissingSecrets
	case "SECRET_UNAVAILABLE":
		return ErrPermissionDenied
	default:
		return nil
	}
}

type serviceEnvelope struct {
	OK      *bool           `json:"ok"`
	Service string          `json:"service,omitempty"`
	Method  string          `json:"method,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   *ServiceError   `json:"error,omitempty"`
}

// --- Trakt (`service.trakt`) typed request/response models ---

type TraktAccount struct {
	Username    string          `json:"username"`
	Slug        string          `json:"slug"`
	Name        string          `json:"name,omitempty"`
	Private     bool            `json:"private"`
	VIP         bool            `json:"vip"`
	VIPEp       bool            `json:"vipEp"`
	Location    string          `json:"location,omitempty"`
	Avatar      string          `json:"avatar,omitempty"`
	Account     json.RawMessage `json:"account,omitempty"`
	Connections json.RawMessage `json:"connections,omitempty"`
}

type TraktSearchRequest struct {
	Query    string `json:"query"`
	Type     string `json:"type,omitempty"` // movie, show, episode, person, list (comma-separated)
	Page     int    `json:"page,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Extended string `json:"extended,omitempty"`
}

type TraktSearchItem struct {
	Type     string          `json:"type"`
	Score    *float64        `json:"score,omitempty"`
	Title    string          `json:"title"`
	Year     *int            `json:"year,omitempty"`
	IDs      json.RawMessage `json:"ids,omitempty"`
	Overview string          `json:"overview,omitempty"`
	Runtime  *int            `json:"runtime,omitempty"`
	Show     json.RawMessage `json:"show,omitempty"`
}

type TraktSearchResult struct {
	Query string            `json:"query"`
	Type  string            `json:"type"`
	Page  int               `json:"page"`
	Limit int               `json:"limit"`
	Items []TraktSearchItem `json:"items"`
}

type TraktWatchlistRequest struct {
	Type     string `json:"type,omitempty"` // movies, shows, seasons, episodes
	Sort     string `json:"sort,omitempty"` // rank, added, released, title
	Page     int    `json:"page,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Extended string `json:"extended,omitempty"`
}

type TraktWatchlistItem struct {
	ID       int64           `json:"id"`
	Rank     *int            `json:"rank,omitempty"`
	ListedAt string          `json:"listedAt,omitempty"`
	Type     string          `json:"type"`
	Title    string          `json:"title"`
	Year     *int            `json:"year,omitempty"`
	IDs      json.RawMessage `json:"ids,omitempty"`
	Notes    string          `json:"notes,omitempty"`
	Show     json.RawMessage `json:"show,omitempty"`
}

type TraktWatchlistResult struct {
	Type  string               `json:"type"`
	Page  int                  `json:"page"`
	Limit int                  `json:"limit"`
	Items []TraktWatchlistItem `json:"items"`
}

type TraktHistoryRequest struct {
	Type     string `json:"type,omitempty"` // movies, shows, seasons, episodes
	ID       string `json:"id,omitempty"`
	Page     int    `json:"page,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	StartAt  string `json:"startAt,omitempty"`
	EndAt    string `json:"endAt,omitempty"`
	Extended string `json:"extended,omitempty"`
}

type TraktHistoryItem struct {
	ID        int64           `json:"id"`
	WatchedAt string          `json:"watchedAt,omitempty"`
	Action    string          `json:"action,omitempty"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Year      *int            `json:"year,omitempty"`
	IDs       json.RawMessage `json:"ids,omitempty"`
	Episode   json.RawMessage `json:"episode,omitempty"`
	Show      json.RawMessage `json:"show,omitempty"`
}

type TraktHistoryResult struct {
	Type  string             `json:"type"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
	Items []TraktHistoryItem `json:"items"`
}

type TraktScrobbleRequest struct {
	Progress   float64 `json:"progress"`
	Movie      any     `json:"movie,omitempty"`
	Episode    any     `json:"episode,omitempty"`
	Show       any     `json:"show,omitempty"`
	AppVersion string  `json:"appVersion,omitempty"`
	AppDate    string  `json:"appDate,omitempty"`
}

type TraktScrobbleResult struct {
	ID       int64           `json:"id"`
	Action   string          `json:"action"`
	Progress float64         `json:"progress"`
	Sharing  json.RawMessage `json:"sharing,omitempty"`
	Movie    json.RawMessage `json:"movie,omitempty"`
	Show     json.RawMessage `json:"show,omitempty"`
	Episode  json.RawMessage `json:"episode,omitempty"`
}

// --- TMDB (`service.tmdb`) typed request/response models ---

type TMDBSearchRequest struct {
	Query        string `json:"query"`
	MediaType    string `json:"mediaType,omitempty"` // multi, movie, tv, person
	Page         int    `json:"page,omitempty"`
	Language     string `json:"language,omitempty"`
	IncludeAdult bool   `json:"includeAdult,omitempty"`
	Year         int    `json:"year,omitempty"`
}

type TMDBSearchItem struct {
	ID            int64    `json:"id"`
	MediaType     string   `json:"mediaType"`
	Title         string   `json:"title"`
	OriginalTitle string   `json:"originalTitle,omitempty"`
	Overview      string   `json:"overview,omitempty"`
	ReleaseDate   string   `json:"releaseDate,omitempty"`
	VoteAverage   *float64 `json:"voteAverage,omitempty"`
	VoteCount     *int     `json:"voteCount,omitempty"`
	Popularity    *float64 `json:"popularity,omitempty"`
	PosterPath    *string  `json:"posterPath,omitempty"`
	PosterURL     *string  `json:"posterUrl,omitempty"`
	BackdropPath  *string  `json:"backdropPath,omitempty"`
	BackdropURL   *string  `json:"backdropUrl,omitempty"`
	ProfilePath   *string  `json:"profilePath,omitempty"`
	ProfileURL    *string  `json:"profileUrl,omitempty"`
	GenreIDs      []int    `json:"genreIds,omitempty"`
}

type TMDBSearchResult struct {
	Query        string           `json:"query"`
	MediaType    string           `json:"mediaType"`
	Language     string           `json:"language"`
	Page         int              `json:"page"`
	TotalPages   int              `json:"totalPages"`
	TotalResults int              `json:"totalResults"`
	Items        []TMDBSearchItem `json:"items"`
}

type TMDBDetailRequest struct {
	ID               int64  `json:"id"`
	MediaType        string `json:"mediaType,omitempty"` // movie, tv
	Language         string `json:"language,omitempty"`
	SeasonNumber     *int   `json:"seasonNumber,omitempty"`
	EpisodeNumber    *int   `json:"episodeNumber,omitempty"`
	AppendToResponse string `json:"appendToResponse,omitempty"`
}

type TMDBGenre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TMDBSeasonSummary struct {
	ID           int64   `json:"id"`
	SeasonNumber int     `json:"seasonNumber"`
	Name         string  `json:"name"`
	EpisodeCount *int    `json:"episodeCount,omitempty"`
	AirDate      string  `json:"airDate,omitempty"`
	PosterPath   *string `json:"posterPath,omitempty"`
	PosterURL    *string `json:"posterUrl,omitempty"`
}

type TMDBEpisodeSummary struct {
	ID            int64    `json:"id"`
	SeasonNumber  int      `json:"seasonNumber"`
	EpisodeNumber int      `json:"episodeNumber"`
	Name          string   `json:"name"`
	Overview      string   `json:"overview,omitempty"`
	AirDate       string   `json:"airDate,omitempty"`
	Runtime       *int     `json:"runtime,omitempty"`
	VoteAverage   *float64 `json:"voteAverage,omitempty"`
	StillPath     *string  `json:"stillPath,omitempty"`
	StillURL      *string  `json:"stillUrl,omitempty"`
}

type TMDBDetailResult struct {
	ID               int64                `json:"id"`
	TVID             int64                `json:"tvId,omitempty"`
	MediaType        string               `json:"mediaType"`
	Title            string               `json:"title,omitempty"`
	Name             string               `json:"name,omitempty"`
	OriginalTitle    string               `json:"originalTitle,omitempty"`
	Tagline          string               `json:"tagline,omitempty"`
	Overview         string               `json:"overview,omitempty"`
	Status           string               `json:"status,omitempty"`
	ReleaseDate      string               `json:"releaseDate,omitempty"`
	AirDate          string               `json:"airDate,omitempty"`
	LastAirDate      string               `json:"lastAirDate,omitempty"`
	Runtime          *int                 `json:"runtime,omitempty"`
	SeasonNumber     *int                 `json:"seasonNumber,omitempty"`
	EpisodeNumber    *int                 `json:"episodeNumber,omitempty"`
	NumberOfSeasons  *int                 `json:"numberOfSeasons,omitempty"`
	NumberOfEpisodes *int                 `json:"numberOfEpisodes,omitempty"`
	VoteAverage      *float64             `json:"voteAverage,omitempty"`
	VoteCount        *int                 `json:"voteCount,omitempty"`
	Genres           []TMDBGenre          `json:"genres,omitempty"`
	PosterPath       *string              `json:"posterPath,omitempty"`
	PosterURL        *string              `json:"posterUrl,omitempty"`
	BackdropPath     *string              `json:"backdropPath,omitempty"`
	BackdropURL      *string              `json:"backdropUrl,omitempty"`
	StillPath        *string              `json:"stillPath,omitempty"`
	StillURL         *string              `json:"stillUrl,omitempty"`
	Seasons          []TMDBSeasonSummary  `json:"seasons,omitempty"`
	Episodes         []TMDBEpisodeSummary `json:"episodes,omitempty"`
}

type TMDBArtworkRequest struct {
	ID                   int64  `json:"id"`
	MediaType            string `json:"mediaType,omitempty"` // movie, tv
	SeasonNumber         *int   `json:"seasonNumber,omitempty"`
	EpisodeNumber        *int   `json:"episodeNumber,omitempty"`
	IncludeImageLanguage string `json:"includeImageLanguage,omitempty"`
	Size                 string `json:"size,omitempty"`
	PreviewSize          string `json:"previewSize,omitempty"`
}

type TMDBArtworkItem struct {
	FilePath    string   `json:"filePath"`
	URL         *string  `json:"url,omitempty"`
	PreviewURL  *string  `json:"previewUrl,omitempty"`
	Width       *int     `json:"width,omitempty"`
	Height      *int     `json:"height,omitempty"`
	AspectRatio *float64 `json:"aspectRatio,omitempty"`
	ISO6391     *string  `json:"iso6391,omitempty"`
	VoteAverage *float64 `json:"voteAverage,omitempty"`
	VoteCount   *int     `json:"voteCount,omitempty"`
}

type TMDBArtworkResult struct {
	ID        int64             `json:"id"`
	MediaType string            `json:"mediaType"`
	BaseURL   string            `json:"baseUrl"`
	Posters   []TMDBArtworkItem `json:"posters"`
	Backdrops []TMDBArtworkItem `json:"backdrops"`
	Logos     []TMDBArtworkItem `json:"logos"`
	Stills    []TMDBArtworkItem `json:"stills"`
}

// --- Bangumi (`service.bangumi`) typed request/response models ---

type BangumiAccount struct {
	ID        int64             `json:"id"`
	Username  string            `json:"username"`
	Nickname  string            `json:"nickname,omitempty"`
	Sign      string            `json:"sign,omitempty"`
	UserGroup int               `json:"userGroup,omitempty"`
	Avatar    map[string]string `json:"avatar,omitempty"`
	URL       string            `json:"url,omitempty"`
}

type BangumiSearchRequest struct {
	Keyword       string         `json:"keyword,omitempty"`
	Query         string         `json:"query,omitempty"`
	Type          any            `json:"type,omitempty"` // int, string ("anime", "book", ...), or slice
	Sort          string         `json:"sort,omitempty"`
	Limit         int            `json:"limit,omitempty"`
	Offset        int            `json:"offset,omitempty"`
	Filter        map[string]any `json:"filter,omitempty"`
	Legacy        bool           `json:"legacy,omitempty"`
	ResponseGroup string         `json:"responseGroup,omitempty"`
	UserAgent     string         `json:"userAgent,omitempty"`
}

type BangumiSubject struct {
	ID      int64           `json:"id"`
	Type    int             `json:"type"`
	Name    string          `json:"name"`
	NameCN  string          `json:"nameCn,omitempty"`
	Summary string          `json:"summary,omitempty"`
	Date    string          `json:"date,omitempty"`
	Score   *float64        `json:"score,omitempty"`
	Rank    *int            `json:"rank,omitempty"`
	Eps     *int            `json:"eps,omitempty"`
	Images  json.RawMessage `json:"images,omitempty"`
	URL     string          `json:"url,omitempty"`
}

type BangumiSearchResult struct {
	Keyword        string           `json:"keyword"`
	Items          []BangumiSubject `json:"items"`
	Total          int              `json:"total"`
	Limit          int              `json:"limit"`
	Offset         int              `json:"offset"`
	FallbackUsed   bool             `json:"fallbackUsed"`
	FallbackReason string           `json:"fallbackReason,omitempty"`
}

type BangumiDetailRequest struct {
	SubjectID       int64  `json:"subjectId"`
	IncludeEpisodes bool   `json:"includeEpisodes,omitempty"`
	EpisodeType     *int   `json:"episodeType,omitempty"`
	EpisodeLimit    int    `json:"episodeLimit,omitempty"`
	EpisodeOffset   int    `json:"episodeOffset,omitempty"`
	UserAgent       string `json:"userAgent,omitempty"`
}

type BangumiEpisode struct {
	ID       int64   `json:"id"`
	Type     int     `json:"type"`
	Sort     float64 `json:"sort"`
	Ep       float64 `json:"ep"`
	Name     string  `json:"name"`
	NameCN   string  `json:"nameCn,omitempty"`
	Airdate  string  `json:"airdate,omitempty"`
	Duration string  `json:"duration,omitempty"`
}

type BangumiSubjectDetail struct {
	ID            int64            `json:"id"`
	Type          int              `json:"type"`
	Name          string           `json:"name"`
	NameCN        string           `json:"nameCn,omitempty"`
	Summary       string           `json:"summary,omitempty"`
	Date          string           `json:"date,omitempty"`
	Platform      string           `json:"platform,omitempty"`
	Eps           *int             `json:"eps,omitempty"`
	TotalEpisodes *int             `json:"totalEpisodes,omitempty"`
	Volumes       *int             `json:"volumes,omitempty"`
	Rating        json.RawMessage  `json:"rating,omitempty"`
	Tags          json.RawMessage  `json:"tags,omitempty"`
	Infobox       json.RawMessage  `json:"infobox,omitempty"`
	Images        json.RawMessage  `json:"images,omitempty"`
	Collection    json.RawMessage  `json:"collection,omitempty"`
	URL           string           `json:"url,omitempty"`
	Episodes      []BangumiEpisode `json:"episodes,omitempty"`
}

type BangumiCollectionItem struct {
	SubjectID   int64           `json:"subjectId"`
	SubjectType int             `json:"subjectType"`
	Type        int             `json:"type"`
	Rate        int             `json:"rate"`
	EpStatus    int             `json:"epStatus"`
	VolStatus   int             `json:"volStatus"`
	Comment     string          `json:"comment,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
	Private     bool            `json:"private"`
	UpdatedAt   string          `json:"updatedAt,omitempty"`
	Subject     *BangumiSubject `json:"subject,omitempty"`
}

type BangumiCollectionGetRequest struct {
	Username    string `json:"username,omitempty"`
	SubjectID   int64  `json:"subjectId,omitempty"`
	SubjectType any    `json:"subjectType,omitempty"`
	Type        any    `json:"type,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty"`
	UserAgent   string `json:"userAgent,omitempty"`
}

type BangumiCollectionGetResult struct {
	Username string                  `json:"username"`
	Item     *BangumiCollectionItem  `json:"item,omitempty"`
	Items    []BangumiCollectionItem `json:"items,omitempty"`
	Total    int                     `json:"total,omitempty"`
	Limit    int                     `json:"limit,omitempty"`
	Offset   int                     `json:"offset,omitempty"`
}

type BangumiCollectionUpdateRequest struct {
	SubjectID int64    `json:"subjectId"`
	Type      any      `json:"type,omitempty"` // 1..5 or wish/collect/doing/on_hold/dropped
	Rate      *int     `json:"rate,omitempty"`
	EpStatus  *int     `json:"epStatus,omitempty"`
	VolStatus *int     `json:"volStatus,omitempty"`
	Comment   *string  `json:"comment,omitempty"`
	Private   *bool    `json:"private,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	UserAgent string   `json:"userAgent,omitempty"`
}

type BangumiCollectionUpdateResult struct {
	SubjectID int64           `json:"subjectId"`
	Updated   bool            `json:"updated"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type BangumiEpisodeUpdateRequest struct {
	Type       any     `json:"type"` // 0..3 or unwatched/wish/done/dropped
	EpisodeID  int64   `json:"episodeId,omitempty"`
	SubjectID  int64   `json:"subjectId,omitempty"`
	EpisodeIDs []int64 `json:"episodeIds,omitempty"`
	UserAgent  string  `json:"userAgent,omitempty"`
}

type BangumiEpisodeUpdateResult struct {
	Mode       string  `json:"mode"`
	SubjectID  int64   `json:"subjectId,omitempty"`
	EpisodeID  int64   `json:"episodeId,omitempty"`
	EpisodeIDs []int64 `json:"episodeIds,omitempty"`
	Type       int     `json:"type"`
	Updated    bool    `json:"updated"`
}

// Client is a typed, supervised client bound to one installed service package.
//
// It enforces permission checks, isolated account secret checks, secret
// redaction, envelope unwrapping, and request cancellation/timeouts through
// Core's Supervisor.
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

// RequireAnySecret verifies that at least one key in candidateKeys is present
// and non-empty in the service's isolated account store (e.g. TMDB accepts
// either `api_read_token` or `api_key`).
func (c *Client) RequireAnySecret(candidateKeys ...string) error {
	if len(candidateKeys) == 0 {
		return nil
	}
	accounts, err := OpenAccounts(c.dataDir, c.serviceID)
	if err != nil {
		return err
	}
	missing, err := accounts.Missing(candidateKeys...)
	if err != nil {
		return err
	}
	if len(missing) < len(candidateKeys) {
		return nil
	}
	return &MissingSecretsError{ServiceID: c.serviceID, Keys: missing}
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
// req to JSON, calls method, unwraps the standard `{ok, service, method, data, error}`
// envelope when present, and decodes the response into resp.
func (c *Client) CallJSON(ctx context.Context, method string, req any, resp any, requiredPerms []service.Permission, requiredSecrets []string) error {
	return c.callJSON(ctx, method, req, resp, requiredPerms, requiredSecrets, nil)
}

func (c *Client) callJSON(ctx context.Context, method string, req any, resp any, requiredPerms []service.Permission, allSecrets []string, anyOfSecrets []string) error {
	if len(requiredPerms) > 0 {
		if _, err := c.RequirePermissions(requiredPerms...); err != nil {
			return err
		}
	}
	if err := c.RequireSecrets(allSecrets...); err != nil {
		return err
	}
	if err := c.RequireAnySecret(anyOfSecrets...); err != nil {
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
	target, err := c.unwrapEnvelope(method, out)
	if err != nil {
		return err
	}
	if resp == nil {
		return nil
	}
	if err := json.Unmarshal(target, resp); err != nil {
		return c.sanitize(fmt.Errorf("service: decode %s response: %w", method, err))
	}
	return nil
}

func (c *Client) unwrapEnvelope(method string, raw json.RawMessage) (json.RawMessage, error) {
	var env serviceEnvelope
	if err := json.Unmarshal(raw, &env); err != nil || env.OK == nil {
		return raw, nil
	}
	// Only treat the response as a standard `{ok, service, method, data, error}`
	// service envelope when envelope metadata/fields are actually present, so
	// raw objects like `{status, ok, body}` pass through untouched.
	if env.Service == "" && env.Method == "" && len(env.Data) == 0 && env.Error == nil {
		return raw, nil
	}
	if !*env.OK {
		svcErr := env.Error
		if svcErr == nil {
			svcErr = &ServiceError{Code: "UNKNOWN_ERROR", Message: "service returned ok=false"}
		}
		svcErr.Service = env.Service
		if svcErr.Service == "" {
			svcErr.Service = c.serviceID
		}
		svcErr.Method = env.Method
		if svcErr.Method == "" {
			svcErr.Method = method
		}
		if accounts, err := OpenAccounts(c.dataDir, c.serviceID); err == nil {
			svcErr.Message = redactString(accounts, svcErr.Message)
		}
		return nil, svcErr
	}
	if len(env.Data) == 0 {
		return json.RawMessage("null"), nil
	}
	return env.Data, nil
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

func (t *TraktClient) AccountMe(ctx context.Context) (*TraktAccount, error) {
	var out TraktAccount
	err := t.callJSON(ctx, "account.me", nil, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		[]string{SecretKeyClientID, SecretKeyAccessToken}, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *TraktClient) MetadataSearch(ctx context.Context, req TraktSearchRequest) (*TraktSearchResult, error) {
	var out TraktSearchResult
	err := t.callJSON(ctx, "metadata.search", req, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		[]string{SecretKeyClientID}, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *TraktClient) Watchlist(ctx context.Context, req TraktWatchlistRequest) (*TraktWatchlistResult, error) {
	var out TraktWatchlistResult
	err := t.callJSON(ctx, "watchlist.list", req, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		[]string{SecretKeyClientID, SecretKeyAccessToken}, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *TraktClient) History(ctx context.Context, req TraktHistoryRequest) (*TraktHistoryResult, error) {
	var out TraktHistoryResult
	err := t.callJSON(ctx, "history.list", req, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		[]string{SecretKeyClientID, SecretKeyAccessToken}, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *TraktClient) ScrobbleStart(ctx context.Context, req TraktScrobbleRequest) (*TraktScrobbleResult, error) {
	var out TraktScrobbleResult
	err := t.callJSON(ctx, "scrobble.start", req, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		[]string{SecretKeyClientID, SecretKeyAccessToken}, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *TraktClient) ScrobbleStop(ctx context.Context, req TraktScrobbleRequest) (*TraktScrobbleResult, error) {
	var out TraktScrobbleResult
	err := t.callJSON(ctx, "scrobble.stop", req, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		[]string{SecretKeyClientID, SecretKeyAccessToken}, nil)
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

func (t *TMDBClient) MetadataSearch(ctx context.Context, req TMDBSearchRequest) (*TMDBSearchResult, error) {
	var out TMDBSearchResult
	err := t.callJSON(ctx, "metadata.search", req, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		nil, []string{SecretKeyAPIReadToken, SecretKeyAPIKey})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *TMDBClient) MetadataDetail(ctx context.Context, req TMDBDetailRequest) (*TMDBDetailResult, error) {
	var out TMDBDetailResult
	err := t.callJSON(ctx, "metadata.detail", req, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		nil, []string{SecretKeyAPIReadToken, SecretKeyAPIKey})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *TMDBClient) ArtworkList(ctx context.Context, req TMDBArtworkRequest) (*TMDBArtworkResult, error) {
	var out TMDBArtworkResult
	err := t.callJSON(ctx, "artwork.list", req, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		nil, []string{SecretKeyAPIReadToken, SecretKeyAPIKey})
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

func (b *BangumiClient) AccountMe(ctx context.Context) (*BangumiAccount, error) {
	var out BangumiAccount
	err := b.callJSON(ctx, "account.me", nil, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		[]string{SecretKeyAccessToken}, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (b *BangumiClient) MetadataSearch(ctx context.Context, req BangumiSearchRequest) (*BangumiSearchResult, error) {
	var out BangumiSearchResult
	err := b.callJSON(ctx, "metadata.search", req, &out,
		[]service.Permission{service.PermissionNetwork}, nil, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (b *BangumiClient) MetadataDetail(ctx context.Context, req BangumiDetailRequest) (*BangumiSubjectDetail, error) {
	var out BangumiSubjectDetail
	err := b.callJSON(ctx, "metadata.detail", req, &out,
		[]service.Permission{service.PermissionNetwork}, nil, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (b *BangumiClient) CollectionGet(ctx context.Context, req BangumiCollectionGetRequest) (*BangumiCollectionGetResult, error) {
	var out BangumiCollectionGetResult
	perms := []service.Permission{service.PermissionNetwork}
	var secrets []string
	if strings.TrimSpace(req.Username) == "" {
		perms = append(perms, service.PermissionSecrets)
		secrets = []string{SecretKeyAccessToken}
	}
	err := b.callJSON(ctx, "collection.get", req, &out, perms, secrets, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (b *BangumiClient) CollectionUpdate(ctx context.Context, req BangumiCollectionUpdateRequest) (*BangumiCollectionUpdateResult, error) {
	var out BangumiCollectionUpdateResult
	err := b.callJSON(ctx, "collection.update", req, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		[]string{SecretKeyAccessToken}, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (b *BangumiClient) EpisodeUpdate(ctx context.Context, req BangumiEpisodeUpdateRequest) (*BangumiEpisodeUpdateResult, error) {
	var out BangumiEpisodeUpdateResult
	err := b.callJSON(ctx, "episode.update", req, &out,
		[]service.Permission{service.PermissionNetwork, service.PermissionSecrets},
		[]string{SecretKeyAccessToken}, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
