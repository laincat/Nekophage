package ui

import (
	"bytes"
	"encoding/json"
	"log"
	"os"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/laincat/veneracat-core/backup"
	"github.com/laincat/veneracat-core/plugin"
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
		err := a.writeBackup(path)
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

// writeBackup builds the archive from what this app holds.
func (a *App) writeBackup(path string) error {
	records := a.core.Records()
	sources := make([]backup.WriteSource, 0, len(records))
	for _, record := range records {
		// A rule or a video source is not a comic source: the official app's
		// comic_source/ entries are comic scripts only.
		if record.Type != plugin.TypeComic || len(record.Body) == 0 {
			continue
		}
		// The source's own stored object travels beside its script: without it a
		// restore brings back code that has lost its tokens and its chosen domain.
		data, err := a.core.Store().AllData(record.Key)
		if err != nil {
			return err
		}
		sources = append(sources, backup.WriteSource{
			Name: record.Key + ".js",
			Body: []byte(record.Body),
			Data: data,
		})
	}

	// The store keeps the newest first; the archive lists them oldest first.
	history := []string{}
	if a.searches != nil {
		items := a.searches.Items(200)
		for i := len(items) - 1; i >= 0; i-- {
			history = append(history, items[i])
		}
	}

	// Settings are kept per source in this client, so there is no app-level
	// document to put here; an empty object is honest about that.
	var buf bytes.Buffer
	if err := backup.Write(&buf, backup.WriteOptions{
		Sources:       sources,
		Settings:      map[string]json.RawMessage{},
		SearchHistory: history,
	}); err != nil {
		return err
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		log.Printf("nekophage: exporting %s: %v", path, err)
		return err
	}
	return nil
}

// backupRow draws the export control beside the import one.
func (a *App) exportRow(c *ui.Context, t *ui.Theme) {
	m := &a.manage
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
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
