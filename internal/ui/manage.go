package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"

	"github.com/egoist/mygo/ui"
	"github.com/laincat/nekophage/internal/archive"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/veneracat-core/plugin"
	"github.com/laincat/veneracat-core/subscription"
)

// manageState is the source manager window: install from a script URL or an
// index, see the installed sources, turn them on or off, remove them.
// recordCallback stores what one callback answered, under its own key.
//
// Per key rather than one message for the panel: a source declares several
// callbacks, and one shared line would put a press's result beside all of them.
func (m *manageState) recordCallback(settingKey, answer string, err error) {
	m.callbackBusy = ""
	if m.callbackMsgs == nil {
		m.callbackMsgs = map[string]string{}
	}
	switch {
	case err != nil:
		// The source's own words: "no network", "Login expired", or whatever it
		// chose.
		m.callbackMsgs[settingKey] = explain(err)
	case answer == "":
		m.callbackMsgs[settingKey] = "已完成"
	default:
		m.callbackMsgs[settingKey] = answer
	}
}

type manageState struct {
	sourceKind string
	scriptURL  string
	indexURL   string
	busy       string
	errText    string
	notice     string
	rows       []subscription.Row
	rowsTitle  string
	removing   string

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

	// A callback setting is a function inside the source: it cannot be carried
	// across the boundary, but the source can run it, so the panel offers a
	// button and shows what it said.
	callbackBusy string
	// Answers are per setting: one message for the whole panel would show the
	// result of a press next to every callback the source declares.
	callbackMsgs map[string]string

	// backupImport is an archive the user picked, waiting to be applied.
	backupImport *archive.Backup
	// backupCookies restores the official app's jar into source partitions.
	// Off by default: they are credentials.
	backupCookies bool
	backupSum     archive.Summary
	backupBusy    bool
	// The export half: where the file went, or why it did not.
	backupErr    string
	backupNotice string
	lifetime     context.Context
	cancel       context.CancelFunc
	mu           sync.Mutex
}

// openManage shows the window, creating it the first time.
func (a *App) openManage() {
	if a.manageWin == nil || a.manageWin.IsDestroyed() {
		a.manageWin = mygo.NewWindow(mygo.WindowOptions{
			Title: "源管理", Width: 860, Height: 720, Content: ui.View(a.manageView),
		})
		a.refreshServices()
		a.refreshCollections()
		a.manageWin.OnClosed(func() {
			a.closeServiceSupervisor()
			a.manage.mu.Lock()
			if a.manage.cancel != nil {
				a.manage.cancel()
				a.manage.lifetime, a.manage.cancel = nil, nil
			}
			a.manage.mu.Unlock()
		})
		a.manageWin.Show()
		return
	}
	a.manageWin.Show()
	a.manageWin.Focus()
}

// manageContext belongs to the open source manager. Closing it cancels
// downloads and index reads derived from this context.
func (a *App) manageContext() (context.Context, context.CancelFunc) {
	m := &a.manage
	m.mu.Lock()
	defer m.mu.Unlock()
	if a.manageWin == nil || a.manageWin.IsDestroyed() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx, func() {}
	}
	if m.lifetime == nil {
		m.lifetime, m.cancel = context.WithCancel(context.Background())
	}
	return context.WithCancel(m.lifetime)
}

