package ui

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/laincat/nekophage/internal/core"
)

// The source log panel.
//
// A source's own console output — and Core's failures on its behalf, such as a
// value that could not be converted — used to go to stderr and nowhere else. A
// user who starts the app from a launcher therefore had no way to answer "why did
// this source fail": the line naming the real cause was invisible, and the app
// showed only the error the source produced afterwards, which is usually a
// misleading one such as a null dereference.
//
// Core keeps the last lines, so this is only a view over them.

// logPanel is the window's state; the zero value means "not opened yet".
type logPanel struct {
	win  *mygo.Window
	open bool
}

// openSourceLogs shows the panel, or focuses it when it is already up.
func (a *App) OpenSourceLogs() {
	if a.logs.win != nil && a.logs.open {
		a.logs.win.Show()
		a.logs.win.Focus()
		// A fresh read on every open: the point is what happened last.
		a.logs.win.Update(func() {})
		return
	}
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:   "源日志",
		Width:   760,
		Height:  480,
		Content: ui.View(func(c *ui.Context) { a.logView(c) }),
	})
	a.logs.win, a.logs.open = win, true
	win.OnClosed(func() { a.logs.open = false })
	win.Show()
	win.Focus()
}

// logView draws the lines and the controls over them.
func (a *App) logView(c *ui.Context) {
	t := c.Theme()
	lines := a.core.LogLines()
	ui.Column(c).Gap(8).Padding(12).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			if ui.Button(c, "复制全部").Clicked() {
				mygo.Clipboard.WriteText(sourceLogText(lines))
			}
			if ui.Button(c, "清空").Clicked() {
				a.core.ClearLogs()
				a.logs.win.Update(func() {})
			}
			if ui.Button(c, "刷新").Clicked() {
				a.logs.win.Update(func() {})
			}
			ui.Textf(c, "%d 行(最多保留 %d 行)", len(lines), core.MaxLogLines()).
				FontSize(11).TextColor(t.TextMuted)
		})
		if len(lines) == 0 {
			ui.Text(c, "还没有源输出日志。源打印的内容,以及 Core 代表源产生的错误,都会出现在这里。").
				FontSize(12).TextColor(t.TextMuted)
			return
		}
		// Oldest first, as a terminal shows it: the line before a failure is
		// usually the explanation for it.
		ui.Column(c).Gap(2).Children(func() {
			for _, line := range lines {
				ui.Text(c, formatLogLine(line)).FontSize(12).TextColor(logColor(t, line.Level))
			}
		})
	})
}

// logColor keeps errors apart from chatter without hiding either.
func logColor(t *ui.Theme, level string) ui.Color {
	switch level {
	case "error":
		return t.Danger
	case "warn", "warning":
		return t.Warning
	}
	return t.Text
}

// formatLogLine is one row: time, source, then what was said.
func formatLogLine(l core.LogLine) string {
	key := l.Key
	if key == "" {
		key = "(app)"
	}
	title := l.Title
	if title == "" {
		title = "JS"
	}
	// A source may log a multi-line value; one line per row keeps the panel
	// readable and the timestamp meaningful.
	content := strings.ReplaceAll(l.Content, "\r\n", " ⏎ ")
	content = strings.ReplaceAll(content, "\n", " ⏎ ")
	return fmt.Sprintf("%s  %-16s %s: %s", l.Time.Format("15:04:05"), key, title, content)
}

// sourceLogText is what 复制全部 puts on the clipboard, in a form that can be
// pasted into a bug report.
func sourceLogText(lines []core.LogLine) string {
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(formatLogLine(l))
		b.WriteString("\n")
	}
	return b.String()
}
