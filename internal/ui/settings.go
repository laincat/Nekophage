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
	"github.com/laincat/nekophage/internal/browser"
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
	a.manage.accountMessage = ""
	a.manage.accountUser, a.manage.accountPass = "", ""
	a.manageWin.Update(func() {})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		translations, terr := a.core.Translations(ctx, key)
		if terr != nil {
			translations = nil
		}
		a.manageWin.Update(func() {
			a.manage.translations = translations
		})
		styles := a.core.AccountSupport(ctx, key)
		fields, ferr := a.core.AccountCookieFields(ctx, key)
		if ferr != nil {
			styles.Cookies = false
		}
		blank := make([]string, len(fields))
		a.manageWin.Update(func() {
			a.manage.account = styles
			a.manage.accountFields = fields
			a.manage.accountCookies = blank
			a.manage.accountEditing = -1
		})
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

// accountPanel draws the source's own login, when it declares one. The source
// does the talking: what it answers, or the error it throws, is what the panel
// shows, because the reason usually names the field it did not like.
func (a *App) accountPanel(c *ui.Context, t *ui.Theme) {
	m := &a.manage
	if m.settingsFor == "" {
		return
	}
	if !m.account.Password && !m.account.Cookies && !m.account.Webview {
		return
	}
	key := m.settingsFor
	// The source logs in on its own page. The window is a real browser with the
	// source's own partition, so what it earns here is what its requests send.
	if m.account.Webview {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "登录").FontSize(11).TextColor(t.TextMuted).Width(96)
			if m.accountBusy {
				ui.Spinner(c)
				ui.Text(c, "正在打开登录页面…").FontSize(11).TextColor(t.TextMuted)
				return
			}
			if ui.PrimaryButton(c, "在网页里登录").Clicked() {
				a.webviewLogin(key)
			}
			ui.Text(c, "会打开该源的登录页,登录完成后窗口自动关闭。").FontSize(11).TextColor(t.TextMuted)
		})
	}
	if m.account.Cookies && len(m.accountCookies) > 0 {
		a.cookieLoginPanel(c, t, key)
	}
	if m.account.Password {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "账号").FontSize(11).TextColor(t.TextMuted).Width(96)
			ui.TextInput(c, &m.accountUser).Width(180)
			ui.Text(c, "密码").FontSize(11).TextColor(t.TextMuted)
			ui.TextInput(c, &m.accountPass).Width(160)
			if m.accountBusy {
				ui.Spinner(c)
				return
			}
			if ui.PrimaryButton(c, "登录").Clicked() {
				user, pass := m.accountUser, m.accountPass
				if user == "" || pass == "" {
					m.accountMessage = "账号和密码都要填。"
					a.manageWin.Update(func() {})
					return
				}
				a.loginAccount(key, user, pass)
			}
		})
	}
	a.accountFooter(c, t, key)
}

// accountFooter draws what every login style shares: the source's own logout and
// the last answer. Both used to sit inside the password branch, so a source that
// logs in with pasted cookies reported "these cookies are not usable" to nobody
// and offered no way to log out.
func (a *App) accountFooter(c *ui.Context, t *ui.Theme, key string) {
	m := &a.manage
	if m.account.Logout {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "").Width(96)
			if m.accountBusy {
				return
			}
			if ui.Button(c, "登出").Clicked() {
				a.logoutAccount(key)
			}
		})
	}
	if m.accountMessage != "" {
		ui.Text(c, m.accountMessage).FontSize(11).Selectable()
	}
}

// cookieLoginPanel draws one field per cookie name the source asked for and
// hands the values to the source's own validate, which sets them.
func (a *App) cookieLoginPanel(c *ui.Context, t *ui.Theme, key string) {
	m := &a.manage
	for i := range m.accountCookies {
		i := i
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			label := ""
			if i < len(m.accountFields) {
				label = m.accountFields[i]
			}
			ui.Text(c, label).FontSize(11).TextColor(t.TextMuted).Width(96)
			ui.TextInput(c, &m.accountCookies[i]).Width(300)
		})
	}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "").Width(96)
		if m.accountBusy {
			ui.Spinner(c)
			return
		}
		if ui.PrimaryButton(c, "校验并保存").Clicked() {
			a.loginWithCookies(key, append([]string(nil), m.accountCookies...))
		}
	})
}

// loginWithCookies asks the source to validate pasted cookies. A false answer
// means the source looked and found them unusable, which is not the same as an
// error and is reported as such.
func (a *App) loginWithCookies(key string, values []string) {
	a.manage.accountBusy = true
	a.manage.accountMessage = ""
	a.manageWin.Update(func() {})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		ok, err := a.core.LoginWithCookies(ctx, key, values)
		a.manageWin.Update(func() {
			m := &a.manage
			m.accountBusy = false
			switch {
			case err != nil:
				m.accountMessage = explain(err)
			case ok:
				m.accountMessage = "登录：已保存"
			default:
				m.accountMessage = "源校验未通过：这组 Cookie 不能用。"
			}
		})
	}()
}

