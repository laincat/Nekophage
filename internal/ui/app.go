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
	"math/rand"
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
	open   bool
	detail core.Details
	// detailCanComment is read once per comic: asking Core inside the render
	// path would run a source call on every repaint.
	detailCanComment bool
	detailComic      core.Comic

	// covers and bitmaps: downloaded bytes and the decoded images the grid
	// draws, bounded by imageCache so a long session does not keep every cover.
	images imageCache

	grid ui.GridState

	// browse: mode 0 searches, mode 1 browses a source's explore pages, mode 3
	// browses its category page.
	mode         int
	explorePages []core.ExplorePage
	exploreIndex int
	explore      core.ExploreResult
	explorePage  int
	exploreBusy  bool
	// category: a source's own category page. Each value of each part is its
	// own entry, because that is what the loader takes — a name and the
	// parameter it maps to — not one combination of every part.
	category  core.Category
	catLoaded bool
	catPart   int
	catValue  int
	catPage   int
	catMax    int
	// A ranking page is a second way into a source's comics: one flat option
	// plus paging, loaded by categoryComics.ranking.load.
	catRank       bool
	catRankOption string

	// A source's own favorites, the ones it keeps on the site. The client has a
	// local shelf too; this is a different list, and eleven official sources
	// declare one.
	// jsui is the handler Core asks for source UI; the window shows its spinners.
	jsui *JSHandler

	// The details view keeps the source's own favorites in reach: whether the
	// source supports them, which folders it has, and which of them hold this
	// comic.
	srcFav        core.FavoritesSupport
	srcFavLoaded  bool
	srcFavFolders map[string]string
	srcFavFolder  string
	srcFavHeld    []string
	// srcFavID is the source's own handle for this comic's favorite entry,
	// which some sources need to remove it.
	srcFavID   string
	srcFavBusy bool
	srcFavMsg  string

	favSup          core.FavoritesSupport
	favLoaded       bool
	favFolders      map[string]string
	sourceFavFolder string
	favComics       []core.Comic
	favPage         int
	favMax          int
	favBusy         bool
	favErr          string
	favGen          generation
	catComics       []core.Comic
	catOptions      []string
	catBusy         bool
	catGen          generation

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

	// reqSeq guards against a slow reply replacing a newer screen. Each new
	// search, detail or browse takes a ticket; a reply whose ticket is no
	// longer current is dropped instead of overwriting what the user is
	// looking at now.
	searchGen  generation
	detailGen  generation
	tagGen     generation
	exploreGen generation

	manageWin *mygo.Window
	// comments is the open comment thread, if any.
	comments commentsState
	// logs is the 源日志 panel's window.
	logs logPanel
	// aiWin is the connection to the Magpie installed on this machine.
	aiWin  *mygo.Window
	ai     aiState
	manage manageState
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
	// AI opens the AI window, which talks to the Magpie installed on this
	// machine. AITest also presses 测试连接 and 读取模型, so a capture shows a
	// real catalogue rather than the empty first frame.
	AI     bool
	AITest bool
	// AIPrompt, with -ai, sends this text once the connection and the
	// catalogue are in, so a capture shows a real streamed answer.
	AIPrompt string
	// AIModel fills 默认模型 before a send, since a task needs one.
	AIModel string
	// BackupCookies starts the sources window with the backup import's cookie
	// checkbox ticked, so a capture shows the state the checkbox reports rather
	// than only that it renders.
	BackupCookies bool
	// Category opens the source's category page before capturing, so the
	// tools can show it without a human clicking 分类.
	Category bool
	// Rank picks a ranking option after the category page opens, so a capture
	// shows the ranking list rather than only its buttons.
	Rank string
	// Favorites opens the source's own favorites, so a capture shows what the
	// source answers without a human clicking 源收藏.
	Favorites bool
	// Callback runs one of the source's own setting callbacks, so a tool can
	// check that the source's UI reaches it at all.
	Callback string
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
	// Favourite keeps the first search result, so the tools can check that
	// the result grid marks what is already on the shelf.
	Favourite bool
	// OpenChapter opens that chapter in the reader once details arrive; -1
	// leaves the reader closed. The tools use it to check the reader path.
	OpenChapter int
	// Browse opens the browse view instead of searching; ExploreIndex and
	// ExplorePage choose which browse and which page of it to load.
	Browse       bool
	ExploreIndex int
	ExplorePage  int

	opened     bool
	favourited bool
}

