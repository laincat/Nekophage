package library

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/laincat/nekophage/internal/core"
)

// A source that needs an account must be recognised as such, and the way it
// wants to log in has to be read correctly: ehentai has no password login at
// all, only a webview and a cookie form, so a client that only knew about
// account.login would offer it nothing.
//
// Gated by NK_LIVE_JS like the other live tests.
func TestLiveAccountStyles(t *testing.T) {
	script := os.Getenv("NK_LIVE_JS")
	if script == "" {
		t.Skip("set NK_LIVE_JS to a source file")
	}
	body, err := os.ReadFile(script)
	if err != nil {
		t.Fatal(err)
	}
	c, err := core.Open(core.Options{DataDir: t.TempDir(), AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	rec, err := c.Install(ctx, string(body), script, "")
	if err != nil {
		t.Fatal(err)
	}
	styles := c.AccountSupport(ctx, rec.Key)
	if !styles.Password && !styles.Webview && !styles.Cookies {
		t.Skip("SKIP: source declares no account")
	}
	t.Logf("  styles: password=%v webview=%v cookies=%v logout=%v",
		styles.Password, styles.Webview, styles.Cookies, styles.Logout)
	if styles.Cookies {
		fields, err := c.AccountCookieFields(ctx, rec.Key)
		if err != nil {
			t.Fatalf("cookie fields: %v", err)
		}
		t.Logf("  cookie fields: %v", fields)
		if len(fields) == 0 {
			t.Error("a cookie login with no fields cannot be filled in")
		}
		// Empty values must be refused by the source, not accepted silently.
		ok, err := c.LoginWithCookies(ctx, rec.Key, make([]string, len(fields)))
		if err != nil {
			t.Fatalf("validate: %v", err)
		}
		if ok {
			t.Error("empty cookies were accepted")
		}
	}
	if styles.Password {
		// Bogus credentials on purpose: what matters is that the source's own
		// refusal comes back as text instead of being swallowed.
		_, err := c.LoginAccount(ctx, rec.Key, "not-a-user@example.invalid", "not-a-password")
		if err == nil {
			t.Error("a refused login must be an error")
		} else {
			t.Logf("  refused login: %v", err)
		}
	}
}
