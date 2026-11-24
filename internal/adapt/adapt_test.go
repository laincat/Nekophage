package adapt

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/magpie"
)

var testPNG = []byte{
	0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n',
	0x00, 0x00, 0x00, 0x0d, 'I', 'H', 'D', 'R',
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0a, 'I', 'D', 'A', 'T',
	0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00, 0x05,
	0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00,
	0x00, 0x00, 'I', 'E', 'N', 'D', 0xae, 0x42, 0x60, 0x82,
}

var testPNGAlt = func() []byte {
	out := append([]byte(nil), testPNG...)
	out = append(out, 0x00) // distinct byte content while remaining valid PNG header
	return out
}()

type stubTextGen struct {
	fn func(ctx context.Context, req core.AdaptGenerateRequest) (*core.AdaptGenerateResponse, error)
}

func (s stubTextGen) Generate(ctx context.Context, req core.AdaptGenerateRequest) (*core.AdaptGenerateResponse, error) {
	return s.fn(ctx, req)
}

type stubImageGen struct {
	fn func(ctx context.Context, req core.AdaptPanelImageRequest) (*core.AdaptPanelImageResult, error)
}

func (s stubImageGen) GenerateImage(ctx context.Context, req core.AdaptPanelImageRequest) (*core.AdaptPanelImageResult, error) {
	return s.fn(ctx, req)
}

func newTestManager(t *testing.T) (*Manager, *core.Core, *Store, string) {
	t.Helper()
	dir := t.TempDir()
	c, err := core.Open(core.Options{DataDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	st, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	mgr, err := NewManager(c, st)
	if err != nil {
		t.Fatal(err)
	}
	return mgr, c, st, dir
}

func TestAdaptChapterMagpieAndCrossChapterCharacters(t *testing.T) {
	var seenAuth string
	var seenModel string
	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		seenAuth = r.Header.Get("Authorization")
		var body struct {
			Model string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		seenModel = body.Model
		n := callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		content := `{"scenes":[{"id":"s1","setting":"Harbor","mood":"misty","panels":[{"id":"p1","shot":"wide","action":"Lin boards the boat","characters":["Lin"],"dialogues":[{"speaker":"Lin","text":"Departure.","kind":"speech"}],"prompt":"misty harbor anime panel"}]}],"newCharacters":[{"id":"c1","name":"Lin","appearance":"short black hair","outfit":"navy coat"}]}`
		if n == 2 {
			content = `{"scenes":[{"id":"s1","setting":"Deck","mood":"tense","panels":[{"id":"p1","shot":"close-up","action":"Mina speaks to Lin","characters":["Lin","Mina"],"prompt":"ship deck conversation"}]}],"newCharacters":[{"id":"c2","name":"Mina","appearance":"silver braid","outfit":"white cloak"}]}`
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]any{"content": content}}},
			"usage":   map[string]any{"prompt_tokens": 100, "completion_tokens": 50, "total_tokens": 150},
		})
	}))
	defer srv.Close()

	mgr, c, _, dir := newTestManager(t)
	ms := magpie.Settings{
		BaseURL:  srv.URL,
		Protocol: magpie.Chat,
		Token:    "local-magpie-secret-token",
		Model:    "default-group",
		Tiers:    map[string]string{string(magpie.TaskComplex): "complex-group"},
	}
	if err := magpie.SaveSettings(filepath.Join(dir, "magpie.json"), ms); err != nil {
		t.Fatal(err)
	}

	out1, err := mgr.AdaptChapter(context.Background(), AdaptChapterInput{
		NovelSourceKey: "novel_src",
		NovelID:        "book_1",
		NovelTitle:     "Sea of Stars",
		Chapter:        core.NovelChapter{ID: "ch1", Title: "Chapter 1"},
		Content:        core.NovelChapterContent{HTML: "<p>Lin stepped onto the harbor dock.</p>"},
		Text:           TextBackend{Mode: TextModeMagpie, MagpieTier: magpie.TaskComplex},
	})
	if err != nil {
		t.Fatalf("chapter 1 adapt failed: %v", err)
	}
	if seenAuth != "Bearer local-magpie-secret-token" || seenModel != "complex-group" {
		t.Fatalf("unexpected magpie request auth=%q model=%q", seenAuth, seenModel)
	}
	if out1.Phase != PhaseStoryboardReady || len(out1.Workspace.Project.Characters.Characters) != 1 {
		t.Fatalf("unexpected chapter 1 output: phase=%s chars=%+v", out1.Phase, out1.Workspace.Project.Characters)
	}

	// Reopen store & manager cleanly from disk to verify cross-chapter persistence.
	st2, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	mgr2, err := NewManager(c, st2)
	if err != nil {
		t.Fatal(err)
	}
	out2, err := mgr2.AdaptChapter(context.Background(), AdaptChapterInput{
		NovelSourceKey: "novel_src",
		NovelID:        "book_1",
		NovelTitle:     "Sea of Stars",
		Chapter:        core.NovelChapter{ID: "ch2", Title: "Chapter 2"},
		Content:        core.NovelChapterContent{HTML: "<p>Mina greeted Lin on the ship deck.</p>"},
		Text:           TextBackend{Mode: TextModeMagpie, MagpieTier: magpie.TaskComplex},
	})
	if err != nil {
		t.Fatalf("chapter 2 adapt failed: %v", err)
	}
	if len(out2.Workspace.Project.Characters.Characters) != 2 {
		t.Fatalf("expected 2 merged cross-chapter characters, got %+v", out2.Workspace.Project.Characters.Characters)
	}
	if len(out2.Workspace.Project.Storyboards) != 2 {
		t.Fatalf("expected 2 persisted storyboards, got %d", len(out2.Workspace.Project.Storyboards))
	}
	if out2.Workspace.Project.Usage.Calls != 2 || out2.Workspace.Project.Usage.TotalTokens != 300 {
		t.Fatalf("unexpected cumulative usage: %+v", out2.Workspace.Project.Usage)
	}
}

