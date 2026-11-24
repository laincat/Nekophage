package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/laincat/nekophage/internal/adapt"
	"github.com/laincat/nekophage/internal/core"
)

type novelAdaptTab int

const (
	adaptTabStoryboard novelAdaptTab = iota
	adaptTabCharacters
)

// novelAdaptState belongs to the UI thread inside novelPanel.
// Credentials (textAPIKey, imageAPIKey) remain strictly in memory and are never
// persisted to disk.
type novelAdaptState struct {
	gen          generation
	imgGen       generation
	activeCancel context.CancelFunc

	workspace      *adapt.Workspace
	settings       adapt.Settings
	settingsLoaded bool
	showConfig     bool
	tab            novelAdaptTab

	phase          adapt.Phase
	running        bool
	busyText       string
	notice         string
	errText        string
	contentBlocked bool
	budgetExceeded bool

	// In-memory-only credentials and form inputs.
	textAPIKey         string
	imageAPIKey        string
	maxCallsText       string
	maxImageCallsText  string
	maxTotalTokensText string
	styleRefPath       string

	// Local rendered panel image cache (decoded on UI thread, loaded off-thread).
	panelImages imageCache
	imageEpoch  int

	// Package-private test hooks (nil in production).
	textGen       core.AdaptTextGenerator
	imageGen      core.AdaptImageGenerator
	maxChunkRunes int
}

func (s *novelAdaptState) reset() {
	if s.activeCancel != nil {
		s.activeCancel()
		s.activeCancel = nil
	}
	s.gen.next()
	s.imgGen.next()
	s.running = false
	s.busyText = ""
	s.notice = ""
	s.errText = ""
	s.contentBlocked = false
	s.budgetExceeded = false
	s.workspace = nil
	s.phase = adapt.PhaseIdle
	s.panelImages = newImageCache()
}

func (s *novelAdaptState) ensureImageCache() {
	if s.panelImages.covers == nil {
		s.panelImages = newImageCache()
	}
}

func (p *novelPanel) ensureAdaptManager() (*adapt.Manager, error) {
	p.adaptMgrMu.Lock()
	defer p.adaptMgrMu.Unlock()
	if p.adaptMgr != nil {
		return p.adaptMgr, nil
	}
	if p.core == nil {
		return nil, fmt.Errorf("adapt: core 未初始化")
	}
	store, err := adapt.OpenStore(p.core.DataDir())
	if err != nil {
		return nil, err
	}
	mgr, err := adapt.NewManager(p.core, store)
	if err != nil {
		return nil, err
	}
	p.adaptMgr = mgr
	return mgr, nil
}

func (p *novelPanel) currentAdaptChapter() (core.NovelChapter, bool) {
	if p.detail == nil || p.chapter < 0 || p.chapter >= len(p.detail.Chapters) {
		return core.NovelChapter{}, false
	}
	return p.detail.Chapters[p.chapter], true
}

func (p *novelPanel) openAdapt() {
	chapter, ok := p.currentAdaptChapter()
	if !ok || strings.TrimSpace(p.chapterContent.HTML) == "" {
		return
	}
	p.screen = novelAdapt
	s := &p.adapt
	s.ensureImageCache()
	s.errText = ""
	s.notice = ""
	s.contentBlocked = false
	s.budgetExceeded = false
	s.tab = adaptTabStoryboard

	ticket := s.gen.next()
	ctx := s.gen.requestContext(ticket)
	s.running = true
	s.busyText = "正在加载改编进度…"

	book := p.book
	chapterID := chapter.ID
	go func() {
		mgr, err := p.ensureAdaptManager()
		if err != nil {
			p.post(func() {
				if !s.gen.current(ticket) {
					return
				}
				s.running, s.busyText, s.errText = false, "", err.Error()
			})
			return
		}
		settings, setErr := mgr.Store().LoadSettings()
		ws, wsErr := mgr.LoadWorkspace(book.Source, book.ID, book.Title)
		if ctx.Err() != nil {
			return
		}
		p.post(func() {
			if !s.gen.current(ticket) {
				return
			}
			s.running, s.busyText = false, ""
			if setErr != nil {
				s.errText = adapt.ExplainError(setErr)
				return
			}
			if wsErr != nil {
				s.errText = adapt.ExplainError(wsErr)
				return
			}
			s.applyLoadedSettings(settings, ws)
			p.applyWorkspace(ws, chapterID)
		})
	}()
}

