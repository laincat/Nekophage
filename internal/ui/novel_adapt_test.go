package ui

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/laincat/nekophage/internal/adapt"
	"github.com/laincat/nekophage/internal/core"
)

var uiTestPNG = []byte{
	0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n',
	0x00, 0x00, 0x00, 0x0d, 'I', 'H', 'D', 'R',
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0a, 'I', 'D', 'A', 'T',
	0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00, 0x05,
	0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00,
	0x00, 0x00, 'I', 'E', 'N', 'D', 0xae, 0x42, 0x60, 0x82,
}

type uiStubTextGen struct {
	fn func(ctx context.Context, req core.AdaptGenerateRequest) (*core.AdaptGenerateResponse, error)
}

func (g uiStubTextGen) Generate(ctx context.Context, req core.AdaptGenerateRequest) (*core.AdaptGenerateResponse, error) {
	return g.fn(ctx, req)
}

type uiStubImageGen struct {
	fn func(ctx context.Context, req core.AdaptPanelImageRequest) (*core.AdaptPanelImageResult, error)
}

func (g uiStubImageGen) GenerateImage(ctx context.Context, req core.AdaptPanelImageRequest) (*core.AdaptPanelImageResult, error) {
	return g.fn(ctx, req)
}

func drainNovelUntil(t *testing.T, queued chan func(), cond func() bool) {
	t.Helper()
	deadline := time.After(8 * time.Second)
	for !cond() {
		select {
		case apply := <-queued:
			apply()
		case <-deadline:
			t.Fatal("timed out waiting for novel UI state transition")
		}
	}
}

