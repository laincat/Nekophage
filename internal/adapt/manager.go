package adapt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/laincat/nekophage/internal/core"
)

// Manager coordinates VeneraCat Core's static comic adaptation and rendering
// entry points with profile-scoped persistence, per-project serialization,
// budget pause/resume checkpoints, and secret redaction.
type Manager struct {
	core    *core.Core
	store   *Store
	dataDir string

	locksMu sync.Mutex
	locks   map[string]chan struct{}
}

// NewManager creates a Manager backed by c and store.
func NewManager(c *core.Core, store *Store) (*Manager, error) {
	if c == nil {
		return nil, errors.New("adapt: core is required")
	}
	if store == nil {
		return nil, errors.New("adapt: store is required")
	}
	return &Manager{
		core:    c,
		store:   store,
		dataDir: c.DataDir(),
		locks:   map[string]chan struct{}{},
	}, nil
}

// Store returns the underlying profile-scoped persistence store.
func (m *Manager) Store() *Store { return m.store }

// LoadWorkspace loads or initializes the adaptation workspace for one novel.
func (m *Manager) LoadWorkspace(novelSourceKey, novelID, title string) (*Workspace, error) {
	ws, err := m.store.LoadOrCreateProject(novelSourceKey, novelID, title)
	if err != nil {
		return nil, err
	}
	return ws.Clone(), nil
}

// SaveBudget updates and persists a novel project's budget limits.
func (m *Manager) SaveBudget(ctx context.Context, novelSourceKey, novelID, title string, budget core.AdaptBudget) (*Workspace, error) {
	projectID := ProjectID(novelSourceKey, novelID)
	release, err := m.acquireProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	defer release()

	ws, err := m.store.LoadOrCreateProject(novelSourceKey, novelID, title)
	if err != nil {
		return nil, err
	}
	ws.Project.Budget = budget
	if err := m.store.SaveProject(ws); err != nil {
		return nil, err
	}
	return ws.Clone(), nil
}

// SaveStyleRef updates and persists a novel project's local style reference metadata.
func (m *Manager) SaveStyleRef(ctx context.Context, novelSourceKey, novelID, title string, styleRef *core.AdaptStyleRef) (*Workspace, error) {
	projectID := ProjectID(novelSourceKey, novelID)
	release, err := m.acquireProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	defer release()

	ws, err := m.store.LoadOrCreateProject(novelSourceKey, novelID, title)
	if err != nil {
		return nil, err
	}
	if styleRef != nil {
		cp := *styleRef
		if len(styleRef.ReferencePageIDs) > 0 {
			cp.ReferencePageIDs = append([]string(nil), styleRef.ReferencePageIDs...)
		}
		ws.Project.StyleRef = &cp
	} else {
		ws.Project.StyleRef = nil
	}
	if err := m.store.SaveProject(ws); err != nil {
		return nil, err
	}
	return ws.Clone(), nil
}

// ReadPanelImage reads a persisted panel image through the store's path guard.
func (m *Manager) ReadPanelImage(projectID, refID string) ([]byte, string, error) {
	return m.store.ReadPanelImage(projectID, refID)
}