func (p *novelPanel) closeAdapt() {
	p.adapt.reset()
	p.screen = novelReading
}

func (s *novelAdaptState) applyLoadedSettings(settings adapt.Settings, ws *adapt.Workspace) {
	if !s.settingsLoaded {
		s.settings = settings
		s.settingsLoaded = true
		if len(settings.StyleRefPaths) > 0 {
			s.styleRefPath = settings.StyleRefPaths[0]
		}
	}
	budget := s.settings.Budget
	if ws != nil && ws.Project != nil && ws.Project.Budget != (core.AdaptBudget{}) {
		budget = ws.Project.Budget
	}
	s.maxCallsText = formatLimit(budget.MaxCalls)
	s.maxImageCallsText = formatLimit(budget.MaxImageCalls)
	s.maxTotalTokensText = formatLimit(budget.MaxTotalTokens)
}

func (p *novelPanel) applyWorkspace(ws *adapt.Workspace, chapterID string) {
	s := &p.adapt
	s.workspace = ws
	if ws == nil {
		s.phase = adapt.PhaseIdle
		return
	}
	snap := ws.ChapterSnapshot(chapterID)
	s.phase = snap.Phase
	s.contentBlocked = snap.BlockedPanels > 0
	s.budgetExceeded = snap.Phase == adapt.PhasePausedBudget
	p.prefetchPanelImages(ws, chapterID)
}

func (p *novelPanel) parseBudgetInputs() (core.AdaptBudget, error) {
	s := &p.adapt
	maxCalls, err := parseLimitInput(s.maxCallsText, "最大文本次数")
	if err != nil {
		return core.AdaptBudget{}, err
	}
	maxImgCalls, err := parseLimitInput(s.maxImageCallsText, "最大生图次数")
	if err != nil {
		return core.AdaptBudget{}, err
	}
	maxTokens, err := parseLimitInput(s.maxTotalTokensText, "最大总 Token")
	if err != nil {
		return core.AdaptBudget{}, err
	}
	return core.AdaptBudget{
		MaxCalls:       maxCalls,
		MaxImageCalls:  maxImgCalls,
		MaxTotalTokens: maxTokens,
	}, nil
}

func (p *novelPanel) snapshotSettings(budget core.AdaptBudget) adapt.Settings {
	s := &p.adapt
	cfg := s.settings
	if cfg.TextMode == "" {
		cfg.TextMode = adapt.TextModeMagpie
	}
	if cfg.TextProtocol == "" {
		cfg.TextProtocol = core.AIProtocolChat
	}
	if cfg.ImageProvider == "" {
		cfg.ImageProvider = core.AdaptImageGemini
	}
	cfg.Budget = budget
	if path := strings.TrimSpace(s.styleRefPath); path != "" {
		cfg.StyleRefPaths = []string{path}
	} else {
		cfg.StyleRefPaths = nil
	}
	s.settings = cfg
	return cfg
}

