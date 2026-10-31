package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/egoist/mygo"

	"github.com/egoist/mygo/ui"
	"github.com/laincat/nekophage/internal/archive"
	"github.com/laincat/nekophage/internal/cookies"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/shelf"
	"github.com/laincat/veneracat-core/backup"
	"github.com/laincat/veneracat-core/subscription"
)

// manageState is the source manager window: install from a script URL or an
// index, see the installed sources, turn them on or off, remove them.
type manageState struct {
	scriptURL string
	indexURL  string
	busy      string
	errText   string
	notice    string
	rows      []subscription.Row
	rowsTitle string
	removing  string

	// settingsFor is the source whose settings panel is open, with its
	// declaration and the values in force.
	settingsFor    string
	settings       []core.SourceSetting
	settingValues  map[string]json.RawMessage
	settingsBusy   bool
	settingsErr    string
	editingSetting string
	editText       string

	// A source may declare its own login, in one of three styles. Without these
	// the panel could not tell such a source from one that needs no account.
	// translations holds the UI strings the source declares for this locale.
	// A source writes `title: "domain"` and supplies the readable text here;
	// without it the panel shows the raw key.
	translations map[string]string

	account        core.AccountStyles
	accountUser    string
	accountPass    string
	accountCookies []string
	// accountFields holds the cookie names, read once when the panel opens: the
	// declaration lives in JS, so reading it per frame would eval every draw.
	accountFields []string
	// accountEditing is the cookie field being typed into.
	accountEditing int
	accountBusy    bool
	accountMessage string

	// backupImport is an archive the user picked, waiting to be applied.
	backupImport *archive.Backup
	// backupCookies restores the official app's jar into source partitions.
	// Off by default: they are credentials.
	backupCookies bool
	backupSum     archive.Summary
	backupBusy    bool
}

// openManage shows the window, creating it the first time.
func (a *App) openManage() {
	if a.manageWin == nil {
		a.manageWin = mygo.NewWindow(mygo.WindowOptions{
			Title: "漫画源", Width: 860, Height: 720, Content: ui.View(a.manageView),
		})
		a.manageWin.Show()
		return
	}
	a.manageWin.Show()
	a.manageWin.Focus()
}

func (a *App) manageView(c *ui.Context) {
	t := c.Theme()
	m := a.manage
	if m.indexURL == "" {
		m.indexURL = a.defaultIndex
	}

	ui.Column(c).Fill().Children(func() {
		ui.Column(c).Padding(16).Gap(8).Children(func() {
			ui.Text(c, "添加漫画源").FontSize(16).Bold()
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.TextInput(c, &m.scriptURL).Placeholder("脚本链接 https://…/source.js").Grow(1)
				if ui.PrimaryButton(c, "安装").Disabled(m.busy != "").Clicked() {
					a.installScript(m.scriptURL)
				}
			})
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.TextInput(c, &m.indexURL).Placeholder("订阅 https://…/index.json").Grow(1)
				if ui.Button(c, "读取订阅").Disabled(m.busy != "").Clicked() {
					a.loadIndex(m.indexURL)
				}
			})
			if m.errText != "" {
				ui.Text(c, m.errText).TextColor(ui.RGB(224, 49, 49)).FontSize(12).Selectable()
			}
			a.backupSection(c, t)
			if m.notice != "" {
				ui.Text(c, m.notice).TextColor(t.TextMuted).FontSize(12)
			}
			if m.busy != "" {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Spinner(c)
					ui.Text(c, m.busy).FontSize(12).TextColor(t.TextMuted)
				})
			}
		})
		ui.Divider(c)

		if len(m.rows) > 0 {
			ui.Text(c, m.rowsTitle).Padding(8, 16).FontSize(13).Bold()
			ui.List(c, nil, len(m.rows), func(i int) {
				a.indexRow(c, t, m.rows[i])
			}).MaxHeight(240)
			ui.Divider(c)
		}

		a.settingsPanel(c, t)

		installed := a.lib.Sources()
		ui.Textf(c, "已安装 %d 个源", len(installed)).Padding(8, 16).FontSize(13).Bold()
		ui.List(c, nil, len(installed), func(i int) {
			a.sourceRow(c, t, installed[i].Key, installed[i].Name, installed[i].Version, installed[i].Enabled)
		}).Grow(1).Children(func() {
			if len(installed) == 0 {
				ui.Text(c, "还没有漫画源。").Padding(12, 16).TextColor(t.TextMuted)
			}
		})
	})
}

