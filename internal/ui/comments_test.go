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
