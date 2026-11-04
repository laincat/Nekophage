package ui

import (
	"context"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/laincat/nekophage/internal/collections"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/services"
	"github.com/laincat/nekophage/internal/store"
)

// userAgent is what the collection store tells servers this client is. The
// build sets it once; the default keeps a test honest.
var userAgent = "Nekophage"

// SetUserAgent records the client identity for collection requests.
func SetUserAgent(value string) {
	if value != "" {
		userAgent = value
	}
}

// collectionPanel is the state of the shared-source collection section: the
// subscriptions on disk, what each one offers, and the last result.
type collectionPanel struct {
	url        string
	rows       []collections.View
	installing string
	busy       string
	errText    string
	notice     string
}

// openCollections opens the collection manager for one action. The store is a
// file and the manager owns no goroutine of its own, so an action opens it,
// runs, and closes it: the window never holds a half-live cache.
func (a *App) openCollections() (*collections.Manager, error) {
	s, err := store.Open(a.core.DataDir())
	if err != nil {
		return nil, err
	}
	m, err := collections.Open(s, core.NewTransport(nil, userAgent, 0))
	if err != nil {
		return nil, err
	}
	return m, nil
}

// refreshCollections reads the subscriptions from disk.
func (a *App) refreshCollections() {
	m, err := a.openCollections()
	if err != nil {
		a.mu.Lock()
		a.cols.errText = explain(err)
		a.mu.Unlock()
		return
	}
	rows, listErr := m.List(collections.Platform(), collections.Capabilities())
	m.Close()
	a.mu.Lock()
	defer a.mu.Unlock()
	if listErr != nil {
		a.cols.errText = explain(listErr)
		return
	}
	a.cols.rows = rows
}

// collectionsSection draws the subscriptions and their entries.
func (a *App) collectionsSection(c *ui.Context, t *ui.Theme) {
	p := &a.cols
	ui.Divider(c)
	ui.Row(c).Padding(8, 16).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Textf(c, "统一源合集 %d 个", len(p.rows)).FontSize(13).Bold().Grow(1)
	})
	ui.Row(c).Padding(0, 16, 8, 16).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.TextInput(c, &p.url).Placeholder("合集链接 https://…/source.json").Grow(1)
		if ui.PrimaryButton(c, "订阅").Disabled(p.busy != "").Key("collection-follow").Clicked() {
			a.followCollection()
		}
	})
	if p.errText != "" {
		ui.Text(c, p.errText).Padding(0, 16).TextColor(ui.RGB(224, 49, 49)).FontSize(12).Selectable()
	}
	if p.notice != "" {
		ui.Text(c, p.notice).Padding(0, 16).TextColor(t.TextMuted).FontSize(12).Selectable()
	}
	if p.busy != "" {
		ui.Row(c).Padding(0, 16).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c)
			ui.Text(c, p.busy).FontSize(12).TextColor(t.TextMuted)
		})
	}
	if len(p.rows) == 0 {
		ui.Text(c, "还没有订阅合集。").Padding(6, 16).TextColor(t.TextMuted).FontSize(12)
		return
	}
	for _, view := range p.rows {
		a.collectionGroup(c, t, view)
	}
}

// collectionGroup draws one subscription and its entries.
func (a *App) collectionGroup(c *ui.Context, t *ui.Theme, view collections.View) {
	ui.Row(c).Padding(6, 16).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.Text(c, view.Name).SingleLine()
			ui.Text(c, view.ID+" · "+view.URL).FontSize(11).TextColor(t.TextMuted).SingleLine()
		}).Grow(1)
		if ui.Button(c, "刷新").Key("collection-refresh:" + view.ID).Clicked() {
			a.refreshCollection(view.ID)
		}
		if ui.Button(c, "取消订阅").Key("collection-unfollow:" + view.ID).Clicked() {
			a.unfollowCollection(view.ID)
		}
	})
	for _, entry := range view.Entries {
		a.collectionEntry(c, t, view, entry)
	}
}

// collectionEntry draws one entry: what it is, whether this client can install
// it, and the install button when it can.
func (a *App) collectionEntry(c *ui.Context, t *ui.Theme, view collections.View, entry collections.Entry) {
	key := view.ID + "/" + entry.ID
	ui.Row(c).Padding(4, 32).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.Text(c, entry.Name).FontSize(12).SingleLine()
			ui.Text(c, entryLine(entry)).FontSize(11).TextColor(t.TextMuted).SingleLine()
		}).Grow(1)
		if a.cols.installing == key {
			ui.Spinner(c)
			return
		}
		if !entry.Compatible {
			return
		}
		if entry.Kind != "comic-source.js" && entry.Kind != "service.js" {
			return
		}
		if ui.Button(c, "安装").Key("collection-install:" + key).Clicked() {
			a.installCollectionEntry(view.ID, entry.ID)
		}
	})
}

