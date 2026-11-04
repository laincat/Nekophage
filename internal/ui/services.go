package ui

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/laincat/veneracat-core/service"

	"github.com/laincat/nekophage/internal/services"
)

// servicePanel is the state of the service-plugin section of the source
// manager. It is small on purpose: the list is what is installed, and the
// editor edits one account key at a time.
type servicePanel struct {
	rows     []services.Record
	selected string
	key      string
	value    string
	removing string
	busy     string
	errText  string
	notice   string
	// methods is what each service said it offers, filled on demand.
	methods map[string][]service.Method
	// running names the "id/method" call in flight, result its last answer.
	running string
	result  map[string]string
	// payload is the JSON text handed to the method, and resultFull says which
	// answers are shown in full rather than truncated.
	payload    string
	resultFull map[string]bool
	// cancel stops the method that is running, if one is.
	cancel context.CancelFunc
}

// methodLabel is what a method button says: its title when it has one, its
// name otherwise.
func methodLabel(m service.Method) string {
	if strings.TrimSpace(m.Title) != "" {
		return m.Title
	}
	return m.Name
}

// shortResult trims a JSON answer to something a panel can show without
// pushing everything else off the window.
func shortResult(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 240 {
		return text[:240] + "…"
	}
	return text
}

// refreshServices reads the installed services from the profile.
func (a *App) refreshServices() {
	rows, err := services.List(a.core.DataDir())
	a.mu.Lock()
	defer a.mu.Unlock()
	if err != nil {
		a.svc.errText = explain(err)
		return
	}
	a.svc.rows = rows
	if a.svc.selected == "" && len(rows) > 0 {
		a.svc.selected = rows[0].ID
	}
	if a.svc.selected != "" {
		found := false
		for _, rec := range rows {
			if rec.ID == a.svc.selected {
				found = true
			}
		}
		if !found {
			a.svc.selected = ""
		}
	}
}

// servicesSection draws the service plugins: what is installed, where to
// install another one, and the account values of the selected one.
func (a *App) servicesSection(c *ui.Context, t *ui.Theme) {
	p := &a.svc
	ui.Divider(c)
	ui.Row(c).Padding(8, 16).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Textf(c, "服务插件 %d 个", len(p.rows)).FontSize(13).Bold().Grow(1)
		if ui.Button(c, "安装服务包…").Disabled(p.busy != "").Key("service-install").Clicked() {
			a.installServiceFile()
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
		ui.Text(c, "还没有安装服务。服务包是带 service.json 的 zip。").Padding(6, 16).TextColor(t.TextMuted).FontSize(12)
		return
	}
	for _, rec := range p.rows {
		a.serviceRow(c, t, rec)
		a.serviceMethods(c, t, rec)
	}
	a.serviceAccountEditor(c, t)
}

// serviceRow shows one installed service.
func (a *App) serviceRow(c *ui.Context, t *ui.Theme, rec services.Record) {
	line := serviceLine(rec)
	id := rec.ID
	ui.Row(c).Padding(6, 16).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.Text(c, rec.Name).SingleLine()
			ui.Text(c, line).FontSize(11).TextColor(t.TextMuted).SingleLine()
		}).Grow(1)
		if ui.Button(c, "方法").Key("service-methods:" + id).Clicked() {
			a.loadServiceMethods(id)
		}
		if ui.Button(c, "账号").Key("service-accounts:" + id).Clicked() {
			a.mu.Lock()
			a.svc.selected = id
			a.mu.Unlock()
		}
		if a.svc.removing == id {
			ui.Spinner(c)
		} else if ui.Button(c, "移除").Key("service-remove:" + id).Clicked() {
			a.removeService(id)
		}
	})
}

