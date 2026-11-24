package ui

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/laincat/nekophage/internal/adapt"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/novel"
	"github.com/laincat/nekophage/internal/shelf"
	"github.com/laincat/veneracat-core/plugin"
)

type novelScreen int

const (
	novelCatalog novelScreen = iota
	novelBooks
	novelDetails
	novelReading
	novelAdapt
)

// novelPanel belongs to the UI thread. Workers hand results back through post.
// The panel outlives its window so an accepted shelf write can finish on close.
type novelPanel struct {
	core   *core.Core
	shelf  *shelf.Shelf
	post   func(func())
	gen    generation
	writes sync.WaitGroup

	sources        []plugin.Record
	key            string
	query          string
	searched       string
	catalog        *core.NovelPage
	catalogPage    int
	screen         novelScreen
	detailFrom     novelScreen
	book           shelf.Novel
	detail         *core.NovelDetail
	chapter        int
	chapterContent core.NovelChapterContent
	pages          []string
	page           int
	entries        []shelf.NovelEntry
	busy           bool
	err            error
	saving         bool
	shelfErr       error
	adapt          novelAdaptState
	adaptMgrMu     sync.Mutex
	adaptMgr       *adapt.Manager
}

func (a *App) initNovelPanel(shelfErr error) {
	a.novels = novelPanel{core: a.core, shelf: a.shelf, shelfErr: shelfErr, post: a.updateNovel}
	if a.shelf != nil {
		a.novels.entries = a.shelf.Novels()
	}
}

func (a *App) updateNovel(fn func()) {
	a.win.Update(func() {
		fn()
		if a.novelWin != nil {
			a.novelWin.Invalidate()
		}
	})
}

func (a *App) openNovel() {
	if a.novelWin != nil {
		a.novelWin.Show()
		a.novelWin.Focus()
		return
	}
	p := &a.novels
	win := mygo.NewWindow(mygo.WindowOptions{
		Title: "小说", Width: 860, Height: 720, Content: ui.View(p.view),
	})
	a.novelWin = win
	win.OnClosed(func() {
		a.novelWin = nil
		p.clearRequest()
		p.adapt.reset()
		p.screen, p.key, p.query = novelCatalog, "", ""
		p.catalog, p.detail, p.pages, p.chapterContent = nil, nil, nil, core.NovelChapterContent{}
	})
	a.refreshNovelSources()
	win.Show()
}

func (a *App) refreshNovelSources() {
	if a.novelWin == nil {
		return
	}
	var sources []plugin.Record
	for _, rec := range a.lib.Sources() {
		if rec.Type == plugin.TypeNovelJS && rec.Enabled {
			sources = append(sources, rec)
		}
	}
	a.novels.setSources(sources)
	a.novelWin.Invalidate()
}

func (p *novelPanel) setSources(sources []plugin.Record) {
	p.sources = sources
	changed := !p.hasSource(p.key)
	if changed {
		p.key, p.query, p.catalog = "", "", nil
		if len(sources) > 0 {
			p.key = sources[0].Key
		}
	}
	if p.detail != nil && !p.hasSource(p.book.Source) {
		p.backTo(p.detailFrom)
	} else if changed && p.screen == novelCatalog {
		p.backTo(novelCatalog)
	}
}

func (p *novelPanel) hasSource(key string) bool {
	return slices.ContainsFunc(p.sources, func(rec plugin.Record) bool { return rec.Key == key })
}

func (p *novelPanel) clearRequest() {
	p.gen.next()
	p.busy, p.err = false, nil
}

func (p *novelPanel) backTo(screen novelScreen) {
	p.clearRequest()
	p.adapt.reset()
	p.screen, p.pages, p.chapterContent = screen, nil, core.NovelChapterContent{}
	if screen != novelDetails {
		p.detail = nil
	}
	if screen == novelCatalog && p.catalog == nil && p.key != "" {
		p.loadCatalog("", 1)
	}
}

// request returns an action for the UI thread. A cancelled or replaced
// request cannot publish an error, open a chapter or write reading progress.
func (p *novelPanel) request(work func(context.Context) (func(), error)) {
	ticket := p.gen.next()
	ctx, cancel := context.WithTimeout(p.gen.requestContext(ticket), 90*time.Second)
	p.busy, p.err = true, nil
	go func() {
		defer cancel()
		apply, err := work(ctx)
		if err == nil {
			err = ctx.Err()
		}
		p.post(func() {
			if !p.gen.current(ticket) {
				return
			}
			p.busy, p.err = false, err
			if err == nil {
				apply()
			}
		})
	}()
}

func (p *novelPanel) loadCatalog(query string, number int) {
	key := p.key
	if key == "" {
		return
	}
	query = strings.TrimSpace(query)
	p.request(func(ctx context.Context) (func(), error) {
		var page *core.NovelPage
		var err error
		if query == "" {
			page, err = p.core.NovelPopular(ctx, key, number)
		} else {
			page, err = p.core.NovelSearch(ctx, key, query, number)
		}
		return func() {
			p.catalog, p.searched, p.catalogPage = page, query, number
		}, err
	})
}