// AdaptChapter runs or resumes chapter storyboard adaptation through Core,
// persisting storyboards, cross-chapter character sheets, usage, and any
// multi-chunk checkpoint on partial failure (budget limit, cancellation, or
// transient error). Credentials are never persisted.
func (m *Manager) AdaptChapter(ctx context.Context, in AdaptChapterInput) (*AdaptChapterOutput, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(in.NovelSourceKey) == "" || strings.TrimSpace(in.NovelID) == "" {
		return nil, errors.New("adapt: novel source key and novel ID are required")
	}
	if strings.TrimSpace(in.Chapter.ID) == "" {
		return nil, errors.New("adapt: chapter ID is required")
	}

	projectID := ProjectID(in.NovelSourceKey, in.NovelID)
	release, err := m.acquireProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	defer release()

	ws, err := m.store.LoadOrCreateProject(in.NovelSourceKey, in.NovelID, in.NovelTitle)
	if err != nil {
		return nil, err
	}
	if in.NovelTitle != "" {
		ws.Project.Title = in.NovelTitle
	}
	if in.Budget != nil {
		ws.Project.Budget = *in.Budget
	}
	if in.StyleRef != nil {
		sr := *in.StyleRef
		if len(in.StyleRef.ReferencePageIDs) > 0 {
			sr.ReferencePageIDs = append([]string(nil), in.StyleRef.ReferencePageIDs...)
		}
		ws.Project.StyleRef = &sr
	}

	var textCfg core.AdaptTextConfig
	var secrets []string
	if in.Text.Generator == nil {
		textCfg, secrets, err = ResolveTextConfig(in.Text, m.dataDir)
		if err != nil {
			return nil, err
		}
	} else if strings.TrimSpace(in.Text.Config.APIKey) != "" {
		secrets = []string{strings.TrimSpace(in.Text.Config.APIKey)}
	}

	var existingCP *core.AdaptCheckpoint
	if !in.Restart && ws.Checkpoints != nil {
		existingCP = cloneCheckpoint(ws.Checkpoints[in.Chapter.ID])
	}
	if in.Restart && ws.Checkpoints != nil {
		delete(ws.Checkpoints, in.Chapter.ID)
		_ = m.store.DeleteCheckpoint(ws.Project.ID, in.Chapter.ID)
	}

	emitProgress(in.OnProgress, ProgressEvent{
		ProjectID: ws.Project.ID,
		ChapterID: in.Chapter.ID,
		Phase:     PhaseAdapting,
	})

	res, runErr := m.core.AdaptChapter(ctx, core.AdaptChapterRequest{
		Project: ws.Project,
		Chapter: in.Chapter,
		Content: in.Content,
		Text:    textCfg,
		Options: core.AdaptOptions{
			MaxChunkRunes: in.MaxChunkRunes,
			Checkpoint:    existingCP,
		},
		Generator: in.Text.Generator,
	})

	scrubProjectSecrets(ws.Project, secrets...)
	if ws.Checkpoints == nil {
		ws.Checkpoints = map[string]*core.AdaptCheckpoint{}
	}

	if runErr != nil {
		cleanErr := redactError(runErr, secrets...)
		if res != nil && res.Checkpoint != nil {
			scrubStoryboardSecrets(res.Checkpoint.Storyboard, secrets...)
			if hasCheckpointProgress(res.Checkpoint) {
				ws.Checkpoints[in.Chapter.ID] = cloneCheckpoint(res.Checkpoint)
			} else if existingCP != nil && hasCheckpointProgress(existingCP) {
				ws.Checkpoints[in.Chapter.ID] = existingCP
			}
		}
		if saveErr := m.store.SaveProject(ws); saveErr != nil {
			return nil, fmt.Errorf("adapt: persist partial progress after %v: %w", cleanErr, saveErr)
		}

		phase := PhaseFailed
		budgetHit := IsBudgetExceeded(cleanErr)
		canceled := IsCanceled(cleanErr)
		switch {
		case budgetHit:
			phase = PhasePausedBudget
		case canceled && ws.Checkpoints[in.Chapter.ID] != nil:
			phase = PhaseCheckpointReady
		case ws.Checkpoints[in.Chapter.ID] != nil:
			phase = PhaseFailed
		}
		msg := ExplainError(cleanErr, secrets...)
		emitProgress(in.OnProgress, ProgressEvent{
			ProjectID: ws.Project.ID,
			ChapterID: in.Chapter.ID,
			Phase:     phase,
			Err:       cleanErr,
		})
		return &AdaptChapterOutput{
			Workspace:      ws.Clone(),
			Result:         res,
			Phase:          phase,
			BudgetExceeded: budgetHit,
			Canceled:       canceled,
			UserMessage:    msg,
		}, cleanErr
	}

	delete(ws.Checkpoints, in.Chapter.ID)
	if err := m.store.SaveProject(ws); err != nil {
		return nil, fmt.Errorf("adapt: save adapted chapter: %w", err)
	}
	snap := ws.ChapterSnapshot(in.Chapter.ID)
	emitProgress(in.OnProgress, ProgressEvent{
		ProjectID: ws.Project.ID,
		ChapterID: in.Chapter.ID,
		Phase:     snap.Phase,
	})
	return &AdaptChapterOutput{
		Workspace:   ws.Clone(),
		Result:      res,
		Phase:       snap.Phase,
		UserMessage: "分镜生成完成。",
	}, nil
}

