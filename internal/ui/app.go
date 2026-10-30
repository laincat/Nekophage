// Package ui is the app's main window: pick a source, search, open a comic,
// pick a chapter, read it.
//
// The window is native UI drawn by MyGo from Go; the reader opens beside it
// as a window of its own holding a WebView. Work that leaves the machine runs
// on goroutines and reaches the window through Window.Update, so a slow
// request never blocks a frame.
package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/library"
	"github.com/laincat/nekophage/internal/reader"
	"github.com/laincat/nekophage/internal/searches"
	"github.com/laincat/nekophage/internal/shelf"
	"github.com/laincat/veneracat-core/plugin"
)

// App is the main window's state.
type App struct {
	core     *core.Core
	lib      *library.Library
	shelf    *shelf.Shelf
	searches *searches.Store
	win      *mygo.Window
	scheme   *reader.Server

	mu sync.Mutex

	// sources and the chosen one
	sources   []plugin.Record
	options   []string
	sourceKey string

	// search
	query   string
	results []core.Comic
	// resultSources is the source key of each result when several sources
	// were searched at once; empty when the results are one source's.
	resultSources []string
	searchAll     bool
	page          int
	maxPage       int
	busy          string // what is running: "search", "details", "install"
	errText       string
	notice        string

	// details
	open        bool
	detail      core.Details
	detailComic core.Comic

	// covers, loaded once each: "source/id" → data or failure
	covers  map[string][]byte
	coverIn map[string]bool
	bitmaps map[string]*ui.Bitmap

	grid ui.GridState

	// browse: mode 0 searches, mode 1 browses a source's explore pages.
	mode         int
	explorePages []core.ExplorePage
	exploreIndex int
	explore      core.ExploreResult
	explorePage  int
	exploreBusy  bool

	// chapterQuery filters the chapter list of the open comic.
	chapterQuery string
	// md renders descriptions; one set of options is shared by every view.
	// Links keep markdown's default: http and https open in the browser.
	md markdown.Options

	// reader window
	readerWin     *mygo.Window
	readerView    *mygo.WebView
	readerTitle   string
	readerChapter int
	readerList    []core.Chapter
	readerComic   string

	manageWin *mygo.Window
	manage    manageState
	// defaultIndex is the index URL tried first in the manager.
	defaultIndex string

	// gridIndex is the card chosen in the results grid.
	gridIndex int

	// devExplore picks the browse the tools asked for, once.
	devExploreUsed bool
	devChapterUsed bool
	devAdvanced    bool
	devAutoUsed    bool
	devKept        bool

	// resumeChapter is the chapter a shelf entry asked for, or -1.
	resumeChapter int

	// search filters: the groups the current source offers and the value
	// chosen for each, plus whether the row is open.
	searchGroups []core.SearchOptionGroup
	searchValues []string
	searchKey    string
	filtersOpen  bool

	// shelf editing: which folder to keep into, the name being typed for a
	// new folder, the folder being renamed and its new name, and the folder
	// each row's move control shows.
	favFolder     string
	folderNew     string
	renaming      string
	renameTo      string
	movePick      map[string]string
	openedChapter bool
	// settings remembers the source and the query between launches.
	settings settings
	// dev is what the tools that check the interface ask for; the app leaves
	// it empty.
	dev DevOptions
}

// DevOptions drives the window from a tool (cmd/uicheck): search on start,
// and open the first result. Nothing in the app sets it.
type DevOptions struct {
	Query     string
	OpenFirst bool
	// SetSetting changes one declared setting as "key=value" after the panel
	// opens, so the tools can check that a choice is stored.
	SetSetting string
	// OpenSettings opens the sources window with the first source's settings
	// panel, so the tools can capture it.
	OpenSettings bool
	// Filters opens the source's search filters before capturing.
	Filters bool
	// SearchAll starts with every source selected.
	SearchAll bool
	// KeepPage clicks the reader's 收藏本页 button once.
	KeepPage bool
	// OpenLast opens the last result, which in a merged search belongs to a
	// source other than the chosen one.
	OpenLast bool
	// Filter picks a search filter as "index=key", then searches again, so
	// the tools can check that the choice reaches the source.
	Filter string
	// Shelf opens the shelf instead of searching.
	Shelf bool
	// ScrollPercent scrolls the reader that far into the chapter once it
	// has loaded, so the tools can check the position it reports.
	ScrollPercent int
	// AutoNext turns the reader's automatic next chapter on and scrolls to
	// the end, so the tools can check that it moves on by itself.
	AutoNext bool
	// AdvanceChapter clicks the reader's 下一话 button once, so the tools can
	// check that Go drives the chapter change.
	AdvanceChapter bool
	// ShelfDemo edits folders at startup (create, move, rename) so the tools
	// can check that the shelf's own editing works end to end.
	ShelfDemo bool
	// OpenChapter opens that chapter in the reader once details arrive; -1
	// leaves the reader closed. The tools use it to check the reader path.
	OpenChapter int
	// Browse opens the browse view instead of searching; ExploreIndex and
	// ExplorePage choose which browse and which page of it to load.
	Browse       bool
	ExploreIndex int
	ExplorePage  int

	opened bool
}

// New builds the app.
func New(c *core.Core, lib *library.Library, srv *reader.Server) *App {
	a := &App{core: c, lib: lib, scheme: srv,
		covers: map[string][]byte{}, coverIn: map[string]bool{}, bitmaps: map[string]*ui.Bitmap{},
		movePick: map[string]string{}, favFolder: shelf.DefaultFolder,
		// Both mean "no chapter asked for"; zero would open the first one.
		resumeChapter: -1}
	a.dev.OpenChapter = -1
	return a
}

