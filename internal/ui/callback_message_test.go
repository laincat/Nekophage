package ui

import (
	"errors"
	"testing"
)

// A source declares several callbacks, and the panel used to keep one message for
// all of them: pressing one button put its answer beside every callback row.
func TestCallbackAnswerStaysOnItsOwnRow(t *testing.T) {
	var m manageState
	m.callbackBusy = "run"

	m.recordCallback("run", "ran", nil)
	if got := m.callbackMsgs["run"]; got != "ran" {
		t.Fatalf("run = %q", got)
	}
	if _, ok := m.callbackMsgs["quiet"]; ok {
		t.Fatal("a press wrote an answer for a callback that was never pressed")
	}
	if m.callbackBusy != "" {
		t.Fatalf("still busy after the answer: %q", m.callbackBusy)
	}

	m.recordCallback("quiet", "", nil)
	if got := m.callbackMsgs["quiet"]; got != "已完成" {
		t.Fatalf("an empty answer should read as done, got %q", got)
	}
	// The earlier answer is untouched by a later press.
	if got := m.callbackMsgs["run"]; got != "ran" {
		t.Fatalf("run was rewritten to %q", got)
	}
}

// The source's own text is what a person needs to see; a generic failure would
// hide "Login expired" or "no network".
func TestCallbackFailureKeepsTheSourcesWords(t *testing.T) {
	var m manageState
	m.recordCallback("refresh", "", errors.New("no network"))

	got := m.callbackMsgs["refresh"]
	if got == "" || got == "已完成" {
		t.Fatalf("failure = %q, want the source's reason", got)
	}
}
