package ui

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/library"
)

const talkSource = `class Talk extends ComicSource {
  name = 'Talk'; key = 'talk'; version = '1'
  search = {load: async () => ({comics: [], maxPage: 1})}
  comic = {
    loadInfo: async (id) => ({title: 'T', subId: 's1', chapters: {}}),
    loadComments: async (comicId, subId, page, replyTo) => ({
      comments: [{id: 'c' + page, userName: 'ada', content: 'p' + page, time: 'now'}], maxPage: 3
    }),
    sendComment: async () => 'ok',
  }
}`

func commentsApp(t *testing.T) *App {
	t.Helper()
	c, err := core.Open(core.Options{DataDir: filepath.Join(t.TempDir(), "d"), AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	if _, err := c.Install(context.Background(), talkSource, "file", ""); err != nil {
		t.Fatal(err)
	}
	a := New(c, library.New(c), nil)
	a.mu.Lock()
	a.detailComic = core.Comic{ID: "42", Title: "T"}
	a.detail = core.Details{Title: "T", SubID: "s1"}
	a.sourceKey = "talk"
	a.mu.Unlock()
	return a
}

// The thread reached the state the window draws, with the sub id the source
// keyed it by.
func TestCommentsLoadIntoTheWindowState(t *testing.T) {
	a := commentsApp(t)
	a.comments = commentsState{key: "talk", id: "42", sub: "s1", page: 1,
		canSend: a.core.CanSendComment(context.Background(), "talk")}
	if !a.comments.canSend {
		t.Fatal("the source accepts comments but the window says it cannot")
	}
	res, err := a.core.Comments(context.Background(), "talk", "42", "s1", 2, "")
	if err != nil {
		t.Fatalf("comments: %v", err)
	}
	if len(res.Comments) != 1 || res.Comments[0].ID != "c2" || res.MaxPage != 3 {
		t.Fatalf("page = %+v", res)
	}
	if err := a.core.SendComment(context.Background(), "talk", "42", "s1", "hi", ""); err != nil {
		t.Errorf("send: %v", err)
	}
}

// A source without comments offers no button, so the details view must know
// before it draws.
func TestCommentButtonFollowsTheSource(t *testing.T) {
	a := commentsApp(t)
	a.mu.Lock()
	a.detailCanComment = a.core.CommentsSupported(context.Background(), "talk")
	a.mu.Unlock()
	if !a.detailCanComment {
		t.Error("the source declares loadComments but the button would be hidden")
	}
	if _, err := a.core.Install(context.Background(), `class Quiet extends ComicSource {
  name = 'Quiet'; key = 'quiet'; version = '1'
  search = {load: async () => ({comics: [], maxPage: 1})}
}`, "file", ""); err != nil {
		t.Fatal(err)
	}
	if a.core.CommentsSupported(context.Background(), "quiet") {
		t.Error("a source without comments reported support")
	}
}

// A source that votes on comments gets a control; one that does not keeps it
// away, and the vote reaches the source with the flag it expects.
func TestCommentVoteFollowsTheSource(t *testing.T) {
	c, err := core.Open(core.Options{DataDir: filepath.Join(t.TempDir(), "d"), AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	ctx := context.Background()
	if _, err := c.Install(ctx, `class Votes extends ComicSource {
  name = 'Votes'; key = 'votes'; version = '1'
  search = {load: async () => ({comics: [], maxPage: 1})}
  comic = {
    loadComments: async () => ({comments: [{id: 'c1', userName: 'a', content: 'x', isLiked: false}], maxPage: 1}),
    likeComment: async (comicId, subId, commentId, isLike) => {
      if (commentId !== 'c1') throw new Error('wrong comment id')
      return isLike ? 'liked' : 'unliked'
    },
  }
}`, "file", ""); err != nil {
		t.Fatal(err)
	}
	if !c.CanLikeComment(ctx, "votes") {
		t.Fatal("the source votes on comments but the client says it does not")
	}
	if err := c.LikeComment(ctx, "votes", "42", "sub", "c1", true); err != nil {
		t.Errorf("like: %v", err)
	}
	// The same call with false is how a vote is taken back.
	if err := c.LikeComment(ctx, "votes", "42", "sub", "c1", false); err != nil {
		t.Errorf("unlike: %v", err)
	}
	// A source without likeComment is told apart rather than asked.
	if _, err := c.Install(ctx, `class Quiet extends ComicSource {
  name = 'Quiet'; key = 'quiet'; version = '1'
  search = {load: async () => ({comics: [], maxPage: 1})}
}`, "file", ""); err != nil {
		t.Fatal(err)
	}
	if c.CanLikeComment(ctx, "quiet") {
		t.Error("a source without likeComment reported support")
	}
	if err := c.LikeComment(ctx, "quiet", "1", "", "c1", true); err == nil {
		t.Error("voting on a source that does not vote succeeded")
	}
}

// "N 条回复" is the source's word that replies exist; reading them is a second
// call with replyTo set, and the way back has to exist.
func TestRepliesAreALoadedWithReplyTo(t *testing.T) {
	c, err := core.Open(core.Options{DataDir: filepath.Join(t.TempDir(), "d"), AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	ctx := context.Background()
	if _, err := c.Install(ctx, `class Threads extends ComicSource {
  name = 'Threads'; key = 'threads'; version = '1'
  search = {load: async () => ({comics: [], maxPage: 1})}
  comic = {
    loadComments: async (comicId, subId, page, replyTo) => ({
      comments: replyTo === null
        ? [{id: 'c1', userName: '阿达', content: 'top', replyCount: 2}]
        : [{id: 'r1', userName: '鲍勃', content: 'reply to ' + replyTo}],
      maxPage: 1,
    }),
  }
}`, "file", ""); err != nil {
		t.Fatal(err)
	}
	thread, err := c.Comments(ctx, "threads", "42", "", 1, "")
	if err != nil || len(thread.Comments) != 1 || thread.Comments[0].ReplyCount != 2 {
		t.Fatalf("thread = %+v, %v", thread, err)
	}
	// Passing null reads the thread; passing an id reads that comment's replies.
	replies, err := c.Comments(ctx, "threads", "42", "", 1, "c1")
	if err != nil {
		t.Fatalf("replies: %v", err)
	}
	if len(replies.Comments) != 1 || replies.Comments[0].Content != "reply to c1" {
		t.Fatalf("replies = %+v", replies.Comments)
	}
}

// The check that runs in a window reads the thread through this method, so it has
// to report what the window holds rather than what a screenshot looks like: an
// empty thread and a real one capture to almost the same number of bytes.
func TestCommentsForCheckReportsWhatTheWindowHolds(t *testing.T) {
	a := commentsApp(t)

	count, author, body, errText := a.CommentsForCheck()
	if count != 0 || author != "" || body != "" || errText != "" {
		t.Fatalf("an empty window reported %d %q %q %q", count, author, body, errText)
	}

	a.mu.Lock()
	a.comments.list = []core.Comment{
		{UserName: "ada", Content: "first"},
		{UserName: "bob", Content: "second"},
	}
	a.comments.errText = "kept from an earlier page"
	a.mu.Unlock()

	count, author, body, errText = a.CommentsForCheck()
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	// The first comment is the one a reader sees first, so it is the one reported.
	if author != "ada" || body != "first" {
		t.Errorf("first comment = %q %q", author, body)
	}
	if errText != "kept from an earlier page" {
		t.Errorf("errText = %q", errText)
	}
}

// The window checks read the view state through these, so what they report has to
// be the view's own contents rather than something derived from a screenshot.
func TestFavoritesForCheckReportsTheListing(t *testing.T) {
	a := commentsApp(t)

	count, first, errText := a.FavoritesForCheck()
	if count != 0 || first != "" || errText != "" {
		t.Fatalf("an empty view reported %d %q %q", count, first, errText)
	}

	a.mu.Lock()
	a.favComics = []core.Comic{{Title: "ada"}, {Title: "bob"}}
	a.favErr = "限制"
	a.mu.Unlock()

	count, first, errText = a.FavoritesForCheck()
	if count != 2 || first != "ada" || errText != "限制" {
		t.Fatalf("count=%d first=%q err=%q", count, first, errText)
	}
}

// The chip rule lives here: one section draws no chips, two draw one each. A
// capture shows a row or no row without saying which the view believes it drew.
func TestBrowseForCheckFollowsTheChipRule(t *testing.T) {
	a := commentsApp(t)

	sections, chips, comics, _ := a.BrowseForCheck()
	if sections != 0 || chips != 0 || comics != 0 {
		t.Fatalf("empty browse reported %d/%d/%d", sections, chips, comics)
	}

	fill := func(pages int) {
		a.mu.Lock()
		a.explorePages = make([]core.ExplorePage, pages)
		a.explore = core.ExploreResult{Sections: []core.ExploreSection{
			{Title: "one", Comics: []core.Comic{{Title: "a"}, {Title: "b"}}},
			{Title: "two", Comics: []core.Comic{{Title: "c"}}},
		}}
		a.mu.Unlock()
	}

	fill(1)
	sections, chips, comics, _ = a.BrowseForCheck()
	if sections != 1 || chips != 0 || comics != 3 {
		t.Fatalf("one section: %d/%d/%d, want 1/0/3", sections, chips, comics)
	}

	fill(2)
	sections, chips, comics, _ = a.BrowseForCheck()
	if sections != 2 || chips != 2 || comics != 3 {
		t.Fatalf("two sections: %d/%d/%d, want 2/2/3", sections, chips, comics)
	}
}

// The category check reports both halves: the parts a reader chooses from and the
// comics the loader returned. A page that draws its choosers and loads nothing is
// the fault it exists to name, so the two counts have to be independent.
func TestCategoryForCheckSeparatesPartsFromComics(t *testing.T) {
	a := commentsApp(t)

	parts, values, _, comics, _ := a.CategoryForCheck()
	if parts != 0 || values != 0 || comics != 0 {
		t.Fatalf("empty category reported %d/%d/%d", parts, values, comics)
	}

	a.mu.Lock()
	a.category = core.Category{Parts: []core.CategoryPart{
		{Name: "Genre", Categories: []string{"probe", "test"}},
	}}
	a.catComics = nil
	a.mu.Unlock()

	parts, values, _, comics, _ = a.CategoryForCheck()
	if parts != 1 || values != 2 || comics != 0 {
		t.Fatalf("parts without results: %d/%d/%d, want 1/2/0", parts, values, comics)
	}

	a.mu.Lock()
	a.catComics = []core.Comic{{Title: "Probe Comic"}}
	a.mu.Unlock()

	parts, values, _, comics, _ = a.CategoryForCheck()
	if parts != 1 || values != 2 || comics != 1 {
		t.Fatalf("loaded page: %d/%d/%d, want 1/2/1", parts, values, comics)
	}
}