// serviceLine is what a row says about a service: identity, requested
// permissions and how many account values exist.
func serviceLine(rec services.Record) string {
	parts := []string{rec.ID, "v" + rec.Version}
	if len(rec.Permissions) > 0 {
		names := make([]string, 0, len(rec.Permissions))
		for _, p := range rec.Permissions {
			names = append(names, string(p))
		}
		parts = append(parts, "权限 "+strings.Join(names, ","))
	} else {
		parts = append(parts, "无权限")
	}
	if rec.Accounts > 0 {
		parts = append(parts, "账号 "+strconv.Itoa(rec.Accounts))
	}
	if rec.Origin != "" {
		parts = append(parts, rec.Origin)
	}
	return strings.Join(parts, " · ")
}

// serviceMethods shows what a service said it offers, once it was asked.
func (a *App) serviceMethods(c *ui.Context, t *ui.Theme, rec services.Record) {
	p := &a.svc
	methods, asked := p.methods[rec.ID]
	if !asked {
		return
	}
	if len(methods) == 0 {
		ui.Text(c, "该服务没有提供可运行的方法。").Padding(0, 32, 6, 16).FontSize(11).TextColor(t.TextMuted)
		return
	}
	ui.Row(c).Padding(0, 32, 6, 16).Gap(6).AlignItems(ui.Center).Children(func() {
		// No Key here: MyGo initialises an input's state at construction, so a
		// key set afterwards panics. The binding pointer identifies it.
		ui.TextInput(c, &p.payload).Placeholder("参数 JSON（可空）").Width(220)
		for _, method := range methods {
			key := rec.ID + "/" + method.Name
			if p.running == key {
				ui.Spinner(c)
				continue
			}
			method := method
			if ui.Button(c, methodLabel(method)).Key("service-run:" + key).Clicked() {
				a.runServiceMethod(rec.ID, method.Name)
			}
		}
		if p.running != "" {
			if ui.Button(c, "取消").Key("service-cancel").Clicked() {
				if p.cancel != nil {
					p.cancel()
				}
			}
		}
	})
	if result := p.result[rec.ID]; result != "" {
		full := p.resultFull[rec.ID]
		ui.Row(c).Padding(0, 32, 6, 16).Gap(6).AlignItems(ui.Center).Children(func() {
			if full || len(shortResult(result)) < len(result) {
				if ui.Button(c, map[bool]string{true: "收起", false: "展开"}[full]).Key("service-result:" + rec.ID).Clicked() {
					if p.resultFull == nil {
						p.resultFull = map[string]bool{}
					}
					p.resultFull[rec.ID] = !full
				}
			}
			if full {
				ui.Text(c, result).FontSize(11).TextColor(t.TextMuted).Selectable().Grow(1)
			} else {
				ui.Text(c, shortResult(result)).FontSize(11).TextColor(t.TextMuted).SingleLine().Grow(1)
			}
		})
	}
}

// methodPayload reads the argument field: empty means no payload, anything else
// has to be JSON. Sending invalid text would make the service fail with a
// parsing error that says nothing about the field the user typed in.
func methodPayload(text string) (json.RawMessage, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	if !json.Valid([]byte(text)) {
		return nil, errors.New("参数不是合法 JSON")
	}
	return json.RawMessage(text), nil
}

// serviceSupervisor opens, once per window, the supervisor that runs services.
// Keeping one means a service stays compiled between calls and a call can be
// cancelled by interrupting the runtime that is running it.
func (a *App) serviceSupervisor() (*service.Supervisor, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.svcSup != nil {
		return a.svcSup, nil
	}
	sup, err := services.NewSupervisor(a.core.DataDir(), "Nekophage")
	if err != nil {
		return nil, err
	}
	a.svcSup = sup
	return sup, nil
}

// closeServiceSupervisor unloads every service; the window calls it when it
// closes so a background script does not outlive the panel.
func (a *App) closeServiceSupervisor() {
	a.mu.Lock()
	sup := a.svcSup
	a.svcSup = nil
	a.mu.Unlock()
	if sup != nil {
		sup.Close()
	}
}