func (p *novelPanel) startAdapt(restart bool) {
	chapter, ok := p.currentAdaptChapter()
	if !ok || p.adapt.running {
		return
	}
	budget, err := p.parseBudgetInputs()
	if err != nil {
		p.adapt.errText = err.Error()
		return
	}
	settings := p.snapshotSettings(budget)
	s := &p.adapt
	ticket := s.gen.next()
	ctx, cancel := context.WithCancel(s.gen.requestContext(ticket))
	s.activeCancel = cancel
	s.running = true
	s.phase = adapt.PhaseAdapting
	s.busyText = "正在生成章节分镜…"
	s.errText = ""
	s.notice = ""
	s.contentBlocked = false
	s.budgetExceeded = false

	book := p.book
	content := p.chapterContent
	dataDir := p.core.DataDir()
	textAPIKey := s.textAPIKey
	textGen := s.textGen
	maxChunkRunes := s.maxChunkRunes

	go func() {
		defer cancel()
		mgr, err := p.ensureAdaptManager()
		if err != nil {
			p.post(func() {
				if !s.gen.current(ticket) {
					return
				}
				s.running, s.activeCancel, s.busyText, s.errText = false, nil, "", err.Error()
			})
			return
		}
		_ = mgr.Store().SaveSettings(settings)
		backend := settings.BuildTextBackend(dataDir, textAPIKey)
		backend.Generator = textGen
		var styleRef *core.AdaptStyleRef
		if notes := strings.TrimSpace(settings.StyleNotes); notes != "" {
			styleRef = &core.AdaptStyleRef{StyleNotes: notes}
		}
		out, runErr := mgr.AdaptChapter(ctx, adapt.AdaptChapterInput{
			NovelSourceKey: book.Source,
			NovelID:        book.ID,
			NovelTitle:     book.Title,
			Chapter:        chapter,
			Content:        content,
			Text:           backend,
			Budget:         &budget,
			StyleRef:       styleRef,
			MaxChunkRunes:  maxChunkRunes,
			Restart:        restart,
		})
		p.post(func() {
			if !s.gen.current(ticket) {
				return
			}
			s.running, s.activeCancel, s.busyText = false, nil, ""
			if out != nil && out.Workspace != nil {
				p.applyWorkspace(out.Workspace, chapter.ID)
			}
			if runErr != nil {
				if out != nil {
					s.phase = out.Phase
					s.budgetExceeded = out.BudgetExceeded
					if out.BudgetExceeded || out.Canceled {
						s.notice = out.UserMessage
						s.errText = ""
					} else {
						s.errText = out.UserMessage
					}
				} else {
					s.phase = adapt.PhaseFailed
					s.errText = adapt.ExplainError(runErr, textAPIKey)
				}
				return
			}
			s.errText = ""
			if out != nil {
				s.phase = out.Phase
				s.notice = out.UserMessage
			}
		})
	}()
}

func (p *novelPanel) startRenderPanel(sceneID, panelID string, force bool) {
	chapter, ok := p.currentAdaptChapter()
	if !ok || p.adapt.running {
		return
	}
	budget, err := p.parseBudgetInputs()
	if err != nil {
		p.adapt.errText = err.Error()
		return
	}
	settings := p.snapshotSettings(budget)
	s := &p.adapt
	ticket := s.gen.next()
	ctx, cancel := context.WithCancel(s.gen.requestContext(ticket))
	s.activeCancel = cancel
	s.running = true
	s.phase = adapt.PhaseRendering
	s.busyText = fmt.Sprintf("正在渲染 %s…", panelID)
	s.errText = ""
	s.notice = ""
	s.contentBlocked = false
	s.budgetExceeded = false

	book := p.book
	imageAPIKey := s.imageAPIKey
	imageGen := s.imageGen

	go func() {
		defer cancel()
		mgr, err := p.ensureAdaptManager()
		if err != nil {
			p.post(func() {
				if !s.gen.current(ticket) {
					return
				}
				s.running, s.activeCancel, s.busyText, s.errText = false, nil, "", err.Error()
			})
			return
		}
		_ = mgr.Store().SaveSettings(settings)
		backend := settings.BuildImageBackend(imageAPIKey)
		backend.Generator = imageGen
		out, runErr := mgr.RenderPanel(ctx, adapt.RenderPanelInput{
			NovelSourceKey: book.Source,
			NovelID:        book.ID,
			NovelTitle:     book.Title,
			ChapterID:      chapter.ID,
			SceneID:        sceneID,
			PanelID:        panelID,
			Image:          backend,
			Budget:         &budget,
			ForceRerender:  force,
		})
		p.post(func() {
			if !s.gen.current(ticket) {
				return
			}
			s.running, s.activeCancel, s.busyText = false, nil, ""
			if force {
				s.imageEpoch++
			}
			if out != nil && out.Workspace != nil {
				p.applyWorkspace(out.Workspace, chapter.ID)
			}
			if runErr != nil {
				if out != nil {
					s.phase = out.Phase
					s.contentBlocked = out.ContentBlocked
					s.budgetExceeded = out.BudgetExceeded
					if out.BudgetExceeded || out.Canceled {
						s.notice = out.UserMessage
						s.errText = ""
					} else {
						s.errText = out.UserMessage
					}
				} else {
					s.phase = adapt.PhaseFailed
					s.contentBlocked = adapt.IsContentBlocked(runErr)
					s.errText = adapt.ExplainError(runErr, imageAPIKey)
				}
				return
			}
			s.errText = ""
			if out != nil {
				s.phase = out.Phase
				s.notice = out.UserMessage
			}
		})
	}()
}

