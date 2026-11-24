// Package adapt manages desktop novel-to-static-comic adaptation state,
// profile-scoped persistence, backend resolution, and rendering workflows
// backed by VeneraCat Core.
//
// Secrets (API keys and bearer tokens) are never persisted to project,
// storyboard, checkpoint, or settings files and are redacted from stored panel
// errors and returned error messages.
package adapt

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/magpie"
)

// Phase describes the current lifecycle state of one chapter's static comic
// adaptation and rendering workflow.
type Phase string

const (
	PhaseIdle            Phase = "idle"
	PhaseAdapting        Phase = "adapting"
	PhasePausedBudget    Phase = "paused_budget"
	PhaseCheckpointReady Phase = "checkpoint_ready"
	PhaseStoryboardReady Phase = "storyboard_ready"
	PhaseRendering       Phase = "rendering"
	PhaseBlocked         Phase = "blocked"
	PhaseCompleted       Phase = "completed"
	PhaseFailed          Phase = "failed"
)

// Label returns a concise Chinese display label for the phase.
func (p Phase) Label() string {
	switch p {
	case PhaseIdle:
		return "待改编"
	case PhaseAdapting:
		return "正在生成分镜"
	case PhasePausedBudget:
		return "达到预算上限（可续跑）"
	case PhaseCheckpointReady:
		return "已保存断点（可续跑）"
	case PhaseStoryboardReady:
		return "分镜已就绪"
	case PhaseRendering:
		return "正在渲染画面"
	case PhaseBlocked:
		return "内容策略拦截（未重试）"
	case PhaseCompleted:
		return "全部格图已完成"
	case PhaseFailed:
		return "执行失败（进度已保留）"
	default:
		return string(p)
	}
}

// TextMode selects how chapter storyboard generation reaches a text model.
type TextMode string

const (
	// TextModeMagpie routes text adaptation through the local Magpie gateway.
	TextModeMagpie TextMode = "magpie"
	// TextModeAPI routes text adaptation directly to a standard AI API endpoint.
	TextModeAPI TextMode = "api"
)

// Settings holds non-secret adaptation preferences stored under the profile
// directory. Credentials (API keys / tokens) are intentionally excluded and
// never written to disk.
type Settings struct {
	TextMode              TextMode                `json:"textMode,omitempty"`
	MagpieTier            magpie.Task             `json:"magpieTier,omitempty"`
	MagpieModel           string                  `json:"magpieModel,omitempty"`
	TextProtocol          core.AIProtocol         `json:"textProtocol,omitempty"`
	TextBaseURL           string                  `json:"textBaseUrl,omitempty"`
	TextModel             string                  `json:"textModel,omitempty"`
	TextAnthropicVersion  string                  `json:"textAnthropicVersion,omitempty"`
	TextMaxTokens         int                     `json:"textMaxTokens,omitempty"`
	TextJSONMode          bool                    `json:"textJsonMode,omitempty"`
	TextReasoningEffort   string                  `json:"textReasoningEffort,omitempty"`
	ImageProvider         core.AdaptImageProvider `json:"imageProvider,omitempty"`
	ImageBaseURL          string                  `json:"imageBaseUrl,omitempty"`
	ImageModel            string                  `json:"imageModel,omitempty"`
	ImageSize             string                  `json:"imageSize,omitempty"`
	ImageAspectRatio      string                  `json:"imageAspectRatio,omitempty"`
	ImageQuality          string                  `json:"imageQuality,omitempty"`
	ImageOutputFormat     string                  `json:"imageOutputFormat,omitempty"`
	ImageResponseFormat   string                  `json:"imageResponseFormat,omitempty"`
	FeedbackCharacterRefs bool                    `json:"feedbackCharacterRefs,omitempty"`
	StyleRefPaths         []string                `json:"styleRefPaths,omitempty"`
	CharacterRefPaths     map[string][]string     `json:"characterRefPaths,omitempty"`
	StyleNotes            string                  `json:"styleNotes,omitempty"`
	Budget                core.AdaptBudget        `json:"budget,omitempty"`
}