func (a *App) indexRow(c *ui.Context, t *ui.Theme, row subscription.Row) {
	ui.Row(c).Padding(6, 16).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.Text(c, row.Name).SingleLine()
			ui.Textf(c, "v%s · %s", row.Version, statusText(row)).FontSize(11).TextColor(t.TextMuted).SingleLine()
		}).Grow(1)
		switch row.Status {
		case subscription.StatusNew:
			if ui.Button(c, "安装").Clicked() {
				a.installEntry(row)
			}
		case subscription.StatusUpdate:
			if ui.Button(c, "更新").Clicked() {
				a.installEntry(row)
			}
		case subscription.StatusInstalled:
			ui.Text(c, "已安装").FontSize(11).TextColor(t.TextMuted)
		case subscription.StatusOther:
			ui.Text(c, "已从其他订阅安装").FontSize(11).TextColor(t.TextMuted)
		case subscription.StatusTooNew:
			ui.Text(c, "需要更新 App").FontSize(11).TextColor(ui.RGB(230, 140, 0))
		}
	})
}

func statusText(row subscription.Row) string {
	switch row.Status {
	case subscription.StatusUpdate:
		return "可更新"
	case subscription.StatusInstalled:
		return "已安装"
	case subscription.StatusOther:
		return "来自其他订阅"
	case subscription.StatusTooNew:
		return "需要更新 App"
	default:
		return "未安装"
	}
}

func (a *App) sourceRow(c *ui.Context, t *ui.Theme, key, name, version string, enabled bool) {
	on := enabled
	ui.Row(c).Padding(6, 16).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.Text(c, name).SingleLine()
			ui.Textf(c, "%s · v%s", key, version).FontSize(11).TextColor(t.TextMuted).SingleLine()
		}).Grow(1)
		if ui.Switch(c, &on).Label("启用").Changed() {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
				err := a.lib.SetEnabled(ctx, key, on)
				a.manageWin.Update(func() {
					a.manage.done(err, "")
					a.refreshSources()
				})
			}()
		}
		if ui.Button(c, "设置").Clicked() {
			a.openSourceSettings(key)
		}
		if a.manage.removing == key {
			ui.Spinner(c)
		} else if ui.Button(c, "删除").Clicked() {
			a.manage.removing = key
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
				err := a.lib.Remove(ctx, key)
				a.manageWin.Update(func() {
					a.manage.removing = ""
					a.manage.done(err, "已删除 "+key)
					a.refreshSources()
				})
			}()
		}
	})
}

// --- actions ---

func (a *App) installScript(rawURL string) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return
	}
	a.manage.start("正在下载脚本…")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		rec, err := a.lib.InstallURL(ctx, rawURL)
		a.manageWin.Update(func() {
			if err != nil {
				a.manage.done(err, "")
				return
			}
			a.manage.done(nil, "已安装 "+rec.Name)
			a.manage.scriptURL = ""
			a.refreshSources()
		})
	}()
}

