package ui

import "testing"

func TestSourceLinksOnlyOpenAbsoluteHTTP(t *testing.T) {
	var opened []string
	handler := &JSHandler{openURL: func(raw string) error { opened = append(opened, raw); return nil }}
	for _, raw := range []string{"file:///tmp/test", "javascript:void(0)", "//example.test", "/chapter", "https:///chapter", "https://user@example.test"} {
		handler.LaunchURL(raw)
	}
	if len(opened) != 0 {
		t.Fatalf("invalid links opened: %v", opened)
	}
	for _, raw := range []string{"https://example.test/chapter", "http://example.test/"} {
		handler.LaunchURL(raw)
	}
	if len(opened) != 2 {
		t.Fatalf("links opened: %v", opened)
	}
}
