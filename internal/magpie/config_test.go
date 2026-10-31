package magpie

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func tempSettings(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "magpie.json")
}

func TestSettingsRoundTrip(t *testing.T) {
	path := tempSettings(t)
	want := Settings{
		BaseURL:  "http://127.0.0.1:3425",
		Protocol: Messages,
		Model:    "default-group",
		Tiers:    map[string]string{"complex": "strong-group", "simple": "cheap-group"},
	}
	if err := SaveSettings(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.BaseURL != want.BaseURL || got.Protocol != want.Protocol || got.Model != want.Model {
		t.Fatalf("loaded = %+v", got)
	}
	if got.ModelFor(TaskComplex) != "strong-group" || got.ModelFor(TaskSimple) != "cheap-group" {
		t.Fatalf("tiers = %+v", got.Tiers)
	}
	// A tier with no choice falls back to the explicit model.
	if got.ModelFor(TaskNormal) != "default-group" {
		t.Fatalf("normal = %q, want the explicit model", got.ModelFor(TaskNormal))
	}
}

func TestMissingSettingsAreTheDefault(t *testing.T) {
	got, err := LoadSettings(tempSettings(t))
	if err != nil {
		t.Fatal(err)
	}
	if got.BaseURL != "http://"+DefaultAddr || got.Protocol != Chat {
		t.Fatalf("defaults = %+v", got)
	}
	if got.ModelFor(TaskComplex) != "" {
		t.Fatal("an unconfigured tier should let the gateway choose")
	}
}

// A damaged file must not be silently replaced by defaults: that would drop the
// user's address and look like Magpie had stopped working.
func TestDamagedSettingsAreAnErrorAndSurvive(t *testing.T) {
	path := tempSettings(t)
	damaged := []byte(`{"baseUrl":`)
	if err := os.WriteFile(path, damaged, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSettings(path); err == nil {
		t.Fatal("a damaged settings file was accepted")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(damaged) {
		t.Fatalf("the damaged file was rewritten: %s, %v", got, err)
	}
}

func TestUnknownTierIsRejected(t *testing.T) {
	s := DefaultSettings()
	s.Tiers["hard"] = "strong-group"
	if err := s.Validate(); err == nil {
		t.Fatal("a misspelled tier was accepted")
	}
	if err := SaveSettings(tempSettings(t), s); err == nil {
		t.Fatal("a misspelled tier was saved")
	}
}

func TestSettingsRejectBadAddressAndProtocol(t *testing.T) {
	s := DefaultSettings()
	s.BaseURL = "ftp://127.0.0.1:3425"
	if err := s.Validate(); err == nil {
		t.Fatal("accepted a non-http address")
	}
	s = DefaultSettings()
	s.Protocol = "gemini"
	if err := s.Validate(); err == nil {
		t.Fatal("accepted an unknown protocol")
	}
	s = DefaultSettings()
	s.Token = "bad\ntoken"
	if err := s.Validate(); err == nil {
		t.Fatal("accepted a token with a control character")
	}
}

// A tier mapped to whitespace means "no choice here", not "send an empty model".
func TestBlankTierFallsBack(t *testing.T) {
	s := DefaultSettings()
	s.Model = "fallback-group"
	s.Tiers["complex"] = "   "
	if got := s.ModelFor(TaskComplex); got != "fallback-group" {
		t.Fatalf("ModelFor = %q", got)
	}
}

func TestBlankTiersAreDroppedFromTheFile(t *testing.T) {
	path := tempSettings(t)
	s := DefaultSettings()
	s.Tiers["simple"] = "  "
	s.Tiers["complex"] = "strong-group"
	if err := SaveSettings(path, s); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, `"simple"`) {
		t.Fatalf("a blank tier was written: %s", text)
	}
	if !strings.Contains(text, `"complex"`) {
		t.Fatalf("the real tier is missing: %s", text)
	}
}