// DefaultSettings returns the initial non-secret adaptation preferences.
func DefaultSettings() Settings {
	return Settings{
		TextMode:              TextModeMagpie,
		MagpieTier:            magpie.TaskComplex,
		TextProtocol:          core.AIProtocolChat,
		TextJSONMode:          true,
		ImageProvider:         core.AdaptImageGemini,
		ImageAspectRatio:      "3:4",
		FeedbackCharacterRefs: true,
	}
}

// TextBackend configures the runtime text model backend for one adaptation run.
// Credentials in Config.APIKey stay in memory only.
type TextBackend struct {
	Mode               TextMode
	MagpieSettings     *magpie.Settings
	MagpieSettingsPath string
	MagpieTier         magpie.Task
	Config             core.AdaptTextConfig
	Generator          core.AdaptTextGenerator
}

// ImageBackend configures the runtime image rendering backend for panel or
// storyboard rendering. Credentials in Config.APIKey stay in memory only.
type ImageBackend struct {
	Config                core.AdaptImageConfig
	AspectRatio           string
	Size                  string
	MaxRetries            int
	ContinueOnError       bool
	FeedbackCharacterRefs bool
	StyleRefPaths         []string
	StyleReferences       []core.AdaptImageRef
	CharacterRefPaths     map[string][]string
	CharacterReferences   map[string][]core.AdaptImageRef
	Generator             core.AdaptImageGenerator
}

// Workspace is the persisted adaptation workspace for one novel, combining the
// Core project (storyboards, cross-chapter character sheet, budget, usage) with
// active multi-chunk chapter checkpoints.
type Workspace struct {
	Project     *core.AdaptProject               `json:"project"`
	Checkpoints map[string]*core.AdaptCheckpoint `json:"checkpoints,omitempty"`
	UpdatedAt   time.Time                        `json:"updatedAt,omitempty"`
}

// ChapterSnapshot summarizes the persisted state machine status of one chapter.
type ChapterSnapshot struct {
	ChapterID       string
	Phase           Phase
	HasStoryboard   bool
	HasCheckpoint   bool
	TotalChunks     int
	CompletedChunks int
	TotalPanels     int
	DonePanels      int
	PendingPanels   int
	FailedPanels    int
	BlockedPanels   int
}

// ChapterSnapshot computes the derived state-machine snapshot for chapterID.
func (w *Workspace) ChapterSnapshot(chapterID string) ChapterSnapshot {
	snap := ChapterSnapshot{
		ChapterID: chapterID,
		Phase:     PhaseIdle,
	}
	if w == nil || chapterID == "" {
		return snap
	}
	if cp := w.Checkpoints[chapterID]; cp != nil {
		snap.HasCheckpoint = true
		snap.TotalChunks = cp.TotalChunks
		snap.CompletedChunks = cp.CompletedChunks
		snap.Phase = PhaseCheckpointReady
		if cp.Storyboard != nil {
			countPanels(cp.Storyboard, &snap)
		}
		if w.Project != nil && w.Project.Usage.CheckNextCall(w.Project.Budget) != nil {
			snap.Phase = PhasePausedBudget
		}
		return snap
	}
	if w.Project == nil || w.Project.Storyboards == nil {
		return snap
	}
	sb := w.Project.Storyboards[chapterID]
	if sb == nil {
		return snap
	}
	snap.HasStoryboard = true
	countPanels(sb, &snap)
	switch {
	case snap.TotalPanels > 0 && snap.DonePanels == snap.TotalPanels:
		snap.Phase = PhaseCompleted
	case snap.BlockedPanels > 0:
		snap.Phase = PhaseBlocked
	case snap.FailedPanels > 0:
		snap.Phase = PhaseFailed
	case snap.PendingPanels > 0 && w.Project.Usage.CheckNextImageCall(w.Project.Budget) != nil:
		snap.Phase = PhasePausedBudget
	default:
		snap.Phase = PhaseStoryboardReady
	}
	return snap
}

