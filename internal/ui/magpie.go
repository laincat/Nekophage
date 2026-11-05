package ui

import (
	"context"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/laincat/nekophage/internal/magpie"
)

// The AI window connects to the Magpie installed on this machine.
//
// It is a connection screen, not an agent: it shows what the gateway offers,
// sends one task and prints the answer as it streams in. Routing, accounts,
// quota and masking stay inside Magpie, which is the whole reason this is a
// thin client rather than a second plugin host.
type aiState struct {
	addr     string
	protocol string
	model    string
	prompt   string
	// tier is which difficulty the user is asking for; tierModels is the model
	// or group each tier maps to. The mapping is the user's, not a classifier:
	// deciding what "complex" means is not this window's job.
	tier       int
	tierModels [3]string

	session *magpie.Session
	models  []magpie.Model

	busy    string
	errText string
	notice  string

	answer  string
	running bool
	cancel  context.CancelFunc
	// lifetime ends when the AI window closes: the connection test, the model
	// catalog and a running answer all belong to that window, not to the app.
	// stopLifetime ends it; cancel stops only the running answer.
	lifetime     context.Context
	stopLifetime context.CancelFunc
	// lastFlush throttles redraws: a stream that redraws the window for every
	// token spends more time painting than the network does.
	lastFlush time.Time
}

// openAI shows the AI window, creating it once.
func (a *App) openAI() {
	if a.aiWin == nil || a.aiWin.IsDestroyed() {
		a.aiWin = mygo.NewWindow(mygo.WindowOptions{
			Title: "AI · 本机 Magpie", Width: 760, Height: 720, Content: ui.View(a.aiView),
		})
		a.mu.Lock()
		a.ai.lifetime, a.ai.stopLifetime = context.WithCancel(context.Background())
		a.mu.Unlock()
		a.aiWin.OnClosed(func() {
			a.mu.Lock()
			if a.ai.stopLifetime != nil {
				a.ai.stopLifetime()
				a.ai.lifetime, a.ai.stopLifetime = nil, nil
			}
			if a.ai.cancel != nil {
				a.ai.cancel()
				a.ai.cancel = nil
			}
			a.ai.running = false
			a.mu.Unlock()
		})
		a.aiWin.Show()
		a.loadAISettings()
		return
	}
	a.aiWin.Show()
	a.aiWin.Focus()
}

// aiContext derives one request's context from the AI window's lifetime, so
// closing the window stops a test, a catalog read or a streamed answer. The
// caller must not hold a.mu: the parent is read under it.
func (a *App) aiContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	a.mu.Lock()
	parent := a.ai.lifetime
	a.mu.Unlock()
	return aiContextFrom(parent, timeout)
}

// aiContextLocked is aiContext for a caller that already holds a.mu.
func aiContextLocked(lifetime context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return aiContextFrom(lifetime, timeout)
}

func aiContextFrom(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	if timeout <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, timeout)
}

// loadAISettings reads the stored connection and fills the fields.
func (a *App) loadAISettings() {
	path := a.aiSettingsPath()
	session, err := magpie.OpenSession(path)
	a.mu.Lock()
	a.ai.errText = ""
	if err != nil {
		a.ai.errText = err.Error()
		a.mu.Unlock()
		a.aiWin.Update(func() {})
		return
	}
	settings := session.Settings()
	a.ai.session = session
	a.ai.addr = settings.BaseURL
	a.ai.protocol = string(settings.Protocol)
	a.ai.model = settings.Model
	for i, task := range magpie.Tasks {
		a.ai.tierModels[i] = settings.Tiers[string(task)]
	}
	a.mu.Unlock()
	a.aiWin.Update(func() {})
}

func (a *App) aiSettingsPath() string {
	return a.core.DataDir() + "/magpie.json"
}

