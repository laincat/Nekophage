package main

// Drives the browser-login window end to end against a local page, so the whole
// chain runs without a person: window, WebView, polling, the source's own
// checkStatus, the cookie import and onLoginSuccess.
//
//	go run ./cmd/loginprobe
//
// The page logs itself in after a moment, which is what a real site does when the
// user finishes typing; nothing here replaces the human, it only proves the client
// moves when the source says the login happened.

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/egoist/mygo"
	"github.com/laincat/nekophage/internal/browser"
	"github.com/laincat/nekophage/internal/core"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<!doctype html><title>Sign in</title>
<script>setTimeout(function(){location.href="/done"},1200)</script>sign in`)
	})
	mux.HandleFunc("/done", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "sid", Value: "abc123", Path: "/"})
		fmt.Fprint(w, `<!doctype html><title>Logged in</title>ok`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dir, err := os.MkdirTemp("", "loginprobe")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	c, err := core.Open(core.Options{DataDir: dir, AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		panic(err)
	}
	defer c.Close()

	src := `class Probe extends ComicSource {
    name = "Probe"; key = "probe"; version = "1.0.0"
    init() { this.saveData("done", false) }
    account = {
        loginWithWebview: {
            url: "` + srv.URL + `/login",
            checkStatus: (url, title) => title === "Logged in",
            onLoginSuccess: async () => {
                const cs = await Network.getCookies("` + srv.URL + `/")
                this.saveData("sawCookie", cs.length > 0)
            },
        },
    }
    search = { load: async () => ({comics: [], maxPage: 1}) }
}`
	ctx := context.Background()
	rec, err := c.Install(ctx, src, "file", "")
	if err != nil {
		fmt.Println("FAIL install:", err)
		os.Exit(1)
	}

	// The app hands the login window Core's own jar; a second jar would collect
	// the cookies where no source ever looks for them.
	jar := c.Jar()
	login := browser.NewLoginWindow(c, jar, rec.Key, core.ComicPartition(rec.Key))

	// App.Run owns the main goroutine; the work starts once the toolkit is ready
	// and reports through the channel after Run returns.
	result := make(chan error, 1)
	mygo.App.WhenReady(func() {
		go func() {
			runCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
			defer cancel()
			_, err := login.Run(runCtx)
			result <- err
			mygo.App.Quit()
		}()
	})
	if err := mygo.App.Run(); err != nil {
		fmt.Println("FAIL app:", err)
		os.Exit(1)
	}
	if err := <-result; err != nil {
		fmt.Println("FAIL run:", err)
		os.Exit(1)
	}
	got := jar.Get(core.ComicPartition(rec.Key), srv.URL+"/")
	if len(got) == 0 {
		fmt.Println("FAIL the partition has no cookie from the login page")
		os.Exit(1)
	}
	raw, ok, err := c.Store().LoadData(rec.Key, "sawCookie")
	if err != nil || !ok {
		fmt.Println("FAIL onLoginSuccess did not run:", err)
		os.Exit(1)
	}
	fmt.Printf("PASS login window: %d cookie(s) kept, onLoginSuccess saw them (%s)\n", len(got), raw)
}
