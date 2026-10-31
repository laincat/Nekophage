package ui

import (
	"testing"
	"time"

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