func (a *App) loadIndex(rawURL string) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return
	}
	a.manage.start("正在读取订阅…")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		idx, rows, err := a.lib.Index(ctx, rawURL)
		a.manageWin.Update(func() {
			m := &a.manage
			m.busy = ""
			if err != nil {
				m.done(err, "")
				return
			}
			a.defaultIndex = rawURL
			m.rows, m.rowsTitle = rows, "订阅中的源"
			m.errText, m.notice = "", ""
			// Entries the parser skipped are reported with the rows, not
			// hidden: a hand-edited index often has one bad line.
			if len(idx.Problems) > 0 {
				m.notice = fmt.Sprintf("订阅中有 %d 项被跳过", len(idx.Problems))
			}
			a.manage.indexURL = rawURL
		})
	}()
}

func (a *App) installEntry(row subscription.Row) {
	indexURL := strings.TrimSpace(a.manage.indexURL)
	a.manage.start("正在安装 " + row.Name + "…")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		rec, err := a.lib.InstallEntry(ctx, row.Entry, indexURL)
		a.manageWin.Update(func() {
			if err != nil {
				a.manage.done(err, "")
				return
			}
			a.manage.done(nil, "已安装 "+rec.Name)
			a.refreshSources()
			// Refresh the statuses from the same index.
			a.loadIndex(indexURL)
		})
	}()
}

// --- state helpers ---

func (m *manageState) start(what string) { m.busy, m.errText, m.notice = what, "", "" }

// done records the result of an action: an error, or a notice.
func (m *manageState) done(err error, notice string) {
	m.busy = ""
	if err != nil {
		m.errText, m.notice = explain(err), ""
		return
	}
	m.errText = ""
	m.notice = notice
}

// backupSection shows the import flow: pick an official .venera backup, see
// what it holds, then apply what this app can.
func (a *App) backupSection(c *ui.Context, t *ui.Theme) {
	m := &a.manage
	if m.backupImport == nil {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			if ui.Button(c, "导入 .venera 备份").Disabled(m.backupBusy || m.busy != "").Clicked() {
				a.pickBackup()
			}
			// Asked before the file is read, because the plan is built with or
			// without cookies. Off by default: they are credentials.
			ui.Checkbox(c, &m.backupCookies, "同时恢复登录状态(Cookie)")
			if m.backupBusy {
				ui.Spinner(c)
				ui.Text(c, "正在读取备份…").FontSize(12).TextColor(t.TextMuted)
			}
		})
		return
	}

	sum := m.backupSum
	b := m.backupImport
	ui.Column(c).Padding(8, 0).Gap(2).Children(func() {
		ui.Text(c, "备份内容").FontSize(13).Bold()
		ui.Textf(c, "漫画源 %d 个(其中带数据 %d)", sum.Sources, sum.WithData).FontSize(12)
		ui.Textf(c, "阅读历史 %d 条 · 收藏夹 %d 个/%d 条 · 图片收藏 %d 条", sum.History, sum.FavoriteFolders, sum.FavoriteItems, sum.ImageFavorites).FontSize(12).TextColor(t.TextMuted)
		ui.Textf(c, "搜索历史 %d 条 · 设置 %d 项", sum.SearchHistory, sum.Settings).FontSize(12).TextColor(t.TextMuted)
		if sum.Unresolved > 0 {
			ui.Textf(c, "有 %d 组记录认不出来源,无法匹配到具体源", sum.Unresolved).FontSize(12).TextColor(ui.RGB(230, 140, 0))
		}
		for _, w := range sum.Warnings {
			ui.Text(c, w).FontSize(11).TextColor(t.TextMuted)
		}
		if m.backupCookies && len(b.Plan.Cookies) > 0 {
			ui.Textf(c, "会把 %d 条登录状态写入 %s(每条 cookie 只在该源访问同名域名时送出)",
				len(b.Plan.Cookies), strings.Join(b.Plan.CookieTargets, "、")).FontSize(11).TextColor(t.TextMuted)
		} else if len(b.Content.Cookies) > 0 {
			ui.Textf(c, "备份里有 %d 条登录状态,当前不导入。", len(b.Content.Cookies)).FontSize(11).TextColor(t.TextMuted)
		}
		ui.Text(c, "会导入漫画源脚本、源数据、阅读历史、收藏、搜索历史与图片收藏;设置还只读。").FontSize(11).TextColor(t.TextMuted)
		if sum.History > 0 || sum.FavoriteItems > 0 {
			ui.Textf(c, "其中可还原:阅读记录 %d 条、收藏 %d 部(其余因来源未识别暂时跳过)", sum.History, sum.FavoriteItems).FontSize(11).TextColor(t.TextMuted)
		}
		ui.Row(c).Gap(8).Children(func() {
			if ui.PrimaryButton(c, "导入源与数据").Disabled(m.backupBusy).Clicked() {
				a.applyBackup()
			}
			if ui.Button(c, "取消").Disabled(m.backupBusy).Clicked() {
				m.backupImport, m.backupSum = nil, archive.Summary{}
			}
		})
	})
}

