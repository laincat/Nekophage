package library

import (
	"errors"
	"testing"
)

// Core reports a taken key inside its error text and nothing else carries it, so
// this parsing is load-bearing: get it wrong and an update silently becomes a
// failed install.
func TestKeyOf(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"the usual shape", errors.New("plugin: already exists: copy_manga"), "copy_manga"},
		{"no separator", errors.New("plugin: broken"), "broken"},
		{"trailing spaces", errors.New("plugin: exists:  spaced  "), "spaced"},
		{"last segment wins", errors.New("plugin: exists: a: b"), "b"},
		{"empty message", errors.New(""), ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := keyOf(c.err); got != c.want {
				t.Fatalf("keyOf(%q) = %q; want %q", c.err, got, c.want)
			}
		})
	}
}