// Run shows the window and blocks until the app quits.
func (a *App) Run() error {
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:   "Nekophage",
			Width:   1100,
			Height:  760,
			Content: ui.View(a.view),
		})
		if sh, err := shelf.Open(a.core.DataDir()); err != nil {
			a.mu.Lock()
			a.notice = "读取书架失败:" + err.Error()
			a.mu.Unlock()
		} else {
			a.shelf = sh
		}
		if a.scheme != nil {
			// The reader reports its position, asks for the next chapter and
			// keeps pages; the window answers all three so its own state
			// follows.
			a.scheme.OnProgress(a.progressFromReader)
			a.scheme.OnSwitch(a.switchChapter)
			a.scheme.OnKeep(a.keepPage)
		}
		if st, err := searches.Open(a.core.DataDir()); err != nil {
			log.Printf("nekophage: reading search history: %v", err)
		} else {
			a.searches = st
		}
		a.loadSettings()
		a.refreshSources()
		if a.dev.Browse {
			a.setMode(1)
		}
		if a.dev.Shelf {
			a.setMode(2)
		}
		if a.dev.SearchAll {
			a.mu.Lock()
			a.searchAll = true
			a.mu.Unlock()
		}
		if a.dev.OpenSettings {
			a.openManage()
			// The panel reads the declaration, which is JS work: show the
			// window first, then ask.
			go func() {
				time.Sleep(500 * time.Millisecond)
				key := a.currentKey()
				a.openSourceSettings(key)
				if spec := a.dev.SetSetting; spec != "" {
					time.Sleep(1500 * time.Millisecond)
					a.devSetSetting(key, spec)
				}
			}()
		}
		if a.dev.Filters {
			a.mu.Lock()
			a.filtersOpen = true
			a.mu.Unlock()
		}
		if a.dev.ShelfDemo {
			a.shelfDemo()
		}
		if q := a.dev.Query; q != "" {
			a.query = q
			// With a filter to apply, the search waits for the filters, so
			// the source is not asked twice.
			if a.dev.Filter == "" {
				a.startSearch()
			}
		}
	})
	return mygo.App.Run()
}

// SetDevOptions drives the window from a tool. It must be called before Run.
func (a *App) SetDevOptions(opts DevOptions) { a.dev = opts }

// Window is the main window, once Run has made it.
func (a *App) Window() *mygo.Window { return a.win }

// ManageWindow is the sources window, for the development tools that capture
// it.
func (a *App) ManageWindow() *mygo.Window { return a.manageWin }

// --- data ---

// refreshSources reloads the installed list on the window's thread.
func (a *App) refreshSources() {
	a.mu.Lock()
	a.sources = a.lib.Sources()
	a.options = make([]string, 0, len(a.sources))
	keep := false
	for _, s := range a.sources {
		label := s.Name
		if !s.Enabled {
			label += "(已停用)"
		}
		a.options = append(a.options, label)
		if s.Key == a.sourceKey {
			keep = true
		}
	}
	if !keep {
		a.sourceKey = ""
		for _, s := range a.sources {
			if s.Enabled {
				a.sourceKey = s.Key
				break
			}
		}
	}
	key := a.sourceKey
	a.mu.Unlock()
	// The filters belong to the source, so they are read whenever the list of
	// sources changes under them.
	go func() { a.loadSearchOptions(key) }()
}

// sourceName is a source's display name.
func (a *App) sourceName(key string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, s := range a.sources {
		if s.Key == key {
			return s.Name
		}
	}
	return key
}

// currentKey returns the key of the chosen source.
func (a *App) currentKey() string {
	if a.sourceKey != "" {
		return a.sourceKey
	}
	for _, s := range a.sources {
		if s.Enabled {
			return s.Key
		}
	}
	return ""
}

// startSearch searches the current source for the typed query.
func (a *App) startSearch() {
	q := strings.TrimSpace(a.query)
	key := a.currentKey()
	if q == "" || key == "" {
		return
	}
	a.mu.Lock()
	a.busy, a.errText, a.page, a.maxPage, a.notice = "search", "", 0, 0, ""
	a.open = false
	a.mu.Unlock()
	a.saveSettings(key, q)
	if a.searches != nil {
		go func() { _ = a.searches.Add(q) }()
	}
	if a.searchAll {
		a.searchEverySource(q)
		return
	}
	a.searchPage(key, q, 1)
}