// RenderPanel renders or force-rerenders one panel via Core, saving the image
// bytes to the profile store and persisting updated status/ref/usage.
// ErrContentBlocked is never retried and is surfaced explicitly to the caller.
func (m *Manager) RenderPanel(ctx context.Context, in RenderPanelInput) (*RenderPanelOutput, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(in.NovelSourceKey) == "" || strings.TrimSpace(in.NovelID) == "" {
		return nil, errors.New("adapt: novel source key and novel ID are required")
	}
	if strings.TrimSpace(in.ChapterID) == "" || strings.TrimSpace(in.PanelID) == "" {
		return nil, errors.New("adapt: chapter ID and panel ID are required")
	}

	projectID := ProjectID(in.NovelSourceKey, in.NovelID)
	release, err := m.acquireProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	defer release()

	ws, err := m.store.LoadOrCreateProject(in.NovelSourceKey, in.NovelID, in.NovelTitle)
	if err != nil {
		return nil, err
	}
	if in.Budget != nil {
		ws.Project.Budget = *in.Budget
	}
	if ws.Project.Storyboards[in.ChapterID] == nil {
		return nil, fmt.Errorf("adapt: storyboard not found for chapter %q", in.ChapterID)
	}

	var imgCfg core.AdaptImageConfig
	var secrets []string
	if in.Image.Generator == nil {
		imgCfg, secrets, err = ResolveImageConfig(in.Image)
		if err != nil {
			return nil, err
		}
	} else if strings.TrimSpace(in.Image.Config.APIKey) != "" {
		secrets = []string{strings.TrimSpace(in.Image.Config.APIKey)}
	}

	styleRefs, charRefs, err := LoadRenderReferences(in.Image)
	if err != nil {
		return nil, err
	}

	sink := core.AdaptImageSinkFunc(func(_ context.Context, meta core.AdaptPanelImageMeta, mimeType string, data []byte) (string, error) {
		return m.store.SavePanelImage(ws.Project.ID, meta, mimeType, data)
	})

	opts := core.AdaptRenderOptions{
		AspectRatio:         strings.TrimSpace(in.Image.AspectRatio),
		Size:                firstNonEmpty(in.Image.Size, imgCfg.Size),
		MaxRetries:          in.Image.MaxRetries,
		ForceRerender:       in.ForceRerender,
		StyleReferences:     styleRefs,
		CharacterReferences: charRefs,
		OnStatusChange: func(sceneID, panelID string, status core.AdaptImageStatus, stErr error) {
			phase := PhaseRendering
			if status == core.AdaptImageFailed && IsContentBlocked(stErr) {
				phase = PhaseBlocked
			}
			emitProgress(in.OnProgress, ProgressEvent{
				ProjectID: ws.Project.ID,
				ChapterID: in.ChapterID,
				SceneID:   sceneID,
				PanelID:   panelID,
				Phase:     phase,
				Status:    status,
				Err:       redactError(stErr, secrets...),
			})
		},
	}

	res, runErr := m.core.RenderPanel(ctx, core.RenderPanelRequest{
		Project:   ws.Project,
		ChapterID: in.ChapterID,
		SceneID:   in.SceneID,
		PanelID:   in.PanelID,
		Image:     imgCfg,
		Options:   opts,
		Sink:      sink,
		Generator: in.Image.Generator,
	})

	scrubProjectSecrets(ws.Project, secrets...)
	if saveErr := m.store.SaveProject(ws); saveErr != nil {
		if runErr != nil {
			return nil, fmt.Errorf("adapt: persist panel state after %v: %w", redactError(runErr, secrets...), saveErr)
		}
		return nil, fmt.Errorf("adapt: persist panel state: %w", saveErr)
	}

	if runErr != nil {
		cleanErr := redactError(runErr, secrets...)
		blocked := IsContentBlocked(cleanErr)
		budgetHit := IsBudgetExceeded(cleanErr)
		canceled := IsCanceled(cleanErr)
		phase := PhaseFailed
		switch {
		case blocked:
			phase = PhaseBlocked
		case budgetHit:
			phase = PhasePausedBudget
		case canceled:
			phase = ws.ChapterSnapshot(in.ChapterID).Phase
		}
		return &RenderPanelOutput{
			Workspace:      ws.Clone(),
			Result:         res,
			Phase:          phase,
			ContentBlocked: blocked,
			BudgetExceeded: budgetHit,
			Canceled:       canceled,
			UserMessage:    ExplainError(cleanErr, secrets...),
		}, cleanErr
	}

	snap := ws.ChapterSnapshot(in.ChapterID)
	msg := "单格渲染完成。"
	if res != nil && res.Skipped {
		msg = "该格已完成渲染，已跳过。"
	}
	return &RenderPanelOutput{
		Workspace:   ws.Clone(),
		Result:      res,
		Phase:       snap.Phase,
		UserMessage: msg,
	}, nil
}