func (p *novelPanel) startRenderStoryboard(force bool) {
	chapter, ok := p.currentAdaptChapter()
	if !ok || p.adapt.running {
		return
	}
	budget, err := p.parseBudgetInputs()
	if err != nil {
		p.adapt.errText = err.Error()
		return
	}
	settings := p.snapshotSettings(budget)
	s := &p.adapt
	ticket := s.gen.next()
	ctx, cancel := context.WithCancel(s.gen.requestContext(ticket))
	s.activeCancel = cancel
	s.running = true
	s.phase = adapt.PhaseRendering
	s.busyText = "正在渲染整章分镜…"
	s.errText = ""
	s.notice = ""
	s.contentBlocked = false
	s.budgetExceeded = false

	book := p.book
	imageAPIKey := s.imageAPIKey
	imageGen := s.imageGen

	go func() {
		defer cancel()
		mgr, err := p.ensureAdaptManager()
		if err != nil {
			p.post(func() {
				if !s.gen.current(ticket) {
					return
				}
				s.running, s.activeCancel, s.busyText, s.errText = false, nil, "", err.Error()
			})
			return
		}
		_ = mgr.Store().SaveSettings(settings)
		backend := settings.BuildImageBackend(imageAPIKey)
		backend.Generator = imageGen
		out, runErr := mgr.RenderStoryboard(ctx, adapt.RenderStoryboardInput{
			NovelSourceKey: book.Source,
			NovelID:        book.ID,
			NovelTitle:     book.Title,
			ChapterID:      chapter.ID,
			Image:          backend,
			Budget:         &budget,
			ForceRerender:  force,
			OnProgress: func(ev adapt.ProgressEvent) {
				if ev.PanelID == "" {
					return
				}
				p.post(func() {
					if !s.gen.current(ticket) || !s.running {
						return
					}
					s.busyText = fmt.Sprintf("正在渲染 %s（%s）…", ev.PanelID, panelStatusLabel(ev.Status, ""))
				})
			},
		})
		p.post(func() {
			if !s.gen.current(ticket) {
				return
			}
			s.running, s.activeCancel, s.busyText = false, nil, ""
			if force {
				s.imageEpoch++
			}
			if out != nil && out.Workspace != nil {
				p.applyWorkspace(out.Workspace, chapter.ID)
			}
			if runErr != nil {
				if out != nil {
					s.phase = out.Phase
					s.contentBlocked = out.ContentBlocked
					s.budgetExceeded = out.BudgetExceeded
					if out.BudgetExceeded || out.Canceled {
						s.notice = out.UserMessage
						s.errText = ""
					} else {
						s.errText = out.UserMessage
					}
				} else {
					s.phase = adapt.PhaseFailed
					s.contentBlocked = adapt.IsContentBlocked(runErr)
					s.errText = adapt.ExplainError(runErr, imageAPIKey)
				}
				return
			}
			s.errText = ""
			if out != nil {
				s.phase = out.Phase
				s.contentBlocked = out.ContentBlocked
				s.notice = out.UserMessage
			}
		})
	}()
}