// searchEverySource searches every enabled source at once and shows the
// results in one grid, each card knowing where it came from.
func (a *App) searchEverySource(query string) {
	a.mu.Lock()
	sources := append([]plugin.Record(nil), a.sources...)
	a.busy, a.errText, a.notice, a.page, a.maxPage = "search", "", "", 1, 0
	a.results, a.resultSources = nil, nil
	a.open = false
	a.mu.Unlock()

	go func() {
		var enabled []plugin.Record
		for _, s := range sources {
			if s.Enabled {
				enabled = append(enabled, s)
			}
		}
		found := make([][]core.Comic, len(enabled))
		failed := make([]string, len(enabled))
		var wg sync.WaitGroup
		// A few sources at a time: a search should not open ten requests at
		// once, and a slow source must not hold up the others.
		sem := make(chan struct{}, 3)
		for i, s := range enabled {
			wg.Add(1)
			go func(i int, key string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()
				options, ok := a.optionsFor(key)
				if !ok {
					options, _ = a.core.DefaultOptions(ctx, key)
				}
				page, err := a.core.Search(ctx, key, query, options, 1)
				if err != nil {
					failed[i] = key
					return
				}
				found[i] = page.Comics
			}(i, s.Key)
		}
		wg.Wait()

		a.win.Update(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			a.busy = ""
			a.results, a.resultSources = nil, nil
			empty := 0
			for i := range enabled {
				if len(found[i]) == 0 {
					empty++
					continue
				}
				for _, comic := range found[i] {
					a.results = append(a.results, comic)
					a.resultSources = append(a.resultSources, enabled[i].Key)
				}
			}
			a.page, a.maxPage = 1, 0
			switch {
			case len(a.results) == 0:
				a.notice = "没有结果"
			case empty > 0:
				a.notice = itoa(len(a.results)) + " 部,其中 " + itoa(empty) + " 个源没有结果"
			}
			if a.dev.OpenLast && !a.dev.opened && len(a.results) > 0 {
				a.dev.opened = true
				last := len(a.results) - 1
				comic, key := a.results[last], a.sourceOf(last)
				go func() { a.openComicAt(key, comic) }()
			} else if a.dev.OpenFirst && !a.dev.opened && len(a.results) > 0 {
				a.dev.opened = true
				first, key := a.results[0], a.resultSources[0]
				go func() { a.openComicAt(key, first) }()
			}
		})
	}()
}

// sourceOf returns the source a result came from.
func (a *App) sourceOf(index int) string {
	if index >= 0 && index < len(a.resultSources) {
		return a.resultSources[index]
	}
	return a.currentKey()
}

// searchPage fetches one page of results off the window's thread.
func (a *App) searchPage(key, query string, page int) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		// One value per search group: the ones chosen in the filter row, or
		// the source's defaults for a source whose filters have not loaded.
		options, ok := a.optionsFor(key)
		var err error
		if !ok {
			options, err = a.core.DefaultOptions(ctx, key)
		}
		var res core.Page
		if err == nil {
			res, err = a.core.Search(ctx, key, query, options, page)
		}
		a.win.Update(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			a.busy = ""
			if err != nil {
				a.errText = err.Error()
				return
			}
			if page <= 1 {
				a.results = res.Comics
				a.resultSources = nil
			} else {
				seen := map[string]bool{}
				for _, c := range a.results {
					seen[c.ID] = true
				}
				for _, c := range res.Comics {
					if !seen[c.ID] {
						a.results = append(a.results, c)
					}
				}
			}
			a.page = page
			a.maxPage = res.MaxPage
			if len(a.results) == 0 {
				a.notice = "没有结果"
			}
			if a.dev.OpenFirst && !a.dev.opened && len(a.results) > 0 {
				a.dev.opened = true
				first := a.results[0]
				go func() { a.openComic(first) }()
			}
		})
	}()
}

// openComicAt opens a comic that may belong to a source other than the
// chosen one; the window follows the comic's source so the reader and the
// shelf keep recording the right place.
func (a *App) openComicAt(key string, c core.Comic) {
	if key != "" && key != a.currentKey() {
		a.mu.Lock()
		a.sourceKey = key
		query := a.query
		a.mu.Unlock()
		a.saveSettings(key, query)
	}
	a.openComic(c)
}

// openComic loads a comic's details.
func (a *App) openComic(c core.Comic) {
	key := a.currentKey()
	a.mu.Lock()
	a.open, a.detailComic, a.busy, a.errText, a.notice = true, c, "details", "", ""
	a.detail = core.Details{}
	a.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		d, err := a.core.Details(ctx, key, c.ID)
		a.win.Update(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			a.busy = ""
			if err != nil {
				a.errText = err.Error()
				return
			}
			a.detail = d
			if n := a.dev.OpenChapter; n >= 0 && n < len(d.Chapters) && !a.devChapterUsed {
				a.devChapterUsed = true
				go func() { a.openReader(n) }()
			} else if n := a.resumeChapter; n >= 0 && n < len(d.Chapters) {
				a.resumeChapter = -1
				go func() { a.openReader(n) }()
			}
		})
	}()
}

// openReader opens the reader window on one chapter of the open comic.
func (a *App) openReader(index int) {
	key := a.currentKey()
	a.mu.Lock()
	list := a.detail.Chapters
	comicID := a.detailComic.ID
	title := a.detailComic.Title
	a.readerChapter, a.readerList, a.readerComic = index, list, comicID
	a.readerTitle = title
	a.mu.Unlock()
	if a.readerWin == nil {
		a.readerWin = mygo.NewWindow(mygo.WindowOptions{
			Title: "阅读", Width: 820, Height: 900, Content: ui.View(a.readerView_),
		})
		view, err := a.readerWin.NewWebView(mygo.WebViewOptions{URL: "about:blank"})
		if err != nil {
			a.readerWin.SetTitle("阅读(无法打开阅读视图)")
			return
		}
		a.readerView = view
		a.readerWin.Show()
	}
	a.loadChapter(key)
}

