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