func countPanels(sb *core.AdaptStoryboard, snap *ChapterSnapshot) {
	if sb == nil {
		return
	}
	for _, sc := range sb.Scenes {
		for _, p := range sc.Panels {
			snap.TotalPanels++
			switch p.ImageStatus {
			case core.AdaptImageDone:
				snap.DonePanels++
			case core.AdaptImageFailed:
				snap.FailedPanels++
				if IsContentBlockedText(p.ImageError) {
					snap.BlockedPanels++
				}
			default:
				snap.PendingPanels++
			}
		}
	}
}

// Clone returns a deep copy of w safe to hand to another goroutine or UI frame.
func (w *Workspace) Clone() *Workspace {
	if w == nil {
		return nil
	}
	out := &Workspace{
		Project:   cloneProject(w.Project),
		UpdatedAt: w.UpdatedAt,
	}
	if len(w.Checkpoints) > 0 {
		out.Checkpoints = make(map[string]*core.AdaptCheckpoint, len(w.Checkpoints))
		for k, cp := range w.Checkpoints {
			out.Checkpoints[k] = cloneCheckpoint(cp)
		}
	}
	return out
}

// ProgressEvent reports a state transition during adaptation or rendering.
type ProgressEvent struct {
	ProjectID string
	ChapterID string
	SceneID   string
	PanelID   string
	Phase     Phase
	Status    core.AdaptImageStatus
	Err       error
}

// AdaptChapterInput configures one chapter storyboard adaptation or resume run.
type AdaptChapterInput struct {
	NovelSourceKey string
	NovelID        string
	NovelTitle     string
	Chapter        core.NovelChapter
	Content        core.NovelChapterContent
	Text           TextBackend
	Budget         *core.AdaptBudget
	StyleRef       *core.AdaptStyleRef
	MaxChunkRunes  int
	Restart        bool
	OnProgress     func(ProgressEvent)
}

// AdaptChapterOutput reports the outcome of AdaptChapter, including any
// persisted checkpoint and updated workspace snapshot.
type AdaptChapterOutput struct {
	Workspace      *Workspace
	Result         *core.AdaptResult
	Phase          Phase
	BudgetExceeded bool
	Canceled       bool
	UserMessage    string
}

// RenderPanelInput configures rendering or force-rerendering one panel.
type RenderPanelInput struct {
	NovelSourceKey string
	NovelID        string
	NovelTitle     string
	ChapterID      string
	SceneID        string
	PanelID        string
	Image          ImageBackend
	Budget         *core.AdaptBudget
	ForceRerender  bool
	OnProgress     func(ProgressEvent)
}

// RenderPanelOutput reports the outcome of RenderPanel.
type RenderPanelOutput struct {
	Workspace      *Workspace
	Result         *core.AdaptPanelRenderResult
	Phase          Phase
	ContentBlocked bool
	BudgetExceeded bool
	Canceled       bool
	UserMessage    string
}

// RenderStoryboardInput configures rendering all pending panels of a chapter.
type RenderStoryboardInput struct {
	NovelSourceKey string
	NovelID        string
	NovelTitle     string
	ChapterID      string
	Image          ImageBackend
	Budget         *core.AdaptBudget
	ForceRerender  bool
	OnProgress     func(ProgressEvent)
}

// RenderStoryboardOutput reports the outcome of RenderStoryboard.
type RenderStoryboardOutput struct {
	Workspace      *Workspace
	Result         *core.AdaptRenderResult
	Phase          Phase
	ContentBlocked bool
	BlockedCount   int
	BudgetExceeded bool
	Canceled       bool
	UserMessage    string
}

// IsContentBlocked reports whether err indicates an image policy block that
// must not be retried automatically.
func IsContentBlocked(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, core.ErrAdaptContentBlocked) || IsContentBlockedText(err.Error())
}