// New builds the app.
// SetJSUI gives the app the source UI handler, so the window can show what a
// source is waiting on.
func (a *App) SetJSUI(h *JSHandler) { a.jsui = h }

func New(c *core.Core, lib *library.Library, srv *reader.Server) *App {
	a := &App{core: c, lib: lib, scheme: srv,
		images:   newImageCache(),
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
			a.setNotice("读取书架失败:" + explain(err))
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
		if a.dev.BackupCookies {
			a.mu.Lock()
			a.manage.backupCookies = true
			a.mu.Unlock()
		}
		if a.dev.AI {
			a.openAI()
			// The window reads the stored connection first; connecting before
			// that would race it for the session.
			prompt := a.dev.AIPrompt
			test := a.dev.AITest
			model := a.dev.AIModel
			if test || prompt != "" {
				go func() {
					time.Sleep(600 * time.Millisecond)
					a.aiSaveAndTest()
					if prompt == "" {
						return
					}
					// 测试连接 and 读取模型 run in the background; the send needs
					// a session, so wait for one instead of sending into a nil.
					for i := 0; i < 100; i++ {
						time.Sleep(100 * time.Millisecond)
						a.mu.Lock()
						ready := a.ai.session != nil && !a.ai.running
						a.mu.Unlock()
						if ready {
							break
						}
					}
					a.mu.Lock()
					a.ai.prompt = prompt
					if model != "" {
						a.ai.model = model
					}
					a.mu.Unlock()
					a.aiSend()
				}()
			}
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
		if a.dev.Category {
			a.setMode(3)
		}
		if a.dev.Favorites {
			a.setMode(4)
		}
		if cb := a.dev.Callback; cb != "" {
			go func() {
				// The source must be installed and its declaration read first.
				time.Sleep(3 * time.Second)
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				answer, err := a.core.CallSettingCallback(ctx, a.currentKey(), cb)
				log.Printf("nekophage: callback %s -> %q err=%v", cb, answer, err)
			}()
		}
		if rank := a.dev.Rank; rank != "" {
			// The declaration loads as JS work, so give it a moment first.
			go func() {
				time.Sleep(2500 * time.Millisecond)
				a.loadRankingPage(a.currentKey(), rank, 1)
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

// LogWindow is the source log panel, or nil before it is opened.
func (a *App) LogWindow() *mygo.Window { return a.logs.win }

// AIWindow is the AI window, or nil before it is opened.
func (a *App) AIWindow() *mygo.Window { return a.aiWin }

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
	ticket := a.searchGen.next()
	a.mu.Unlock()
	a.saveSettings(key, q)
	if a.searches != nil {
		go func() { _ = a.searches.Add(q) }()
	}
	if a.searchAll {
		a.searchEverySource(q, ticket)
		return
	}
	a.searchPage(key, q, 1, ticket)
}

// quietSources says what happened to the sources that contributed nothing, so
// a source that errored is never reported as one that simply had no results.
func quietSources(empty, broken int) string {
	var parts []string
	if empty > 0 {
		parts = append(parts, itoa(empty)+" 个源没有结果")
	}
	if broken > 0 {
		parts = append(parts, itoa(broken)+" 个源出错")
	}
	if len(parts) == 0 {
		return ""
	}
	return ",其中 " + strings.Join(parts, "、")
}

// searchEverySource searches every enabled source at once and shows the
// results in one grid, each card knowing where it came from.
func (a *App) searchEverySource(query string, ticket int) {
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
			if !a.searchGen.current(ticket) {
				return
			}
			a.busy = ""
			a.results, a.resultSources = nil, nil
			empty, broken := 0, 0
			for i := range enabled {
				// A source that errored is not a source that found nothing:
				// one says the source is broken, the other says the query is.
				if failed[i] != "" {
					broken++
					continue
				}
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
			if len(a.results) == 0 {
				a.notice = "没有结果" + quietSources(empty, broken)
			} else if tail := quietSources(empty, broken); tail != "" {
				a.notice = itoa(len(a.results)) + " 部" + tail
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

// searchTicket is the ticket a pagination request belongs to. A new search has
// already replaced the results, so an old page must not be appended to a list
// it was never part of.
func (a *App) searchTicket() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.searchGen.peek()
}

// searchPage fetches one page of results off the window's thread. ticket names
// the search it belongs to; a reply for an older search is history.
func (a *App) searchPage(key, query string, page, ticket int) {
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
			if !a.searchGen.current(ticket) {
				// A newer search owns the screen; this reply is history.
				return
			}
			a.busy = ""
			if err != nil {
				a.errText = explain(err)
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
			if a.dev.Favourite && !a.dev.favourited && len(a.results) > 0 {
				a.dev.favourited = true
				first := a.results[0]
				if a.shelf != nil {
					if _, err := a.shelf.Add(shelf.DefaultFolder, shelfComic(key, first, core.Details{})); err != nil {
						a.setNotice(explain(err))
					}
				}
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
	a.detailCanComment = a.core.CommentsSupported(context.Background(), key)
	a.srcFavLoaded, a.srcFavFolders, a.srcFavHeld, a.srcFavMsg = false, nil, nil, ""
	ticket := a.detailGen.next()
	a.mu.Unlock()
	a.loadSourceFavorites(key, c.ID)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		d, err := a.core.Details(ctx, key, c.ID)
		a.win.Update(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			if !a.detailGen.current(ticket) {
				return
			}
			a.busy = ""
			if err != nil {
				a.errText = explain(err)
				return
			}
			a.detail = d
			if n := a.dev.OpenChapter; n >= 0 && !a.devChapterUsed && (n < len(d.Chapters) || len(d.Chapters) == 0 && n == 0) {
				a.devChapterUsed = true
				go func() { a.openReader(n) }()
			} else if n := a.resumeChapter; n >= 0 && n < len(d.Chapters) {
				a.resumeChapter = -1
				go func() { a.openReader(n) }()
			}
		})
	}()
}

// tagOutcome is what a tapped tag turns into: a search to run, or a sentence
// explaining why nothing runs.
//
// Opening a source's category page needs the parameter its own browse entry
// expects, which this view cannot pass yet; saying so beats searching for a
// word the source did not ask for.
func tagOutcome(action core.TagAction, tag string) (query, notice string) {
	switch action.Action {
	case "search":
		if q := strings.TrimSpace(action.Keyword); q != "" {
			return q, ""
		}
		return "", "这个源没有给出「" + tag + "」的搜索词"
	case "category":
		return "", "这个源把「" + tag + "」归到分类页,当前版本还打不开分类参数"
	}
	return "", "这个源没有说明「" + tag + "」该做什么"
}

// tagClicked asks the source what a tapped tag means.
//
// The meaning belongs to the source: copy_manga turns an author into a search
// for "作者:<name>", manhuagui turns a 类型 into a category parameter, and a
// source that does not handle the namespace throws with its own message. That
// message is shown as it is, because it is the only honest answer.
func (a *App) tagClicked(namespace, tag string) {
	key := a.currentKey()
	a.mu.Lock()
	ticket := a.tagGen.next()
	a.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		action, err := a.core.ClickTag(ctx, key, namespace, tag)
		a.win.Update(func() {
			a.mu.Lock()
			current := a.tagGen.current(ticket)
			a.mu.Unlock()
			if !current {
				return
			}
			if err != nil {
				a.mu.Lock()
				a.notice = explain(err)
				a.mu.Unlock()
				return
			}
			query, notice := tagOutcome(action, tag)
			if query != "" {
				a.mu.Lock()
				a.query = query
				a.mu.Unlock()
				a.startSearch()
				return
			}
			a.mu.Lock()
			a.notice = notice
			a.mu.Unlock()
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
	if len(list) == 0 {
		// A source may describe a comic with no chapter list at all: the
		// gallery is the single reading unit and its loadEp takes an empty
		// chapter id. Without this the detail view offers 开始阅读 and the
		// reader then loads nothing, so such a source can never be read.
		if index != 0 {
			return ""
		}
		return reader.ReaderURLNear(a.currentKey(), comic, "", base,
			reader.Neighbour{}, reader.Neighbour{})
	}
	if index < 0 || index >= len(list) {
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
	start := a.images.want(name, url)
	a.mu.Unlock()
	if !start {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		data, _, err := a.core.FetchImage(ctx, source, url, comicID, chapterID)
		a.win.Update(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			if err == nil && len(data) > 0 {
				a.images.finishCover(name, data)
			} else {
				a.images.finishCover(name, nil)
			}
		})
	}()
}

// cover asks for a cover's bytes once; the view decodes it when it shows.
func (a *App) cover(key, id, url string) {
	name := key + "/" + id
	a.mu.Lock()
	start := a.images.want(name, url)
	a.mu.Unlock()
	if !start {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		data, ctype, err := a.core.FetchThumbnail(ctx, key, url, id)
		debugLog("cover %s/%s: %d bytes, %s, err %v", key, id, len(data), ctype, err)
		a.win.Update(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			if err == nil {
				a.images.finishCover(name, data)
			} else {
				// Failed: remembered so it is not retried this session.
				a.images.finishCover(name, nil)
			}
		})
	}()
}

// bitmap decodes a cover once, for the image element. A cover that has not
// arrived yet must not be remembered as missing: the frame that first draws
// a card runs before the download finishes, so only decoded bitmaps and
// failed downloads are remembered.
func (a *App) bitmap(name string) ui.ImageSource {
	return a.images.decoded(name)
}

// --- view ---

func (a *App) view(c *ui.Context) {
	t := c.Theme()
	a.mu.Lock()
	sources, options, open, busy, errText, notice := a.sources, a.options, a.open, a.busy, a.errText, a.notice
	a.mu.Unlock()
	loading := 0
	if a.jsui != nil {
		loading = a.jsui.LoadingCount()
	}

	ui.Column(c).Fill().Children(func() {
		a.header(c, t, sources, options)

		if errText != "" {
			ui.Text(c, errText).TextColor(ui.RGB(224, 49, 49)).Padding(12, 16).Selectable()
		}
		if notice != "" {
			ui.Text(c, notice).TextColor(t.TextMuted).Padding(12, 16)
		}
		if loading > 0 {
			// A source asked for a spinner and has not cancelled it: the work is
			// still running somewhere, so say so rather than looking idle.
			ui.Row(c).Padding(8, 16).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c)
				ui.Text(c, "源正在处理…").FontSize(12).TextColor(t.TextMuted)
			})
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
		case a.mode == 3:
			a.categoryView(c, t)
		case a.mode == 4:
			a.favoritesView(c, t)
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
					a.favGen.next()
					a.favLoaded, a.favBusy = false, false
					a.favFolders, a.favComics, a.favErr = nil, nil, ""
					a.mu.Unlock()
					if a.mode == 4 {
						a.loadFavorites(a.currentKey())
					}
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
		if ui.Button(c, "分类").Clicked() {
			a.setMode(3)
		}
		if ui.Button(c, "源收藏").Clicked() {
			a.setMode(4)
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
		if ui.Button(c, "AI").Clicked() {
			a.openAI()
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
	// The cell has to fit the cover, the title and — in a merged search — the
	// source name. At 240 the name was clipped off below the first row, and a
	// result looked like it had come from nowhere.
	grid := ui.GridView(c, &a.grid, len(results), 150, 260, func(i int) {
		a.mu.Lock()
		comic := results[i]
		source := key
		if i < len(a.resultSources) {
			source = a.resultSources[i]
		}
		a.mu.Unlock()
		// Result cards say when the comic is already on the shelf, so the
		// reader does not look for it twice.
		held := false
		if a.shelf != nil {
			_, held = a.shelf.Holds(source, comic.ID)
		}
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
			if held {
				ui.Text(c, "已在书架").FontSize(10).TextColor(t.Accent).SingleLine()
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
				a.searchPage(key, query, page-1, a.searchTicket())
			}
		}
		if maxPage == 0 || page < maxPage {
			if ui.Button(c, "下一页").Clicked() {
				a.mu.Lock()
				query := a.query
				a.busy = "search"
				a.mu.Unlock()
				a.searchPage(key, query, page+1, a.searchTicket())
			}
		}
	})
}

// loadCategoryDecl reads the current source's category page and shows its
// first entry. A source without one leaves the mode alone: the button is not
// drawn for it.
func (a *App) loadCategoryDecl(key string) {
	a.mu.Lock()
	a.catBusy, a.errText = true, ""
	ticket := a.catGen.next()
	a.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		cat, err := a.core.CategoryPage(ctx, key)
		a.win.Update(func() {
			a.mu.Lock()
			if !a.catGen.current(ticket) {
				a.mu.Unlock()
				return
			}
			a.catBusy = false
			if err != nil {
				a.errText = explain(err)
				a.mu.Unlock()
				return
			}
			a.category = cat
			a.catLoaded = true
			a.catPart, a.catValue, a.catPage, a.catMax = 0, 0, 1, 0
			a.catComics = nil
			// The options start from what the source declared, one value per
			// group, exactly like a search's filters.
			a.catOptions = nil
			for _, g := range cat.Options {
				a.catOptions = append(a.catOptions, g.Default)
			}
			a.mu.Unlock()
			if len(cat.Parts) > 0 && len(cat.Parts[0].Categories) > 0 {
				a.loadCategoryPage(key, 0, 0, 1)
			}
		})
	}()
}

// loadCategoryPage loads one entry of the category page.
func (a *App) loadCategoryPage(key string, part, value, page int) {
	a.mu.Lock()
	if part < 0 || part >= len(a.category.Parts) {
		a.mu.Unlock()
		return
	}
	categories := a.category.Parts[part].Categories
	if value < 0 || value >= len(categories) {
		a.mu.Unlock()
		return
	}
	p := a.category.Parts[part]
	// The loader's first argument is the chosen value, not the dimension's
	// name: nhentai looks the value up ("Chinese"), and only the parameter
	// says which group it belongs to.
	name := categories[value]
	param := categories[value]
	// A source may map each value onto its own parameter; a part with a
	// groupParam wants that name instead (nhentai's loader switches on
	// "language"); otherwise the value itself is the parameter.
	if len(p.Params) == len(p.Categories) {
		param = p.Params[value]
	} else if p.GroupParam != "" {
		param = p.GroupParam
	}
	options := a.catOptions
	a.catBusy, a.errText = true, ""
	a.catRank = false
	a.catPart, a.catValue, a.catPage = part, value, page
	ticket := a.catGen.next()
	a.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		res, err := a.core.CategoryComics(ctx, key, name, param, options, page)
		a.win.Update(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			if !a.catGen.current(ticket) {
				return
			}
			a.catBusy = false
			if err != nil {
				a.errText = explain(err)
				return
			}
			a.catComics, a.catMax = res.Comics, res.MaxPage
		})
	}()
}

// loadRankingPage reads one page of the source's ranking page, which is a flat
// option list rather than dimensions.
func (a *App) loadRankingPage(key, option string, page int) {
	if key == "" || option == "" {
		return
	}
	a.mu.Lock()
	a.catBusy, a.errText = true, ""
	a.catRank, a.catRankOption, a.catPage, a.catMax = true, option, page, 0
	ticket := a.catGen.next()
	a.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		res, err := a.core.RankingComics(ctx, key, option, page)
		a.win.Update(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			if !a.catGen.current(ticket) {
				return
			}
			a.catBusy = false
			if err != nil {
				a.errText = explain(err)
				return
			}
			a.catComics, a.catMax = res.Comics, res.MaxPage
		})
	}()
}

// chooseCatOption changes one of the loader's options and reloads the entry
// that is open, so the choice takes effect the way a search filter does.
func (a *App) chooseCatOption(group int, key string, multi bool) {
	a.mu.Lock()
	if group < len(a.catOptions) {
		if multi {
			current := multiValues(a.catOptions[group])
			out := make([]string, 0, len(current)+1)
			had := false
			for _, v := range current {
				if v == key {
					had = true
					continue
				}
				out = append(out, v)
			}
			if !had {
				out = append(out, key)
			}
			a.catOptions[group] = encodeMulti(out)
		} else {
			a.catOptions[group] = key
		}
	}
	part, value, page := a.catPart, a.catValue, a.catPage
	a.mu.Unlock()
	if a.win != nil {
		a.win.Update(func() {})
	}
	a.loadCategoryPage(a.currentKey(), part, value, page)
}

// randomCategory searches for one value of a random part.
//
// The declared randomNumber is not the number of values to join: the source's
// own note says it is "the number of comics to display at the same time", and
// a comic search treats several tags as all-of-them. Joining twenty nhentai
// tags asks for works carrying all twenty, which is nothing: measured on the
// live site, eight tags still answered 20 comics, ten answered none, and twenty
// were rejected outright. One value per click is what the site accepts, and
// clicking again explores another.
func (a *App) randomCategory(part core.CategoryPart) {
	values := append([]string(nil), part.Categories...)
	if len(values) == 0 {
		return
	}
	rand.Shuffle(len(values), func(i, j int) { values[i], values[j] = values[j], values[i] })
	a.mu.Lock()
	a.query = values[0]
	a.mu.Unlock()
	a.setMode(0)
	a.startSearch()
}

// categoryView shows a source's category page: its parts, its options, then the
// comics the chosen entry returned.
func (a *App) categoryView(c *ui.Context, t *ui.Theme) {
	a.mu.Lock()
	cat, loaded, busy := a.category, a.catLoaded, a.catBusy
	part, value, page, maxPage := a.catPart, a.catValue, a.catPage, a.catMax
	rank, rankOption := a.catRank, a.catRankOption
	comics := a.catComics
	key := a.currentKey()
	a.mu.Unlock()

	if !loaded {
		ui.Row(c).Padding(16).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c)
			ui.Text(c, "正在读取分类…").TextColor(t.TextMuted)
		})
		return
	}
	if len(cat.Parts) == 0 {
		ui.Column(c).Fill().Center().Children(func() {
			ui.Text(c, "这个源没有分类页。").TextColor(t.TextMuted)
		})
		return
	}
	if cat.Title != "" {
		ui.Text(c, cat.Title).FontSize(14).Bold().Padding(10, 16)
	}
	// One row per part. Each value is its own entry, not a filter to combine:
	// the loader takes one name and one parameter.
	for i, p := range cat.Parts {
		i, p := i, p
		ui.Row(c).Padding(6, 16).Gap(6).Wrap().AlignItems(ui.Center).Children(func() {
			ui.Text(c, p.Name).FontSize(12).TextColor(t.TextMuted).Width(64)
			if p.Type == "random" {
				// A random part is not loaded: it searches for one of its
				// values, which is what the site accepts (see randomCategory).
				if ui.Button(c, "随机一个标签").Clicked() {
					a.randomCategory(p)
				}
				return
			}
			for j, name := range p.Categories {
				j, name := j, name
				if i == part && j == value {
					if ui.PrimaryButton(c, name).Clicked() {
						a.loadCategoryPage(key, i, j, 1)
					}
					continue
				}
				if ui.Button(c, name).Clicked() {
					a.loadCategoryPage(key, i, j, 1)
				}
			}
		})
	}
	// The ranking page, when the source declares one: one flat option, and the
	// same list and pager as a dimension.
	if cat.Ranking != nil && len(cat.Ranking.Options) > 0 {
		ui.Row(c).Padding(6, 16).Gap(6).Wrap().AlignItems(ui.Center).Children(func() {
			ui.Text(c, "排行").FontSize(12).TextColor(t.TextMuted).Width(64)
			for _, o := range cat.Ranking.Options {
				o := o
				if rank && rankOption == o.Value {
					if ui.PrimaryButton(c, o.Label).Clicked() {
						a.loadRankingPage(key, o.Value, 1)
					}
					continue
				}
				if ui.Button(c, o.Label).Clicked() {
					a.loadRankingPage(key, o.Value, 1)
				}
			}
		})
	}
	// The loader's own options, drawn the way a search's filters are.
	a.mu.Lock()
	groups := append([]core.SearchOptionGroup(nil), cat.Options...)
	values := append([]string(nil), a.catOptions...)
	a.mu.Unlock()
	if len(groups) > 0 {
		optionChips(c, t, groups, values, a.chooseCatOption)
	}
	ui.Divider(c)

	if busy {
		ui.Row(c).Padding(16).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c)
			ui.Text(c, "正在加载…").TextColor(t.TextMuted)
		})
		return
	}
	if len(comics) == 0 {
		ui.Column(c).Fill().Center().Children(func() {
			ui.Text(c, "这个分类没有内容。").TextColor(t.TextMuted)
		})
		return
	}
	ui.Scroll(c).Grow(1).Padding(12, 16).Children(func() {
		for _, comic := range comics {
			a.browseRow(c, t, key, comic)
		}
		if maxPage > 0 {
			ui.Row(c).Padding(8, 0).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Textf(c, "第 %d / %d 页", page, maxPage).FontSize(12).TextColor(t.TextMuted)
				ui.Spacer(c)
				if page > 1 {
					if ui.Button(c, "上一页").Clicked() {
						if rank {
							a.loadRankingPage(key, rankOption, page-1)
						} else {
							a.loadCategoryPage(key, part, value, page-1)
						}
					}
				}
				if page < maxPage {
					if ui.Button(c, "下一页").Clicked() {
						if rank {
							a.loadRankingPage(key, rankOption, page+1)
						} else {
							a.loadCategoryPage(key, part, value, page+1)
						}
					}
				}
			})
		}
	})
}

