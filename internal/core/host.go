package core

import (
	"runtime"

	"github.com/laincat/nekophage/internal/sourceapi"
	"github.com/laincat/veneracat-core/gohost"
	"github.com/laincat/veneracat-core/novel"
)

type (
	Core                = gohost.Core
	Options             = gohost.Options
	Transport           = gohost.Transport
	Comic               = gohost.Comic
	Page                = gohost.Page
	Details             = gohost.Details
	Chapter             = gohost.Chapter
	TagAction           = gohost.TagAction
	Category            = gohost.Category
	CategoryPart        = gohost.CategoryPart
	Comment             = gohost.Comment
	ExplorePage         = gohost.ExplorePage
	ExploreResult       = gohost.ExploreResult
	ExploreSection      = gohost.ExploreSection
	AccountStyles       = gohost.AccountStyles
	FavoritesSupport    = gohost.FavoritesSupport
	SearchOptionGroup   = gohost.SearchOptionGroup
	SourceSetting       = gohost.SourceSetting
	WebviewLogin        = gohost.WebviewLogin
	LogLine             = gohost.LogLine
	ImageRequest        = gohost.ImageRequest
	NovelItem           = novel.Item
	NovelPage           = novel.Page
	NovelChapter        = novel.Chapter
	NovelDetail         = novel.Detail
	NovelChapterContent = novel.ChapterContent
	NovelInfo           = novel.Info
)

var (
	ErrNoTagAction    = gohost.ErrNoTagAction
	ErrNovelNotFound  = novel.ErrNotFound
	ComicPartition    = gohost.ComicPartition
	MaxLogLines       = gohost.MaxLogLines
	NewTransport      = gohost.NewTransport
	WebviewHandleGone = gohost.WebviewHandleGone
)

func Open(opts Options) (*Core, error) {
	if opts.Platform == "" {
		opts.Platform = runtime.GOOS
		if opts.Platform == "darwin" {
			opts.Platform = "macos"
		}
	}
	if opts.AppVersion == "" {
		opts.AppVersion = sourceapi.Level
	}
	return gohost.Open(opts)
}