// aiView draws the window.
func (a *App) aiView(c *ui.Context) {
	t := c.Theme()
	a.mu.Lock()
	state := a.ai
	a.mu.Unlock()

	ui.Column(c).Fill().Children(func() {
		ui.Column(c).Padding(16).Gap(8).Children(func() {
			ui.Text(c, "连接本机 Magpie").FontSize(16).Bold()
			ui.Text(c, "Nekophage 只负责连接:插件、供应商账号、登录刷新与路由都在 Magpie 里。先启动 Magpie,再点下面的按钮。").FontSize(12).TextColor(t.TextMuted)

			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.TextInput(c, &a.ai.addr).Placeholder("127.0.0.1:3425").Width(240)
				if ui.Button(c, "测试连接").Disabled(state.busy != "").Clicked() {
					a.aiSaveAndTest()
				}
				if ui.Button(c, "读取模型").Disabled(state.busy != "" || state.session == nil).Clicked() {
					a.aiLoadModels()
				}
				if ui.Button(c, "取消").Disabled(!state.running).Clicked() {
					a.aiStop()
				}
			})

			if state.busy != "" {
				ui.Text(c, state.busy).FontSize(12).TextColor(t.TextMuted)
			}
			if state.errText != "" {
				ui.Text(c, state.errText).FontSize(12).TextColor(ui.RGB(224, 49, 49)).Selectable()
			}
			if state.notice != "" {
				ui.Text(c, state.notice).FontSize(12).TextColor(t.TextMuted)
			}
		})

		ui.Divider(c)

		ui.Column(c).Padding(16).Gap(8).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "这次按").FontSize(12).TextColor(t.TextMuted)
				if ui.Segmented(c, &a.ai.tier, "简单", "普通", "复杂").Width(180).Changed() {
					a.aiWin.Update(func() {})
				}
				ui.Text(c, tierNote(state)).FontSize(12).TextColor(t.TextMuted)
			})
			// Each tier names a model or routing group. An empty one falls back
			// to the single model below, then to Magpie's own default route.
			for i, task := range magpie.Tasks {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, tierLabel(task)).FontSize(12).TextColor(t.TextMuted).Width(48)
					ui.TextInput(c, &a.ai.tierModels[i]).Placeholder("这个档位用哪个模型或组").Grow(1)
				})
			}
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "默认模型").FontSize(12).TextColor(t.TextMuted).Width(48)
				ui.TextInput(c, &a.ai.model).Placeholder("档位没填时用这个,再没有就交给 Magpie").Grow(1)
			})
			if len(state.models) > 0 {
				ui.Text(c, "目录中有 "+itoa(len(state.models))+" 个模型或路由组:"+modelPreview(state.models)).FontSize(12).TextColor(t.TextMuted)
			}
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.TextInput(c, &a.ai.prompt).Placeholder("要 Magpie 做什么").Grow(1)
				if ui.PrimaryButton(c, "发送").Disabled(state.running || strings.TrimSpace(state.prompt) == "").Clicked() {
					a.aiSend()
				}
			})
		})

		ui.Divider(c)

		ui.Column(c).Fill().Padding(16).Children(func() {
			ui.Scroll(c).Children(func() {
				if state.answer == "" {
					ui.Text(c, "回答会显示在这里。").FontSize(12).TextColor(t.TextMuted)
					return
				}
				ui.Text(c, state.answer).Selectable()
			})
		})
	})
}

// tierMapping is what the three fields hold, as the settings keep it.
func (a *App) tierMapping() map[string]string {
	out := map[string]string{}
	for i, task := range magpie.Tasks {
		if value := strings.TrimSpace(a.ai.tierModels[i]); value != "" {
			out[string(task)] = value
		}
	}
	return out
}

// applyAIFields copies what is on screen into settings, so pressing 测试连接
// or 发送 uses the address and model the user just typed. Reading only the
// stored settings made the 默认模型 field decorative: the value stayed empty,
// and the gateway answered HTTP 400 for a request with no model.
//
// The caller holds a.mu.
func (a *App) applyAIFields(settings magpie.Settings) magpie.Settings {
	settings.BaseURL = strings.TrimSpace(a.ai.addr)
	settings.Model = strings.TrimSpace(a.ai.model)
	settings.Tiers = a.tierMapping()
	return settings
}

// clampTier keeps an out-of-range index from panicking; 1 is "normal", the
// documented default.
func clampTier(tier int) int {
	if tier < 0 || tier >= len(magpie.Tasks) {
		return 1
	}
	return tier
}

func tierLabel(task magpie.Task) string {
	switch task {
	case magpie.TaskSimple:
		return "简单"
	case magpie.TaskComplex:
		return "复杂"
	default:
		return "普通"
	}
}

// tierNote says where the chosen tier currently points, so an unmapped tier is
// visible before the request goes out.
func tierNote(state aiState) string {
	index := clampTier(state.tier)
	if chosen := strings.TrimSpace(state.tierModels[index]); chosen != "" {
		return "→ " + chosen
	}
	if fallback := strings.TrimSpace(state.model); fallback != "" {
		return "→ " + fallback + "(默认模型)"
	}
	return "→ 还没选模型，先读取模型目录"
}

// modelPreview lists the first few IDs: enough to show the connection works
// without turning the window into a catalog browser.
func modelPreview(models []magpie.Model) string {
	limit := len(models)
	if limit > 6 {
		limit = 6
	}
	names := make([]string, 0, limit)
	for _, model := range models[:limit] {
		names = append(names, model.ID)
	}
	out := strings.Join(names, "、")
	if len(models) > limit {
		out += " 等"
	}
	return out
}

