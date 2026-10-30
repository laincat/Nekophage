package ui

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestMultiValuesReadsBothShapes(t *testing.T) {
	cases := map[string][]string{
		`["a","b"]`: {"a", "b"},
		`"a"`:       {"a"},
		`a`:         {"a"},
		``:          nil,
		`[]`:        {},
	}
	for raw, want := range cases {
		if got := multiValues(raw); !reflect.DeepEqual(got, want) {
			t.Errorf("multiValues(%q) = %#v, want %#v", raw, got, want)
		}
	}
}

func TestEncodeMultiWritesOneString(t *testing.T) {
	if got := encodeMulti([]string{"a", "b"}); got != `["a","b"]` {
		t.Errorf("encodeMulti = %q", got)
	}
	if got := encodeMulti(nil); got != "" {
		t.Errorf("empty encodeMulti = %q", got)
	}
	// What is written can be read back.
	round := multiValues(encodeMulti([]string{"x", "y"}))
	if !reflect.DeepEqual(round, []string{"x", "y"}) {
		t.Errorf("round trip = %#v", round)
	}
}

func TestChooseOptionSingleAndMulti(t *testing.T) {
	a := &App{searchValues: []string{"any", `["a"]`},
		searchGroups: nil}
	a.win = nil

	// A single-choice group replaces its value.
	a.chooseOption(0, "rating", false)
	if a.searchValues[0] != "rating" {
		t.Errorf("single = %q", a.searchValues[0])
	}
	// A multi-choice group adds a key, keeping the order of the others.
	a.chooseOption(1, "b", true)
	if got := multiValues(a.searchValues[1]); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("multi add = %#v", got)
	}
	// Choosing it again drops it.
	a.chooseOption(1, "a", true)
	if got := multiValues(a.searchValues[1]); !reflect.DeepEqual(got, []string{"b"}) {
		t.Errorf("multi remove = %#v", got)
	}
	// A group index that does not exist is ignored, not a panic.
	a.chooseOption(9, "x", false)
}

func TestSourceOfNamesWhereAResultCameFrom(t *testing.T) {
	a := &App{resultSources: []string{"komiic", "manga_dex"}}
	a.sourceKey = "copy_manga"
	if got := a.sourceOf(0); got != "komiic" {
		t.Errorf("sourceOf(0) = %q", got)
	}
	if got := a.sourceOf(1); got != "manga_dex" {
		t.Errorf("sourceOf(1) = %q", got)
	}
	// A merged list that is shorter than the results falls back to the
	// chosen source rather than guessing.
	if got := a.sourceOf(5); got != "copy_manga" {
		t.Errorf("sourceOf(5) = %q", got)
	}
	if got := a.sourceOf(-1); got != "copy_manga" {
		t.Errorf("sourceOf(-1) = %q", got)
	}
}

func TestPickLabelNamesBothStates(t *testing.T) {
	if got := pickLabel(true, "全部源", "当前源"); got != "全部源" {
		t.Errorf("on = %q", got)
	}
	if got := pickLabel(false, "全部源", "当前源"); got != "当前源" {
		t.Errorf("off = %q", got)
	}
}

func TestSettingsRememberTheFavouriteFolder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ui.json")
	a := &App{favFolder: "以后看"}
	a.saveSettingsTo(path, "copy_manga", "海贼王")

	got, ok := readSettings(path)
	if !ok {
		t.Fatal("settings not written")
	}
	if got.Source != "copy_manga" || got.Query != "海贼王" || got.Folder != "以后看" {
		t.Errorf("settings = %+v", got)
	}
}
