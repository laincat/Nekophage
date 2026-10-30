package main

import (
	"runtime"
	"strings"
	"testing"
)

// The report has to name the build a user is running: "it does not start" is far
// easier to act on with the commit and the runtime in hand.
func TestBuildInfoNamesTheBuild(t *testing.T) {
	got := buildInfo()
	for _, want := range []string{version, sourceVersion, runtime.Version(), runtime.GOOS, runtime.GOARCH, "commit "} {
		if !strings.Contains(got, want) {
			t.Errorf("buildInfo() = %q, missing %q", got, want)
		}
	}
}
