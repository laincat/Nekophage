package ui

import (
	"log"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/reader"
	"github.com/laincat/nekophage/internal/shelf"
)

// rememberChapter writes the reading position the reader just opened, so the
// shelf can offer it again.
func (a *App) rememberChapter(key string) {
	if a.shelf == nil || key == "" {
		return
	}
	a.mu.Lock()
	comic, d := a.detailComic, a.detail
	index, list := a.readerChapter, a.readerList
	a.mu.Unlock()
	if comic.ID == "" {
		return
	}
	chapter := ""
	if index >= 0 && index < len(list) {
		chapter = list[index].Title
	}
	entry := shelf.History{
		Comic: shelf.Comic{Source: key, ID: comic.ID,
			Title:  firstNonEmpty(d.Title, comic.Title),
			Author: firstNonEmpty(d.Subtitle, comic.Subtitle),
			Cover:  firstNonEmpty(d.Cover, comic.Cover)},
		Chapter:      chapter,
		ChapterIndex: index,
		Time:         time.Now().Unix(),
	}
	go func() { _ = a.shelf.Record(entry) }()
}

// shelfComic is the shelf's copy of a comic the window is showing.
func shelfComic(key string, comic core.Comic, d core.Details) shelf.Comic {
	return shelf.Comic{Source: key, ID: comic.ID,
		Title:  firstNonEmpty(d.Title, comic.Title),
		Author: firstNonEmpty(d.Subtitle, comic.Subtitle),
		Cover:  firstNonEmpty(d.Cover, comic.Cover)}
}

// progressFromReader writes the reader's position into the shelf. The page
// knows page numbers and percentages; the chapter's name and place in the
// list come from the window, which loaded it.
func (a *App) progressFromReader(p reader.Progress) {
	if a.shelf == nil || p.Comic == "" {
		return
	}
	a.mu.Lock()
	comicID, list, mode, current := a.readerComic, a.readerList, a.mode, a.readerChapter
	a.mu.Unlock()
	if p.Comic != comicID || len(list) == 0 {
		return
	}
	// A page that has been replaced still reports where it was; that is not
	// where reading is now.
	if current >= 0 && current < len(list) && list[current].ID != p.Chapter {
		return
	}
	index, title := -1, ""
	for i, ch := range list {
		if ch.ID == p.Chapter {
			index, title = i, ch.Title
			break
		}
	}
	if index < 0 {
		return
	}
	ok, err := a.shelf.UpdateProgress(p.Source, p.Comic, title, index, p.Page, p.Percent)
	if err != nil {
		log.Printf("nekophage: saving reading position: %v", err)
		return
	}
	// Only the shelf shows the position, so only it needs a redraw.
	if ok && mode == 2 {
		a.win.Update(func() {})
	}
}

// switchChapter answers the reader's request for another chapter: it moves
// the window's own idea of the current chapter and returns the address to
// load, which the page then navigates to.
func (a *App) switchChapter(sw reader.Switch) (string, bool) {
	a.mu.Lock()
	comic, list := a.readerComic, a.readerList
	a.mu.Unlock()
	if sw.Comic != comic || len(list) == 0 {
		return "", false
	}
	for i, ch := range list {
		if ch.ID != sw.Chapter {
			continue
		}
		a.mu.Lock()
		a.readerChapter = i
		title := a.readerTitle
		a.mu.Unlock()
		title += " · " + ch.Title
		a.readerWin.SetTitle(title)
		a.rememberChapter(sw.Source)
		return a.readerURLFor(i), true
	}
	return "", false
}

// keepPage stores the page the reader kept. The comic and chapter it belongs
// to come from the window, which loaded them.
func (a *App) keepPage(k reader.Keep) bool {
	if a.shelf == nil {
		return false
	}
	a.mu.Lock()
	comic, d := a.detailComic, a.detail
	index, list := a.readerChapter, a.readerList
	a.mu.Unlock()
	if comic.ID == "" || comic.ID != k.Comic {
		return false
	}
	name := ""
	if index >= 0 && index < len(list) && list[index].ID == k.Chapter {
		name = list[index].Title
	}
	err := a.shelf.KeepPage(shelf.ImageFavourite{
		Source:       k.Source,
		ComicID:      k.Comic,
		ComicTitle:   firstNonEmpty(d.Title, comic.Title),
		ChapterID:    k.Chapter,
		ChapterIndex: index,
		ChapterName:  name,
		Page:         k.Page,
		URL:          k.URL,
	})
	if err != nil {
		a.setNotice("收藏本页失败:" + err.Error())
		return false
	}
	a.setNotice("已收藏第 " + itoa(k.Page) + " 页")
	if a.mode == 2 {
		a.win.Update(func() {})
	}
	return true
}

