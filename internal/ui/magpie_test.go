package ui

import (
	"testing"

	"github.com/laincat/nekophage/internal/magpie"
)

// The note above the tier picker must say where the request will actually go:
// an unmapped tier is the case that surprises people, so it has to be visible
// before sending rather than after.
func TestTierNoteSaysWhereTheRequestGoes(t *testing.T) {
	state := aiState{model: "default-group"}
	state.tierModels[0] = "cheap-group"
	state.tier = 0
	if got := tierNote(state); got != "→ cheap-group" {
		t.Fatalf("mapped tier = %q", got)
	}
	state.tier = 2 // complex, not mapped
	if got := tierNote(state); got != "→ default-group(默认模型)" {
		t.Fatalf("unmapped tier with a default = %q", got)
	}
	state.model = ""
	if got := tierNote(state); got != "→ 还没选模型，先读取模型目录" {
		t.Fatalf("nothing configured = %q", got)
	}
}

// An index from the segmented control is a UI value; it must never index the
// task list out of range.
func TestClampTierKeepsTheIndexInRange(t *testing.T) {
	for _, bad := range []int{-1, 3, 99} {
		if got := clampTier(bad); got != 1 {
			t.Fatalf("clampTier(%d) = %d", bad, got)
		}
	}
	for want := range magpie.Tasks {
		if got := clampTier(want); got != want {
			t.Fatalf("clampTier(%d) = %d", want, got)
		}
	}
}

// The mapping written to the settings holds only tiers the user filled in, so
// an empty field is "no choice" rather than an empty model name.
func TestTierMappingSkipsEmptyFields(t *testing.T) {
	a := &App{}
	a.ai.tierModels = [3]string{"cheap", "", "  "}
	got := a.tierMapping()
	if len(got) != 1 || got[string(magpie.TaskSimple)] != "cheap" {
		t.Fatalf("mapping = %+v", got)
	}
	if _, ok := got[string(magpie.TaskNormal)]; ok {
		t.Fatal("an empty tier was written")
	}
}

func TestTierLabelsAreChinese(t *testing.T) {
	if tierLabel(magpie.TaskSimple) != "简单" || tierLabel(magpie.TaskNormal) != "普通" || tierLabel(magpie.TaskComplex) != "复杂" {
		t.Fatal("tier labels changed")
	}
}

func TestModelPreviewStaysShort(t *testing.T) {
	many := make([]magpie.Model, 0, 10)
	for i := 0; i < 10; i++ {
		many = append(many, magpie.Model{ID: string(rune('a' + i))})
	}
	got := modelPreview(many)
	if len(got) > 40 {
		t.Fatalf("preview is too long: %q", got)
	}
	if modelPreview(nil) != "" {
		t.Fatal("an empty catalog produced a preview")
	}
}

// The 默认模型 field is what the user typed, and pressing 发送 must use it.
// Taking only the stored settings left Model empty, and the installed Magpie
// answers HTTP 400 for a request with no model — which is what a real send did
// before this was fixed.
func TestTheTypedModelReachesTheRequest(t *testing.T) {
	a := &App{}
	a.ai.addr = "  127.0.0.1:3425  "
	a.ai.model = "  group/laincat  "
	a.ai.tierModels = [3]string{"", "group/cheap", ""}

	settings := a.applyAIFields(magpie.Settings{Tiers: map[string]string{}})
	if settings.BaseURL != "127.0.0.1:3425" {
		t.Errorf("address = %q, want the trimmed one", settings.BaseURL)
	}
	if settings.Model != "group/laincat" {
		t.Errorf("model = %q, want what was typed", settings.Model)
	}
	if got := settings.ModelFor(magpie.TaskNormal); got != "group/cheap" {
		t.Errorf("tier normal = %q, want its own choice to win", got)
	}
	if got := settings.ModelFor(magpie.TaskSimple); got != "group/laincat" {
		t.Errorf("tier simple = %q, want the typed default", got)
	}
}