func (a *App) manageView(c *ui.Context) {
	t := c.Theme()
	m := &a.manage
	if m.sourceKind == "" {
		m.sourceKind = "漫画源"
	}
	if m.indexURL == "" {
		m.indexURL = a.defaultIndex
	}

	ui.Column(c).Fill().Children(func() {
		ui.Column(c).Padding(16).Gap(8).Children(func() {
			ui.Text(c, "添加源").FontSize(16).Bold()
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Select(c, &m.sourceKind, []string{"漫画源", "视频源", "Kazumi 规则"}).Width(150).Label("源类型").Disabled(m.busy != "").Changed()
				ui.TextInput(c, &m.scriptURL).Placeholder("源文件链接 https://…").Grow(1)
				if ui.PrimaryButton(c, "安装").Disabled(m.busy != "").Clicked() {
					a.installScript(m.scriptURL)
				}
				if ui.Button(c, "从文件安装").Disabled(m.busy != "").Clicked() {
					a.pickSourceFile()
				}
			})
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.TextInput(c, &m.indexURL).Placeholder("漫画订阅 https://…/index.json").Grow(1)
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

		a.collectionsSection(c, t)

		a.servicesSection(c, t)

		installed := a.lib.Sources()
		ui.Textf(c, "已安装 %d 个源", len(installed)).Padding(8, 16).FontSize(13).Bold()
		ui.List(c, nil, len(installed), func(i int) {
			a.sourceRow(c, t, installed[i])
		}).Grow(1).Children(func() {
			if len(installed) == 0 {
				ui.Text(c, "还没有安装源。").Padding(12, 16).TextColor(t.TextMuted)
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

func (a *App) sourceRow(c *ui.Context, t *ui.Theme, rec plugin.Record) {
	key, name, version := rec.Key, rec.Name, rec.Version
	on := rec.Enabled
	ui.Row(c).Padding(6, 16).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.Text(c, name).SingleLine()
			ui.Textf(c, "%s · %s · v%s", sourceKindLabel(rec.Type), key, version).FontSize(11).TextColor(t.TextMuted).SingleLine()
		}).Grow(1)
		if ui.Switch(c, &on).Label("启用").Changed() {
			go func() {
				ctx, cancel := a.manageContext()
				defer cancel()
				ctx, cancel = context.WithTimeout(ctx, 20*time.Second)
				defer cancel()
				err := a.lib.SetEnabled(ctx, key, on)
				a.manageWin.Update(func() {
					a.manage.done(err, "")
					a.refreshSources()
				})
			}()
		}
		if rec.Type == plugin.TypeComic || rec.Type == plugin.TypeVideoJS {
			if ui.Button(c, "设置").Clicked() {
				a.openSourceSettings(key)
			}
		}
		if rec.Type == plugin.TypeComic || rec.Type == plugin.TypeVideoJS || rec.Type == plugin.TypeKazumi {
			if ui.Button(c, "查看源码").Key("source-code:" + key).Clicked() {
				a.openSourceCode(key)
			}
		}
		if a.manage.removing == key {
			ui.Spinner(c)
		} else if ui.Button(c, "删除").Clicked() {
			a.manage.removing = key
			go func() {
				ctx, cancel := a.manageContext()
				defer cancel()
				ctx, cancel = context.WithTimeout(ctx, 20*time.Second)
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
	typ := a.manage.installType()
	a.manage.start("正在下载源文件…")
	go func() {
		ctx, cancel := a.manageContext()
		defer cancel()
		ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
		defer cancel()
		rec, err := a.lib.InstallTypedURL(ctx, typ, rawURL)
		a.manageWin.Update(func() {
			if err != nil {
				a.manage.done(err, "")
				return
			}
			a.manage.done(nil, "已安装 "+rec.Name)
			a.manage.scriptURL = ""
			a.refreshSources()
			if a.videoWin != nil {
				a.videoWin.Update(func() {})
			}
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
		ctx, cancel := a.manageContext()
		defer cancel()
		ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
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
		ctx, cancel := a.manageContext()
		defer cancel()
		ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
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

// pickSourceFile captures the selected kind before the dialog opens, then
// imports off the UI thread through the same durable path as URL installs.
func (a *App) pickSourceFile() {
	typ := a.manage.installType()
	a.manage.start("正在选择源文件…")
	go func() {
		extensions := []string{"js"}
		if typ == plugin.TypeKazumi {
			extensions = []string{"json"}
		}
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent: a.manageWin, Title: "选择" + sourceKindLabel(typ),
			Filters: []mygo.FileFilter{{Name: sourceKindLabel(typ), Extensions: extensions}},
		})
		if err != nil || len(paths) == 0 {
			a.manageWin.Update(func() { a.manage.done(err, "") })
			return
		}
		ctx, cancel := a.manageContext()
		defer cancel()
		ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
		defer cancel()
		rec, err := a.lib.InstallFile(ctx, typ, paths[0])
		a.manageWin.Update(func() {
			if err != nil {
				a.manage.done(err, "")
				return
			}
			a.manage.done(nil, "已安装 "+rec.Name)
			a.refreshSources()
			if a.videoWin != nil {
				a.videoWin.Update(func() {})
			}
		})
	}()
}

func (m *manageState) installType() plugin.Type {
	switch m.sourceKind {
	case "视频源":
		return plugin.TypeVideoJS
	case "Kazumi 规则":
		return plugin.TypeKazumi
	default:
		return plugin.TypeComic
	}
}

func sourceKindLabel(typ plugin.Type) string {
	switch typ {
	case plugin.TypeVideoJS:
		return "视频源"
	case plugin.TypeKazumi:
		return "Kazumi 规则"
	case plugin.TypeBuiltin:
		return "内置视频源"
	default:
		return "漫画源"
	}
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
		// Export sits beside import: what a backup is for, in both directions.
		a.exportRow(c, t)
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
		ctx, cancel := a.manageContext()
		defer cancel()
		ctx, cancel = context.WithTimeout(ctx, 2*time.Minute)
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
		ctx, cancel := a.manageContext()
		defer cancel()
		ctx, cancel = context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		result, firstErr := archive.Apply(ctx, b, a.lib, a.shelf, a.searches)
		a.manageWin.Update(func() {
			m := &a.manage
			m.backupBusy = false
			m.backupImport, m.backupSum = nil, archive.Summary{}
			if firstErr != nil {
				m.done(firstErr, "")
			} else {
				notice := fmt.Sprintf("已导入 %d 个源、%d 项源数据、%d 条阅读记录、%d 部收藏、%d 条搜索历史、%d 张图片",
					result.Sources, result.Fields, result.History, result.Favourites, result.Searches, result.Pages)
				if result.Cookies > 0 {
					notice += fmt.Sprintf("、%d 条登录状态", result.Cookies)
				}
				m.done(nil, notice)
			}
			a.refreshSources()
		})
	}()
}