// keepComic keeps the open comic in the chosen folder, or takes it out of
// whichever folder holds it.
func (a *App) keepComic() {
	if a.shelf == nil {
		return
	}
	a.mu.Lock()
	key, comic, d, folder := a.currentKey(), a.detailComic, a.detail, a.favFolder
	a.mu.Unlock()
	if comic.ID == "" {
		return
	}
	if heldIn, held := a.shelf.Holds(key, comic.ID); held {
		if err := a.shelf.Remove(heldIn, key, comic.ID); err != nil {
			a.setNotice("取消收藏失败:" + err.Error())
			return
		}
		a.setNotice("已取消收藏")
		a.win.Update(func() {})
		return
	}
	if _, err := a.shelf.Move(shelfComic(key, comic, d), folder); err != nil {
		a.setNotice("收藏失败:" + err.Error())
		return
	}
	a.setNotice("已收藏到「" + folder + "」")
	a.saveSettings(key, a.query)
	a.win.Update(func() {})
}

// folderNames lists the folders, with the default one always offered.
func (a *App) folderNames() []string {
	names := []string{shelf.DefaultFolder}
	seen := map[string]bool{shelf.DefaultFolder: true}
	if a.shelf == nil {
		return names
	}
	for _, f := range a.shelf.Folders() {
		if !seen[f.Name] {
			seen[f.Name] = true
			names = append(names, f.Name)
		}
	}
	return names
}

// setNotice puts a line under the header and redraws.
func (a *App) setNotice(text string) {
	a.mu.Lock()
	a.notice = text
	a.mu.Unlock()
	a.win.Update(func() {})
}

// shelfDemo runs the folder edits a reader would click, for the tools.
func (a *App) shelfDemo() {
	if a.shelf == nil {
		a.setNotice("书架打不开")
		return
	}
	if err := a.shelf.CreateFolder("以后看"); err != nil {
		a.setNotice("新建失败:" + err.Error())
		return
	}
	var moved bool
	for _, f := range a.shelf.Folders() {
		for _, it := range f.Items {
			if _, err := a.shelf.Move(it, "以后看"); err != nil {
				a.setNotice("移动失败:" + err.Error())
				return
			}
			moved = true
			break
		}
		if moved {
			break
		}
	}
	if err := a.shelf.RenameFolder("以后看", "以后再看"); err != nil {
		a.setNotice("重命名失败:" + err.Error())
		return
	}
	a.setNotice("书架编辑自检完成")
	a.win.Update(func() {})
}

// shelfView is the reader's own list: what was read, then what was kept.
func (a *App) shelfView(c *ui.Context, t *ui.Theme) {
	if a.shelf == nil {
		ui.Column(c).Fill().Center().Children(func() {
			ui.Text(c, "书架打不开。").TextColor(t.TextMuted)
		})
		return
	}
	history := a.shelf.History(40)
	folders := a.shelf.Folders()
	pictures := a.shelf.Pages(24)

	if len(history) == 0 && len(folders) == 0 && len(pictures) == 0 {
		ui.Column(c).Fill().Center().Gap(8).Children(func() {
			ui.Text(c, "书架还是空的").TextColor(t.TextMuted)
			ui.Text(c, "打开的章节会记在这里,详情页可以收藏漫画,阅读器可以收藏单页。").FontSize(12).TextColor(t.TextMuted)
		})
		return
	}

	ui.Scroll(c).Grow(1).Padding(12, 16).Children(func() {
		// Adding a folder: the name goes in a field, then a button.
		ui.Row(c).Padding(6, 0).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.TextInput(c, &a.folderNew).Width(180).Placeholder("新收藏夹的名字")
			disabled := strings.TrimSpace(a.folderNew) == ""
			if ui.Button(c, "新建收藏夹").Disabled(disabled).Clicked() {
				if err := a.shelf.CreateFolder(a.folderNew); err != nil {
					a.setNotice("新建失败:" + err.Error())
				} else {
					a.favFolder = strings.TrimSpace(a.folderNew)
					a.setNotice("已新建「" + a.favFolder + "」")
					a.folderNew = ""
				}
				a.win.Update(func() {})
			}
		})
		if len(history) > 0 {
			ui.Text(c, "继续阅读").FontSize(14).Bold().Padding(8, 0)
			for _, h := range history {
				a.historyRow(c, t, h)
			}
		}
		if len(pictures) > 0 {
			ui.Textf(c, "图片收藏 · %d 张", len(pictures)).FontSize(14).Bold().Padding(16, 0, 8, 0)
			for _, p := range pictures {
				a.pictureRow(c, t, p)
			}
		}
		for _, f := range folders {
			ui.Row(c).Padding(16, 0, 6, 0).Gap(8).AlignItems(ui.Center).Children(func() {
				if a.renaming == f.Name {
					ui.TextInput(c, &a.renameTo).Width(180)
					if ui.Button(c, "确定").Clicked() {
						if err := a.shelf.RenameFolder(f.Name, a.renameTo); err != nil {
							a.setNotice("重命名失败:" + err.Error())
						} else {
							a.renaming = ""
						}
						a.win.Update(func() {})
					}
					if ui.Button(c, "取消").Clicked() {
						a.renaming = ""
						a.win.Update(func() {})
					}
					return
				}
				ui.Textf(c, "%s · %d 部", f.Name, len(f.Items)).FontSize(14).Bold()
				ui.Spacer(c)
				if ui.Button(c, "重命名").Clicked() {
					a.renaming, a.renameTo = f.Name, f.Name
					a.win.Update(func() {})
				}
				if ui.Button(c, "删除").Clicked() {
					if err := a.shelf.DeleteFolder(f.Name); err != nil {
						a.setNotice("删除失败:" + err.Error())
					} else {
						a.setNotice("已删除「" + f.Name + "」")
					}
					a.win.Update(func() {})
				}
			})
			if len(f.Items) == 0 {
				ui.Text(c, "（空）").FontSize(12).TextColor(t.TextMuted).Padding(6, 0)
			}
			for _, it := range f.Items {
				a.favouriteRow(c, t, f.Name, it)
			}
		}
	})
}

