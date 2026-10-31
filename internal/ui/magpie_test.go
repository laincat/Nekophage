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
	if got := tierNote(state); got != "→ 交给 Magpie 的默认路由" {
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