// callSettingCallback asks the source to run one of its own callbacks.
func (a *App) callSettingCallback(key, settingKey string) {
	a.manage.callbackBusy = settingKey
	if a.manage.callbackMsgs == nil {
		a.manage.callbackMsgs = map[string]string{}
	}
	delete(a.manage.callbackMsgs, settingKey)
	a.manageWin.Update(func() {})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		answer, err := a.core.CallSettingCallback(ctx, key, settingKey)
		a.manageWin.Update(func() {
			m := &a.manage
			m.callbackBusy = ""
			if m.callbackMsgs == nil {
				m.callbackMsgs = map[string]string{}
			}
			switch {
			case err != nil:
				// The source's own words: "no network", "Login expired", or
				// whatever it chose.
				m.callbackMsgs[settingKey] = explain(err)
			case answer == "":
				m.callbackMsgs[settingKey] = "已完成"
			default:
				m.callbackMsgs[settingKey] = answer
			}
		})
	}()
}

// loginAccount asks the source to log in and reports what it said.
func (a *App) loginAccount(key, account, password string) {
	a.manage.accountBusy = true
	a.manage.accountMessage = ""
	a.manageWin.Update(func() {})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		message, err := a.core.LoginAccount(ctx, key, account, password)
		a.manageWin.Update(func() {
			m := &a.manage
			m.accountBusy = false
			if err != nil {
				// The source's own words: the reason usually names the field.
				m.accountMessage = explain(err)
				return
			}
			if message == "" {
				message = "已完成"
			}
			m.accountMessage = "登录：" + message
			m.accountPass = ""
		})
	}()
}

// logoutAccount asks the source to forget the session it holds.
func (a *App) logoutAccount(key string) {
	a.manage.accountBusy = true
	a.manage.accountMessage = ""
	a.manageWin.Update(func() {})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := a.core.LogoutAccount(ctx, key)
		a.manageWin.Update(func() {
			m := &a.manage
			m.accountBusy = false
			if err != nil {
				m.accountMessage = explain(err)
				return
			}
			m.accountMessage = "已登出"
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
		a.accountPanel(c, t)
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
	// A source may declare a key and leave the readable text to its translation
	// table, which is why ehentai's panel used to read "domain" and "ehevent".
	if text, ok := m.translations[title]; ok && text != "" {
		title = text
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
			// Option text may be a key too.
			for i, label := range labels {
				if text, ok := m.translations[label]; ok && text != "" {
					labels[i] = text
				}
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
			// A callback is a function in the source. Nothing can be carried
			// across by value, but the source can run it, so the panel does that
			// and shows what came back.
			//
			// The button stays while it runs. Replacing it with a spinner takes the
			// control away mid-press, which reads as a glitch rather than as work in
			// progress; the press is ignored while a callback is already running.
			// A source names its own button ("Refresh"); falling back to the
			// generic word keeps a source that names none usable.
			label := s.ButtonText
			if label == "" {
				label = "执行"
			}
			if ui.Button(c, label).Clicked() && m.callbackBusy == "" {
				a.callSettingCallback(m.settingsFor, s.Key)
			}
			if m.callbackBusy == s.Key {
				ui.Row(c).Gap(8).Children(func() {
					ui.Spinner(c)
					ui.Text(c, "正在执行…").FontSize(11).TextColor(t.TextMuted)
				})
			}
			if msg := m.callbackMsgs[s.Key]; msg != "" {
				ui.Text(c, msg).FontSize(11).TextColor(t.TextMuted).Selectable()
			}
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

// webviewLogin opens the source's own login page in a browser window and keeps
// whatever cookies it earns in the source's partition.
func (a *App) webviewLogin(key string) {
	m := &a.manage
	if m.accountBusy {
		return
	}
	a.mu.Lock()
	m.accountBusy = true
	m.errText, m.notice = "", ""
	a.mu.Unlock()
	a.manageWin.Update(func() {})
	go func() {
		login := browser.NewLoginWindow(a.core, a.lib.Jar(), key, core.ComicPartition(key))
		stored, err := login.Run(context.Background())
		a.manageWin.Update(func() {
			a.mu.Lock()
			m.accountBusy = false
			if err != nil {
				m.accountMessage = explain(err)
			} else {
				// The cookies are in the partition now; whether the source agrees
				// it is logged in is its own answer.
				m.accountMessage = fmt.Sprintf("登录状态已保存(%d 个 Cookie),可以点「校验」让源自己确认", stored)
			}
			a.mu.Unlock()
		})
	}()
}