// loadServiceMethods asks a service what it offers.
func (a *App) loadServiceMethods(id string) {
	a.mu.Lock()
	a.svc.busy = "正在询问 " + id + "…"
	a.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		sup, err := a.serviceSupervisor()
		var methods []service.Method
		if err == nil {
			methods, err = sup.Describe(ctx, id)
		}
		a.manageWin.Update(func() {
			a.svc.busy = ""
			if a.svc.methods == nil {
				a.svc.methods = map[string][]service.Method{}
			}
			if err != nil {
				a.svc.errText = explain(err)
				return
			}
			a.svc.errText = ""
			a.svc.methods[id] = methods
			a.svc.notice = ""
		})
	}()
}

// runServiceMethod runs one method and shows its answer.
func (a *App) runServiceMethod(id, method string) {
	key := id + "/" + method
	a.mu.Lock()
	a.svc.running = key
	a.mu.Unlock()
	payload, payloadErr := methodPayload(a.svc.payload)
	if payloadErr != nil {
		a.svc.running = ""
		a.svc.errText = payloadErr.Error()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	a.mu.Lock()
	a.svc.cancel = cancel
	a.mu.Unlock()
	go func() {
		defer cancel()
		sup, err := a.serviceSupervisor()
		var out []byte
		if err == nil {
			out, err = sup.Call(ctx, id, method, payload)
		}
		a.manageWin.Update(func() {
			a.svc.running = ""
			a.svc.cancel = nil
			if a.svc.result == nil {
				a.svc.result = map[string]string{}
			}
			if err != nil {
				a.svc.errText = explain(err)
				return
			}
			a.svc.errText = ""
			a.svc.result[id] = string(out)
		})
	}()
}

// serviceAccountEditor edits one account key of the selected service. Values
// are write-only here: the list shows which keys exist, never what they hold.
func (a *App) serviceAccountEditor(c *ui.Context, t *ui.Theme) {
	p := &a.svc
	ids := make([]string, 0, len(p.rows))
	for _, rec := range p.rows {
		ids = append(ids, rec.ID)
	}
	if p.selected == "" {
		p.selected = ids[0]
	}
	ui.Row(c).Padding(8, 16).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Select(c, &p.selected, ids).Width(180).Label("服务").Disabled(p.busy != "").Changed()
	})
	ui.Row(c).Padding(0, 16, 8, 16).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.TextInput(c, &p.key).Placeholder("账号键，如 user").Width(180)
		ui.TextInput(c, &p.value).Placeholder("值").Password().Grow(1)
		if ui.PrimaryButton(c, "保存").Disabled(p.busy != "" || p.selected == "").Key("service-account-save").Clicked() {
			a.saveServiceAccount()
		}
		if ui.Button(c, "删除").Disabled(p.busy != "" || p.selected == "").Key("service-account-delete").Clicked() {
			a.deleteServiceAccount()
		}
	})
	keys := a.serviceAccountKeys(p.selected)
	if len(keys) == 0 {
		ui.Text(c, "该服务还没有账号值；secrets 权限只有保存后才会生效。").Padding(0, 16, 8, 16).FontSize(12).TextColor(t.TextMuted)
		return
	}
	ui.Textf(c, "已保存的键：%s（值不显示）", strings.Join(keys, "、")).Padding(0, 16, 8, 16).FontSize(12).TextColor(t.TextMuted)
}

// serviceAccountKeys reads the stored key names of one service.
func (a *App) serviceAccountKeys(id string) []string {
	if id == "" {
		return nil
	}
	accounts, err := services.OpenAccounts(a.core.DataDir(), id)
	if err != nil {
		return nil
	}
	keys, err := accounts.Keys()
	if err != nil {
		return nil
	}
	return keys
}