// historyRow is one reading position: cover, title, chapter, when.
func (a *App) historyRow(c *ui.Context, t *ui.Theme, h shelf.History) {
	name := h.Source + "/" + h.ID
	a.cover(h.Source, h.ID, h.Cover)
	ui.Row(c).Padding(6, 0).Gap(10).AlignItems(ui.Center).Children(func() {
		a.shelfThumb(c, t, name)
		ui.Column(c).Gap(2).Children(func() {
			ui.Text(c, h.Title).SingleLine()
			line := h.Chapter
			if line == "" {
				line = "第 " + itoa(h.ChapterIndex+1) + " 话"
			}
			if h.Page > 0 {
				line += " · 第 " + itoa(h.Page) + " 页"
			}
			if h.Percent > 0 {
				line += " · " + itoa(h.Percent) + "%"
			}
			ui.Textf(c, "%s · %s", line, when(h.Time)).FontSize(12).TextColor(t.TextMuted).SingleLine()
		})
		ui.Spacer(c)
		if ui.Button(c, "继续").Clicked() {
			a.resume(h)
		}
		if ui.Button(c, "移除").Clicked() {
			if err := a.shelf.ForgetHistory(h.Source, h.ID); err != nil {
				a.setNotice("移除失败:" + err.Error())
				return
			}
			a.win.Update(func() {})
		}
	})
}

// favouriteRow is one kept comic.
func (a *App) favouriteRow(c *ui.Context, t *ui.Theme, folder string, it shelf.Comic) {
	name := it.Source + "/" + it.ID
	a.cover(it.Source, it.ID, it.Cover)
	ui.Row(c).Padding(6, 0).Gap(10).AlignItems(ui.Center).Children(func() {
		a.shelfThumb(c, t, name)
		ui.Column(c).Gap(2).Children(func() {
			ui.Text(c, it.Title).SingleLine()
			if it.Author != "" {
				ui.Text(c, it.Author).FontSize(12).TextColor(t.TextMuted).SingleLine()
			}
			// A kept comic may also be a comic being read; the position is
			// what makes "open" useful rather than a fresh start.
			if line := a.positionLine(it.Source, it.ID); line != "" {
				ui.Text(c, line).FontSize(12).TextColor(t.TextMuted).SingleLine()
			}
		})
		ui.Spacer(c)
		if names := a.folderNames(); len(names) > 1 {
			// The control shows the folder a row is in until it is moved.
			pick := a.movePick[name]
			if pick == "" {
				pick = folder
			}
			if ui.Select(c, &pick, names).Width(120).Changed() {
				if _, err := a.shelf.Move(it, pick); err != nil {
					a.setNotice("移动失败:" + err.Error())
				} else {
					a.movePick[name] = pick
				}
				a.win.Update(func() {})
			}
		}
		if ui.Button(c, "打开").Clicked() {
			a.openShelfComic(it)
		}
		if ui.Button(c, "取消收藏").Clicked() {
			if err := a.shelf.Remove(folder, it.Source, it.ID); err != nil {
				a.setNotice("取消收藏失败:" + err.Error())
				return
			}
			a.win.Update(func() {})
		}
	})
}

// positionLine describes where reading stopped in one comic, empty when the
// shelf has no position for it.
func (a *App) positionLine(source, id string) string {
	h, ok := a.shelf.HistoryEntry(source, id)
	if !ok {
		return ""
	}
	var parts []string
	if h.Page > 0 {
		parts = append(parts, "第 "+itoa(h.Page)+" 页")
	}
	if h.Percent > 0 {
		parts = append(parts, itoa(h.Percent)+"%")
	}
	if len(parts) == 0 {
		if h.Chapter != "" {
			parts = append(parts, h.Chapter)
		} else {
			parts = append(parts, "第 "+itoa(h.ChapterIndex+1)+" 话")
		}
	}
	return "读到 " + strings.Join(parts, " · ")
}

