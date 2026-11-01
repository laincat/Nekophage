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
	"github.com/laincat/nekophage/internal/browser"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/veneracat-core/host"
)

func main() {
	mode := flag.String("mode", "clear", "clear or dismiss")
	flag.Parse()
	switch *mode {
	case "clear", "dismiss":
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

	partition := core.ComicPartition("probe")
	verifier := browser.NewVerifier(c.Jar())
	ch := &host.Challenge{
		Kind:      host.ChallengeCloudflare,
		URL:       srv.URL + "/challenge",
		Partition: partition,
		Done: host.DoneRule{
			AbsentMarkers:   []string{"just-a-moment-checking"},
			RequiredCookies: []string{"cf_clearance"},
		},
	}

	result := make(chan error, 1)
	var clearance *host.Clearance
	mygo.App.WhenReady(func() {
		go func() {
			bound := 40 * time.Second
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
	got := c.Jar().Get(partition, srv.URL+"/")

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