func TestNovelAdaptUIWorkflowBudgetResumeRenderAndCharacters(t *testing.T) {
	longHTML := "<p>" + strings.Repeat("星海航行记录。", 14) + "</p><p>" + strings.Repeat("星际跃迁正文。", 14) + "</p>"
	p, queued := novelFixture(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/book/one":
			fmt.Fprint(w, longHTML)
		default:
			http.NotFound(w, r)
		}
	})
	p.setSources(p.core.Plugins())
	novelResult(t, queued)

	var textCalls atomic.Int32
	p.adapt.maxChunkRunes = 120
	p.adapt.textGen = uiStubTextGen{fn: func(_ context.Context, _ core.AdaptGenerateRequest) (*core.AdaptGenerateResponse, error) {
		n := textCalls.Add(1)
		if n == 1 {
			return &core.AdaptGenerateResponse{
				Text:  `{"scenes":[{"id":"s1","setting":"舰桥","mood":"宁静","panels":[{"id":"p1","shot":"远景","action":"林站在窗前","characters":["林"],"dialogues":[{"speaker":"林","text":"出发吧","kind":"speech"}],"prompt":"sci-fi bridge"}]}],"newCharacters":[{"id":"c1","name":"林","appearance":"黑短发","outfit":"深蓝舰服"}]}`,
				Usage: core.AdaptUsage{Calls: 1, TotalTokens: 100},
			}, nil
		}
		return &core.AdaptGenerateResponse{
			Text:  `{"scenes":[{"id":"s2","setting":"通道","mood":"紧张","panels":[{"id":"p2","shot":"特写","action":"警报亮起","characters":["林"],"prompt":"red alert corridor"}]}]}`,
			Usage: core.AdaptUsage{Calls: 1, TotalTokens: 120},
		}, nil
	}}

	var imgCalls atomic.Int32
	p.adapt.imageGen = uiStubImageGen{fn: func(_ context.Context, req core.AdaptPanelImageRequest) (*core.AdaptPanelImageResult, error) {
		imgCalls.Add(1)
		return &core.AdaptPanelImageResult{
			MIMEType: "image/png",
			Data:     uiTestPNG,
			Usage:    core.AdaptUsage{ImageCalls: 1},
		}, nil
	}}

	tt := ui.NewTester(p.view, 900, 760)
	click := func(name string) {
		t.Helper()
		tt.Frame()
		if err := tt.Click(name); err != nil {
			t.Fatalf("click %q failed: %v (texts=%q)", name, err, strings.Join(tt.Texts(), " | "))
		}
	}

	// Open book -> open chapter -> enter static comic adaptation from reader.
	click("热门 1")
	novelResult(t, queued)
	click("起航")
	novelResult(t, queued) // chapter loaded
	novelResult(t, queued) // reading progress saved

	click("改编为静态漫画")
	drainNovelUntil(t, queued, func() bool { return !p.adapt.running })
	if p.screen != novelAdapt {
		t.Fatalf("expected novelAdapt screen, got %d", p.screen)
	}

	// Cap text budget to 1 call so multi-chunk adaptation pauses with a checkpoint.
	p.adapt.maxCallsText = "1"
	click("生成分镜")
	drainNovelUntil(t, queued, func() bool { return !p.adapt.running })
	if p.adapt.phase != adapt.PhasePausedBudget || !p.adapt.budgetExceeded {
		t.Fatalf("expected budget pause, phase=%s budgetExceeded=%v notice=%q err=%q",
			p.adapt.phase, p.adapt.budgetExceeded, p.adapt.notice, p.adapt.errText)
	}
	tt.Frame()
	joined := strings.Join(tt.Texts(), "\n")
	if !strings.Contains(joined, "继续生成分镜") || !strings.Contains(joined, "预算上限") {
		t.Fatalf("expected resume button and budget notice, got:\n%s", joined)
	}

	// Raise budget and resume without re-billing chunk 1.
	p.adapt.maxCallsText = "3"
	click("继续生成分镜")
	drainNovelUntil(t, queued, func() bool { return !p.adapt.running })
	if textCalls.Load() != 2 || p.adapt.phase != adapt.PhaseStoryboardReady {
		t.Fatalf("expected 2 total text calls & PhaseStoryboardReady, calls=%d phase=%s", textCalls.Load(), p.adapt.phase)
	}

	// Inspect character sheet tab.
	click("角色设定表 (1)")
	tt.Frame()
	if !strings.Contains(strings.Join(tt.Texts(), "\n"), "黑短发") {
		t.Fatalf("character sheet did not display character details: %v", tt.Texts())
	}

	// Return to storyboard tab, render single panel p1, then verify local image is decoded.
	click("分镜与画面")
	click("渲染 p1")
	drainNovelUntil(t, queued, func() bool {
		if p.adapt.running {
			return false
		}
		sb := p.adapt.workspace.Project.Storyboards["one"]
		if sb == nil {
			return false
		}
		ref := sb.Scenes[0].Panels[0].ImageRefID
		if ref == "" {
			return false
		}
		key := panelImageCacheKey(p.adapt.imageEpoch, ref)
		_, ok := p.adapt.panelImages.covers[key]
		return ok
	})
	tt.Frame()
	sb := p.adapt.workspace.Project.Storyboards["one"]
	p1Key := panelImageCacheKey(p.adapt.imageEpoch, sb.Scenes[0].Panels[0].ImageRefID)
	if p.adapt.panelImages.decoded(p1Key) == nil {
		t.Fatal("expected decoded bitmap for rendered local panel p1")
	}

	// Render entire storyboard: skips p1 and renders p2.
	click("渲染整章")
	drainNovelUntil(t, queued, func() bool { return !p.adapt.running && p.adapt.phase == adapt.PhaseCompleted })
	if imgCalls.Load() != 2 {
		t.Fatalf("expected 2 total image calls (p1 skipped on storyboard render), got %d", imgCalls.Load())
	}

	// Force-rerender p1 via "重画 p1".
	prevEpoch := p.adapt.imageEpoch
	click("重画 p1")
	drainNovelUntil(t, queued, func() bool { return !p.adapt.running })
	if imgCalls.Load() != 3 || p.adapt.imageEpoch != prevEpoch+1 {
		t.Fatalf("expected ForceRerender 3rd call and epoch bump, calls=%d epoch=%d", imgCalls.Load(), p.adapt.imageEpoch)
	}

	// Return to novel reader: reading state remains intact.
	click("返回阅读")
	if p.screen != novelReading || len(p.pages) == 0 {
		t.Fatalf("expected return to novelReading with pages intact, screen=%d pages=%d", p.screen, len(p.pages))
	}
}