func TestAdaptChapterMultiChunkCheckpointBudgetAndResume(t *testing.T) {
	mgr, c, _, dir := newTestManager(t)

	var calls atomic.Int32
	gen := stubTextGen{fn: func(ctx context.Context, req core.AdaptGenerateRequest) (*core.AdaptGenerateResponse, error) {
		n := calls.Add(1)
		return &core.AdaptGenerateResponse{
			Text: fmt.Sprintf(`{"scenes":[{"id":"s%d","setting":"Hall %d","panels":[{"id":"p%d","shot":"medium","action":"Chunk %d action","characters":["Hero"],"prompt":"chunk %d panel"}]}],"newCharacters":[{"id":"c1","name":"Hero","appearance":"scar"}]}`, n, n, n, n, n),
			Usage: core.AdaptUsage{
				Calls:        1,
				InputTokens:  40,
				OutputTokens: 20,
				TotalTokens:  60,
			},
		}, nil
	}}

	html := "<p>" + strings.Repeat("A", 120) + "</p><p>" + strings.Repeat("B", 120) + "</p><p>" + strings.Repeat("C", 120) + "</p>"

	// Cap budget at 1 call: chunk 1 completes, chunk 2 hits ErrBudgetExceeded.
	budget1 := core.AdaptBudget{MaxCalls: 1}
	out1, err := mgr.AdaptChapter(context.Background(), AdaptChapterInput{
		NovelSourceKey: "src",
		NovelID:        "n1",
		NovelTitle:     "Epic",
		Chapter:        core.NovelChapter{ID: "ch1", Title: "Long Chapter"},
		Content:        core.NovelChapterContent{HTML: html},
		Text:           TextBackend{Generator: gen},
		Budget:         &budget1,
		MaxChunkRunes:  140,
	})
	if !IsBudgetExceeded(err) {
		t.Fatalf("expected ErrBudgetExceeded, got %v", err)
	}
	if out1 == nil || !out1.BudgetExceeded || out1.Phase != PhasePausedBudget {
		t.Fatalf("unexpected paused output: %+v", out1)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected exactly 1 call before budget pause, got %d", calls.Load())
	}
	if cp := out1.Workspace.Checkpoints["ch1"]; cp == nil || cp.CompletedChunks != 1 || cp.TotalChunks != 3 {
		t.Fatalf("unexpected checkpoint: %+v", cp)
	}

	// Reload from disk in a fresh Store/Manager, raise budget, and resume.
	st2, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	mgr2, err := NewManager(c, st2)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := mgr2.LoadWorkspace("src", "n1", "Epic")
	if err != nil {
		t.Fatal(err)
	}
	snap := loaded.ChapterSnapshot("ch1")
	if !snap.HasCheckpoint || snap.CompletedChunks != 1 || snap.TotalChunks != 3 || snap.Phase != PhasePausedBudget {
		t.Fatalf("unexpected reloaded snapshot: %+v", snap)
	}

	budget3 := core.AdaptBudget{MaxCalls: 3}
	out2, err := mgr2.AdaptChapter(context.Background(), AdaptChapterInput{
		NovelSourceKey: "src",
		NovelID:        "n1",
		NovelTitle:     "Epic",
		Chapter:        core.NovelChapter{ID: "ch1", Title: "Long Chapter"},
		Content:        core.NovelChapterContent{HTML: html},
		Text:           TextBackend{Generator: gen},
		Budget:         &budget3,
		MaxChunkRunes:  140,
	})
	if err != nil {
		t.Fatalf("resume adapt failed: %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("expected 3 total calls (chunk 1 not re-billed), got %d", calls.Load())
	}
	if out2.Phase != PhaseStoryboardReady {
		t.Fatalf("expected PhaseStoryboardReady, got %s", out2.Phase)
	}
	if len(out2.Workspace.Checkpoints) != 0 {
		t.Fatalf("expected checkpoint to be cleared after completion, got %+v", out2.Workspace.Checkpoints)
	}
	sb := out2.Workspace.Project.Storyboards["ch1"]
	if sb == nil || len(sb.Scenes) != 3 {
		t.Fatalf("expected 3 merged scenes, got %+v", sb)
	}
	// Verify checkpoint file was deleted from disk too.
	if _, err := st2.LoadCheckpoint(out2.Workspace.Project.ID, "ch1"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected checkpoint file deleted, got err=%v", err)
	}
}

func TestAdaptChapterCancellationPreservesCheckpoint(t *testing.T) {
	mgr, _, _, _ := newTestManager(t)
	ctx, cancel := context.WithCancel(context.Background())

	var calls atomic.Int32
	gen := stubTextGen{fn: func(callCtx context.Context, _ core.AdaptGenerateRequest) (*core.AdaptGenerateResponse, error) {
		n := calls.Add(1)
		if n == 2 {
			cancel()
			return nil, callCtx.Err()
		}
		return &core.AdaptGenerateResponse{
			Text: `{"scenes":[{"id":"s1","setting":"Forest","panels":[{"id":"p1","shot":"wide","action":"walk","prompt":"forest"}]}]}`,
			Usage: core.AdaptUsage{
				Calls: 1, TotalTokens: 25,
			},
		}, nil
	}}

	html := "<p>" + strings.Repeat("X", 120) + "</p><p>" + strings.Repeat("Y", 120) + "</p>"
	out, err := mgr.AdaptChapter(ctx, AdaptChapterInput{
		NovelSourceKey: "src",
		NovelID:        "n_cancel",
		NovelTitle:     "Cancel Test",
		Chapter:        core.NovelChapter{ID: "ch1", Title: "Ch 1"},
		Content:        core.NovelChapterContent{HTML: html},
		Text:           TextBackend{Generator: gen},
		MaxChunkRunes:  140,
	})
	if !IsCanceled(err) {
		t.Fatalf("expected canceled error, got %v", err)
	}
	if out == nil || !out.Canceled || out.Phase != PhaseCheckpointReady {
		t.Fatalf("expected PhaseCheckpointReady on cancel with 1 chunk done, got %+v", out)
	}
	if cp := out.Workspace.Checkpoints["ch1"]; cp == nil || cp.CompletedChunks != 1 {
		t.Fatalf("expected checkpoint with 1 completed chunk, got %+v", cp)
	}
}

func TestRenderPanelStoryboardForceRerenderAndLocalRefs(t *testing.T) {
	mgr, _, _, dir := newTestManager(t)

	// Prepare local style and character reference PNG files.
	stylePath := filepath.Join(dir, "style_ref.png")
	charPath := filepath.Join(dir, "char_lin.png")
	if err := os.WriteFile(stylePath, testPNG, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(charPath, testPNGAlt, 0o600); err != nil {
		t.Fatal(err)
	}

	// Seed a 2-panel storyboard.
	_, err := mgr.AdaptChapter(context.Background(), AdaptChapterInput{
		NovelSourceKey: "src",
		NovelID:        "book_render",
		NovelTitle:     "Render Novel",
		Chapter:        core.NovelChapter{ID: "ch1", Title: "One"},
		Content:        core.NovelChapterContent{HTML: "<p>Lin walks under neon lights.</p>"},
		Text: TextBackend{Generator: stubTextGen{fn: func(_ context.Context, _ core.AdaptGenerateRequest) (*core.AdaptGenerateResponse, error) {
			return &core.AdaptGenerateResponse{
				Text: `{"scenes":[{"id":"s1","setting":"Alley","mood":"neon","panels":[{"id":"p1","shot":"wide","action":"Lin enters","characters":["Lin"],"prompt":"neon alley"},{"id":"p2","shot":"close","action":"Lin smiles","characters":["Lin"],"prompt":"close up smile"}]}],"newCharacters":[{"id":"c1","name":"Lin","appearance":"blue eyes"}]}`,
			}, nil
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}

	var calls atomic.Int32
	var sawRefs int
	imgGen := stubImageGen{fn: func(_ context.Context, req core.AdaptPanelImageRequest) (*core.AdaptPanelImageResult, error) {
		n := calls.Add(1)
		sawRefs = len(req.References)
		data := testPNG
		if n >= 3 {
			data = testPNGAlt
		}
		return &core.AdaptPanelImageResult{
			MIMEType: "image/png",
			Data:     data,
			Usage:    core.AdaptUsage{ImageCalls: 1},
		}, nil
	}}

	// Budget limit of 1 image call: renders p1, pauses before p2 with ErrBudgetExceeded.
	budget1 := core.AdaptBudget{MaxImageCalls: 1}
	sbOut1, err := mgr.RenderStoryboard(context.Background(), RenderStoryboardInput{
		NovelSourceKey: "src",
		NovelID:        "book_render",
		ChapterID:      "ch1",
		Budget:         &budget1,
		Image: ImageBackend{
			Generator:         imgGen,
			StyleRefPaths:     []string{stylePath},
			CharacterRefPaths: map[string][]string{"Lin": {charPath}},
		},
	})
	if !IsBudgetExceeded(err) {
		t.Fatalf("expected ErrBudgetExceeded, got %v", err)
	}
	if sbOut1 == nil || !sbOut1.BudgetExceeded || sbOut1.Phase != PhasePausedBudget || sbOut1.Result.Rendered != 1 {
		t.Fatalf("unexpected budget-paused render output: %+v", sbOut1)
	}
	if sawRefs != 2 {
		t.Fatalf("expected 2 local references (style + character), got %d", sawRefs)
	}

	// Raise budget and resume RenderStoryboard: p1 skipped, p2 rendered.
	budget3 := core.AdaptBudget{MaxImageCalls: 3}
	sbOut2, err := mgr.RenderStoryboard(context.Background(), RenderStoryboardInput{
		NovelSourceKey: "src",
		NovelID:        "book_render",
		ChapterID:      "ch1",
		Budget:         &budget3,
		Image:          ImageBackend{Generator: imgGen},
	})
	if err != nil {
		t.Fatalf("resumed RenderStoryboard failed: %v", err)
	}
	if sbOut2.Phase != PhaseCompleted || sbOut2.Result.Skipped != 1 || sbOut2.Result.Rendered != 1 {
		t.Fatalf("expected 1 skipped + 1 rendered, got %+v", sbOut2)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 total image calls, got %d", calls.Load())
	}

	// Verify saved panel image bytes readable through store guard.
	p1Ref := sbOut2.Workspace.Project.Storyboards["ch1"].Scenes[0].Panels[0].ImageRefID
	raw1, mime1, err := mgr.ReadPanelImage(sbOut2.Workspace.Project.ID, p1Ref)
	if err != nil || mime1 != "image/png" || !bytes.Equal(raw1, testPNG) {
		t.Fatalf("ReadPanelImage p1 failed: mime=%q len=%d err=%v", mime1, len(raw1), err)
	}

	// ForceRerender p1: must invoke generator again and overwrite image bytes.
	panelOut, err := mgr.RenderPanel(context.Background(), RenderPanelInput{
		NovelSourceKey: "src",
		NovelID:        "book_render",
		ChapterID:      "ch1",
		SceneID:        "s1",
		PanelID:        "p1",
		ForceRerender:  true,
		Image:          ImageBackend{Generator: imgGen},
	})
	if err != nil {
		t.Fatalf("ForceRerender failed: %v", err)
	}
	if panelOut.Result.Skipped || calls.Load() != 3 {
		t.Fatalf("expected ForceRerender to execute 3rd call, calls=%d res=%+v", calls.Load(), panelOut.Result)
	}
	raw2, _, err := mgr.ReadPanelImage(panelOut.Workspace.Project.ID, panelOut.Result.RefID)
	if err != nil || !bytes.Equal(raw2, testPNGAlt) {
		t.Fatalf("expected overwritten panel bytes after ForceRerender, err=%v len=%d", err, len(raw2))
	}
}

func TestRenderPanelContentBlockedNoRetry(t *testing.T) {
	mgr, _, _, _ := newTestManager(t)

	_, err := mgr.AdaptChapter(context.Background(), AdaptChapterInput{
		NovelSourceKey: "src",
		NovelID:        "book_block",
		NovelTitle:     "Block Novel",
		Chapter:        core.NovelChapter{ID: "ch1", Title: "Ch 1"},
		Content:        core.NovelChapterContent{HTML: "<p>A tense scene.</p>"},
		Text: TextBackend{Generator: stubTextGen{fn: func(_ context.Context, _ core.AdaptGenerateRequest) (*core.AdaptGenerateResponse, error) {
			return &core.AdaptGenerateResponse{
				Text: `{"scenes":[{"id":"s1","setting":"Roof","panels":[{"id":"p1","shot":"wide","action":"duel","prompt":"dramatic duel"}]}]}`,
			}, nil
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}

	var attempts atomic.Int32
	blockedGen := stubImageGen{fn: func(_ context.Context, _ core.AdaptPanelImageRequest) (*core.AdaptPanelImageResult, error) {
		attempts.Add(1)
		return nil, fmt.Errorf("%w: safety filter triggered", core.ErrAdaptContentBlocked)
	}}

	out, err := mgr.RenderPanel(context.Background(), RenderPanelInput{
		NovelSourceKey: "src",
		NovelID:        "book_block",
		ChapterID:      "ch1",
		SceneID:        "s1",
		PanelID:        "p1",
		Image: ImageBackend{
			Generator:  blockedGen,
			MaxRetries: 3, // must NOT retry ErrContentBlocked
		},
	})
	if !IsContentBlocked(err) {
		t.Fatalf("expected ErrContentBlocked, got %v", err)
	}
	if attempts.Load() != 1 {
		t.Fatalf("ErrContentBlocked must not be retried, got %d attempts", attempts.Load())
	}
	if out == nil || !out.ContentBlocked || out.Phase != PhaseBlocked {
		t.Fatalf("expected ContentBlocked & PhaseBlocked, got %+v", out)
	}
	if !strings.Contains(out.UserMessage, "内容安全策略") || !strings.Contains(out.UserMessage, "未自动重试") {
		t.Fatalf("unexpected user message for content block: %q", out.UserMessage)
	}

	// Reload from disk: chapter snapshot must still distinguish PhaseBlocked.
	ws, err := mgr.LoadWorkspace("src", "book_block", "Block Novel")
	if err != nil {
		t.Fatal(err)
	}
	snap := ws.ChapterSnapshot("ch1")
	if snap.Phase != PhaseBlocked || snap.BlockedPanels != 1 {
		t.Fatalf("reloaded snapshot lost blocked state: %+v", snap)
	}
}

func TestRenderProvidersGeminiAndOpenAIAndSecretIsolation(t *testing.T) {
	const textSecret = "sk-super-secret-text-key-987654321"
	const imageSecret = "gm-super-secret-image-key-123456789"

	var sawGeminiHeader string
	var sawOpenAIAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/chat/completions":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{"message": map[string]any{"content": `{"scenes":[{"id":"s1","setting":"Lab","panels":[{"id":"p1","shot":"wide","action":"test","prompt":"lab"},{"id":"p2","shot":"close","action":"fail","prompt":"lab2"}]}]}`}},
				},
			})
		case strings.Contains(r.URL.Path, ":generateContent"):
			sawGeminiHeader = r.Header.Get("x-goog-api-key")
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"candidates": []map[string]any{
					{
						"content": map[string]any{
							"parts": []map[string]any{
								{"inlineData": map[string]any{
									"mimeType": "image/png",
									"data":     base64.StdEncoding.EncodeToString(testPNG),
								}},
							},
						},
					},
				},
			})
		case r.URL.Path == "/v1/images/generations":
			sawOpenAIAuth = r.Header.Get("Authorization")
			// Deliberately echo the secret in an upstream error to verify redaction on disk!
			w.WriteHeader(http.StatusBadGateway)
			_, _ = fmt.Fprintf(w, "upstream failure leaking %s", imageSecret)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	mgr, _, st, dir := newTestManager(t)

	// Persist non-secret settings first.
	settings := DefaultSettings()
	settings.TextMode = TextModeAPI
	settings.TextBaseURL = srv.URL + "/v1"
	settings.TextModel = "test-text-model"
	settings.ImageProvider = core.AdaptImageGemini
	settings.ImageBaseURL = srv.URL
	settings.ImageModel = "gemini-2.5-flash-image"
	if err := st.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}

	// Adapt via standard API endpoint with secret key in memory.
	_, err := mgr.AdaptChapter(context.Background(), AdaptChapterInput{
		NovelSourceKey: "src",
		NovelID:        "secret_book",
		NovelTitle:     "Secret Audit",
		Chapter:        core.NovelChapter{ID: "ch1", Title: "Ch 1"},
		Content:        core.NovelChapterContent{HTML: "<p>Inside the lab.</p>"},
		Text:           settings.BuildTextBackend(dir, textSecret),
	})
	if err != nil {
		t.Fatalf("API text adapt failed: %v", err)
	}

	// Render p1 via Gemini.
	if _, err := mgr.RenderPanel(context.Background(), RenderPanelInput{
		NovelSourceKey: "src",
		NovelID:        "secret_book",
		ChapterID:      "ch1",
		SceneID:        "s1",
		PanelID:        "p1",
		Image:          settings.BuildImageBackend(imageSecret),
	}); err != nil {
		t.Fatalf("Gemini render failed: %v", err)
	}
	if sawGeminiHeader != imageSecret {
		t.Fatalf("Gemini did not receive API key header")
	}

	// Render p2 via OpenAI where upstream echoes the secret in an HTTP 502 error.
	settings.ImageProvider = core.AdaptImageOpenAI
	settings.ImageBaseURL = srv.URL + "/v1"
	settings.ImageModel = "gpt-image-1"
	outFail, err := mgr.RenderPanel(context.Background(), RenderPanelInput{
		NovelSourceKey: "src",
		NovelID:        "secret_book",
		ChapterID:      "ch1",
		SceneID:        "s1",
		PanelID:        "p2",
		Image:          settings.BuildImageBackend(imageSecret),
	})
	if err == nil {
		t.Fatal("expected OpenAI 502 error")
	}
	if sawOpenAIAuth != "Bearer "+imageSecret {
		t.Fatalf("OpenAI did not receive Bearer header")
	}
	if strings.Contains(err.Error(), imageSecret) || strings.Contains(outFail.UserMessage, imageSecret) {
		t.Fatalf("secret leaked in returned error: err=%q msg=%q", err.Error(), outFail.UserMessage)
	}

	// Walk EVERY persisted file under dataDir and confirm zero secrets were written.
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if bytes.Contains(data, []byte(textSecret)) || bytes.Contains(data, []byte(imageSecret)) {
			t.Errorf("secret leaked to disk in %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPersistencePathTraversalSafety(t *testing.T) {
	dir := t.TempDir()
	st, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Adversarial identifiers must remain strictly inside <dir>/adapt/projects/.
	evilSource := "../../etc"
	evilNovel := "..\\..\\windows\\CON"
	evilChapter := "../../../pwned/ch1"
	ws, err := st.LoadOrCreateProject(evilSource, evilNovel, "Evil")
	if err != nil {
		t.Fatal(err)
	}
	ws.Project.Storyboards[evilChapter] = &core.AdaptStoryboard{
		ChapterID: evilChapter,
		Scenes: []core.AdaptScene{{
			ID:     "../scene",
			Panels: []core.AdaptPanel{{ID: "..\\panel", Prompt: "ok", ImageStatus: core.AdaptImagePending}},
		}},
	}
	ws.Checkpoints[evilChapter] = &core.AdaptCheckpoint{
		ChapterID:       evilChapter,
		TotalChunks:     2,
		CompletedChunks: 1,
	}
	if err := st.SaveProject(ws); err != nil {
		t.Fatal(err)
	}

	refID, err := st.SavePanelImage(ws.Project.ID, core.AdaptPanelImageMeta{
		ProjectID: ws.Project.ID,
		ChapterID: evilChapter,
		SceneID:   "../scene",
		PanelID:   "..\\panel",
	}, "image/png", testPNG)
	if err != nil {
		t.Fatalf("SavePanelImage with adversarial IDs failed: %v", err)
	}
	resolved, err := st.ResolvePanelImagePath(ws.Project.ID, refID)
	if err != nil {
		t.Fatalf("ResolvePanelImagePath failed: %v", err)
	}
	rel, err := filepath.Rel(st.Root(), resolved)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		t.Fatalf("saved panel escaped adapt root: root=%s resolved=%s rel=%s", st.Root(), resolved, rel)
	}

	// Reject traversal refIDs on ReadPanelImage / ResolvePanelImagePath.
	badRefs := []string{
		"",
		"../project.json",
		"panels/../../escape.png",
		"panels/ch/..\\escape.png",
		"/etc/passwd",
		`C:\Windows\win.ini`,
		"panels/ch/sub/deep.png",
		"other/ch/p.png",
	}
	for _, bad := range badRefs {
		if _, err := st.ResolvePanelImagePath(ws.Project.ID, bad); !errors.Is(err, ErrUnsafePath) {
			t.Errorf("ResolvePanelImagePath(%q) expected ErrUnsafePath, got %v", bad, err)
		}
		if _, _, err := st.ReadPanelImage(ws.Project.ID, bad); !errors.Is(err, ErrUnsafePath) {
			t.Errorf("ReadPanelImage(%q) expected ErrUnsafePath, got %v", bad, err)
		}
	}

	// Reject non-image local reference files.
	txtPath := filepath.Join(dir, "not_an_image.txt")
	if err := os.WriteFile(txtPath, []byte("plain text, not an image"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadImageRef(txtPath, core.AdaptImageRefStyle, "bad"); err == nil {
		t.Fatal("LoadImageRef should reject non-image files")
	}
}

func TestRenderStoryboardCancellationAndBlockedContinue(t *testing.T) {
	mgr, _, _, _ := newTestManager(t)

	_, err := mgr.AdaptChapter(context.Background(), AdaptChapterInput{
		NovelSourceKey: "src",
		NovelID:        "book_mid_cancel",
		NovelTitle:     "Mid Cancel",
		Chapter:        core.NovelChapter{ID: "ch1", Title: "Ch 1"},
		Content:        core.NovelChapterContent{HTML: "<p>Two panels.</p>"},
		Text: TextBackend{Generator: stubTextGen{fn: func(_ context.Context, _ core.AdaptGenerateRequest) (*core.AdaptGenerateResponse, error) {
			return &core.AdaptGenerateResponse{
				Text: `{"scenes":[{"id":"s1","setting":"Bridge","panels":[{"id":"p1","shot":"wide","action":"enter","prompt":"bridge 1"},{"id":"p2","shot":"close","action":"look","prompt":"bridge 2"}]}]}`,
			}, nil
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32
	gen := stubImageGen{fn: func(callCtx context.Context, req core.AdaptPanelImageRequest) (*core.AdaptPanelImageResult, error) {
		n := calls.Add(1)
		if n == 1 {
			return &core.AdaptPanelImageResult{MIMEType: "image/png", Data: testPNG}, nil
		}
		if n == 2 {
			cancel()
			return nil, callCtx.Err()
		}
		return &core.AdaptPanelImageResult{MIMEType: "image/png", Data: testPNGAlt}, nil
	}}

	outCancel, err := mgr.RenderStoryboard(ctx, RenderStoryboardInput{
		NovelSourceKey: "src",
		NovelID:        "book_mid_cancel",
		ChapterID:      "ch1",
		Image:          ImageBackend{Generator: gen},
	})
	if !IsCanceled(err) {
		t.Fatalf("expected canceled error, got %v", err)
	}
	if outCancel == nil || !outCancel.Canceled || outCancel.Result.Rendered != 1 {
		t.Fatalf("expected 1 rendered before cancel, got %+v", outCancel)
	}

	// Resume after cancellation: p1 is already done and skipped, p2 hits ErrContentBlocked with ContinueOnError.
	var blockedAttempts atomic.Int32
	blockGen := stubImageGen{fn: func(_ context.Context, req core.AdaptPanelImageRequest) (*core.AdaptPanelImageResult, error) {
		if req.PanelID == "p2" {
			blockedAttempts.Add(1)
			return nil, fmt.Errorf("%w: blocked p2", core.ErrAdaptContentBlocked)
		}
		t.Fatalf("unexpected render call for already-done panel %s", req.PanelID)
		return nil, nil
	}}
	outResume, err := mgr.RenderStoryboard(context.Background(), RenderStoryboardInput{
		NovelSourceKey: "src",
		NovelID:        "book_mid_cancel",
		ChapterID:      "ch1",
		Image: ImageBackend{
			Generator:       blockGen,
			MaxRetries:      2,
			ContinueOnError: true,
		},
	})
	if err != nil {
		t.Fatalf("ContinueOnError storyboard render returned unexpected error: %v", err)
	}
	if blockedAttempts.Load() != 1 {
		t.Fatalf("blocked panel must not retry, got %d attempts", blockedAttempts.Load())
	}
	if !outResume.ContentBlocked || outResume.BlockedCount != 1 || outResume.Phase != PhaseBlocked || outResume.Result.Skipped != 1 {
		t.Fatalf("unexpected resume/blocked output: %+v", outResume)
	}
}