// pictureRow is one kept page: its thumbnail, where it came from, and a way
// back to it.
func (a *App) pictureRow(c *ui.Context, t *ui.Theme, p shelf.ImageFavourite) {
	name := pageName(p.Source, p.URL)
	a.pageImage(p.Source, p.ComicID, p.ChapterID, p.URL)
	ui.Row(c).Padding(6, 0).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Width(44).Height(60).Radius(4).Background(t.Surface).Clip().Children(func() {
			if b := a.bitmap(name); b != nil {
				ui.Image(c, b).Fit(ui.Cover).Fill()
			}
		})
		ui.Column(c).Gap(2).Children(func() {
			title := p.ComicTitle
			if title == "" {
				title = p.ComicID
			}
			ui.Text(c, title).SingleLine()
			line := p.ChapterName
			if line == "" {
				line = "第 " + itoa(p.ChapterIndex+1) + " 话"
			}
			ui.Textf(c, "%s · 第 %d 页", line, p.Page).FontSize(12).TextColor(t.TextMuted).SingleLine()
		})
		ui.Spacer(c)
		if ui.Button(c, "回到这一话").Clicked() {
			a.mu.Lock()
			a.resumeChapter = p.ChapterIndex
			a.mu.Unlock()
			a.openShelfComic(shelf.Comic{Source: p.Source, ID: p.ComicID, Title: p.ComicTitle})
		}
		if ui.Button(c, "移除").Clicked() {
			if err := a.shelf.ForgetPage(p.Source, p.ComicID, p.URL); err != nil {
				a.setNotice("移除失败:" + err.Error())
				return
			}
			a.win.Update(func() {})
		}
	})
}

// shelfThumb draws a cover at list size, or an empty frame while it loads.
func (a *App) shelfThumb(c *ui.Context, t *ui.Theme, name string) {
	ui.Box(c).Width(44).Height(60).Radius(4).Background(t.Surface).Clip().Children(func() {
		if b := a.bitmap(name); b != nil {
			ui.Image(c, b).Fit(ui.Cover).Fill()
		}
	})
}

// resume reopens the comic of a history entry at the chapter it remembers.
func (a *App) resume(h shelf.History) {
	a.mu.Lock()
	a.resumeChapter = h.ChapterIndex
	a.mu.Unlock()
	a.openShelfComic(shelf.Comic{Source: h.Source, ID: h.ID, Title: h.Title,
		Author: h.Author, Cover: h.Cover})
}

// openShelfComic opens a comic the shelf holds, switching source if needed.
func (a *App) openShelfComic(it shelf.Comic) {
	if it.Source != a.currentKey() {
		a.mu.Lock()
		a.sourceKey = it.Source
		a.results, a.explorePages, a.exploreIndex, a.explorePage = nil, nil, 0, 1
		query := a.query
		a.mu.Unlock()
		a.saveSettings(it.Source, query)
	}
	a.openComic(core.Comic{ID: it.ID, Title: it.Title, Subtitle: it.Author, Cover: it.Cover})
}

// recentSearches shows what was searched before as chips that search again.
// It is only drawn on the empty results screen, so results keep the room.
func (a *App) recentSearches(c *ui.Context, t *ui.Theme) {
	if a.searches == nil {
		return
	}
	items := a.searches.Items(12)
	if len(items) == 0 {
		return
	}
	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "最近搜索").FontSize(12).TextColor(t.TextMuted)
		for _, q := range items {
			query := q
			if ui.Button(c, query).Clicked() {
				a.mu.Lock()
				a.query = query
				a.mu.Unlock()
				a.startSearch()
			}
		}
		if ui.Button(c, "清空").Clicked() {
			if err := a.searches.Clear(); err != nil {
				a.setNotice("清空失败:" + err.Error())
				return
			}
			a.win.Update(func() {})
		}
	})
}

// when renders a timestamp as something short to read.
func when(sec int64) string {
	if sec <= 0 {
		return ""
	}
	d := time.Since(time.Unix(sec, 0))
	switch {
	case d < time.Minute:
		return "刚刚"
	case d < time.Hour:
		return itoa(int(d.Minutes())) + " 分钟前"
	case d < 24*time.Hour:
		return itoa(int(d.Hours())) + " 小时前"
	case d < 30*24*time.Hour:
		return itoa(int(d.Hours()/24)) + " 天前"
	default:
		return time.Unix(sec, 0).Format("2006-01-02")
	}
}