// RenderStoryboard renders all pending panels of a chapter storyboard in order,
// persisting progress after each completed/failed panel so budget pauses,
// cancellations, or restarts resume cleanly without re-rendering done panels
// unless ForceRerender is set.
func (m *Manager) RenderStoryboard(ctx context.Context, in RenderStoryboardInput) (*RenderStoryboardOutput, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(in.NovelSourceKey) == "" || strings.TrimSpace(in.NovelID) == "" {
		return nil, errors.New("adapt: novel source key and novel ID are required")
	}
	if strings.TrimSpace(in.ChapterID) == "" {
		return nil, errors.New("adapt: chapter ID is required")
	}

	projectID := ProjectID(in.NovelSourceKey, in.NovelID)
	release, err := m.acquireProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	defer release()

	ws, err := m.store.LoadOrCreateProject(in.NovelSourceKey, in.NovelID, in.NovelTitle)
	if err != nil {
		return nil, err
	}
	if in.Budget != nil {
		ws.Project.Budget = *in.Budget
	}
	if ws.Project.Storyboards[in.ChapterID] == nil {
		return nil, fmt.Errorf("adapt: storyboard not found for chapter %q", in.ChapterID)
	}

	var imgCfg core.AdaptImageConfig
	var secrets []string
	if in.Image.Generator == nil {
		imgCfg, secrets, err = ResolveImageConfig(in.Image)
		if err != nil {
			return nil, err
		}
	} else if strings.TrimSpace(in.Image.Config.APIKey) != "" {
		secrets = []string{strings.TrimSpace(in.Image.Config.APIKey)}
	}

	styleRefs, charRefs, err := LoadRenderReferences(in.Image)
	if err != nil {
		return nil, err
	}

	sink := core.AdaptImageSinkFunc(func(_ context.Context, meta core.AdaptPanelImageMeta, mimeType string, data []byte) (string, error) {
		return m.store.SavePanelImage(ws.Project.ID, meta, mimeType, data)
	})

	opts := core.AdaptRenderOptions{
		AspectRatio:           strings.TrimSpace(in.Image.AspectRatio),
		Size:                  firstNonEmpty(in.Image.Size, imgCfg.Size),
		MaxRetries:            in.Image.MaxRetries,
		ForceRerender:         in.ForceRerender,
		ContinueOnError:       in.Image.ContinueOnError,
		FeedbackCharacterRefs: in.Image.FeedbackCharacterRefs,
		StyleReferences:       styleRefs,
		CharacterReferences:   charRefs,
		OnStatusChange: func(sceneID, panelID string, status core.AdaptImageStatus, stErr error) {
			if status == core.AdaptImageDone || status == core.AdaptImageFailed {
				scrubProjectSecrets(ws.Project, secrets...)
				_ = m.store.SaveProject(ws)
			}
			phase := PhaseRendering
			if status == core.AdaptImageFailed && IsContentBlocked(stErr) {
				phase = PhaseBlocked
			}
			emitProgress(in.OnProgress, ProgressEvent{
				ProjectID: ws.Project.ID,
				ChapterID: in.ChapterID,
				SceneID:   sceneID,
				PanelID:   panelID,
				Phase:     phase,
				Status:    status,
				Err:       redactError(stErr, secrets...),
			})
		},
	}

	res, runErr := m.core.RenderStoryboard(ctx, core.RenderStoryboardRequest{
		Project:   ws.Project,
		ChapterID: in.ChapterID,
		Image:     imgCfg,
		Options:   opts,
		Sink:      sink,
		Generator: in.Image.Generator,
	})

	scrubProjectSecrets(ws.Project, secrets...)
	if saveErr := m.store.SaveProject(ws); saveErr != nil {
		if runErr != nil {
			return nil, fmt.Errorf("adapt: persist storyboard render state after %v: %w", redactError(runErr, secrets...), saveErr)
		}
		return nil, fmt.Errorf("adapt: persist storyboard render state: %w", saveErr)
	}

	blockedCount := 0
	if res != nil {
		blockedCount = res.Blocked
		for i := range res.Failures {
			res.Failures[i].Err = redactError(res.Failures[i].Err, secrets...)
		}
	}

	if runErr != nil {
		cleanErr := redactError(runErr, secrets...)
		blocked := blockedCount > 0 || IsContentBlocked(cleanErr)
		if blocked && blockedCount == 0 {
			blockedCount = 1
		}
		budgetHit := IsBudgetExceeded(cleanErr)
		canceled := IsCanceled(cleanErr)
		phase := PhaseFailed
		switch {
		case budgetHit:
			phase = PhasePausedBudget
		case blocked:
			phase = PhaseBlocked
		case canceled:
			phase = ws.ChapterSnapshot(in.ChapterID).Phase
		}
		return &RenderStoryboardOutput{
			Workspace:      ws.Clone(),
			Result:         res,
			Phase:          phase,
			ContentBlocked: blocked,
			BlockedCount:   blockedCount,
			BudgetExceeded: budgetHit,
			Canceled:       canceled,
			UserMessage:    ExplainError(cleanErr, secrets...),
		}, cleanErr
	}

	snap := ws.ChapterSnapshot(in.ChapterID)
	blocked := blockedCount > 0 || snap.BlockedPanels > 0
	msg := "整章分镜渲染完成。"
	if blocked {
		msg = fmt.Sprintf("整章渲染结束，但有 %d 格被内容安全策略拦截（未自动重试），请调整提示词后单独重画。", max(blockedCount, snap.BlockedPanels))
	} else if res != nil && res.Failed > 0 {
		msg = fmt.Sprintf("整章渲染结束：成功 %d 格，跳过 %d 格，失败 %d 格。", res.Rendered, res.Skipped, res.Failed)
	}
	return &RenderStoryboardOutput{
		Workspace:      ws.Clone(),
		Result:         res,
		Phase:          snap.Phase,
		ContentBlocked: blocked,
		BlockedCount:   max(blockedCount, snap.BlockedPanels),
		UserMessage:    msg,
	}, nil
}

func (m *Manager) acquireProject(ctx context.Context, projectID string) (func(), error) {
	m.locksMu.Lock()
	sem := m.locks[projectID]
	if sem == nil {
		sem = make(chan struct{}, 1)
		m.locks[projectID] = sem
	}
	m.locksMu.Unlock()

	select {
	case sem <- struct{}{}:
		return func() { <-sem }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func hasCheckpointProgress(cp *core.AdaptCheckpoint) bool {
	if cp == nil {
		return false
	}
	if cp.CompletedChunks > 0 {
		return true
	}
	return cp.Storyboard != nil && len(cp.Storyboard.Scenes) > 0
}

func emitProgress(fn func(ProgressEvent), ev ProgressEvent) {
	if fn != nil {
		fn(ev)
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