func (p *novelPanel) cancelAdapt() {
	s := &p.adapt
	if s.activeCancel != nil {
		s.busyText = "正在取消并保留已完成进度…"
		s.activeCancel()
	}
}

func (p *novelPanel) prefetchPanelImages(ws *adapt.Workspace, chapterID string) {
	if ws == nil || ws.Project == nil || ws.Project.Storyboards == nil {
		return
	}
	sb := ws.Project.Storyboards[chapterID]
	if sb == nil {
		return
	}
	s := &p.adapt
	s.ensureImageCache()
	ticket := s.imgGen.next()
	ctx := s.imgGen.requestContext(ticket)
	projectID := ws.Project.ID
	epoch := s.imageEpoch

	for _, sc := range sb.Scenes {
		for _, panel := range sc.Panels {
			if panel.ImageRefID == "" {
				continue
			}
			cacheKey := panelImageCacheKey(epoch, panel.ImageRefID)
			if !s.panelImages.want(cacheKey, panel.ImageRefID) {
				continue
			}
			refID := panel.ImageRefID
			go func() {
				if ctx.Err() != nil {
					return
				}
				mgr, err := p.ensureAdaptManager()
				if err != nil {
					return
				}
				data, _, err := mgr.ReadPanelImage(projectID, refID)
				if err != nil {
					data = nil
				}
				if ctx.Err() != nil {
					return
				}
				p.post(func() {
					if !s.imgGen.current(ticket) {
						return
					}
					s.panelImages.finishCover(cacheKey, data)
				})
			}()
		}
	}
}