// loadChapter points the reader at the chapter it should show.
func (a *App) loadChapter(key string) {
	if a.readerView == nil {
		return
	}
	a.mu.Lock()
	index := a.readerChapter
	title := a.readerTitle
	list := a.readerList
	a.mu.Unlock()
	u := a.readerURLFor(index)
	if u == "" {
		return
	}
	if index >= 0 && index < len(list) {
		title += " · " + list[index].Title
	}
	if err := a.readerView.Page().LoadURL(u); err != nil {
		log.Printf("nekophage: opening the reader: %v (%s)", err, u)
	}
	a.readerWin.SetTitle(title)
	a.rememberChapter(key)
	if a.dev.AutoNext && !a.devAutoUsed {
		a.devAutoUsed = true
		go func() {
			page := a.readerView.Page()
			time.Sleep(4 * time.Second)
			_, _ = page.Eval("localStorage.setItem('nkreader.auto','1'); location.reload(); 0")
			time.Sleep(10 * time.Second)
			_, _ = page.Eval("scrollTo(0, document.body.scrollHeight); 0")
			time.Sleep(2 * time.Second)
			_, _ = page.Eval("scrollTo(0, document.body.scrollHeight); 0")

		}()
	}
	if a.dev.KeepPage && !a.devKept {
		a.devKept = true
		go func() {
			time.Sleep(16 * time.Second)
			if _, err := a.readerView.Page().Eval("document.getElementById('keep').click(); 0"); err != nil {
				log.Printf("nekophage: keeping a page: %v", err)
			}

		}()
	}
	if a.dev.AdvanceChapter && !a.devAdvanced {
		a.devAdvanced = true
		go func() {
			time.Sleep(10 * time.Second)
			if _, err := a.readerView.Page().Eval("document.getElementById('next').click(); 0"); err != nil {
				log.Printf("nekophage: asking for the next chapter: %v", err)
			}
		}()
	}
	if pct := a.dev.ScrollPercent; pct > 0 {
		// The chapter grows as its lazy images take their room, so the
		// target moves: scroll a few times, as reading would.
		go func() {
			js := "scrollTo(0, (document.body.scrollHeight - innerHeight) * " +
				itoa(pct) + " / 100); 0"
			for i := 0; i < 4; i++ {
				time.Sleep(3 * time.Second)
				if _, err := a.readerView.Page().Eval(js); err != nil {
					log.Printf("nekophage: scrolling the reader: %v", err)
					return
				}
			}
		}()
	}
}

// readerURLFor is the reader's address for one chapter of the open comic,
// with the chapters on either side so the page can offer them.
func (a *App) readerURLFor(index int) string {
	a.mu.Lock()
	list, comic, base := a.readerList, a.readerComic, a.readerTitle
	a.mu.Unlock()
	if len(list) == 0 || index < 0 || index >= len(list) {
		return ""
	}
	var prev, next reader.Neighbour
	if index > 0 {
		ch := list[index-1]
		prev = reader.Neighbour{ID: ch.ID, Title: ch.Title}
	}
	if index+1 < len(list) {
		ch := list[index+1]
		next = reader.Neighbour{ID: ch.ID, Title: ch.Title}
	}
	title := base + " · " + list[index].Title
	return reader.ReaderURLNear(a.currentKey(), comic, list[index].ID, title, prev, next)
}

// step moves the reader to the next or previous chapter.
func (a *App) step(delta int) {
	a.mu.Lock()
	next := a.readerChapter + delta
	ok := next >= 0 && next < len(a.readerList)
	a.mu.Unlock()
	if !ok {
		return
	}
	a.mu.Lock()
	a.readerChapter = next
	key := a.currentKey()
	a.mu.Unlock()
	a.loadChapter(key)
}

// --- covers ---

// pageName is the cache key of one kept page's image.
func pageName(source, url string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(url))
	return "page/" + source + "/" + strconv.FormatUint(h.Sum64(), 16)
}

// pageImage fetches one kept page through the source's own rules, so a page
// that needs a Referer or a login still shows.
func (a *App) pageImage(source, comicID, chapterID, url string) {
	name := pageName(source, url)
	a.mu.Lock()
	if _, ok := a.covers[name]; ok || a.coverIn[name] || url == "" {
		a.mu.Unlock()
		return
	}
	a.coverIn[name] = true
	a.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		data, _, err := a.core.FetchImage(ctx, source, url, comicID, chapterID)
		a.win.Update(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			delete(a.coverIn, name)
			if err == nil && len(data) > 0 {
				a.covers[name] = data
			} else {
				a.covers[name] = nil
			}
		})
	}()
}

// cover asks for a cover's bytes once; the view decodes it when it shows.
func (a *App) cover(key, id, url string) {
	name := key + "/" + id
	a.mu.Lock()
	if _, ok := a.covers[name]; ok || a.coverIn[name] || url == "" {
		a.mu.Unlock()
		return
	}
	a.coverIn[name] = true
	a.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		data, ctype, err := a.core.FetchThumbnail(ctx, key, url, id)
		debugLog("cover %s/%s: %d bytes, %s, err %v", key, id, len(data), ctype, err)
		a.win.Update(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			delete(a.coverIn, name)
			if err == nil {
				a.covers[name] = data
			} else {
				a.covers[name] = nil // failed: do not try again this session
			}
		})
	}()
}

// bitmap decodes a cover once, for the image element. A cover that has not
// arrived yet must not be remembered as missing: the frame that first draws
// a card runs before the download finishes, so only decoded bitmaps and
// failed downloads are remembered.
func (a *App) bitmap(name string) ui.ImageSource {
	if b, ok := a.bitmaps[name]; ok {
		return b
	}
	data, ok := a.covers[name]
	if !ok || data == nil {
		return nil
	}
	b, err := ui.DecodeBitmap(data)
	if err != nil {
		debugLog("decode %s: %v", name, err)
		a.covers[name] = nil // a corrupt image is not decoded again
		return nil
	}
	w, h := b.Size()
	debugLog("decoded %s: %dx%d", name, w, h)
	a.bitmaps[name] = b
	return b
}

