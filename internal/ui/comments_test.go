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