// pickBackup asks for the file and reads it off the interface's thread.
func (a *App) pickBackup() {
	a.manage.backupBusy = true
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:  a.manageWin,
			Title:   "选择 .venera 备份",
			Filters: []mygo.FileFilter{{Name: "Venera 备份", Extensions: []string{"venera", "zip"}}},
		})
		if err != nil || len(paths) == 0 {
			a.manageWin.Update(func() { a.manage.backupBusy = false })
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		known := make([]string, 0, len(a.lib.Sources()))
		for _, s := range a.lib.Sources() {
			known = append(known, s.Key)
		}
		b, err := archive.Read(ctx, paths[0], known, archive.Options{Cookies: a.manage.backupCookies})
		a.manageWin.Update(func() {
			m := &a.manage
			m.backupBusy = false
			if err != nil {
				m.done(err, "")
				return
			}
			m.backupImport = b
			m.backupSum = b.Summarize()
			m.errText, m.notice = "", ""
		})
	}()
}

// applyBackup installs the archive's sources and writes their data.
func (a *App) applyBackup() {
	m := &a.manage
	b := m.backupImport
	if b == nil {
		return
	}
	m.backupBusy = true
	m.errText, m.notice = "", ""
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		sources, fields := 0, 0
		var firstErr error
		for _, src := range b.Plan.Sources {
			if src.Script != "" {
				if _, err := a.lib.InstallText(ctx, src.Script, "backup:"+src.Key); err != nil {
					if firstErr == nil {
						firstErr = fmt.Errorf("%s: %w", src.Key, err)
					}
					continue
				}
				sources++
			}
		}
		recorded, kept := 0, 0
		for _, p := range b.Positions() {
			chapter := ""
			index := p.Episode - 1
			if p.Episode > 0 {
				chapter = "第 " + itoa(p.Episode) + " 话"
			}
			ok, err := a.shelf.Merge(shelf.History{
				Comic:   shelf.Comic{Source: p.Source, ID: p.ID, Title: p.Title, Author: p.Author, Cover: p.Cover},
				Chapter: chapter, ChapterIndex: index, Page: p.Page, Time: p.Time})
			if err != nil && firstErr == nil {
				firstErr = err
			}
			if ok {
				recorded++
			}
		}
		for _, f := range b.Favourites() {
			for _, it := range f.Items {
				added, err := a.shelf.Add(f.Name, shelf.Comic{Source: it.Source, ID: it.ID,
					Title: it.Title, Author: it.Author, Cover: it.Cover})
				if err != nil && firstErr == nil {
					firstErr = err
				}
				if added {
					kept++
				}
			}
		}

		pages := 0
		for _, kept := range mustPages(b) {
			err := a.shelf.KeepPage(shelf.ImageFavourite{
				Source:       kept.Source,
				ComicID:      kept.ComicID,
				ComicTitle:   kept.ComicTitle,
				ChapterID:    kept.ChapterID,
				ChapterIndex: kept.ChapterIndex,
				ChapterName:  kept.ChapterName,
				Page:         kept.Page,
				URL:          kept.URL,
				Time:         kept.Time,
			})
			if err != nil && firstErr == nil {
				firstErr = err
			}
			if err == nil {
				pages++
			}
		}

		installed := make([]string, 0, len(a.lib.Sources()))
		for _, src := range a.lib.Sources() {
			installed = append(installed, src.Key)
		}
		cookieRows := restoreCookies(b.Plan, a.lib.Jar(), installed, time.Now())

		searched := 0
		if a.searches != nil && len(b.Content.SearchHistory) > 0 {
			n, err := a.searches.Merge(b.Content.SearchHistory)
			if err != nil && firstErr == nil {
				firstErr = err
			}
			searched = n
		}

		data, err := b.SourceData()
		if err != nil && firstErr == nil {
			firstErr = err
		}
		for _, d := range data {
			values := make(map[string]json.RawMessage, len(d.Fields))
			for _, f := range d.Fields {
				values[f.Name] = f.Value
			}
			n, err := a.lib.ImportSourceData(d.Key, values)
			fields += n
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}
		a.manageWin.Update(func() {
			m := &a.manage
			m.backupBusy = false
			m.backupImport, m.backupSum = nil, archive.Summary{}
			if firstErr != nil {
				m.done(firstErr, "")
			} else {
				notice := fmt.Sprintf("已导入 %d 个源、%d 项源数据、%d 条阅读记录、%d 部收藏、%d 条搜索历史、%d 张图片",
					sources, fields, recorded, kept, searched, pages)
				if cookieRows > 0 {
					notice += fmt.Sprintf("、%d 条登录状态", cookieRows)
				}
				m.done(nil, notice)
			}
			a.refreshSources()
		})
	}()
}

