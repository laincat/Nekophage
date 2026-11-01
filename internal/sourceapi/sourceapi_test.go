package sourceapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every official source states the level it needs, and a level below one of them
// is not a style problem: Core refuses the source and the client shows an empty
// list. That is what shipped once on the iOS side, which reported the app's own
// 0.1.0.
func TestLevelMeetsEveryOfficialSourceFloor(t *testing.T) {
	dir := os.Getenv("NK_OFFICIAL_SRC")
	if dir == "" {
		t.Skip("set NK_OFFICIAL_SRC to the official source directory to check them")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	checked := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".js") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		for _, line := range strings.Split(string(body), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "minAppVersion") {
				continue
			}
			_, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			want := strings.Trim(strings.TrimSpace(value), `"';`)
			if want == "" {
				continue
			}
			checked++
			if newer(want, Level) {
				t.Errorf("%s requires %s, this client reports %s", e.Name(), want, Level)
			}
		}
	}
	if checked == 0 {
		t.Skipf("no minAppVersion declarations under %s", dir)
	}
	t.Logf("checked %d declaration(s) against %s", checked, Level)
}

// newer reports whether a is a higher dotted version than b, which is the rule
// Core applies to minAppVersion.
func newer(a, b string) bool {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		var x, y int
		if i < len(as) {
			x = atoi(as[i])
		}
		if i < len(bs) {
			y = atoi(bs[i])
		}
		if x != y {
			return x > y
		}
	}
	return false
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}