// aiSaveAndTest stores the address and asks the gateway for its catalog.
func (a *App) aiSaveAndTest() {
	a.mu.Lock()
	session := a.ai.session
	addr := strings.TrimSpace(a.ai.addr)
	tiers := a.tierMapping()
	model := strings.TrimSpace(a.ai.model)
	a.ai.busy, a.ai.errText, a.ai.notice = "正在连接…", "", ""
	a.mu.Unlock()
	a.aiWin.Update(func() {})

	go func() {
		if session == nil {
			opened, err := magpie.OpenSession(a.aiSettingsPath())
			if err != nil {
				a.aiFinish("", err)
				return
			}
			session = opened
			a.mu.Lock()
			a.ai.session = opened
			a.mu.Unlock()
		}
		settings := session.Settings()
		settings.BaseURL = addr
		settings.Tiers = tiers
		settings.Model = model
		if err := session.Update(settings); err != nil {
			a.aiFinish("", err)
			return
		}
		ctx, cancel := a.aiContext(20 * time.Second)
		defer cancel()
		message, err := session.Test(ctx)
		if err != nil {
			a.aiFinish("", err)
			return
		}
		a.mu.Lock()
		a.ai.busy, a.ai.notice, a.ai.errText = "", message, ""
		a.mu.Unlock()
		a.aiWin.Update(func() {})
		a.aiLoadModels()
	}()
}

// aiLoadModels fills the catalog note under the model field.
func (a *App) aiLoadModels() {
	a.mu.Lock()
	session := a.ai.session
	a.ai.busy = "正在读取模型目录…"
	a.mu.Unlock()
	a.aiWin.Update(func() {})

	go func() {
		if session == nil {
			a.aiFinish("", nil)
			return
		}
		ctx, cancel := a.aiContext(20 * time.Second)
		defer cancel()
		models, err := session.Models(ctx)
		if err != nil {
			a.aiFinish("", err)
			return
		}
		a.mu.Lock()
		a.ai.models = models
		a.ai.busy, a.ai.errText = "", ""
		a.mu.Unlock()
		a.aiWin.Update(func() {})
	}()
}

// aiFinish reports one failure in one place.
func (a *App) aiFinish(note string, err error) {
	a.mu.Lock()
	a.ai.busy = ""
	if err != nil {
		a.ai.errText = explain(err)
	} else {
		a.ai.errText = ""
	}
	a.mu.Unlock()
	a.aiWin.Update(func() {})
}

// aiSend streams one answer into the window.
func (a *App) aiSend() {
	a.mu.Lock()
	session := a.ai.session
	prompt := strings.TrimSpace(a.ai.prompt)
	if session == nil || prompt == "" || a.ai.running {
		a.mu.Unlock()
		return
	}
	task := magpie.Tasks[clampTier(a.ai.tier)]
	settings := a.applyAIFields(session.Settings())
	if err := session.Update(settings); err != nil {
		a.mu.Unlock()
		a.aiFinish("", err)
		return
	}
	ctx, cancel := aiContextLocked(a.ai.lifetime, 0)
	a.ai.running, a.ai.cancel = true, cancel
	a.ai.answer, a.ai.errText = "", ""
	a.ai.prompt = ""
	a.mu.Unlock()
	a.aiWin.Update(func() {})

	go func() {
		defer func() {
			cancel()
			a.mu.Lock()
			a.ai.running, a.ai.cancel = false, nil
			a.mu.Unlock()
			a.aiWin.Update(func() {})
		}()
		events, err := session.Ask(ctx, task, []magpie.Message{{Role: "user", Content: prompt}})
		if err != nil {
			a.mu.Lock()
			a.ai.errText = explain(err)
			a.mu.Unlock()
			return
		}
		for event := range events {
			switch event.Kind {
			case magpie.EventText:
				a.mu.Lock()
				a.ai.answer += event.Text
				now := time.Now()
				due := now.Sub(a.ai.lastFlush) > 80*time.Millisecond
				if due {
					a.ai.lastFlush = now
				}
				a.mu.Unlock()
				if due {
					a.aiWin.Update(func() {})
				}
			case magpie.EventError:
				a.mu.Lock()
				a.ai.errText = event.Text
				a.mu.Unlock()
			}
		}
	}()
}

// aiStop cancels the running request; the gateway sees the connection close.
func (a *App) aiStop() {
	a.mu.Lock()
	cancel := a.ai.cancel
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}
