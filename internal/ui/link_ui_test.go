package ui

import "testing"

// What counts as a pasted address: the search box must not turn a keyword into a
// link lookup, nor a link into a keyword search.
func TestLooksLikeLink(t *testing.T) {
	links := []string{
		"https://nhentai.net/g/123456/",
		"http://a.example/x",
		"nhentai.net/g/123456",
		"www.a.example",
	}
	for _, q := range links {
		if !looksLikeLink(q) {
			t.Errorf("%q was not treated as a link", q)
		}
	}
	// A dot is not enough: a version number and a keyword with a space both stay
	// keywords, so the box does not spend a source call on them.
	keywords := []string{"海贼王", "one piece", "test", "12.5", "v1.2", ""}
	for _, q := range keywords {
		if looksLikeLink(q) {
			t.Errorf("%q was treated as a link", q)
		}
	}
}
