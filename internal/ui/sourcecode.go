package ui

import (
	"fmt"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/laincat/veneracat-core/plugin"
)

// sourceCodePanel owns one read-only snapshot. It never installs, executes or
// saves the text. Closing drops the editor (including selections and history),
// and a late background load cannot bring it back.
type sourceCodePanel struct {
	win    *mygo.Window
	key    string
	code   *editor.State
	name   string
	err    string
	closed bool
}

func sourceCodeState(records []plugin.Record, key string) (*editor.State, string, error) {
	for _, rec := range records {
		if rec.Key != key {
			continue
		}
		language := "JavaScript"
		switch rec.Type {
		case plugin.TypeComic, plugin.TypeVideoJS:
		case plugin.TypeKazumi:
			language = "JSON"
		default:
			return nil, "", fmt.Errorf("此源没有可查看的脚本")
		}
		buf := editor.NewBuffer(rec.Body)
		buf.SetLanguage(editor.LanguageNamed(language))
		return &editor.State{Buffer: buf, ReadOnly: true, Label: "源代码（只读）"}, rec.Name, nil
	}
	return nil, "", fmt.Errorf("源已被移除，请刷新源列表")
}

func (p *sourceCodePanel) loaded(code *editor.State, name string, err error) bool {
	if p.closed {
		return false
	}
	p.code, p.name = code, name
	if err != nil {
		p.err = explain(err)
	}
	return true
}

func (p *sourceCodePanel) release() {
	p.closed = true
	p.win, p.code = nil, nil
	p.key, p.name, p.err = "", "", ""
}

func (a *App) openSourceCode(key string) {
	if old := a.sourceCode; old != nil {
		if old.key == key {
			old.win.Show()
			old.win.Focus()
			return
		}
		old.win.Close()
	}
	p := &sourceCodePanel{key: key}
	win := mygo.NewWindow(mygo.WindowOptions{
		Title: "源码 · " + key, Width: 900, Height: 680,
		Content: ui.View(p.view),
	})
	p.win, a.sourceCode = win, p
	win.OnClosed(func() {
		p.release()
		if a.sourceCode == p {
			a.sourceCode = nil
		}
	})
	win.Show()
	win.Focus()
	// Registry.List copies installed records without evaluating the source.
	// Build the potentially large text buffer away from the UI thread.
	go func() {
		code, name, err := sourceCodeState(a.core.Records(), key)
		win.Update(func() { p.loaded(code, name, err) })
	}()
}

func (p *sourceCodePanel) view(c *ui.Context) {
	ui.Column(c).Fill().Gap(8).Padding(12).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "源码（只读）").Bold()
			ui.Text(c, p.name).Grow(1).SingleLine()
			if ui.Button(c, "关闭").Key("close-source-code").Clicked() && p.win != nil {
				p.win.Close()
			}
		})
		ui.Text(c, "已安装源的只读副本；可选择、复制和查找。重新打开以读取最新版本。").FontSize(12).TextColor(c.Theme().TextMuted)
		switch {
		case p.err != "":
			ui.Text(c, p.err).TextColor(c.Theme().Danger).Selectable()
		case p.code == nil:
			ui.Text(c, "正在读取源码…")
		default:
			editor.View(c, p.code).Grow(1).FillWidth()
		}
	})
}
