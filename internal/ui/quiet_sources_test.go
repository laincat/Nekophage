package ui

import "testing"

// A merged search has to distinguish two ways of contributing nothing: a source
// that found no results, and a source that errored. Reporting the second as the
// first hides a broken source behind a quiet one.
func TestQuietSourcesSeparatesErrorsFromNoResults(t *testing.T) {
	cases := []struct {
		empty, broken int
		want          string
	}{
		{0, 0, ""},
		{2, 0, ",其中 2 个源没有结果"},
		{0, 1, ",其中 1 个源出错"},
		{3, 2, ",其中 3 个源没有结果、2 个源出错"},
	}
	for _, c := range cases {
		if got := quietSources(c.empty, c.broken); got != c.want {
			t.Errorf("quietSources(%d, %d) = %q, want %q", c.empty, c.broken, got, c.want)
		}
	}
}
