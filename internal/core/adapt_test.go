package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCoreAdaptBridge(t *testing.T) {
	pngBytes := []byte{
		0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n',
		0x00, 0x00, 0x00, 0x0d, 'I', 'H', 'D', 'R',
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0a, 'I', 'D', 'A', 'T',
		0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00, 0x05,
		0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00,
		0x00, 0x00, 'I', 'E', 'N', 'D', 0xae, 0x42, 0x60, 0x82,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/chat/completions":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{
						"message": map[string]any{
							"content": `{"scenes":[{"id":"s1","setting":"station","mood":"calm","panels":[{"id":"p1","shot":"wide","action":"arrive","characters":["Lin"],"prompt":"anime train station"}]}],"newCharacters":[{"id":"c1","name":"Lin","appearance":"short hair","outfit":"coat"}]}`,
						},
					},
				},
				"usage": map[string]any{
					"prompt_tokens":     10,
					"completion_tokens": 20,
					"total_tokens":      30,
				},
			})
		case r.URL.Path == "/v1/images/generations":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"b64_json": base64.StdEncoding.EncodeToString(pngBytes)},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c, err := Open(Options{DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	proj := &AdaptProject{
		ID:             "p1",
		NovelSourceKey: "novel_src",
		NovelID:        "book1",
		Title:          "Starlight",
	}

	adaptRes, err := c.AdaptChapter(context.Background(), AdaptChapterRequest{
		Project: proj,
		Chapter: NovelChapter{ID: "ch1", Title: "Arrival"},
		Content: NovelChapterContent{HTML: "<p>Lin stepped onto the quiet station platform.</p>"},
		Text: AdaptTextConfig{
			Protocol: AIProtocolChat,
			BaseURL:  srv.URL + "/v1",
			APIKey:   "test-key",
			Model:    "gpt-test",
			JSONMode: true,
		},
	})
	if err != nil {
		t.Fatalf("AdaptChapter failed: %v", err)
	}
	if adaptRes.Storyboard == nil || len(adaptRes.Storyboard.Scenes) != 1 || len(proj.Characters.Characters) != 1 {
		t.Fatalf("unexpected adapt result: %+v, chars=%+v", adaptRes, proj.Characters)
	}

	saved := map[string][]byte{}
	sink := AdaptImageSinkFunc(func(_ context.Context, meta AdaptPanelImageMeta, _ string, data []byte) (string, error) {
		ref := meta.SceneID + "/" + meta.PanelID + ".png"
		saved[ref] = append([]byte(nil), data...)
		return ref, nil
	})

	imgCfg := AdaptImageConfig{
		Provider: AdaptImageOpenAI,
		BaseURL:  srv.URL + "/v1",
		APIKey:   "test-image-key",
		Model:    "dall-e-3",
	}

	panelRes, err := c.RenderPanel(context.Background(), RenderPanelRequest{
		Project:   proj,
		ChapterID: "ch1",
		SceneID:   "s1",
		PanelID:   "p1",
		Image:     imgCfg,
		Sink:      sink,
	})
	if err != nil {
		t.Fatalf("RenderPanel failed: %v", err)
	}
	if panelRes.Skipped || !strings.HasSuffix(panelRes.RefID, "p1.png") || len(saved[panelRes.RefID]) != len(pngBytes) {
		t.Fatalf("unexpected panel result: %+v", panelRes)
	}

	sbRes, err := c.RenderStoryboard(context.Background(), RenderStoryboardRequest{
		Project:   proj,
		ChapterID: "ch1",
		Image:     imgCfg,
		Sink:      sink,
	})
	if err != nil {
		t.Fatalf("RenderStoryboard failed: %v", err)
	}
	if sbRes.Skipped != 1 || sbRes.Rendered != 0 {
		t.Fatalf("expected already-done panel to be skipped: %+v", sbRes)
	}
}