func (p *novelPanel) adaptView(c *ui.Context) {
	chapter, ok := p.currentAdaptChapter()
	if !ok {
		p.closeAdapt()
		return
	}
	s := &p.adapt
	s.ensureImageCache()
	t := c.Theme()

	var snap adapt.ChapterSnapshot
	var sb *core.AdaptStoryboard
	var cp *core.AdaptCheckpoint
	var chars []core.AdaptCharacter
	var usage core.AdaptUsage
	var budget core.AdaptBudget
	if s.workspace != nil {
		snap = s.workspace.ChapterSnapshot(chapter.ID)
		cp = s.workspace.Checkpoints[chapter.ID]
		if s.workspace.Project != nil {
			sb = s.workspace.Project.Storyboards[chapter.ID]
			chars = s.workspace.Project.Characters.Characters
			usage = s.workspace.Project.Usage
			budget = s.workspace.Project.Budget
		}
	}
	displayPhase := s.phase
	if displayPhase == "" {
		displayPhase = snap.Phase
	}

	// Header row
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		if ui.Button(c, "返回阅读").Clicked() {
			p.closeAdapt()
			return
		}
		ui.Text(c, fmt.Sprintf("静态漫画改编 · %s", chapterTitle(chapter, p.chapter))).FontSize(16).Bold().SingleLine().Grow(1)
		ui.Text(c, displayPhase.Label()).FontSize(12).TextColor(t.TextMuted)
	})
	if p.screen != novelAdapt {
		return
	}

	// Usage & budget bar
	ui.Text(c, formatUsageSummary(usage, budget)).FontSize(12).TextColor(t.TextMuted).Selectable()

	// Status / error / content block / budget banners
	if s.running {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c)
			ui.Text(c, s.busyText).FontSize(12).TextColor(t.TextMuted)
		})
	}
	if s.contentBlocked && s.errText != "" {
		ui.Text(c, "内容策略拦截："+s.errText).TextColor(t.Danger).Selectable()
	} else if s.errText != "" {
		ui.Text(c, s.errText).TextColor(t.Danger).Selectable()
	}
	if s.notice != "" {
		ui.Text(c, s.notice).TextColor(t.TextMuted).Selectable()
	}

	// Action bar
	ui.Row(c).Gap(8).Wrap().AlignItems(ui.Center).Children(func() {
		adaptLabel := "生成分镜"
		if cp != nil {
			adaptLabel = "继续生成分镜"
		}
		if ui.PrimaryButton(c, adaptLabel).Disabled(s.running).Clicked() {
			p.startAdapt(false)
		}
		if sb != nil || cp != nil {
			if ui.Button(c, "重新生成分镜").Disabled(s.running).Clicked() {
				p.startAdapt(true)
			}
		}
		if ui.Button(c, "渲染整章").Disabled(s.running || sb == nil).Clicked() {
			p.startRenderStoryboard(false)
		}
		if ui.Button(c, "取消").Disabled(!s.running).Clicked() {
			p.cancelAdapt()
		}
		cfgLabel := "展开设置"
		if s.showConfig {
			cfgLabel = "收起设置"
		}
		if ui.Button(c, cfgLabel).Clicked() {
			s.showConfig = !s.showConfig
		}
	})

	if s.showConfig {
		p.adaptConfigView(c)
	}

	// Tab selector
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		if s.tab == adaptTabStoryboard {
			ui.PrimaryButton(c, "分镜与画面")
		} else if ui.Button(c, "分镜与画面").Clicked() {
			s.tab = adaptTabStoryboard
		}
		charLabel := fmt.Sprintf("角色设定表 (%d)", len(chars))
		if s.tab == adaptTabCharacters {
			ui.PrimaryButton(c, charLabel)
		} else if ui.Button(c, charLabel).Clicked() {
			s.tab = adaptTabCharacters
		}
	})

	// Content body
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Gap(10).Children(func() {
			if s.tab == adaptTabCharacters {
				p.adaptCharactersView(c, chars)
				return
			}
			p.adaptStoryboardView(c, sb, cp)
		})
	})
}

func (p *novelPanel) adaptConfigView(c *ui.Context) {
	s := &p.adapt
	ui.Column(c).Gap(6).Padding(8).Children(func() {
		ui.Row(c).Gap(8).Wrap().AlignItems(ui.Center).Children(func() {
			textModes := []string{"本机 Magpie", "标准 API"}
			selectedMode := textModes[0]
			if s.settings.TextMode == adapt.TextModeAPI {
				selectedMode = textModes[1]
			}
			if ui.Select(c, &selectedMode, textModes).Width(140).Label("文本模型").Changed() {
				if selectedMode == textModes[1] {
					s.settings.TextMode = adapt.TextModeAPI
				} else {
					s.settings.TextMode = adapt.TextModeMagpie
				}
			}
			if s.settings.TextMode == adapt.TextModeAPI {
				protos := []string{"chat", "responses", "messages"}
				protoSel := string(s.settings.TextProtocol)
				if protoSel == "" {
					protoSel = "chat"
				}
				if ui.Select(c, &protoSel, protos).Width(115).Changed() {
					s.settings.TextProtocol = core.AIProtocol(protoSel)
				}
				ui.TextInput(c, &s.settings.TextBaseURL).Placeholder("文本 API BaseURL").Width(200)
				ui.TextInput(c, &s.settings.TextModel).Placeholder("文本模型").Width(140)
				ui.TextInput(c, &s.textAPIKey).Placeholder("文本 API Key（仅内存）").Password().Width(160)
			} else {
				ui.TextInput(c, &s.settings.MagpieModel).Placeholder("Magpie 模型/路由组（留空用档位）").Width(240)
			}
		})

		ui.Row(c).Gap(8).Wrap().AlignItems(ui.Center).Children(func() {
			providers := []string{"gemini", "openai"}
			provSel := string(s.settings.ImageProvider)
			if provSel == "" {
				provSel = "gemini"
			}
			if ui.Select(c, &provSel, providers).Width(110).Label("生图").Changed() {
				s.settings.ImageProvider = core.AdaptImageProvider(provSel)
			}
			ui.TextInput(c, &s.settings.ImageBaseURL).Placeholder("生图 BaseURL（可选）").Width(180)
			ui.TextInput(c, &s.settings.ImageModel).Placeholder("生图模型").Width(150)
			ui.TextInput(c, &s.imageAPIKey).Placeholder("生图 API Key（仅内存）").Password().Width(160)
		})

		ui.Row(c).Gap(8).Wrap().AlignItems(ui.Center).Children(func() {
			ui.TextInput(c, &s.styleRefPath).Placeholder("本地画风参考图路径（可选）").Width(220)
			ui.TextInput(c, &s.settings.StyleNotes).Placeholder("画风备注（可选）").Width(180)
			ui.TextInput(c, &s.maxCallsText).Placeholder("最大文本次数").Width(100)
			ui.TextInput(c, &s.maxImageCallsText).Placeholder("最大生图次数").Width(100)
			ui.TextInput(c, &s.maxTotalTokensText).Placeholder("最大总Token").Width(100)
		})
	})
}