// --- view ---

func (a *App) view(c *ui.Context) {
	t := c.Theme()
	a.mu.Lock()
	sources, options, open, busy, errText, notice := a.sources, a.options, a.open, a.busy, a.errText, a.notice
	a.mu.Unlock()

	ui.Column(c).Fill().Children(func() {
		a.header(c, t, sources, options)

		if errText != "" {
			ui.Text(c, errText).TextColor(ui.RGB(224, 49, 49)).Padding(12, 16).Selectable()
		}
		if notice != "" {
			ui.Text(c, notice).TextColor(t.TextMuted).Padding(12, 16)
		}

		if !open && a.mode == 0 && a.filtersOpen {
			a.filtersView(c, t)
		}

		switch {
		case open:
			a.detailsView(c, t, busy == "details")
		case a.mode == 1:
			a.browseView(c, t)
		case a.mode == 2:
			a.shelfView(c, t)
		default:
			a.resultsView(c, t, busy == "search")
		}
	})
}

func (a *App) header(c *ui.Context, t *ui.Theme, sources []plugin.Record, options []string) {
	ui.Row(c).Padding(12, 16).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "Nekophage").FontSize(18).Bold()
		ui.Spacer(c)
		if len(sources) == 0 {
			ui.Text(c, "还没有漫画源").TextColor(t.TextMuted)
		} else {
			// The list shows each source's name; the index picks its key.
			sel := a.sourceLabel(sources)
			if ui.Select(c, &sel, options).Width(200).Changed() {
				if i := indexOfOption(options, sel); i >= 0 && i < len(sources) {
					a.mu.Lock()
					a.sourceKey = sources[i].Key
					a.results, a.open, a.notice, a.errText = nil, false, "", ""
					a.explorePages, a.exploreIndex, a.explorePage = nil, 0, 1
					a.explore = core.ExploreResult{}
					a.mu.Unlock()
					if a.mode == 1 {
						a.loadExplorePages()
					}
					a.loadSearchOptions(a.currentKey())
				}
			}
		}
		mode := a.mode
		if ui.Segmented(c, &mode, "搜索", "浏览", "书架").Width(200).Label("模式").Changed() {
			a.setMode(mode)
		}
		if mode == 0 {
			if ui.SearchField(c, &a.query).Width(280).Submitted() {
				a.startSearch()
			}
			if ui.Button(c, "搜索").Clicked() {
				a.startSearch()
			}
			if ui.Button(c, pickLabel(a.searchAll, "全部源", "当前源")).Clicked() {
				a.mu.Lock()
				a.searchAll = !a.searchAll
				all := a.searchAll
				query := a.query
				a.mu.Unlock()
				if all && strings.TrimSpace(query) != "" {
					a.startSearch()
				}
				a.win.Update(func() {})
			}
			if len(a.searchGroups) > 0 {
				if ui.Button(c, "筛选").Clicked() {
					a.mu.Lock()
					a.filtersOpen = !a.filtersOpen
					open := a.filtersOpen
					a.mu.Unlock()
					if open {
						// A source's filters are read once; opening the row
						// asks again if the source changed under it.
						if key := a.currentKey(); key != a.searchKey {
							a.loadSearchOptions(key)
						}
					}
					a.win.Update(func() {})
				}
			}
		}
		if ui.Button(c, "漫画源").Clicked() {
			a.openManage()
		}
	})
	ui.Divider(c)
}

// sourceLabel is the label of the chosen source in options.
func (a *App) sourceLabel(sources []plugin.Record) string {
	for _, s := range sources {
		if s.Key == a.sourceKey {
			if s.Enabled {
				return s.Name
			}
			return s.Name + "(已停用)"
		}
	}
	if len(sources) > 0 {
		return sources[0].Name
	}
	return ""
}

func indexOfOption(options []string, want string) int {
	for i, o := range options {
		if o == want {
			return i
		}
	}
	return -1
}