// installServiceFile asks for a package and installs it.
func (a *App) installServiceFile() {
	a.mu.Lock()
	a.svc.start("正在选择服务包…")
	a.mu.Unlock()
	a.manageWin.Update(func() {})
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent: a.manageWin, Title: "选择服务包",
			Filters: []mygo.FileFilter{{Name: "服务包", Extensions: []string{"zip"}}},
		})
		if err != nil || len(paths) == 0 {
			a.manageWin.Update(func() {
				a.svc.busy, a.svc.errText = "", ""
			})
			return
		}
		a.manageWin.Update(func() { a.svc.busy = "正在安装服务包…" })
		archive, err := os.ReadFile(paths[0])
		if err == nil {
			sum := sha256.Sum256(archive)
			ctx, cancel := a.manageContext()
			defer cancel()
			ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
			defer cancel()
			var rec *services.Record
			rec, err = services.Install(ctx, a.core.DataDir(), "file:"+baseName(paths[0]), hex.EncodeToString(sum[:]), archive)
			a.manageWin.Update(func() {
				a.svc.done(err, "")
				if err == nil {
					a.svc.selected = rec.ID
					a.svc.notice = "已安装 " + rec.Name
				}
				a.refreshServices()
			})
			return
		}
		a.manageWin.Update(func() {
			a.svc.done(err, "")
		})
	}()
}

// removeService uninstalls one service.
func (a *App) removeService(id string) {
	a.mu.Lock()
	a.svc.removing = id
	a.mu.Unlock()
	go func() {
		err := services.Remove(a.core.DataDir(), id)
		a.manageWin.Update(func() {
			a.svc.removing = ""
			a.refreshServices()
			if err != nil {
				a.svc.errText = explain(err)
				return
			}
			a.svc.notice = "已移除 " + id
			a.svc.errText = ""
		})
	}()
}

// saveServiceAccount stores one account value for the selected service.
func (a *App) saveServiceAccount() {
	id, key, value := a.svc.selected, strings.TrimSpace(a.svc.key), a.svc.value
	if key == "" {
		a.svc.errText = "账号键不能为空"
		return
	}
	a.mu.Lock()
	a.svc.busy = "正在保存账号值…"
	a.mu.Unlock()
	go func() {
		accounts, err := services.OpenAccounts(a.core.DataDir(), id)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			err = accounts.Set(ctx, key, value)
		}
		a.manageWin.Update(func() {
			a.svc.busy = ""
			if err != nil {
				a.svc.errText = explain(err)
				return
			}
			a.svc.errText = ""
			a.svc.notice = "已保存账号项 " + key
			a.svc.value = ""
			a.refreshServices()
		})
	}()
}

// deleteServiceAccount removes one account value of the selected service.
func (a *App) deleteServiceAccount() {
	id, key := a.svc.selected, strings.TrimSpace(a.svc.key)
	if key == "" {
		a.svc.errText = "账号键不能为空"
		return
	}
	a.mu.Lock()
	a.svc.busy = "正在删除账号值…"
	a.mu.Unlock()
	go func() {
		accounts, err := services.OpenAccounts(a.core.DataDir(), id)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			err = accounts.Remove(ctx, key)
		}
		a.manageWin.Update(func() {
			a.svc.busy = ""
			if err != nil {
				a.svc.errText = explain(err)
				return
			}
			a.svc.errText = ""
			a.svc.notice = "已删除账号项 " + key
			a.refreshServices()
		})
	}()
}

// start marks what the panel is doing.
func (p *servicePanel) start(what string) {
	p.busy, p.errText, p.notice = what, "", ""
}

// done records the result of one action.
func (p *servicePanel) done(err error, notice string) {
	p.busy = ""
	if err != nil {
		p.errText = explain(err)
		p.notice = ""
		return
	}
	p.errText = ""
	if notice != "" {
		p.notice = notice
	}
}

// baseName is the file name of a path, for an install origin.
func baseName(path string) string {
	if i := strings.LastIndexAny(path, `/\`); i >= 0 {
		return path[i+1:]
	}
	return path
}