// setMode switches between searching and browsing.
func (a *App) setMode(mode int) {
	a.mu.Lock()
	a.mode = mode
	a.open = false
	need := mode == 1 && len(a.explorePages) == 0
	needCategory := mode == 3 && !a.catLoaded
	needFavorites := mode == 4 && !a.favLoaded
	a.mu.Unlock()
	if need {
		a.loadExplorePages()
	}
	if needCategory {
		a.loadCategoryDecl(a.currentKey())
	}
	if needFavorites {
		a.loadFavorites(a.currentKey())
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
	ticket := a.exploreGen.next()
	a.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		pages, err := a.core.ExplorePages(ctx, key)
		a.win.Update(func() {
			a.mu.Lock()
			if !a.exploreGen.current(ticket) {
				a.mu.Unlock()
				return
			}
			a.exploreBusy = false
			if err != nil {
				a.errText = explain(err)
				a.mu.Unlock()
				return
			}
			a.explorePages = pages
			a.exploreIndex = 0
			a.explore = core.ExploreResult{}
			a.explorePage = 1
			index, page := 0, 1
			if !a.devExploreUsed && (a.dev.ExploreIndex > 0 || a.dev.ExplorePage > 1) {
				index, page = a.dev.ExploreIndex, max(1, a.dev.ExplorePage)
				a.devExploreUsed = true
			}
			a.mu.Unlock()
			// Commit the list before advancing the generation for its first
			// page; Update may schedule this callback asynchronously.
			if len(pages) > 0 {
				a.loadExplore(key, index, page)
			}
		})
	}()
}