// IsContentBlockedText recognizes persisted Panel.ImageError text produced by
// Core's ErrContentBlocked sentinel.
func IsContentBlockedText(msg string) bool {
	lower := strings.ToLower(strings.TrimSpace(msg))
	if lower == "" {
		return false
	}
	return strings.Contains(lower, "blocked by content policy") || strings.Contains(lower, "content_policy_violation")
}

// IsBudgetExceeded reports whether err was caused by a configured budget limit.
func IsBudgetExceeded(err error) bool {
	return err != nil && errors.Is(err, core.ErrAdaptBudgetExceeded)
}

// IsCanceled reports whether err was caused by cancellation or deadline expiry.
func IsCanceled(err error) bool {
	return err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
}

// ExplainError maps adaptation and rendering errors to concise Chinese guidance
// without leaking credentials.
func ExplainError(err error, secrets ...string) string {
	if err == nil {
		return ""
	}
	switch {
	case IsContentBlocked(err):
		return "画面生成被内容安全策略拦截，未自动重试；请调整该格提示词、参考图或跳过该格。"
	case IsBudgetExceeded(err):
		return "已达到设定的预算上限，已完成的分镜断点与格图均已保留；提高预算后可继续续跑。"
	case errors.Is(err, context.Canceled):
		return "操作已取消，已完成的分镜断点与格图均已保留。"
	case errors.Is(err, context.DeadlineExceeded):
		return "请求超时，已完成的分镜断点与格图均已保留，可继续续跑。"
	case errors.Is(err, core.ErrAdaptEmptyChapter):
		return "当前章节没有可用于改编的正文内容。"
	case errors.Is(err, core.ErrAdaptInvalidStoryboard):
		return "模型返回的分镜 JSON 格式无效，已完成断点已保留，可重试继续。"
	case errors.Is(err, magpie.ErrNoModel):
		return err.Error()
	case errors.Is(err, magpie.ErrNotRunning):
		return err.Error()
	case errors.Is(err, magpie.ErrRejected):
		return err.Error()
	default:
		return RedactSecrets(err.Error(), secrets...)
	}
}

// RedactSecrets replaces raw, trimmed, and URL-escaped forms of every secret in
// text with [REDACTED].
func RedactSecrets(text string, secrets ...string) string {
	if text == "" || len(secrets) == 0 {
		return text
	}
	clean := text
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		clean = replaceSecretVariants(clean, secret)
		if trimmed := strings.TrimSpace(secret); trimmed != "" && trimmed != secret {
			clean = replaceSecretVariants(clean, trimmed)
		}
	}
	return clean
}

func replaceSecretVariants(text, secret string) string {
	if secret == "" {
		return text
	}
	out := strings.ReplaceAll(text, secret, "[REDACTED]")
	if escaped := url.QueryEscape(secret); escaped != "" && escaped != secret {
		out = strings.ReplaceAll(out, escaped, "[REDACTED]")
	}
	return out
}

func redactError(err error, secrets ...string) error {
	if err == nil || len(secrets) == 0 {
		return err
	}
	orig := err.Error()
	clean := RedactSecrets(orig, secrets...)
	if clean == orig {
		return err
	}
	for _, sentinel := range []error{
		core.ErrAdaptContentBlocked,
		core.ErrAdaptBudgetExceeded,
		core.ErrAdaptInvalidStoryboard,
		core.ErrAdaptEmptyChapter,
		context.Canceled,
		context.DeadlineExceeded,
	} {
		if errors.Is(err, sentinel) {
			return &wrappedRedactedError{sentinel: sentinel, msg: clean}
		}
	}
	return errors.New(clean)
}

type wrappedRedactedError struct {
	sentinel error
	msg      string
}

func (e *wrappedRedactedError) Error() string { return e.msg }
func (e *wrappedRedactedError) Unwrap() error { return e.sentinel }

