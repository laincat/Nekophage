package ui

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/laincat/nekophage/internal/cookies"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/veneracat-core/backup"
)

// A restored cookie has to be one the jar would actually send. An expired row is
// not a sign-in: writing it makes the import summary claim a session that no
// longer exists.
func TestToNetCookiesSkipsExpired(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	ms := func(offset time.Duration) *int64 {
		v := now.Add(offset).UnixMilli()
		return &v
	}
	rows := []backup.Cookie{
		{Name: "old", Value: "v", Domain: "wnacg.com", Expires: ms(-time.Hour)},
		{Name: "live", Value: "v", Domain: "wnacg.com", Expires: ms(time.Hour)},
		{Name: "session", Value: "v", Domain: "wnacg.com"},
		{Name: "zero", Value: "v", Domain: "wnacg.com", Expires: new(int64)},
	}
	got := toNetCookies(rows, now)
	if len(got) != 3 {
		t.Fatalf("cookies = %+v, want the live one, the session one and the zero-expiry one", got)
	}
	for _, c := range got {
		if c.Name == "old" {
			t.Fatal("an expired cookie was restored")
		}
	}
	if got[0].Expires.IsZero() {
		t.Errorf("the live cookie lost its expiry: %+v", got[0])
	}
}

// The import writes cookies only into partitions whose source is installed: a
// source whose script failed has nothing that would ever send them.
func TestRestoreCookiesOnlyWritesInstalledPartitions(t *testing.T) {
	// The jar matches on the real clock, so the cookie has to be live against
	// it; only the import decision is given a fixed time.
	now := time.Now()
	live := now.Add(time.Hour).UnixMilli()
	plan := &backup.Plan{
		Cookies: []backup.Cookie{
			{Name: "sid", Value: "v", Domain: ".wnacg.com", Expires: &live},
		},
		CookieTargets: []string{"copy_manga", "mangadex"},
	}
	dir := t.TempDir()
	jar, err := cookies.Open(filepath.Join(dir, "cookies.json"))
	if err != nil {
		t.Fatal(err)
	}

	// Only copy_manga installed: the other target must be skipped.
	written := restoreCookies(plan, jar, []string{"copy_manga"}, now)
	if written != 1 {
		t.Fatalf("written = %d, want 1", written)
	}
	if got := jar.Get(core.ComicPartition("copy_manga"), "https://wnacg.com/"); len(got) != 1 || got[0].Value != "v" {
		t.Errorf("copy_manga cookies = %+v", got)
	}
	if got := jar.Get(core.ComicPartition("mangadex"), "https://wnacg.com/"); len(got) != 0 {
		t.Errorf("mangadex got cookies for a source that is not installed: %+v", got)
	}
}

func TestRestoreCookiesWithoutTargetsWritesNothing(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	plan := &backup.Plan{
		Cookies: []backup.Cookie{{Name: "sid", Value: "v", Domain: ".wnacg.com"}},
	}
	jar, err := cookies.Open(filepath.Join(t.TempDir(), "cookies.json"))
	if err != nil {
		t.Fatal(err)
	}
	if written := restoreCookies(plan, jar, []string{"copy_manga"}, now); written != 0 {
		t.Fatalf("written = %d, want 0", written)
	}
	if got := jar.Get(core.ComicPartition("copy_manga"), "https://wnacg.com/"); len(got) != 0 {
		t.Errorf("cookies = %+v, want none", got)
	}
}

// Restored cookies have to land in the partition requests use. The transport and
// the challenge flow name "comic:<key>"; a restore that wrote the bare key stored
// cookies nothing would ever send, and a test making the same assumption passed.
func TestRestoreCookiesUsesTheRequestPartition(t *testing.T) {
	now := time.Now()
	plan := &backup.Plan{
		Cookies:       []backup.Cookie{{Name: "sid", Value: "v", Domain: ".wnacg.com"}},
		CookieTargets: []string{"copy_manga"},
	}
	jar, err := cookies.Open(filepath.Join(t.TempDir(), "cookies.json"))
	if err != nil {
		t.Fatal(err)
	}
	restoreCookies(plan, jar, []string{"copy_manga"}, now)

	if got := jar.Get(core.ComicPartition("copy_manga"), "https://wnacg.com/"); len(got) != 1 {
		t.Errorf("the source's partition has %d cookies, want 1", len(got))
	}
	if got := jar.Get("copy_manga", "https://wnacg.com/"); len(got) != 0 {
		t.Errorf("the bare key was written too: %+v", got)
	}
}
