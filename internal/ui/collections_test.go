package ui

import (
	"strings"
	"testing"

	"github.com/laincat/nekophage/internal/collections"
)

// An entry row has to say why it cannot be installed here, not just that it
// cannot: the missing capability is the thing that explains the refusal.
func TestEntryLineExplainsIncompatibility(t *testing.T) {
	ok := collections.Entry{}
	ok.Kind, ok.Status, ok.Key, ok.Compatible = "comic-source.js", "active", "demo", true
	line := entryLine(ok)
	for _, want := range []string{"comic-source.js", "active", "key demo"} {
		if !strings.Contains(line, want) {
			t.Errorf("line %q does not mention %q", line, want)
		}
	}
	if strings.Contains(line, "不支持") {
		t.Errorf("a compatible entry says it is unsupported: %q", line)
	}

	blocked := collections.Entry{}
	blocked.Kind, blocked.Status, blocked.Compatible = "video-source.js", "active", false
	blocked.Missing = []string{"video-runtime"}
	line = entryLine(blocked)
	if !strings.Contains(line, "不支持") || !strings.Contains(line, "video-runtime") {
		t.Errorf("line %q does not explain the refusal", line)
	}
}
