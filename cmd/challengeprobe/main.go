package main

// Drives the challenge window the way a Cloudflare page does, against a local
// server, so the whole path runs with nobody at the keyboard: a page that shows a
// marker and earns its clearance cookie on its own, the poll that decides it is
// over, the cookie import and the exact User-Agent the later requests must send.
//
//	go run ./cmd/challengeprobe
//
// The point is what a screenshot cannot show: the clearance cookie must reach the
// source's partition and the User-Agent must be the one the page actually used,
// because sites bind the two together.

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/laincat/nekophage/internal/browser"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/veneracat-core/host"
)

func main() {
	mode := flag.String("mode", "clear", "clear, dismiss or live")
	live := flag.String("url", "", "the real page to try in live mode")
	flag.Parse()
	switch *mode {
	case "clear", "dismiss", "live", "inspect":
	default:
		fmt.Println("FAIL unknown mode:", *mode)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/challenge", func(w http.ResponseWriter, r *http.Request) {
		// The marker is present until the page decides it has passed, which is
		// the shape a real challenge has.
		if *mode == "dismiss" {
			// A page that never clears itself, so the wait can only end by being
			// given up on.
			fmt.Fprint(w, `<!doctype html><title>Checking your browser</title>`+
				`<body>just-a-moment-checking</body>`)
			return
		}
		fmt.Fprint(w, `<!doctype html><title>Checking your browser</title>`+
			`<body>just-a-moment-checking`+
			`<script>setTimeout(function(){`+
			`document.cookie="cf_clearance=cleared; path=/";`+
			`document.body.innerHTML="welcome"},1500)</script></body>`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dir, err := os.MkdirTemp("", "challengeprobe")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	c, err := core.Open(core.Options{DataDir: dir, AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		panic(err)
	}
	defer c.Close()

	if *mode == "inspect" {
		// What the window actually shows, which decides whether a site has a
		// challenge to clear at all rather than one the client cannot judge.
		if *live == "" {
			fmt.Println("FAIL inspect mode needs -url")
			os.Exit(1)
		}
		report := make(chan string, 1)
		mygo.App.WhenReady(func() {
			go func() {
				win := mygo.NewWindow(mygo.WindowOptions{Title: "inspect", Width: 520, Height: 780,
					Content: ui.View(func(c *ui.Context) { ui.Text(c, "inspect") })})
				view, err := win.NewWebView(mygo.WebViewOptions{URL: *live})
				if err != nil {
					report <- "FAIL open: " + err.Error()
					mygo.App.Quit()
					return
				}
				win.Show()
				time.Sleep(9 * time.Second)
				page := view.Page()
				title, _ := page.Eval("document.title")
				html, _ := page.Eval("document.documentElement.outerHTML")
				body, _ := html.(string)
				short := body
				if len(short) > 160 {
					short = short[:160]
				}
				listed, _ := mygo.Cookies.List(mygo.CookieFilter{URL: *live})
				names := make([]string, 0, len(listed))
				for _, ck := range listed {
					names = append(names, ck.Name)
				}
				report <- fmt.Sprintf("title=%v cookies=%v html=%q", title, names, short)
				win.Close()
				mygo.App.Quit()
			}()
		})
		if err := mygo.App.Run(); err != nil {
			fmt.Println("FAIL app:", err)
			os.Exit(1)
		}
		fmt.Println("INSPECT", <-report)
		return
	}

	partition := core.ComicPartition("probe")
	page := srv.URL + "/challenge"
	done := host.DoneRule{
		AbsentMarkers:   []string{"just-a-moment-checking"},
		RequiredCookies: []string{"cf_clearance"},
	}
	if *mode == "live" {
		// A real interstitial: the markers a Cloudflare challenge page carries,
		// and the cookie it sets when it passes. Whether it passes at all is what
		// this mode is asking.
		if *live == "" {
			fmt.Println("FAIL live mode needs -url")
			os.Exit(1)
		}
		page = *live
		done = host.DoneRule{
			AbsentMarkers:   []string{"Just a moment", "Checking your browser", "cf-browser-verification", "__cf_chl_"},
			RequiredCookies: []string{"cf_clearance"},
		}
	}
	verifier := browser.NewVerifier(c.Jar())
	ch := &host.Challenge{
		Kind:      host.ChallengeCloudflare,
		URL:       page,
		Partition: partition,
		Done:      done,
	}

	result := make(chan error, 1)
	var clearance *host.Clearance
	mygo.App.WhenReady(func() {
		go func() {
			bound := 40 * time.Second
			if *mode == "live" {
				bound = 60 * time.Second
			}
			if *mode == "dismiss" {
				// The window is closed by the person giving up, which the probe
				// stands in for with the deadline.
				bound = 4 * time.Second
			}
			ctx, cancel := context.WithTimeout(context.Background(), bound)
			defer cancel()
			var err error
			clearance, err = verifier.Verify(ctx, ch)
			result <- err
			mygo.App.Quit()
		}()
	})
	if err := mygo.App.Run(); err != nil {
		fmt.Println("FAIL app:", err)
		os.Exit(1)
	}

	err = <-result
	target := srv.URL + "/"
	if *mode == "live" {
		target = *live
	}
	got := c.Jar().Get(partition, target)

	if *mode == "live" {
		if err != nil {
			fmt.Println("NOTE live page did not clear:", err)
			fmt.Println("NOTE cookies seen:", len(got))
			os.Exit(2)
		}
		fmt.Println("PASS live challenge cleared; cookies:", len(got), "ua:", clearance.UserAgent)
		return
	}

	if *mode == "dismiss" {
		if err == nil {
			fmt.Println("FAIL a closed challenge was reported as cleared")
			os.Exit(1)
		}
		if len(got) != 0 {
			fmt.Println("FAIL the clearance was stored although the window was closed:", len(got))
			os.Exit(1)
		}
		fmt.Println("PASS dismiss: nothing was stored and the wait ended with", err)
		return
	}

	if err != nil {
		fmt.Println("FAIL verify:", err)
		os.Exit(1)
	}
	if clearance == nil {
		fmt.Println("FAIL no clearance was returned")
		os.Exit(1)
	}
	if clearance.UserAgent == "" {
		fmt.Println("FAIL the clearance carries no User-Agent to send later")
		os.Exit(1)
	}
	var stored bool
	for _, ck := range got {
		if ck.Name == "cf_clearance" && ck.Value == "cleared" {
			stored = true
		}
	}
	if !stored {
		fmt.Println("FAIL the clearance cookie did not reach the partition:", got)
		os.Exit(1)
	}
	fmt.Println("PASS challenge: cleared, cookie in the partition, User-Agent", clearance.UserAgent)
}