func (p *novelPanel) adaptCharactersView(c *ui.Context, chars []core.AdaptCharacter) {
	if len(chars) == 0 {
		ui.Text(c, "尚无角色设定表。生成分镜后会自动汇总跨章角色外貌与服装。")
		return
	}
	for ci, ch := range chars {
		ui.Column(c).Key(fmt.Sprintf("char:%d:%s", ci, ch.ID)).Gap(2).Padding(6).Children(func() {
			header := ch.Name
			if ch.ID != "" {
				header += " (" + ch.ID + ")"
			}
			if len(ch.Aliases) > 0 {
				header += " · 别名：" + strings.Join(ch.Aliases, " / ")
			}
			ui.Text(c, header).Bold().Selectable()
			if ch.Appearance != "" {
				ui.Text(c, "外貌："+ch.Appearance).FontSize(12).Selectable()
			}
			if ch.Outfit != "" {
				ui.Text(c, "服装："+ch.Outfit).FontSize(12).Selectable()
			}
			if len(ch.Traits) > 0 {
				ui.Text(c, "特征："+strings.Join(ch.Traits, "、")).FontSize(12).Selectable()
			}
		})
	}
}

func (p *novelPanel) adaptStoryboardView(c *ui.Context, sb *core.AdaptStoryboard, cp *core.AdaptCheckpoint) {
	s := &p.adapt
	t := c.Theme()

	if cp != nil {
		ui.Textf(c, "多段断点进度：已完成 %d / %d 段，可点击“继续生成分镜”断点续跑。", cp.CompletedChunks, cp.TotalChunks).TextColor(t.TextMuted).Selectable()
		if sb == nil && cp.Storyboard != nil {
			sb = cp.Storyboard
		}
	}
	if sb == nil || len(sb.Scenes) == 0 {
		ui.Text(c, "尚未生成本章分镜。点击“生成分镜”开始将当前章节改编为静态漫画分镜。")
		return
	}

	canRender := cp == nil && !s.running
	for si, sc := range sb.Scenes {
		scene := sc
		ui.Column(c).Key(fmt.Sprintf("scene:%d:%s", si, scene.ID)).Gap(6).Padding(6).Children(func() {
			heading := fmt.Sprintf("场景 %s · %s", scene.ID, scene.Setting)
			if scene.Mood != "" {
				heading += "（" + scene.Mood + "）"
			}
			ui.Text(c, heading).FontSize(15).Bold().Selectable()

			for pi, pnl := range scene.Panels {
				panel := pnl
				ui.Column(c).Key(fmt.Sprintf("panel:%d:%d:%s", si, pi, panel.ID)).Gap(4).Padding(6).Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, fmt.Sprintf("%s · %s · %s", panel.ID, panel.Shot, panelStatusLabel(panel.ImageStatus, panel.ImageError))).Bold().Grow(1)
						btnLabel := "渲染 " + panel.ID
						force := false
						if panel.ImageStatus == core.AdaptImageDone {
							btnLabel = "重画 " + panel.ID
							force = true
						}
						if ui.Button(c, btnLabel).Key(fmt.Sprintf("btn:%d:%d:%s", si, pi, panel.ID)).Disabled(!canRender).Clicked() {
							p.startRenderPanel(scene.ID, panel.ID, force)
						}
					})
					if panel.Action != "" {
						ui.Text(c, "动作："+panel.Action).FontSize(13).Selectable()
					}
					if len(panel.Characters) > 0 {
						ui.Text(c, "角色："+strings.Join(panel.Characters, "、")).FontSize(12).TextColor(t.TextMuted).Selectable()
					}
					for di, dlg := range panel.Dialogues {
						line := dlg.Text
						if dlg.Speaker != "" {
							line = dlg.Speaker + "：" + line
						}
						ui.Text(c, line).Key(fmt.Sprintf("dlg:%d:%d:%d", si, pi, di)).FontSize(12).Selectable()
					}
					if panel.Prompt != "" {
						ui.Text(c, "画面提示词："+panel.Prompt).FontSize(12).TextColor(t.TextMuted).Selectable()
					}
					if panel.ImageError != "" {
						prefix := "渲染失败："
						if adapt.IsContentBlockedText(panel.ImageError) {
							prefix = "内容策略拦截（未自动重试）："
						}
						ui.Text(c, prefix+panel.ImageError).FontSize(12).TextColor(t.Danger).Selectable()
					}
					if panel.ImageRefID != "" {
						cacheKey := panelImageCacheKey(s.imageEpoch, panel.ImageRefID)
						if bmp := s.panelImages.decoded(cacheKey); bmp != nil {
							ui.Image(c, bmp).Fit(ui.Contain).Width(240).Height(320).Radius(6)
						} else {
							ui.Text(c, "本地格图："+panel.ImageRefID).FontSize(12).TextColor(t.TextMuted).Selectable()
						}
					}
				})
			}
		})
	}
}

