package ui

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/shelf"
	"github.com/laincat/veneracat-core/plugin"
)

func novelFixture(t *testing.T, handler http.HandlerFunc) (*novelPanel, chan func()) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := core.Open(core.Options{DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	const source = `class NovelFixture extends NovelSource {
		name = "Fixture"; key = "novel_fixture"; version = "1"
		async popular(page) { return {items: [{id: "book", title: "热门 " + page}], maxPage: 2} }
		async search(query, page) { return {items: [{id: "book", title: "找到 " + query}], maxPage: 1} }
		async detail(id) {
			return {id, title: "星海故事", author: "作者", summary: "故事简介",
				chapters: [{id: "one", title: "起航"}, {id: "two", title: "抵达"}, {id: "empty", title: "空章"}]}
		}
		async chapter(novelId, chapterId) {
			const res = await Network.get(%q + "/" + novelId + "/" + chapterId)
			return {html: res.body}
		}
	}`
	if _, err = c.InstallTyped(context.Background(), plugin.TypeNovelJS, fmt.Sprintf(source, server.URL), "test", ""); err != nil {
		t.Fatal(err)
	}
	sh, err := shelf.Open(c.DataDir())
	if err != nil {
		t.Fatal(err)
	}
	queued := make(chan func(), 8)
	p := &novelPanel{core: c, shelf: sh, post: func(fn func()) { queued <- fn }}
	t.Cleanup(p.clearRequest)
	return p, queued
}

func novelResult(t *testing.T, queued chan func()) {
	t.Helper()
	select {
	case apply := <-queued:
		apply()
	case <-time.After(8 * time.Second):
		t.Fatal("novel operation did not return")
	}
}

func TestNovelDesktopBrowseReadAndResume(t *testing.T) {
	p, queued := novelFixture(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/book/one":
			fmt.Fprint(w, "<p>"+strings.Repeat("文", 1000)+"</p><script>not chapter text</script><p>第二页正文</p>")
		case "/book/two":
			fmt.Fprint(w, "<p>第二章正文</p>")
		case "/book/empty":
			fmt.Fprint(w, "<script>empty</script>")
		default:
			http.NotFound(w, r)
		}
	})
	p.setSources(p.core.Plugins())
	novelResult(t, queued)
	tt := ui.NewTester(p.view, 860, 720)
	click := func(name string) {
		t.Helper()
		tt.Frame()
		if err := tt.Click(name); err != nil {
			t.Fatal(err)
		}
	}
	click("下一页")
	novelResult(t, queued)
	if p.catalogPage != 2 || p.catalog.Items[0].Title != "热门 2" {
		t.Fatalf("catalog = %+v", p.catalog)
	}
	p.query = "星海"
	click("搜索")
	novelResult(t, queued)
	click("找到 星海")
	novelResult(t, queued)
	click("加入书架")
	novelResult(t, queued)
	click("起航")
	novelResult(t, queued) // display chapter
	novelResult(t, queued) // save first page
	if p.screen != novelReading || len(p.pages) != 2 || p.pages[1] != "第二页正文" {
		t.Fatalf("unexpected reading state: pages=%d, screen=%d", len(p.pages), p.screen)
	}
	click("下一页")
	novelResult(t, queued)
	root := p.shelf.Root()
	reopened, err := shelf.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	entries := reopened.Novels()
	if len(entries) != 1 || !entries[0].Kept || entries[0].ChapterID != "one" || entries[0].Page != 1 {
		t.Fatalf("saved position: %+v", entries)
	}

	// Reopen on the saved book like a new window using the profile on disk.
	resumed := &novelPanel{core: p.core, shelf: reopened, post: p.post, entries: entries, screen: novelBooks}
	resumed.setSources(p.sources)
	p = resumed
	tt = ui.NewTester(p.view, 860, 720)
	click("星海故事 · 作者")
	novelResult(t, queued)
	click("继续阅读")
	novelResult(t, queued)
	novelResult(t, queued)
	tt.Frame()
	if p.page != 1 || !strings.Contains(strings.Join(tt.Texts(), "\n"), "第二页正文") {
		t.Fatalf("resume landed on page %d", p.page)
	}
	click("下一章")
	novelResult(t, queued)
	novelResult(t, queued)
	if p.chapter != 1 || p.page != 0 || p.entries[0].ChapterID != "two" {
		t.Fatalf("chapter switch did not update progress: chapter=%d, entries=%+v", p.chapter, p.entries)
	}
	click("下一章")
	novelResult(t, queued)
	if p.err == nil || p.chapter != 1 || p.entries[0].ChapterID != "two" {
		t.Fatal("an empty chapter replaced the current reading position")
	}
	click("目录")
	click("返回")
	if p.screen != novelBooks {
		t.Fatal("the reader did not return to the bookshelf")
	}
}

func TestNovelLeavingDuringLoadCancelsAndRejectsLateResult(t *testing.T) {
	started := make(chan struct{}, 1)
	cancelled := make(chan struct{}, 1)
	p, queued := novelFixture(t, func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		<-r.Context().Done()
		cancelled <- struct{}{}
	})
	p.setSources(p.core.Plugins())
	novelResult(t, queued)
	p.openBook(shelf.Novel{Source: p.key, ID: "book"})
	novelResult(t, queued)
	p.openChapter(0)
	select {
	case <-started:
	case <-time.After(8 * time.Second):
		t.Fatal("request never reached the server")
	}
	p.backTo(novelCatalog)
	select {
	case <-cancelled:
	case <-time.After(8 * time.Second):
		t.Fatal("leaving the detail screen did not cancel the chapter")
	}
	novelResult(t, queued)
	if p.screen != novelCatalog || p.busy || p.err != nil || p.detail != nil || p.pages != nil || len(p.shelf.Novels()) != 0 {
		t.Fatal("a cancelled chapter changed the view or saved progress")
	}
}
