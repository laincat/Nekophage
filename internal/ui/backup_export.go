package ui

import (
	"context"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/laincat/nekophage/internal/archive"
)

// Exporting a .venera backup.
//
// Core could read this format and never write one, so a user could move into this
// app and not out of it. What travels is the sources and the search history; the
// shelf and the per-source settings do not, because this client keeps them in its
// own shape rather than the official app's.
//
// Cookies never travel: they are per-source credentials, and an export is a file
// people mail to themselves.

// exportBackup writes a backup to the path the user picks.
func (a *App) exportBackup() {
	path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Parent:      a.manageWin,
		Title:       "导出 .venera 备份",
		DefaultPath: "nekophage.venera",
		ButtonLabel: "导出",
		Filters:     []mygo.FileFilter{{Name: "Venera 备份", Extensions: []string{"venera"}}},
	})
	if err != nil {
		a.manage.backupErr = explain(err)
		a.manageWin.Update(func() {})
		return
	}
	if path == "" {
		// The user closed the dialog; nothing to report.
		return
	}
	a.manage.backupBusy = true
	a.manage.backupErr, a.manage.backupNotice = "", ""
	a.manageWin.Update(func() {})
	go func() {
		ctx, cancel := a.manageContext()
		defer cancel()
		err := archive.Export(ctx, path, a.core, a.searches)
		a.manageWin.Update(func() {
			m := &a.manage
			m.backupBusy = false
			if err != nil {
				m.backupErr = explain(err)
				return
			}
			m.backupNotice = "已导出到 " + path
		})
	}()
}

func (a *App) writeBackup(path string) error {
	return archive.Export(context.Background(), path, a.core, a.searches)
}

// exportRow draws the export control beside the import one, and the way into the
// log panel: an export or an import that fails is exactly when the source's own
// lines explain why.
func (a *App) exportRow(c *ui.Context, t *ui.Theme) {
	m := &a.manage
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		if ui.Button(c, "源日志").Clicked() {
			a.OpenSourceLogs()
		}
		if ui.Button(c, "导出 .venera 备份").Disabled(m.backupBusy).Clicked() {
			a.exportBackup()
		}
		ui.Text(c, "导出源、源数据与搜索历史;不含 Cookie").FontSize(11).TextColor(t.TextMuted)
		if m.backupBusy {
			ui.Spinner(c)
			ui.Text(c, "正在写入…").FontSize(12).TextColor(t.TextMuted)
		}
	})
	if m.backupNotice != "" {
		ui.Text(c, m.backupNotice).FontSize(11).TextColor(t.TextMuted).Selectable()
	}
	if m.backupErr != "" {
		ui.Text(c, m.backupErr).FontSize(11).TextColor(ui.RGB(224, 49, 49)).Selectable()
	}
}
