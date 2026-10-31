package ui

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/laincat/nekophage/internal/core"
)

// A source's console output has to reach the buffer the panel reads: it used to
// go to stderr only, where a user who started the app from a launcher never
// sees it.
func TestSourceLogsReachTheBuffer(t *testing.T) {
	c, err := core.Open(core.Options{DataDir: filepath.Join(t.TempDir(), "d"), AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	ctx := context.Background()
	if _, err := c.Install(ctx, `class Talk extends ComicSource {
  name = 'Talk'; key = 'talk'; version = '1'
  search = {load: async () => { console.log('hello from the source'); return {comics: [], maxPage: 1} }}
}`, "file", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Search(ctx, "talk", "x", nil, 1); err != nil {
		t.Fatal(err)
	}
	lines := c.LogLines()
	if len(lines) == 0 {
		t.Fatal("the buffer is empty")
	}
	found := false
	for _, l := range lines {
		if l.Key == "talk" && strings.Contains(l.Content, "hello from the source") {
			found = true
		}
	}
	if !found {
		t.Fatalf("the line did not carry its source: %+v", lines)
	}
}

// The panel must not grow without limit: a source that logs in a loop would
// otherwise fill memory for as long as the app runs.
func TestSourceLogBufferIsBounded(t *testing.T) {
	c, err := core.Open(core.Options{DataDir: filepath.Join(t.TempDir(), "d"), AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	if _, err := c.Install(context.Background(), `class Loud extends ComicSource {
  name = 'Loud'; key = 'loud'; version = '1'
  search = {load: async () => {
    for (let i = 0; i < 500; i++) console.log('line ' + i)
    return {comics: [], maxPage: 1}
  }}
}`, "file", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Search(context.Background(), "loud", "x", nil, 1); err != nil {
		t.Fatal(err)
	}
	lines := c.LogLines()
	if len(lines) != core.MaxLogLines() {
		t.Fatalf("kept %d lines, want %d", len(lines), core.MaxLogLines())
	}
	// The newest lines are the ones kept: the last message has to be there.
	if last := lines[len(lines)-1].Content; !strings.Contains(last, "line 499") {
		t.Errorf("the newest line was dropped: %q", last)
	}
	c.ClearLogs()
	if got := len(c.LogLines()); got != 0 {
		t.Errorf("after clearing: %d lines", got)
	}
}

// One row per line, with the source named, so a pasted log is readable.
func TestLogLineFormatting(t *testing.T) {
	line := core.LogLine{Key: "wnacg", Level: "error", Title: "JS Engine", Content: "a\nb"}
	got := formatLogLine(line)
	if !strings.Contains(got, "wnacg") || !strings.Contains(got, "a ⏎ b") {
		t.Errorf("formatLogLine = %q", got)
	}
	if !strings.Contains(sourceLogText([]core.LogLine{line}), "wnacg") {
		t.Error("sourceLogText dropped the source")
	}
}

// A source that fails without logging leaves the panel empty, which is exactly
// when a user opens it. Core's own failures are recorded with the call that
// failed, so the panel has the one line that names the cause.
func TestFailedCallsAreLoggedWithTheirSource(t *testing.T) {
	c, err := core.Open(core.Options{DataDir: filepath.Join(t.TempDir(), "d"), AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	ctx := context.Background()
	if _, err := c.Install(ctx, `class Broken extends ComicSource {
  name = 'Broken'; key = 'broken'; version = '1'
  search = {load: async () => { throw new Error('the site answered with a login page') }}
}`, "file", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Search(ctx, "broken", "x", nil, 1); err == nil {
		t.Fatal("the search was supposed to fail")
	}
	lines := c.LogLines()
	if len(lines) == 0 {
		t.Fatal("a failed call left no line behind")
	}
	last := lines[len(lines)-1]
	if last.Key != "broken" || !strings.Contains(last.Content, "search") {
		t.Errorf("the line does not name the source and the call: %+v", last)
	}
	if !strings.Contains(last.Content, "login page") {
		t.Errorf("the line lost the cause: %q", last.Content)
	}
}