// restoreCookies writes a plan's cookies into the partitions that exist now.
//
// Split out from the import so the rule can be tested without a window: a
// partition is written only when its source is installed, because a source whose
// script failed has nothing that would ever send the cookie. Returns how many
// cookie rows were written, across all partitions.
func restoreCookies(plan *backup.Plan, jar *cookies.Jar, installed []string, now time.Time) int {
	if plan == nil || len(plan.Cookies) == 0 || len(plan.CookieTargets) == 0 {
		return 0
	}
	present := make(map[string]bool, len(installed))
	for _, key := range installed {
		present[key] = true
	}
	rows := toNetCookies(plan.Cookies, now)
	if len(rows) == 0 {
		return 0
	}
	written := 0
	for _, target := range plan.CookieTargets {
		if !present[target] {
			continue
		}
		// The plan names sources; the jar is keyed by partition. Writing to the
		// bare key would store cookies no request ever reads.
		jar.Import(core.ComicPartition(target), rows)
		written += len(rows)
	}
	return written
}

// toNetCookies turns a backup's cookie rows into the shape the jar imports.
//
// Expiry is milliseconds since the epoch in the backup and a time here; a row
// without one is a session cookie and stays without one.
func toNetCookies(rows []backup.Cookie, now time.Time) []*http.Cookie {
	out := make([]*http.Cookie, 0, len(rows))
	for _, row := range rows {
		c := &http.Cookie{Name: row.Name, Value: row.Value, Domain: row.Domain, Path: row.Path,
			Secure: row.Secure, HttpOnly: row.HTTPOnly}
		if row.Expires != nil && *row.Expires > 0 {
			when := time.UnixMilli(*row.Expires)
			// Already expired: the jar would never send it, yet the summary
			// would count it as a restored sign-in. Skip it instead.
			if !when.After(now) {
				continue
			}
			c.Expires = when
		}
		out = append(out, c)
	}
	return out
}

// mustPages reads the backup's kept pages, reporting a problem through the
// notice line rather than failing the whole import.
func mustPages(b *archive.Backup) []archive.KeptPage {
	pages, err := b.Pages()
	if err != nil {
		log.Printf("nekophage: kept pages in %s: %v", b.Path, err)
		return nil
	}
	return pages
}