func (a *App) resultsView(c *ui.Context, t *ui.Theme, busy bool) {
	a.mu.Lock()
	results := a.results
	page, maxPage := a.page, a.maxPage
	a.mu.Unlock()

	if busy {
		ui.Row(c).Padding(16).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c)
			ui.Text(c, "正在搜索…").TextColor(t.TextMuted)
		})
		return
	}
	if len(results) == 0 {
		ui.Column(c).Fill().Center().Gap(10).Children(func() {
			ui.Text(c, "输入关键词搜索,或从“漫画源”添加一个源。").TextColor(t.TextMuted)
			a.recentSearches(c, t)
		})
		return
	}

	// Covers are requested as the grid builds their items.
	// Ids are only unique within a source, so a merged grid keys on both.
	a.grid.Key = func(i int) any {
		a.mu.Lock()
		defer a.mu.Unlock()
		if i < len(a.results) {
			source := ""
			if i < len(a.resultSources) {
				source = a.resultSources[i]
			}
			return source + "/" + a.results[i].ID
		}
		return i
	}
	key := a.currentKey()
	merged := len(a.resultSources) > 0
	a.grid.Selected = &a.gridIndex
	grid := ui.GridView(c, &a.grid, len(results), 150, 240, func(i int) {
		a.mu.Lock()
		comic := results[i]
		source := key
		if i < len(a.resultSources) {
			source = a.resultSources[i]
		}
		a.mu.Unlock()
		name := source + "/" + comic.ID
		a.cover(source, comic.ID, comic.Cover)
		ui.Column(c).Gap(6).Children(func() {
			ui.Box(c).Height(180).Radius(8).Background(t.Surface).Clip().Children(func() {
				if b := a.bitmap(name); b != nil {
					ui.Image(c, b).Fit(ui.Cover).Fill()
				}
			})
			ui.Text(c, comic.Title).SingleLine()
			if comic.Subtitle != "" {
				ui.Text(c, comic.Subtitle).FontSize(11).TextColor(t.TextMuted).SingleLine()
			}
			if merged {
				ui.Text(c, a.sourceName(source)).FontSize(10).TextColor(t.TextMuted).SingleLine()
			}
		})
	}).Grow(1).Padding(16).Gap(16)
	// A click opens a card, as a launcher does; Enter or a double click does
	// too, for keyboards.
	opened := grid.Submitted() || grid.Changed()
	// A click in the grid chooses a card; opening it follows.
	if opened && a.gridIndex >= 0 && a.gridIndex < len(results) {
		index := a.gridIndex
		a.openComicAt(a.sourceOf(index), results[index])
	}
	if merged {
		// Several sources at once: each shows its first page, so paging is
		// not offered rather than paging one source of the many.
		return
	}

	ui.Row(c).Padding(10, 16).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Textf(c, "第 %d 页", page).TextColor(t.TextMuted)
		ui.Spacer(c)
		if page > 1 {
			if ui.Button(c, "上一页").Clicked() {
				a.mu.Lock()
				query := a.query
				a.mu.Unlock()
				a.searchPage(key, query, page-1)
			}
		}
		if maxPage == 0 || page < maxPage {
			if ui.Button(c, "下一页").Clicked() {
				a.mu.Lock()
				query := a.query
				a.busy = "search"
				a.mu.Unlock()
				a.searchPage(key, query, page+1)
			}
		}
	})
}

// setMode switches between searching and browsing.
func (a *App) setMode(mode int) {
	a.mu.Lock()
	a.mode = mode
	a.open = false
	need := mode == 1 && len(a.explorePages) == 0
	a.mu.Unlock()
	if need {
		a.loadExplorePages()
	}
}

// loadExplorePages lists the current source's browses.
func (a *App) loadExplorePages() {
	key := a.currentKey()
	if key == "" {
		return
	}
	a.mu.Lock()
	a.exploreBusy, a.errText = true, ""
	a.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		pages, err := a.core.ExplorePages(ctx, key)
		a.win.Update(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			a.exploreBusy = false
			if err != nil {
				a.errText = err.Error()
				return
			}
			a.explorePages = pages
			a.exploreIndex = 0
			a.explore = core.ExploreResult{}
			a.explorePage = 1
		})
		if err == nil && len(pages) > 0 {
			// The tools may ask for a browse and a page; the first load after
			// the pages arrive is where that belongs.
			index, page := 0, 1
			a.mu.Lock()
			if !a.devExploreUsed && (a.dev.ExploreIndex > 0 || a.dev.ExplorePage > 1) {
				index, page = a.dev.ExploreIndex, max(1, a.dev.ExplorePage)
				a.devExploreUsed = true
			}
			a.mu.Unlock()
			a.loadExplore(key, index, page)
		}
	}()
}

// loadExplore loads one browse page.
func (a *App) loadExplore(key string, index, page int) {
	a.mu.Lock()
	a.exploreBusy, a.errText = true, ""
	a.exploreIndex, a.explorePage = index, page
	a.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		res, err := a.core.Explore(ctx, key, index, page)
		a.win.Update(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			a.exploreBusy = false
			if err != nil {
				a.errText = err.Error()
				return
			}
			a.explore = res
			if len(res.Sections) == 1 && res.Sections[0].Title != "" {
				// A single titled part reads better as its title above the
				// rows, so it stays as it is.
			}
		})
	}()
}