func (p *novelPanel) openBook(book shelf.Novel) {
	from := p.screen
	p.request(func(ctx context.Context) (func(), error) {
		detail, err := p.core.NovelDetail(ctx, book.Source, book.ID)
		return func() {
			if !p.hasSource(book.Source) {
				p.err = errors.New("小说源已停用或移除")
				return
			}
			if detail.Title != "" {
				book.Title = detail.Title
			}
			if detail.Author != "" {
				book.Author = detail.Author
			}
			p.book, p.detail, p.detailFrom, p.screen = book, detail, from, novelDetails
		}, err
	})
}

func (p *novelPanel) savedBook() shelf.NovelEntry {
	for _, entry := range p.entries {
		if entry.Source == p.book.Source && entry.ID == p.book.ID {
			return entry
		}
	}
	return shelf.NovelEntry{}
}

func (p *novelPanel) openChapter(index int) {
	if p.detail == nil || index < 0 || index >= len(p.detail.Chapters) || p.saving {
		return
	}
	source, id, chapter := p.book.Source, p.detail.ID, p.detail.Chapters[index]
	p.request(func(ctx context.Context) (func(), error) {
		content, err := p.core.NovelChapter(ctx, source, id, chapter.ID)
		if err != nil {
			return nil, err
		}
		paragraphs, err := novel.Paragraphs(content.HTML)
		if err != nil {
			return nil, err
		}
		pages := novel.Pages(paragraphs)
		if len(pages) == 0 {
			return nil, errors.New("章节没有可阅读的正文")
		}
		return func() {
			page := 0
			if saved := p.savedBook(); saved.ChapterID == chapter.ID {
				page = max(0, min(saved.Page, len(pages)-1))
			}
			p.screen, p.chapter, p.pages, p.chapterContent = novelReading, index, pages, *content
			p.readPage(page)
		}, nil
	})
}

func (p *novelPanel) readPage(page int) {
	if page < 0 || page >= len(p.pages) || p.saving {
		return
	}
	p.page = page
	book, chapterID := p.book, p.detail.Chapters[p.chapter].ID
	p.save(func() error { return p.shelf.RecordNovelProgress(book, chapterID, page) })
}

func (p *novelPanel) keepBook(keep bool) {
	book := p.book
	p.save(func() error { return p.shelf.KeepNovel(book, keep) })
}

func (p *novelPanel) save(write func() error) {
	if p.shelf == nil || p.saving {
		return
	}
	p.saving, p.shelfErr = true, nil
	p.writes.Go(func() {
		err := write()
		entries := p.shelf.Novels()
		p.post(func() { p.saving, p.entries, p.shelfErr = false, entries, err })
	})
}

func (p *novelPanel) view(c *ui.Context) {
	ui.Column(c).Fill().Padding(16).Gap(8).Children(func() {
		if p.err != nil {
			ui.Text(c, p.err.Error()).TextColor(c.Theme().Danger).Selectable()
		}
		if p.shelfErr != nil {
			ui.Text(c, "小说书架："+p.shelfErr.Error()).TextColor(c.Theme().Danger).Selectable()
		}
		if p.busy || p.saving {
			ui.Spinner(c)
		}
		switch p.screen {
		case novelCatalog:
			p.catalogView(c)
		case novelBooks:
			p.booksView(c)
		case novelDetails:
			p.detailView(c)
		case novelReading:
			p.readerView(c)
		case novelAdapt:
			p.adaptView(c)
		}
	})
}

func (p *novelPanel) catalogView(c *ui.Context) {
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		if ui.Button(c, "小说书架").Clicked() {
			p.backTo(novelBooks)
		}
		if len(p.sources) == 0 {
			ui.Text(c, "请在源管理安装并启用小说源。")
			return
		}
		labels := make([]string, len(p.sources))
		var selected string
		for i, rec := range p.sources {
			labels[i] = rec.Name + " · " + rec.Key
			if rec.Key == p.key {
				selected = labels[i]
			}
		}
		if ui.Select(c, &selected, labels).Width(205).Label("小说源").Changed() {
			if index := indexOfOption(labels, selected); index >= 0 {
				p.key, p.query, p.catalog = p.sources[index].Key, "", nil
				p.loadCatalog("", 1)
			}
		}
		if ui.SearchField(c, &p.query).Grow(1).Submitted() {
			p.loadCatalog(p.query, 1)
		}
		if ui.Button(c, "搜索").Clicked() {
			p.loadCatalog(p.query, 1)
		}
		if ui.Button(c, "热门").Clicked() {
			p.query = ""
			p.loadCatalog("", 1)
		}
	})
	if p.catalog == nil {
		return
	}
	page := p.catalog
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		if ui.Button(c, "上一页").Disabled(p.busy || p.catalogPage <= 1).Clicked() {
			p.loadCatalog(p.searched, p.catalogPage-1)
		}
		ui.Textf(c, "第 %d 页", p.catalogPage)
		if ui.Button(c, "下一页").Disabled(p.busy || p.catalogPage >= page.MaxPage).Clicked() {
			p.loadCatalog(p.searched, p.catalogPage+1)
		}
	})
	if len(page.Items) == 0 {
		ui.Text(c, "没有找到小说。")
		return
	}
	ui.List(c, nil, len(page.Items), func(i int) {
		item := page.Items[i]
		if ui.Button(c, novelTitle(item.Title, item.Author)).Key(i).Disabled(p.busy).Clicked() {
			p.openBook(shelf.Novel{Source: p.key, ID: item.ID, Title: item.Title, Author: item.Author})
		}
	}).Grow(1)
}

