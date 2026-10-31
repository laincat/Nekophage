package ui

import (
	"net/url"
	"testing"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/reader"
)

// A source may declare a comic with no chapter list: the gallery itself is the
// single reading unit. The detail view offers 开始阅读 in that case, so the
// reader address must exist too, with an empty chapter id that the source's
// loadEp ignores.
func TestReaderURLOpensAComicWithNoChapters(t *testing.T) {
	a := &App{}
	a.readerList = nil
	a.readerComic = "https://e-hentai.org/g/1/abc/"
	a.readerTitle = "某画廊"
	u := a.readerURLFor(0)
	if u == "" {
		t.Fatal("a comic with no chapters has no reader address")
	}
	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatalf("reader address does not parse: %v", err)
	}
	q := parsed.Query()
	if q.Get("comic") != a.readerComic {
		t.Errorf("comic = %q, want %q", q.Get("comic"), a.readerComic)
	}
	if q.Get("chapter") != "" {
		t.Errorf("chapter = %q, want empty: the source's loadEp takes none", q.Get("chapter"))
	}
	if q.Get("title") != "某画廊" {
		t.Errorf("title = %q, want the comic title", q.Get("title"))
	}
	// Only the one unit exists, so there is no second page to open.
	if got := a.readerURLFor(1); got != "" {
		t.Errorf("index 1 = %q, want nothing", got)
	}
}

// A comic that does have chapters keeps its own address.
func TestReaderURLStillUsesChaptersWhenPresent(t *testing.T) {
	a := &App{}
	a.readerList = []core.Chapter{{ID: "ch1", Title: "第一话"}}
	a.readerComic = "c1"
	a.readerTitle = "标题"
	u, err := url.Parse(a.readerURLFor(0))
	if err != nil {
		t.Fatalf("reader address does not parse: %v", err)
	}
	if got := u.Query().Get("chapter"); got != "ch1" {
		t.Errorf("chapter = %q, want ch1", got)
	}
}

// The reader page asks to reopen the unit it is showing. With no chapter list
// the only unit is the comic itself, named by an empty chapter id.
func TestSwitchChapterForAComicWithNoChapters(t *testing.T) {
	a := &App{}
	a.readerList = nil
	a.readerComic = "https://e-hentai.org/g/1/abc/"
	a.readerTitle = "某画廊"

	u, ok := a.switchChapter(reader.Switch{Source: "ehentai", Comic: a.readerComic, Chapter: ""})
	if !ok || u == "" {
		t.Fatalf("reopening the one unit: ok=%v url=%q", ok, u)
	}
	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatalf("url does not parse: %v", err)
	}
	if got := parsed.Query().Get("comic"); got != a.readerComic {
		t.Errorf("comic = %q, want the open comic", got)
	}
	// A chapter id this comic does not have is still refused.
	if _, ok := a.switchChapter(reader.Switch{Source: "ehentai", Comic: a.readerComic, Chapter: "ch1"}); ok {
		t.Error("a comic with no chapters accepted a chapter id")
	}
	// A switch for another comic is refused too.
	if _, ok := a.switchChapter(reader.Switch{Source: "ehentai", Comic: "other", Chapter: ""}); ok {
		t.Error("a switch for another comic was accepted")
	}
}

// A tag's meaning comes from the source; what the interface does with it is
// decided here, so the three cases can be checked without a window.
func TestTagOutcomeFollowsTheSource(t *testing.T) {
	query, notice := tagOutcome(core.TagAction{Action: "search", Keyword: "作者:张三"}, "张三")
	if query != "作者:张三" || notice != "" {
		t.Errorf("search tag = %q, %q", query, notice)
	}
	// A source that names no keyword gets no search.
	if query, notice = tagOutcome(core.TagAction{Action: "search"}, "张三"); query != "" || notice == "" {
		t.Errorf("empty search tag = %q, %q", query, notice)
	}
	// A category tag is not a search: the caller opens the category tab on the
	// parameter instead, so this returns nothing to run here.
	if query, notice = tagOutcome(core.TagAction{Action: "category", Keyword: "长篇", Param: "33"}, "长篇"); query != "" || notice != "" {
		t.Errorf("category tag = %q, %q", query, notice)
	}
	if query, notice = tagOutcome(core.TagAction{Action: ""}, "长篇"); query != "" || notice == "" {
		t.Errorf("unknown tag = %q, %q", query, notice)
	}
}