// entryLine is what an entry row says: kind, status and why it cannot be
// installed here.
func entryLine(entry collections.Entry) string {
	parts := []string{entry.Kind, entry.Status}
	if entry.Key != "" {
		parts = append(parts, "key "+entry.Key)
	}
	if !entry.Compatible {
		parts = append(parts, "不支持:缺 "+strings.Join(entry.Missing, ","))
	}
	return strings.Join(parts, " · ")
}

// followCollection subscribes to the URL in the field.
func (a *App) followCollection() {
	raw := strings.TrimSpace(a.cols.url)
	if raw == "" {
		a.mu.Lock()
		a.cols.errText = "合集链接不能为空"
		a.mu.Unlock()
		return
	}
	a.colsStart("正在订阅合集…")
	go func() {
		m, err := a.openCollections()
		if err == nil {
			ctx, cancel := a.manageContext()
			defer cancel()
			ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
			defer cancel()
			err = m.Follow(ctx, raw)
			m.Close()
		}
		a.manageWin.Update(func() {
			a.colsDone(err, "")
			if err == nil {
				a.cols.notice = "已订阅 " + raw
				a.cols.url = ""
			}
			a.refreshCollections()
		})
	}()
}

// refreshCollection refetches one subscription.
func (a *App) refreshCollection(id string) {
	a.colsStart("正在刷新 " + id + "…")
	go func() {
		m, err := a.openCollections()
		if err == nil {
			ctx, cancel := a.manageContext()
			defer cancel()
			ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
			defer cancel()
			err = m.Refresh(ctx, id)
			m.Close()
		}
		a.manageWin.Update(func() {
			a.colsDone(err, "已刷新 "+id)
			a.refreshCollections()
		})
	}()
}

// unfollowCollection forgets one subscription.
func (a *App) unfollowCollection(id string) {
	a.colsStart("正在取消订阅…")
	go func() {
		m, err := a.openCollections()
		if err == nil {
			err = m.Unfollow(id)
			m.Close()
		}
		a.manageWin.Update(func() {
			a.colsDone(err, "已取消订阅 "+id)
			a.refreshCollections()
		})
	}()
}

// installCollectionEntry installs one entry of a subscription.
func (a *App) installCollectionEntry(collectionID, entryID string) {
	key := collectionID + "/" + entryID
	a.mu.Lock()
	a.cols.installing = key
	a.cols.busy, a.cols.errText, a.cols.notice = "正在安装 "+entryID+"…", "", ""
	a.mu.Unlock()
	go func() {
		m, err := a.openCollections()
		var notice string
		if err == nil {
			ctx, cancel := a.manageContext()
			defer cancel()
			ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
			defer cancel()
			if _, err = collections.InstallEntry(ctx, m, collectionID, entryID, a.core.DataDir(), a.core, a.lib); err == nil {
				notice = "已安装 " + entryID
			}
			m.Close()
		}
		a.manageWin.Update(func() {
			a.cols.installing = ""
			a.cols.busy = ""
			if err != nil {
				a.cols.errText = explain(err)
				a.cols.notice = ""
				return
			}
			a.cols.errText = ""
			a.cols.notice = notice
			a.refreshServices()
			a.refreshSources()
		})
	}()
}

// colsStart marks what the collection section is doing.
func (a *App) colsStart(what string) {
	a.mu.Lock()
	a.cols.busy, a.cols.errText, a.cols.notice = what, "", ""
	a.mu.Unlock()
}

// colsDone records the result of one collection action.
func (a *App) colsDone(err error, notice string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cols.busy = ""
	if err != nil {
		a.cols.errText = explain(err)
		a.cols.notice = ""
		return
	}
	a.cols.errText = ""
	if notice != "" {
		a.cols.notice = notice
	}
}

// installedServices is a small helper for tests: the ids of installed services.
func (a *App) installedServices() []string {
	rows, err := services.List(a.core.DataDir())
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(rows))
	for _, rec := range rows {
		out = append(out, rec.ID)
	}
	return out
}
