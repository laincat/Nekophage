package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/laincat/veneracat-core/comic/js"
)

// A dialog action is source code. A panic in it used to leave the caller waiting
// on a dialog nobody could answer, because the goroutine that owed the result was
// gone; it becomes an error the panel can show instead.
var errFill = errors.New("源拒绝")

func TestInvokeSafelyTurnsAPanicIntoAnError(t *testing.T) {
	err := invokeSafely(js.DialogAction{Invoke: func() error { panic("boom") }})
	if err == nil {
		t.Fatal("a panic must come back as an error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error = %v, want the panic's message", err)
	}
}

func TestInvokeSafelyPassesTheSourceError(t *testing.T) {
	err := invokeSafely(js.DialogAction{Invoke: func() error { return errFill }})
	if err != errFill {
		t.Errorf("error = %v, want the source's own", err)
	}
}

func TestInvokeSafelyWithoutACallback(t *testing.T) {
	if err := invokeSafely(js.DialogAction{}); err != nil {
		t.Errorf("error = %v, want nil", err)
	}
}

// A validator is source code too: rejecting input must not take the window down.
func TestSafeValidateSurvivesAPanic(t *testing.T) {
	if got := safeValidate(func(string) string { panic("bad") }, "x"); got == "" {
		t.Error("a panicking validator must report invalid input")
	}
	if got := safeValidate(func(text string) string {
		if text == "" {
			return "不能为空"
		}
		return ""
	}, ""); got != "不能为空" {
		t.Errorf("hint = %q", got)
	}
	if got := safeValidate(nil, "x"); got != "" {
		t.Errorf("hint = %q, want none", got)
	}
}

// A source that opens a spinner keeps it until it cancels; the main window shows
// one line while any is open, so the app does not look idle during real work.
func TestLoadingCountTracksTheSource(t *testing.T) {
	h := NewJSHandler()
	if got := h.LoadingCount(); got != 0 {
		t.Fatalf("count = %d, want 0", got)
	}
	first := h.ShowLoading(func() {})
	second := h.ShowLoading(func() {})
	if got := h.LoadingCount(); got != 2 {
		t.Errorf("count = %d, want 2", got)
	}
	h.CancelLoading(first)
	if got := h.LoadingCount(); got != 1 {
		t.Errorf("count = %d, want 1", got)
	}
	h.CancelLoading(second)
	h.CancelLoading(second) // cancelling twice must not go negative
	if got := h.LoadingCount(); got != 0 {
		t.Errorf("count = %d, want 0", got)
	}
}
