package ui

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/library"
)

const tagSource = `class Tags extends ComicSource {
  name = 'Tags'; key = 'tags'; version = '1'
  // The declaration lives on category; categoryComics only loads it.
  category = {
    title: '分类',
    parts: [{name: '类型', type: 'fixed', itemType: 'category', categories: ['热血', '恋爱'], categoryParams: ['action', 'romance']}],
  }
  categoryComics = {
    optionList: [],
    load: async (category, param, options, page) => ({comics: [], maxPage: 1}),
  }
  comic = {
    onClickTag: async (namespace, tag) => ({action: 'category', param: tag === '热血' ? 'action' : 'nope'}),
  }
  search = {load: async () => ({comics: [], maxPage: 1})}
}`

// A tag the source maps to a category parameter has to land on that value, not
// on a search and not on the first value of the tab.
func TestTagCategoryClickSelectsTheValue(t *testing.T) {
	c, err := core.Open(core.Options{DataDir: filepath.Join(t.TempDir(), "d"), AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	if _, err := c.Install(context.Background(), tagSource, "file", ""); err != nil {
		t.Fatal(err)
	}
	a := New(c, library.New(c), nil)
	a.sourceKey = "tags"

	action, err := c.ClickTag(context.Background(), "tags", "类型", "热血")
	if err != nil {
		t.Fatalf("click tag: %v", err)
	}
	if action.Action != "category" || action.Param != "action" {
		t.Fatalf("action = %+v", action)
	}
	// The tab's own declaration is what the click looks up, so it has to be
	// readable without a second source call.
	page, err := c.CategoryPage(context.Background(), "tags")
	if err != nil {
		t.Fatalf("category: %v", err)
	}
	if len(page.Parts) != 1 {
		t.Fatalf("declaration = %+v", page.Parts)
	}
	if got := page.Parts[0].Params; len(got) != 2 || got[0] != "action" {
		t.Fatalf("params = %v", got)
	}
	a.mu.Lock()
	a.category, a.catLoaded = page, true
	a.mu.Unlock()
	// The matching parameter selects that value on the category tab.
	a.openCategoryAt("action")
	waitFor(t, func() bool {
		a.mu.Lock()
		defer a.mu.Unlock()
		return a.mode == 3 && a.catPart == 0 && a.catValue == 0
	}, "the category tab never selected the value the tag named")

	// A parameter the declaration does not contain is reported, not silently
	// turned into the first value.
	a.mu.Lock()
	a.notice = ""
	a.catValue = -1
	a.mu.Unlock()
	a.openCategoryAt("not-a-param")
	waitFor(t, func() bool {
		a.mu.Lock()
		defer a.mu.Unlock()
		return a.notice != ""
	}, "an unknown category parameter was dropped without a word")
}

// waitFor polls until cond holds, so a background load can settle without a
// fixed sleep.
func waitFor(t *testing.T, cond func() bool, message string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal(message)
}
