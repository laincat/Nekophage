package main

// Drives the browser-login window end to end against a local page, so the whole
// chain runs without a person: window, WebView, polling, the source's own
// checkStatus, the cookie import and onLoginSuccess.
//
//	go run ./cmd/loginprobe -mode ok|timeout|failcallback
//
// The page logs itself in after a moment, which is what a real site does when the
// user finishes typing. Modes:
//
//	ok            the ordinary path
//	timeout       a page the source never accepts, where the wait has to end
//	              without storing half a login
//	failcallback  a login whose onLoginSuccess throws, where the cookies are
//	              already the source's and the error still has to surface
//	latecookie    a page that agrees before its cookies exist, where the wait has
//	              to keep polling instead of giving up on the empty first look
//	close         a person closing the window, which must end the wait at once and
//	              keep nothing

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/laincat/nekophage/internal/browser"
	"github.com/laincat/nekophage/internal/core"
)

func main() {
	mode := flag.String("mode", "ok", "ok, timeout, failcallback, latecookie or close")
	flag.Parse()
	// An unknown mode would otherwise fall through to the ordinary path and report
	// a pass for a case nobody asked about.
	switch *mode {
	case "ok", "timeout", "failcallback", "latecookie", "close":
	default:
		fmt.Println("FAIL unknown mode:", *mode)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		// The timeout mode stays on this page: the source only accepts the title
		// "Logged in", so the client has to give up rather than decide for it.
		advance := `<script>setTimeout(function(){location.href="/done"},1200)</script>`
		// Both of these stay on a page the source never accepts: one is given up
		// by a deadline, the other by closing the window, and in both cases a
		// client that ignored the ending would hang here.
		if *mode == "timeout" || *mode == "close" {
			advance = ""
			// A pending login may already have cookies; they must not be imported.
			http.SetCookie(w, &http.Cookie{Name: "pending", Value: "secret", Path: "/"})
		}
		fmt.Fprintf(w, "<!doctype html><title>Sign in</title>%s sign in", advance)
	})
	mux.HandleFunc("/done", func(w http.ResponseWriter, r *http.Request) {
		if *mode == "latecookie" {
			// The title is already acceptable while the cookie is three seconds
			// behind it, which is the case the wait has to survive.
			fmt.Fprint(w, `<!doctype html><title>Logged in</title><script>`+
				`setTimeout(function(){document.cookie="sid=late; path=/"},3000)</script>ok`)
			return
		}
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

	callback := `onLoginSuccess: async () => {
                const cs = await Network.getCookies("` + srv.URL + `/")
                this.saveData("sawCookie", cs.length > 0)
            }`
	if *mode == "failcallback" {
		callback = `onLoginSuccess: async () => { throw new Error("probe callback refused") }`
	}
	src := `class Probe extends ComicSource {
    name = "Probe"; key = "probe"; version = "1.0.0"
    init() { this.saveData("done", false) }
    account = {
        loginWithWebview: {
            url: "` + srv.URL + `/login",
            checkStatus: (url, title) => title === "Logged in",
            ` + callback + `,
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
	if *mode == "close" {
		// Stand where the person sits: the page loads, then the window is closed
		// without answering. The wait has to end there.
		login.OnWindow = func(w *mygo.Window) {
			go func() {
				time.Sleep(3 * time.Second)
				w.Close()
			}()
		}
	}

	// App.Run owns the main goroutine; the work starts once the toolkit is ready
	// and reports through the channel after Run returns.
	result := make(chan error, 1)
	var elapsed time.Duration
	mygo.App.WhenReady(func() {
		go func() {
			bound := 90 * time.Second
			if *mode == "timeout" {
				bound = 8 * time.Second
			}
			if *mode == "close" {
				// Long enough that a wait which ignored the close would run into
				// it and report the wrong reason.
				bound = 60 * time.Second
			}
			runCtx, cancel := context.WithTimeout(ctx, bound)
			defer cancel()
			start := time.Now()
			_, err := login.Run(runCtx)
			elapsed = time.Since(start)
			result <- err
			mygo.App.Quit()
		}()
	})
	if err := mygo.App.Run(); err != nil {
		fmt.Println("FAIL app:", err)
		os.Exit(1)
	}

	runErr := <-result
	got := jar.Get(core.ComicPartition(rec.Key), srv.URL+"/")

	if *mode == "close" {
		if runErr == nil {
			fmt.Println("FAIL closing the window was reported as a finished login")
			os.Exit(1)
		}
		if len(got) != 0 {
			fmt.Println("FAIL half a login was kept after the window closed:", len(got))
			os.Exit(1)
		}
		if elapsed > 15*time.Second {
			fmt.Println("FAIL the wait outlived the window by", elapsed, "— it kept polling a closed page")
			os.Exit(1)
		}
		fmt.Println("PASS close: the wait ended in", elapsed.Round(time.Millisecond), "with", runErr)
		return
	}

	switch *mode {
	case "timeout":
		if !errors.Is(runErr, context.DeadlineExceeded) {
			fmt.Println("FAIL expected deadline exceeded, got:", runErr)
			os.Exit(1)
		}
		if len(got) != 0 {
			fmt.Println("FAIL cookies were stored for a login the source never accepted:", len(got))
			os.Exit(1)
		}
		fmt.Println("PASS timeout: the wait ended with an error and nothing was stored")
		return
	case "failcallback":
		if runErr == nil {
			fmt.Println("FAIL a throwing onLoginSuccess was reported as success")
			os.Exit(1)
		}
		if len(got) == 0 {
			fmt.Println("FAIL the cookies were dropped although they were already stored")
			os.Exit(1)
		}
		// The source's own reason, not a deadline: waiting the window out would
		// tell the user nothing and take a minute and a half.
		if !strings.Contains(runErr.Error(), "onLoginSuccess") {
			fmt.Println("FAIL the callback error was replaced by:", runErr)
			os.Exit(1)
		}
		fmt.Println("PASS failing callback surfaced with", len(got), "cookie(s) kept:", runErr)
		return
	}

	if runErr != nil {
		fmt.Println("FAIL run:", runErr)
		os.Exit(1)
	}
	if len(got) == 0 {
		fmt.Println("FAIL the partition has no cookie from the login page")
		os.Exit(1)
	}
	raw, ok, err := c.Store().LoadData(rec.Key, "sawCookie")
	if err != nil || !ok || string(raw) != "true" {
		fmt.Println("FAIL onLoginSuccess did not see the cookie:", string(raw), err)
		os.Exit(1)
	}
	fmt.Println("PASS login window:", len(got), "cookie(s) kept, onLoginSuccess saw them:", string(raw))
}