// browseView shows a source's browses: a tab per explore page, then what it
// returned.
func (a *App) browseView(c *ui.Context, t *ui.Theme) {
	a.mu.Lock()
	pages := a.explorePages
	index, page, busy := a.exploreIndex, a.explorePage, a.exploreBusy
	res := a.explore
	key := a.currentKey()
	a.mu.Unlock()

	if busy && len(pages) == 0 {
		ui.Row(c).Padding(16).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c)
			ui.Text(c, "正在读取分类…").TextColor(t.TextMuted)
		})
		return
	}
	if len(pages) == 0 {
		ui.Column(c).Fill().Center().Children(func() {
			ui.Text(c, "这个源没有分类浏览。").TextColor(t.TextMuted)
		})
		return
	}

	// The tabs of the browses the source offers. A source has a handful of
	// them, so a row is enough and costs less room than a scrolling strip.
	ui.Row(c).Padding(8, 16).Gap(6).AlignItems(ui.Center).Children(func() {
		for i, p := range pages {
			label := p.Title
			if label == "" {
				label = "分类 " + itoa(i+1)
			}
			if i == index {
				// The chosen browse reads as the filled one; clicking it
				// loads it again, which is what a reader expects.
				if ui.PrimaryButton(c, label).Clicked() {
					a.loadExplore(key, i, 1)
				}
				continue
			}
			if ui.Button(c, label).Clicked() {
				a.loadExplore(key, i, 1)
			}
		}
	})
	ui.Divider(c)

	if busy {
		ui.Row(c).Padding(16).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c)
			ui.Text(c, "正在加载…").TextColor(t.TextMuted)
		})
		return
	}
	if len(res.Sections) == 0 {
		ui.Column(c).Fill().Center().Children(func() {
			ui.Text(c, "这个分类没有内容。").TextColor(t.TextMuted)
		})
		return
	}

	ui.Scroll(c).Grow(1).Padding(12, 16).Children(func() {
		for _, section := range res.Sections {
			if section.Title != "" {
				ui.Text(c, section.Title).FontSize(14).Bold().Padding(8, 0)
			}
			for _, comic := range section.Comics {
				a.browseRow(c, t, key, comic)
			}
		}
		if res.MaxPage > 0 {
			ui.Row(c).Padding(8, 0).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Textf(c, "第 %d / %d 页", page, res.MaxPage).FontSize(12).TextColor(t.TextMuted)
				ui.Spacer(c)
				if page > 1 {
					if ui.Button(c, "上一页").Clicked() {
						a.loadExplore(key, index, page-1)
					}
				}
				if page < res.MaxPage {
					if ui.Button(c, "下一页").Clicked() {
						a.loadExplore(key, index, page+1)
					}
				}
			})
		}
	})
}

// browseRow is one comic in a browse list: a small cover, its title, and a
// button that opens it.
func (a *App) browseRow(c *ui.Context, t *ui.Theme, key string, comic core.Comic) {
	name := key + "/" + comic.ID
	a.cover(key, comic.ID, comic.Cover)
	ui.Row(c).Padding(6, 0).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Width(44).Height(60).Radius(4).Background(t.Surface).Clip().Children(func() {
			if b := a.bitmap(name); b != nil {
				ui.Image(c, b).Fit(ui.Cover).Fill()
			}
		})
		ui.Column(c).Gap(2).Children(func() {
			ui.Text(c, comic.Title).SingleLine()
			if comic.Subtitle != "" {
				ui.Text(c, comic.Subtitle).FontSize(11).TextColor(t.TextMuted).SingleLine()
			}
		}).Grow(1)
		if ui.Button(c, "打开").Clicked() {
			a.openComic(comic)
		}
	})
}

func itoa(n int) string { return fmt.Sprint(n) }

func (a *App) detailsView(c *ui.Context, t *ui.Theme, busy bool) {
	a.mu.Lock()
	d, comic, chapters := a.detail, a.detailComic, a.detail.Chapters
	key := a.currentKey()
	sh := a.shelf
	a.mu.Unlock()

	// The chapter read last, so the list shows where reading stopped even
	// after another chapter was opened and closed.
	var lastChapter shelf.History
	hasLast := false
	if sh != nil && comic.ID != "" {
		lastChapter, hasLast = sh.HistoryEntry(key, comic.ID)
	}

	if busy {
		ui.Row(c).Padding(16).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c)
			ui.Text(c, "正在打开…").TextColor(t.TextMuted)
		})
		return
	}

	ui.Row(c).Grow(1).AlignItems(ui.Start).Children(func() {
		// Cover and tags.
		ui.Column(c).Width(240).Padding(16).Gap(10).Children(func() {
			name := key + "/" + comic.ID
			a.cover(key, comic.ID, firstNonEmpty(d.Cover, comic.Cover))
			if b := a.bitmap(name); b != nil {
				ui.Image(c, b).Fit(ui.Cover).Width(200).Height(270).Radius(8)
			}
			for _, g := range d.Tags {
				ui.Text(c, g.Name).FontSize(11).TextColor(t.TextMuted)
				ui.Text(c, strings.Join(g.Values, "、")).FontSize(12)
			}
		})
		ui.Divider(c)
		// Title, description, chapters.
		ui.Column(c).Grow(1).Children(func() {
			ui.Column(c).Padding(16).Gap(8).Children(func() {
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, firstNonEmpty(d.Title, comic.Title)).FontSize(20).Bold()
					ui.Spacer(c)
					if a.shelf != nil {
						// The folder to keep this comic in; choosing another
						// one moves a comic that is already kept.
						names := a.folderNames()
						sel := a.favFolder
						if len(names) > 1 {
							if ui.Select(c, &sel, names).Width(130).Changed() {
								a.mu.Lock()
								a.favFolder = sel
								query := a.query
								a.mu.Unlock()
								a.saveSettings(key, query)
								if _, held := a.shelf.Holds(key, comic.ID); held {
									if _, err := a.shelf.Move(shelfComic(key, comic, d), sel); err != nil {
										a.setNotice("移动失败:" + err.Error())
									}
									a.win.Update(func() {})
								}
							}
						}
						label := "收藏"
						if _, held := a.shelf.Holds(key, comic.ID); held {
							label = "已收藏"
						}
						if ui.Button(c, label).Clicked() {
							a.keepComic()
						}
					}
					if ui.Button(c, "返回").Clicked() {
						a.mu.Lock()
						a.open = false
						a.mu.Unlock()
					}
				})
				if sub := firstNonEmpty(d.Subtitle, comic.Subtitle); sub != "" {
					ui.Text(c, sub).TextColor(t.TextMuted)
				}
				if d.Description != "" {
					// Sources write descriptions in Markdown; a long one
					// scrolls in a box of its own so the chapter list stays
					// in reach.
					ui.Scroll(c).MaxHeight(240).Padding(2).Children(func() {
						markdown.View(c, d.Description, &a.md)
					})
				}
				if len(chapters) == 0 {
					if ui.PrimaryButton(c, "开始阅读").Clicked() {
						a.mu.Lock()
						a.detail.Chapters = nil
						a.mu.Unlock()
						a.openReader(0)
					}
				}
			})
			ui.Divider(c)
			shown, matches := filterChapters(chapters, a.chapterQuery)
			ui.Row(c).Padding(6, 12).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Textf(c, "共 %d 章", len(chapters)).FontSize(12).TextColor(t.TextMuted)
				if a.chapterQuery != "" {
					ui.Textf(c, "· 找到 %d 章", len(matches)).FontSize(12).TextColor(t.TextMuted)
				}
				ui.SearchField(c, &a.chapterQuery).Placeholder("筛选章节标题").Label("筛选章节").Grow(1)
			})
			if len(matches) == 0 {
				ui.Text(c, "没有匹配的章节").Padding(12, 16).TextColor(t.TextMuted)
				return
			}
			ui.List(c, nil, len(shown), func(i int) {
				// shown keeps the chapter's index in the whole list, so the
				// reader opens the chapter the row names.
				idx := shown[i]
				ch := chapters[idx]
				label := ch.Title
				if ch.Group != "" {
					label = ch.Group + " · " + ch.Title
				}
				ui.Row(c).Padding(6, 16).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, label).Grow(1)
					// Mark where reading stopped: the recorded index, or the
					// same title when the list has shifted since.
					if hasLast && (idx == lastChapter.ChapterIndex || ch.Title == lastChapter.Chapter) {
						ui.Text(c, "读到").FontSize(11).TextColor(t.Accent)
					}
					if ui.Button(c, "阅读").Clicked() {
						a.openReader(idx)
					}
				})
			}).Grow(1)
		})
	})
}