func TestNovelAdaptUIContentBlockedAndCancel(t *testing.T) {
	longHTML := "<p>" + strings.Repeat("待取消段落一。", 14) + "</p><p>" + strings.Repeat("待取消段落二。", 14) + "</p>"
	p, queued := novelFixture(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, longHTML)
	})
	p.setSources(p.core.Plugins())
	novelResult(t, queued)

	p.book.Source, p.book.ID, p.book.Title = p.key, "book", "星海故事"
	p.openBook(p.book)
	novelResult(t, queued)
	p.openChapter(0)
	novelResult(t, queued)
	novelResult(t, queued)

	// Test cancellation mid-adaptation preserves chunk 1 checkpoint.
	chunk2Started := make(chan struct{}, 1)
	var textCalls atomic.Int32
	p.adapt.maxChunkRunes = 120
	p.adapt.textGen = uiStubTextGen{fn: func(ctx context.Context, _ core.AdaptGenerateRequest) (*core.AdaptGenerateResponse, error) {
		n := textCalls.Add(1)
		if n == 1 {
			return &core.AdaptGenerateResponse{
				Text: `{"scenes":[{"id":"s1","setting":"港口","panels":[{"id":"p1","shot":"远景","action":"停靠","prompt":"port"}]}]}`,
			}, nil
		}
		if n == 2 {
			chunk2Started <- struct{}{}
			<-ctx.Done()
			return nil, ctx.Err()
		}
		return &core.AdaptGenerateResponse{
			Text: `{"scenes":[{"id":"s2","setting":"广场","panels":[{"id":"p2","shot":"中景","action":"集合","prompt":"square"}]}]}`,
		}, nil
	}}

	p.openAdapt()
	drainNovelUntil(t, queued, func() bool { return !p.adapt.running })

	tt := ui.NewTester(p.view, 900, 760)
	tt.Frame()
	if err := tt.Click("生成分镜"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-chunk2Started:
	case <-time.After(8 * time.Second):
		t.Fatal("chunk 2 did not start")
	}
	tt.Frame()
	if err := tt.Click("取消"); err != nil {
		t.Fatal(err)
	}
	drainNovelUntil(t, queued, func() bool { return !p.adapt.running })
	if p.adapt.phase != adapt.PhaseCheckpointReady {
		t.Fatalf("expected PhaseCheckpointReady after cancel, got %s", p.adapt.phase)
	}

	// Resume and complete storyboard, then hit ErrContentBlocked on p1.
	tt.Frame()
	if err := tt.Click("继续生成分镜"); err != nil {
		t.Fatal(err)
	}
	drainNovelUntil(t, queued, func() bool { return !p.adapt.running })
	if p.adapt.phase != adapt.PhaseStoryboardReady {
		t.Fatalf("expected PhaseStoryboardReady after resume, got %s", p.adapt.phase)
	}

	var blockAttempts atomic.Int32
	p.adapt.imageGen = uiStubImageGen{fn: func(_ context.Context, _ core.AdaptPanelImageRequest) (*core.AdaptPanelImageResult, error) {
		blockAttempts.Add(1)
		return nil, fmt.Errorf("%w: blocked by policy", core.ErrAdaptContentBlocked)
	}}
	tt.Frame()
	if err := tt.Click("渲染 p1"); err != nil {
		t.Fatal(err)
	}
	drainNovelUntil(t, queued, func() bool { return !p.adapt.running })
	if blockAttempts.Load() != 1 || !p.adapt.contentBlocked || p.adapt.phase != adapt.PhaseBlocked {
		t.Fatalf("expected single blocked attempt & PhaseBlocked, attempts=%d blocked=%v phase=%s",
			blockAttempts.Load(), p.adapt.contentBlocked, p.adapt.phase)
	}
	tt.Frame()
	if !strings.Contains(strings.Join(tt.Texts(), "\n"), "内容策略拦截") {
		t.Fatalf("expected content policy block warning in UI, got %v", tt.Texts())
	}
}