func (p *novelPanel) booksView(c *ui.Context) {
	if ui.Button(c, "返回浏览").Clicked() {
		p.backTo(novelCatalog)
	}
	if len(p.entries) == 0 {
		ui.Text(c, "添加到书架的小说和阅读进度会显示在这里。")
		return
	}
	ui.List(c, nil, len(p.entries), func(i int) {
		entry := p.entries[i]
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			available := p.hasSource(entry.Source)
			if ui.Button(c, novelTitle(entry.Title, entry.Author)).Key(i).Disabled(!available || p.busy).Clicked() {
				p.openBook(entry.Novel)
			}
			if entry.Kept {
				ui.Text(c, "已收藏")
			}
			if !available {
				ui.Text(c, "源未启用")
			}
		})
	}).Grow(1)
}

func (p *novelPanel) detailView(c *ui.Context) {
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		if ui.Button(c, "返回").Clicked() {
			p.backTo(p.detailFrom)
			return
		}
		saved := p.savedBook()
		label := "加入书架"
		if saved.Kept {
			label = "取消收藏"
		}
		if ui.Button(c, label).Disabled(p.shelf == nil || p.saving || p.busy).Clicked() {
			p.keepBook(!saved.Kept)
		}
		if index := slices.IndexFunc(p.detail.Chapters, func(ch core.NovelChapter) bool { return ch.ID == saved.ChapterID }); index >= 0 {
			if ui.PrimaryButton(c, "继续阅读").Disabled(p.busy || p.saving).Clicked() {
				p.openChapter(index)
			}
		}
	})
	if p.detail == nil { // The back button can clear it in this frame.
		return
	}
	ui.Text(c, novelTitle(p.book.Title, p.book.Author)).FontSize(18).Bold()
	if p.detail.Summary != "" {
		ui.Scroll(c).MaxHeight(110).Children(func() { ui.Text(c, p.detail.Summary).Selectable() })
	}
	if len(p.detail.Chapters) == 0 {
		ui.Text(c, "这个来源没有提供章节。")
		return
	}
	ui.List(c, nil, len(p.detail.Chapters), func(i int) {
		if ui.Button(c, chapterTitle(p.detail.Chapters[i], i)).Key(i).Disabled(p.saving).Clicked() {
			p.openChapter(i)
		}
	}).Grow(1)
}

func (p *novelPanel) readerView(c *ui.Context) {
	chapter, page := p.chapter, p.page
	detail, pages := p.detail, p.pages
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		if ui.Button(c, "目录").Clicked() {
			p.backTo(novelDetails)
		}
		if ui.Button(c, "上一章").Disabled(p.saving || chapter == 0).Clicked() {
			p.openChapter(chapter - 1)
		}
		ui.Text(c, chapterTitle(detail.Chapters[chapter], chapter)).SingleLine().Grow(1)
		if ui.Button(c, "下一章").Disabled(p.saving || chapter+1 >= len(detail.Chapters)).Clicked() {
			p.openChapter(chapter + 1)
		}
		if ui.Button(c, "改编为静态漫画").Disabled(p.busy || p.saving || p.chapterContent.HTML == "").Clicked() {
			p.openAdapt()
		}
	})
	ui.Scroll(c).Key(fmt.Sprintf("chapter:%d/page:%d", chapter, page)).Grow(1).Children(func() {
		ui.Text(c, pages[page]).FontSize(18).Selectable()
	})
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		if ui.Button(c, "上一页").Disabled(p.busy || p.saving || page == 0).Clicked() {
			p.readPage(page - 1)
		}
		ui.Textf(c, "%d / %d", page+1, len(pages))
		if ui.Button(c, "下一页").Disabled(p.busy || p.saving || page+1 >= len(pages)).Clicked() {
			p.readPage(page + 1)
		}
	})
}

func novelTitle(title, author string) string {
	if author == "" {
		return title
	}
	return title + " · " + author
}

func chapterTitle(ch core.NovelChapter, index int) string {
	if ch.Title != "" {
		return ch.Title
	}
	return fmt.Sprintf("第 %d 章", index+1)
}
