package main

import "testing"

// A seed has to be the same bytes wherever it is built, or "the seed matches the
// tree" stops being a question a hash can answer.
func TestNewlinesAreNormalised(t *testing.T) {
	cases := map[string]string{
		"a\r\nb":     "a\nb",
		"a\nb":       "a\nb",
		"a\rb":       "a\nb",
		"a\r\nb\r\n": "a\nb\n",
		"\r\n\r\n":   "\n\n",
		"":           "",
	}
	for in, want := range cases {
		if got := normaliseNewlines(in); got != want {
			t.Errorf("normaliseNewlines(%q) = %q, want %q", in, got, want)
		}
	}
}