// readerView_ draws the reader window: chapter controls over the WebView.
func (a *App) readerView_(c *ui.Context) {
	t := c.Theme()
	a.mu.Lock()
	index, list, title := a.readerChapter, a.readerList, a.readerTitle
	a.mu.Unlock()

	ui.Column(c).Fill().Children(func() {
		ui.Row(c).Padding(8, 12).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, title).FontSize(13).TextColor(t.TextMuted).SingleLine().Grow(1)
			prevOK := index > 0
			nextOK := index+1 < len(list)
			if ui.Button(c, "上一章").Disabled(!prevOK).Clicked() {
				a.step(-1)
			}
			if ui.Button(c, "下一章").Disabled(!nextOK).Clicked() {
				a.step(1)
			}
		})
		ui.Divider(c)
		if a.readerView != nil {
			ui.WebView(c, a.readerView).Grow(1)
		}
	})
}

// filterChapters returns the chapter indices whose title or group matches
// query, and the same as chapters for counting.
func filterChapters(chapters []core.Chapter, query string) ([]int, []core.Chapter) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		all := make([]int, len(chapters))
		for i := range chapters {
			all[i] = i
		}
		return all, chapters
	}
	var idx []int
	var found []core.Chapter
	for i, ch := range chapters {
		if strings.Contains(strings.ToLower(ch.Title), q) || strings.Contains(strings.ToLower(ch.Group), q) {
			idx = append(idx, i)
			found = append(found, ch)
		}
	}
	return idx, found
}

// debugLog prints when NKDEBUG is set, for the tools that check the interface.
func debugLog(format string, args ...any) {
	if os.Getenv("NKDEBUG") == "" {
		return
	}
	fmt.Fprintln(os.Stderr, fmt.Sprintf(format, args...))
}

// settings is what the window remembers between launches.
type settings struct {
	Source string `json:"source,omitempty"`
	Query  string `json:"query,omitempty"`
	// Folder is the favourite folder kept last, so the next comic goes there
	// without choosing it again.
	Folder string `json:"folder,omitempty"`
}

func (a *App) settingsPath() string { return filepath.Join(a.core.DataDir(), "ui.json") }

func (a *App) loadSettings() {
	s, ok := readSettings(a.settingsPath())
	if !ok {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if s.Source != "" {
		for _, src := range a.core.Plugins() {
			if src.Key == s.Source {
				a.sourceKey = s.Source
				break
			}
		}
	}
	if a.dev.Query == "" && s.Query != "" {
		a.query = s.Query
	}
	if s.Folder != "" {
		a.favFolder = s.Folder
	}
}

// saveSettings writes the source and query; a failure costs only the
// remembered state, so it is not reported.
func (a *App) saveSettings(source, query string) {
	a.saveSettingsTo(a.settingsPath(), source, query)
}

// saveSettingsTo writes the remembered state to a named file, which the
// tests use.
func (a *App) saveSettingsTo(path, source, query string) {
	a.mu.Lock()
	folder := a.favFolder
	a.mu.Unlock()
	writeSettings(path, settings{Source: source, Query: query, Folder: folder})
}

// readSettings loads what the window remembered. A missing or unreadable
// file is not an error: the window starts empty.
func readSettings(path string) (settings, bool) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return settings{}, false
	}
	var s settings
	if json.Unmarshal(data, &s) != nil {
		return settings{}, false
	}
	return s, true
}

func writeSettings(path string, s settings) {
	data, err := json.Marshal(s)
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, data, 0o600)
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}