func panelImageCacheKey(epoch int, refID string) string {
	return fmt.Sprintf("%d:%s", epoch, refID)
}

func panelStatusLabel(status core.AdaptImageStatus, errMsg string) string {
	switch status {
	case core.AdaptImageDone:
		return "已完成"
	case core.AdaptImageGenerating:
		return "渲染中"
	case core.AdaptImageFailed:
		if adapt.IsContentBlockedText(errMsg) {
			return "策略拦截"
		}
		return "失败"
	default:
		return "待渲染"
	}
}

func formatUsageSummary(u core.AdaptUsage, b core.AdaptBudget) string {
	parts := []string{
		fmt.Sprintf("文本调用 %d", u.Calls) + formatLimitSuffix(b.MaxCalls),
		fmt.Sprintf("生图调用 %d", u.ImageCalls) + formatLimitSuffix(b.MaxImageCalls),
		fmt.Sprintf("Token %d", u.TotalTokens) + formatLimitSuffix(b.MaxTotalTokens),
	}
	return "用量 / 预算：" + strings.Join(parts, " · ")
}

func formatLimitSuffix(limit int64) string {
	if limit <= 0 {
		return ""
	}
	return fmt.Sprintf("/%d", limit)
}

func formatLimit(limit int64) string {
	if limit <= 0 {
		return ""
	}
	return strconv.FormatInt(limit, 10)
}

func parseLimitInput(raw, field string) (int64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, nil
	}
	v, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || v < 0 {
		return 0, fmt.Errorf("%s必须是非负整数", field)
	}
	return v, nil
}
