package browser

import (
	"net/http"
	"testing"
)

// A login page also loads third parties, and their cookies are not part of the
// login. Storing them would put another site's cookies in the source's partition.
func TestLoginCookiesForKeepsTheLoginSiteOnly(t *testing.T) {
	cs := []*http.Cookie{
		{Name: "session", Value: "a", Domain: "forums.e-hentai.org"},
		{Name: "shared", Value: "b", Domain: ".e-hentai.org"},
		{Name: "other", Value: "c", Domain: "analytics.test"},
		{Name: "hostonly", Value: "d"},
	}
	got := loginCookiesFor(cs, "https://forums.e-hentai.org/index.php?act=Login&CODE=00")
	names := make([]string, 0, len(got))
	for _, c := range got {
		names = append(names, c.Name)
	}
	if len(names) != 3 {
		t.Fatalf("kept %v, want the session, the parent-domain cookie and the host-only one", names)
	}
	for _, want := range []string{"session", "shared", "hostonly"} {
		found := false
		for _, name := range names {
			if name == want {
				found = true
			}
		}
		if !found {
			t.Errorf("%s was dropped", want)
		}
	}
	for _, name := range names {
		if name == "other" {
			t.Error("a third party's cookie was kept")
		}
	}
}

// A leading dot means the host and its subdomains, and nothing else.
func TestLoginCookiesForRespectsTheDotRule(t *testing.T) {
	cs := []*http.Cookie{
		{Name: "sub", Value: "1", Domain: ".nhentai.net"},
		{Name: "lookalike", Value: "2", Domain: ".notnhentai.net"},
	}
	got := loginCookiesFor(cs, "https://nhentai.net/login")
	if len(got) != 1 || got[0].Name != "sub" {
		t.Fatalf("kept %v, want only the cookie for nhentai.net", got)
	}
	// A host-only cookie for a sibling host is not the login site's.
	got = loginCookiesFor([]*http.Cookie{{Name: "x", Value: "1", Domain: "evilnhentai.net"}},
		"https://nhentai.net/login")
	if len(got) != 0 {
		t.Fatalf("kept %v for a lookalike host", got)
	}
}

// A page that is not a web address keeps nothing, rather than everything.
func TestLoginCookiesForRejectsNonWebURL(t *testing.T) {
	cs := []*http.Cookie{{Name: "a", Value: "1", Domain: "x.test"}}
	if got := loginCookiesFor(cs, "about:blank"); len(got) != 0 {
		t.Fatalf("kept %v for a non-web page", got)
	}
}

// A login can finish on another host than it started on: manga_dex goes from
// forums.mangadex.org back to mangadex.org. Both are the login site, so both
// sets of cookies belong to the source.
func TestLoginCookiesCoverTheStartingAndFinishingHost(t *testing.T) {
	cs := []*http.Cookie{
		{Name: "forum", Value: "1", Domain: "forums.mangadex.org"},
		{Name: "site", Value: "2", Domain: "mangadex.org"},
		{Name: "shared", Value: "3", Domain: ".mangadex.org"},
		{Name: "other", Value: "4", Domain: "analytics.test"},
	}
	start := loginCookiesFor(cs, "https://forums.mangadex.org/login/")
	end := loginCookiesFor(cs, "https://mangadex.org/")
	merged := start
	for _, c := range end {
		if !hasCookie(merged, c) {
			merged = append(merged, c)
		}
	}
	// forum, site and the cookie for the parent domain; not the third party.
	if len(merged) != 3 {
		t.Fatalf("kept %d cookie(s), want the forum's, the site's and the parent-domain one", len(merged))
	}
	for _, c := range merged {
		if c.Name == "other" {
			t.Error("a third party's cookie was kept")
		}
	}
	// The parent-domain cookie matches both hosts, and must be stored once.
	if len(start)+len(end) != len(merged)+1 {
		t.Errorf("start %d + end %d did not collapse one duplicate into %d", len(start), len(end), len(merged))
	}
}