func scrubProjectSecrets(proj *core.AdaptProject, secrets ...string) {
	if proj == nil || len(secrets) == 0 {
		return
	}
	for _, sb := range proj.Storyboards {
		scrubStoryboardSecrets(sb, secrets...)
	}
}

func scrubStoryboardSecrets(sb *core.AdaptStoryboard, secrets ...string) {
	if sb == nil || len(secrets) == 0 {
		return
	}
	for si := range sb.Scenes {
		for pi := range sb.Scenes[si].Panels {
			if sb.Scenes[si].Panels[pi].ImageError != "" {
				sb.Scenes[si].Panels[pi].ImageError = RedactSecrets(sb.Scenes[si].Panels[pi].ImageError, secrets...)
			}
		}
	}
}

func cloneProject(p *core.AdaptProject) *core.AdaptProject {
	if p == nil {
		return nil
	}
	out := &core.AdaptProject{
		ID:             p.ID,
		NovelSourceKey: p.NovelSourceKey,
		NovelID:        p.NovelID,
		Title:          p.Title,
		Characters:     cloneCharacterSheet(p.Characters),
		Budget:         p.Budget,
		Usage:          p.Usage,
	}
	if p.StyleRef != nil {
		sr := *p.StyleRef
		if len(p.StyleRef.ReferencePageIDs) > 0 {
			sr.ReferencePageIDs = append([]string(nil), p.StyleRef.ReferencePageIDs...)
		}
		out.StyleRef = &sr
	}
	if len(p.Storyboards) > 0 {
		out.Storyboards = make(map[string]*core.AdaptStoryboard, len(p.Storyboards))
		for k, sb := range p.Storyboards {
			out.Storyboards[k] = cloneStoryboard(sb)
		}
	}
	return out
}

func cloneCheckpoint(cp *core.AdaptCheckpoint) *core.AdaptCheckpoint {
	if cp == nil {
		return nil
	}
	out := *cp
	out.Storyboard = cloneStoryboard(cp.Storyboard)
	out.Characters = cloneCharacterSheet(cp.Characters)
	out.AddedCharacters = cloneCharacters(cp.AddedCharacters)
	return &out
}

func cloneStoryboard(sb *core.AdaptStoryboard) *core.AdaptStoryboard {
	if sb == nil {
		return nil
	}
	out := &core.AdaptStoryboard{
		ChapterID:    sb.ChapterID,
		ChapterTitle: sb.ChapterTitle,
		Scenes:       make([]core.AdaptScene, len(sb.Scenes)),
	}
	for si, sc := range sb.Scenes {
		cs := core.AdaptScene{
			ID:      sc.ID,
			Setting: sc.Setting,
			Mood:    sc.Mood,
			Panels:  make([]core.AdaptPanel, len(sc.Panels)),
		}
		for pi, p := range sc.Panels {
			cp := p
			if len(p.Characters) > 0 {
				cp.Characters = append([]string(nil), p.Characters...)
			}
			if len(p.Dialogues) > 0 {
				cp.Dialogues = append([]core.AdaptDialogue(nil), p.Dialogues...)
			}
			cs.Panels[pi] = cp
		}
		out.Scenes[si] = cs
	}
	return out
}

func cloneCharacterSheet(sheet core.AdaptCharacterSheet) core.AdaptCharacterSheet {
	return core.AdaptCharacterSheet{Characters: cloneCharacters(sheet.Characters)}
}

func cloneCharacters(in []core.AdaptCharacter) []core.AdaptCharacter {
	if len(in) == 0 {
		return nil
	}
	out := make([]core.AdaptCharacter, len(in))
	for i, c := range in {
		cc := c
		if len(c.Aliases) > 0 {
			cc.Aliases = append([]string(nil), c.Aliases...)
		}
		if len(c.Traits) > 0 {
			cc.Traits = append([]string(nil), c.Traits...)
		}
		out[i] = cc
	}
	return out
}