// loadExplore loads one browse page. cursor is the token a browse that pages by
// token handed back last time; "" starts at the beginning.
func (a *App) loadExplore(key string, index, page int) {
	a.loadExploreAt(key, index, page, "")
}

func (a *App) loadExploreAt(key string, index, page int, cursor string) {
	a.mu.Lock()
	a.exploreBusy, a.errText = true, ""
	a.exploreIndex, a.explorePage = index, page
	ticket := a.exploreGen.next()
	a.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		res, err := a.core.Explore(ctx, key, index, page, cursor)
		a.win.Update(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			if !a.exploreGen.current(ticket) {
				return
			}
			a.exploreBusy = false
			if err != nil {
				a.errText = explain(err)
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
		if res.MaxPage > 0 || res.Next != "" {
			ui.Row(c).Padding(8, 0).Gap(10).AlignItems(ui.Center).Children(func() {
				if res.Next != "" {
					// A cursor browse has no page numbers to show; the count of
					// what has been fetched is the honest thing here.
					ui.Textf(c, "第 %d 批", page).FontSize(12).TextColor(t.TextMuted)
				} else {
					ui.Textf(c, "第 %d / %d 页", page, res.MaxPage).FontSize(12).TextColor(t.TextMuted)
				}
				ui.Spacer(c)
				if page > 1 {
					if ui.Button(c, "上一页").Clicked() {
						a.loadExplore(key, index, page-1)
					}
				}
				if res.Next != "" {
					if ui.Button(c, "下一页").Clicked() {
						a.loadExploreAt(key, index, page+1, res.Next)
					}
				} else if page < res.MaxPage {
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
	srcFav, srcFavFolders, srcFavHeld := a.srcFav, a.srcFavFolders, a.srcFavHeld
	srcFavFolder, srcFavMsg, srcFavBusy := a.srcFavFolder, a.srcFavMsg, a.srcFavBusy
	canComment := a.detailCanComment
	a.mu.Unlock()

	// The chapter read last, so the list shows where reading stopped even
	// after another chapter was opened and closed.
	var lastChapter shelf.History
	if sh != nil && comic.ID != "" {
		if e, ok := sh.HistoryEntry(key, comic.ID); ok {
			lastChapter = e
		}
	}
	// One row carries the marker: the recorded index while it still names
	// the same chapter, otherwise the first row with that title, since a
	// source may repeat a title across groups.
	markIndex := -1
	if lastChapter.Chapter != "" || lastChapter.Time != 0 {
		if i := lastChapter.ChapterIndex; i >= 0 && i < len(chapters) && chapters[i].Title == lastChapter.Chapter {
			markIndex = i
		} else {
			for i, ch := range chapters {
				if ch.Title == lastChapter.Chapter {
					markIndex = i
					break
				}
			}
		}
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
				// A tag is a way into the source's own search or category
				// page, so each value is a button rather than joined text.
				ui.Row(c).Gap(6).Wrap().Children(func() {
					for _, value := range g.Values {
						value, namespace := value, g.Name
						if ui.Button(c, value).Clicked() {
							a.tagClicked(namespace, value)
						}
					}
				})
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
										a.setNotice("移动失败:" + explain(err))
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
					// The source's own favorites are a different list from the
					// local shelf, so they get their own control rather than
					// sharing the 收藏 button.
					// The source's own favorites are a different list from the
					// local shelf, so they get their own control rather than
					// sharing the 收藏 button.
					if srcFav.Supported {
						if srcFav.MultiFolder && len(srcFavFolders) > 1 {
							labels, byLabel, labelOf := folderPicker(srcFavFolders)
							pick := labelOf[srcFavFolder]
							if pick == "" && len(labels) > 0 {
								pick = labels[0]
							}
							if ui.Select(c, &pick, labels).Width(120).Changed() {
								a.mu.Lock()
								a.srcFavFolder = byLabel[pick]
								a.mu.Unlock()
							}
						}
						label := "收藏到源"
						if len(srcFavHeld) > 0 {
							label = "已在源收藏"
						}
						if srcFavBusy {
							ui.Spinner(c)
						} else if ui.Button(c, label).Clicked() {
							a.toggleSourceFavorite()
						}
					}
					// A source that keeps comments gets a way in; one that does
					// not keeps the control away instead of showing an empty
					// thread. The flag is read when the comic opens, not here.
					if canComment {
						if ui.Button(c, "评论").Clicked() {
							a.openComments()
						}
					}
					if ui.Button(c, "返回").Clicked() {
						a.mu.Lock()
						a.open = false
						a.mu.Unlock()
					}
				})
				if srcFavMsg != "" {
					ui.Text(c, srcFavMsg).FontSize(11).Selectable()
				}
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
				if len(chapters) == 0 {
					// A gallery source declares no chapter list: the work is
					// the single reading unit, so a chapter count and a
					// chapter filter would both describe nothing.
					ui.Text(c, "整部作品是一个阅读单元").FontSize(12).TextColor(t.TextMuted)
					return
				}
				ui.Textf(c, "共 %d 章", len(chapters)).FontSize(12).TextColor(t.TextMuted)
				if a.chapterQuery != "" {
					ui.Textf(c, "· 找到 %d 章", len(matches)).FontSize(12).TextColor(t.TextMuted)
				}
				// Where reading stopped, with a way back into it.
				if markIndex >= 0 {
					switch {
					case lastChapter.Page > 0 && lastChapter.Percent > 0:
						ui.Textf(c, "· 上次读到 第 %d 页 · %d%%", lastChapter.Page, lastChapter.Percent).FontSize(12).TextColor(t.Accent)
					case lastChapter.Page > 0:
						ui.Textf(c, "· 上次读到 第 %d 页", lastChapter.Page).FontSize(12).TextColor(t.Accent)
					default:
						ui.Text(c, "· 有未读完的记录").FontSize(12).TextColor(t.Accent)
					}
					if ui.Button(c, "继续阅读").Clicked() {
						a.openReader(markIndex)
					}
				}
				ui.SearchField(c, &a.chapterQuery).Placeholder("筛选章节标题").Label("筛选章节").Grow(1)
			})
			if len(chapters) == 0 {
				// The work is one unit and the reader already has a way into
				// it above, so there is no chapter list to draw.
				return
			}
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
					// Mark where reading stopped.
					if idx == markIndex {
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
