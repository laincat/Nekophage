package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/laincat/nekophage/internal/core"
)

// A source declares its own settings — a CDN line, an ordering, an account
// token — and reads them back through getSetting. The client draws what the
// declaration asks for and stores the chosen value as JSON, which is what the
// source wrote; a number stays a number and a boolean stays a boolean.

// openSourceSettings reads a source's settings declaration and the values in
// force, then shows the panel.
func (a *App) openSourceSettings(key string) {
	a.manage.settingsFor = key
	a.manage.settings = nil
	a.manage.settingValues = nil
	a.manage.settingsBusy = true
	a.manage.settingsErr = ""
	a.manageWin.Update(func() {})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		list, err := a.core.Settings(ctx, key)
		values := map[string]json.RawMessage{}
		if err == nil {
			for _, s := range list {
				raw, ok, lerr := a.core.Store().LoadSetting(key, s.Key)
				switch {
				case lerr != nil:
					err = lerr
				case ok:
					values[s.Key] = raw
				default:
					values[s.Key] = s.Default
				}
				if err != nil {
					break
				}
			}
		}
		a.manageWin.Update(func() {
			m := &a.manage
			m.settingsBusy = false
			if err != nil {
				m.settingsErr = explain(err)
				return
			}
			m.settings, m.settingValues = list, values
		})
	}()
}

// saveSetting writes one value and keeps what is on screen in step.
func (a *App) saveSetting(key, settingKey string, value json.RawMessage) {
	go func() {
		err := a.core.Store().SaveSetting(key, settingKey, value)
		a.manageWin.Update(func() {
			m := &a.manage
			if err != nil {
				m.settingsErr = "保存失败:" + explain(err)
				return
			}
			m.settingsErr = ""
			if m.settingValues == nil {
				m.settingValues = map[string]json.RawMessage{}
			}
			m.settingValues[settingKey] = value
			m.done(nil, "已保存设置")
		})
		if err != nil {
			log.Printf("nekophage: saving %s/%s: %v", key, settingKey, err)
		}
	}()
}

// rawText is a JSON value as text, for a field or a comparison.
func rawText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return strings.TrimSpace(string(raw))
}

// boolValue reads a switch's stored value, falling back to the default.
func boolValue(raw json.RawMessage) bool {
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		return b
	}
	return false
}

// settingsPanel draws the open source's settings.
func (a *App) settingsPanel(c *ui.Context, t *ui.Theme) {
	m := &a.manage
	if m.settingsFor == "" {
		return
	}
	ui.Column(c).Padding(8, 16).Gap(6).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Textf(c, "「%s」的设置", m.settingsFor).FontSize(13).Bold()
			ui.Spacer(c)
			if m.settingsBusy {
				ui.Spinner(c)
			}
			if ui.Button(c, "收起").Clicked() {
				m.settingsFor, m.settings = "", nil
				a.manageWin.Update(func() {})
			}
		})
		if m.settingsErr != "" {
			ui.Text(c, m.settingsErr).FontSize(11).TextColor(ui.RGB(224, 49, 49)).Selectable()
		}
		if !m.settingsBusy && len(m.settings) == 0 && m.settingsErr == "" {
			ui.Text(c, "这个源没有声明设置。").FontSize(12).TextColor(t.TextMuted)
		}
		for _, s := range m.settings {
			a.settingRow(c, t, s)
		}
	})
	ui.Divider(c)
}

// settingRow draws one setting according to its type.
func (a *App) settingRow(c *ui.Context, t *ui.Theme, s core.SourceSetting) {
	m := &a.manage
	current := m.settingValues[s.Key]
	title := s.Title
	if title == "" {
		title = s.Key
	}
	ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Text(c, title).FontSize(12).TextColor(t.TextMuted)
		switch s.Type {
		case "select":
			// A select shows the option text; the value stored is the
			// option's own JSON, so a number stays a number.
			index := -1
			labels := make([]string, 0, len(s.Options))
			for i, o := range s.Options {
				text := o.Text
				if text == "" {
					text = rawText(o.Value)
				}
				labels = append(labels, text)
				if string(o.Value) == string(current) {
					index = i
				}
			}
			if index < 0 && len(labels) > 0 {
				index = 0
			}
			if len(labels) == 0 {
				ui.Text(c, "（没有选项）").FontSize(11).TextColor(t.TextMuted)
				return
			}
			pick := labels[max(0, index)]
			if ui.Select(c, &pick, labels).Width(160).Changed() {
				for i, label := range labels {
					if label == pick {
						a.saveSetting(m.settingsFor, s.Key, s.Options[i].Value)
						break
					}
				}
			}
		case "switch":
			on := boolValue(current)
			if ui.Switch(c, &on).Changed() {
				value, err := json.Marshal(on)
				if err == nil {
					a.saveSetting(m.settingsFor, s.Key, value)
				}
			}
		case "input":
			if m.editingSetting == s.Key {
				ui.TextInput(c, &m.editText).Width(220)
				if ui.Button(c, "保存").Clicked() {
					text := m.editText
					if s.Validator != "" {
						if err := checkPattern(s.Validator, text); err != nil {
							m.settingsErr = explain(err)
							a.manageWin.Update(func() {})
							return
						}
					}
					value, err := json.Marshal(text)
					if err == nil {
						a.saveSetting(m.settingsFor, s.Key, value)
					}
					m.editingSetting = ""
					a.manageWin.Update(func() {})
				}
				if ui.Button(c, "取消").Clicked() {
					m.editingSetting = ""
					a.manageWin.Update(func() {})
				}
				return
			}
			shown := rawText(current)
			if shown == "" {
				shown = "（未设置）"
			}
			ui.Text(c, shown).FontSize(11).SingleLine()
			if ui.Button(c, "修改").Clicked() {
				m.editingSetting, m.editText = s.Key, rawText(current)
				a.manageWin.Update(func() {})
			}
		case "callback":
			// A callback is a function in the source; it cannot be called
			// across the boundary, so the setting is shown but not run.
			ui.Text(c, "（源内的操作，客户端暂不支持）").FontSize(11).TextColor(t.TextMuted)
		default:
			ui.Textf(c, "%s", rawText(current)).FontSize(11).TextColor(t.TextMuted).SingleLine()
		}
	})
}

// checkPattern applies an input's validator, which the source wrote as a
// regular expression.
func checkPattern(pattern, text string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("校验规则无效:%w", err)
	}
	if !re.MatchString(text) {
		return fmt.Errorf("内容不符合要求:%s", pattern)
	}
	return nil
}

// devSetSetting applies "key=value" to a declared setting, the way a click
// would, so the tools can check the round trip.
func (a *App) devSetSetting(sourceKey, spec string) {
	name, value, ok := strings.Cut(spec, "=")
	if !ok {
		log.Printf("nekophage: setting %q is not key=value", spec)
		return
	}
	a.mu.Lock()
	list := append([]core.SourceSetting(nil), a.manage.settings...)
	a.mu.Unlock()
	for _, s := range list {
		if s.Key != name {
			continue
		}
		if s.Type == "select" {
			for _, o := range s.Options {
				if rawText(o.Value) == value {
					a.saveSetting(sourceKey, s.Key, o.Value)
					return
				}
			}
			log.Printf("nekophage: %s has no option %q", name, value)
			return
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return
		}
		a.saveSetting(sourceKey, s.Key, raw)
		return
	}
	log.Printf("nekophage: no setting named %q", name)
}
