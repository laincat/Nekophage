// verifycheck drives the verification window, and the whole verify-and-retry
// flow Core runs, without needing a real Cloudflare challenge.
//
//	go run ./cmd/verifycheck -local   # the window alone, against a page that passes itself
//	go run ./cmd/verifycheck -e2e     # Core -> 403 challenge -> WebView -> retry -> 200
//
// -url opens a real page instead, which needs a challenge the user can solve.
// MyGo needs the main goroutine for its event loop, so this is a program and
// not a test.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/laincat/nekophage/internal/browser"
	"github.com/laincat/nekophage/internal/cookies"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/veneracat-core/host"
)

const (
	// marker sits in the challenge page until the browser passes it.
	marker = "CHALLENGE_STILL_RUNNING"
	// passed is what the site answers once clearance is in place.
	passed = "PASSED-OK"
)

func main() {
	target := flag.String("url", "", "a real challenge page to open")
	cookie := flag.String("cookie", "cf_clearance", "cookie the page must set")
	absent := flag.String("marker", marker, "text that must be gone when the page is passed")
	local := flag.Bool("local", false, "check the window against a page that passes itself")
	e2e := flag.Bool("e2e", false, "check Core's whole challenge flow against a local site")
	timeout := flag.Duration("timeout", 90*time.Second, "how long to wait")
	flag.Parse()

	switch {
	case *e2e:
		runE2E(*timeout)
	case *local || *target == "":
		addr, stop, err := serveLocal()
		if err != nil {
			log.Fatal(err)
		}
		defer stop()
		check(jar(), "http://"+addr+"/", *absent, *cookie, *timeout)
	default:
		if !strings.HasPrefix(*target, "http") {
			log.Fatalf("not an http(s) URL: %s", *target)
		}
		check(jar(), *target, *absent, *cookie, *timeout)
	}
}

func jar() *cookies.Jar {
	j, err := cookies.Open(filepath.Join(os.TempDir(), "verifycheck-cookies.json"))
	if err != nil {
		log.Fatal(err)
	}
	return j
}

// check runs the verifier alone and prints what it found.
func check(j *cookies.Jar, pageURL, absent, cookie string, timeout time.Duration) {
	verifier := browser.NewVerifier(j)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var (
		result *host.Clearance
		verr   error
		done   = make(chan struct{})
	)
	mygo.App.WhenReady(func() {
		go func() {
			defer close(done)
			start := time.Now()
			result, verr = verifier.Verify(ctx, &host.Challenge{
				Kind:      host.ChallengeCloudflare,
				URL:       pageURL,
				Partition: "verifycheck",
				Done:      host.DoneRule{AbsentMarkers: []string{absent}, RequiredCookies: []string{cookie}},
			})
			log.Printf("verify took %s", time.Since(start).Round(time.Millisecond))
			mygo.App.Quit()
		}()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
	<-done
	if verr != nil {
		log.Fatalf("verify failed: %v", verr)
	}
	log.Printf("user agent: %s", result.UserAgent)
	for _, c := range result.Cookies {
		log.Printf("cookie: %s=%s domain=%s path=%s", c.Name, c.Value, c.Domain, c.Path)
	}
	if sent := j.Get("verifycheck", pageURL); len(sent) == 0 {
		log.Fatal("the clearance never reached the jar")
	}
	fmt.Println("ok")
}

// serveLocal starts a page that looks like a challenge: the marker is in the
// HTML at first, then a script removes it and sets the clearance cookie.
func serveLocal() (string, func(), error) {
	page := `<!doctype html><meta charset="utf-8"><title>challenge</title>
<div id="wait">` + marker + `</div>
<script>
setTimeout(() => {
  document.getElementById('wait').remove();
  document.cookie = 'cf_clearance=local-ok; path=/';
  document.title = 'passed';
}, 800);
</script>`
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(page))
	})
	return start(mux)
}

// runE2E drives Core: a request is challenged, Core asks for verification,
// the window passes, and the retry with the WebView's User-Agent gets in.
func runE2E(timeout time.Duration) {
	addr, stop, err := serveCloudflareLike()
	if err != nil {
		log.Fatal(err)
	}
	defer stop()
	site := "http://" + addr
	log.Printf("local site: %s", site)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	dataDir := filepath.Join(os.TempDir(), "verifycheck-e2e")
	_ = os.RemoveAll(dataDir)

	// The source asks for the challenged page and returns what it got.
	script := "class C extends ComicSource { name = 'C'; key = 'c'; version = '1'\n" +
		"  search = { load: async () => { const r = await Network.get(" + quote(site+"/challenged") + ", {})\n" +
		"    return {comics: [], note: r.status + ':' + r.body} } } }"

	var (
		note string
		ferr error
		sent int
		done = make(chan struct{})
	)
	mygo.App.WhenReady(func() {
		go func() {
			defer close(done)
			// The verifier must import clearance into the jar the transport
			// reads, so Core builds it: this is the wiring main.go uses.
			c, err := core.Open(core.Options{DataDir: dataDir, AppVersion: "1.6.0", Locale: "zh_CN",
				VerifierFactory: func(j *cookies.Jar) host.Verifier { return browser.NewVerifier(j) }})
			if err != nil {
				ferr = err
				mygo.App.Quit()
				return
			}
			defer c.Close()
			defer func() { sent = len(c.Jar().Get("comic:c", site+"/challenged")) }()
			if _, err := c.Install(ctx, script, "verifycheck", ""); err != nil {
				ferr = err
				mygo.App.Quit()
				return
			}
			start := time.Now()
			out, err := c.Call(ctx, "c", "search.load")
			log.Printf("call took %s", time.Since(start).Round(time.Millisecond))
			if err != nil {
				ferr = err
			} else {
				note = string(out)
			}
			mygo.App.Quit()
		}()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
	<-done
	if ferr != nil {
		log.Fatalf("core call failed: %v", ferr)
	}
	log.Printf("source got: %s", note)
	if !strings.Contains(note, passed) {
		log.Fatalf("the retry did not get in: %s", note)
	}
	if sent == 0 {
		log.Fatal("clearance is not in the source's partition")
	}
	log.Printf("clearance in the source's partition: %d cookie(s)", sent)
	fmt.Println("ok")
}

// serveCloudflareLike answers like a Cloudflare-protected site: a browser is
// served the challenge page, anything else gets 403 with the markers Core
// detects, and the clearance cookie lets a request through.
func serveCloudflareLike() (string, func(), error) {
	challengePage := `<!doctype html><meta charset="utf-8"><title>Just a moment</title>
<div id="wait">` + marker + `</div>
<script>
// What a managed challenge does once it is satisfied.
setTimeout(() => {
  document.getElementById('wait').remove();
  document.cookie = 'cf_clearance=clear; path=/';
}, 600);
</script>`
	blocked := `<!doctype html><html><head><title>Just a moment...</title>
<script>window._cf_chl_opt={cvId: '3'};</script></head>
<body><form id="challenge-form"></form></body></html>`

	mux := http.NewServeMux()
	mux.HandleFunc("/challenged", func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("NKDEBUG") != "" {
			log.Printf("request ua=%q cookie=%q", r.Header.Get("User-Agent"), r.Header.Get("Cookie"))
		}
		// Clearance in place: the site lets the request through, whoever asks.
		if c, err := r.Cookie("cf_clearance"); err == nil && c.Value != "" {
			_, _ = w.Write([]byte(passed))
			return
		}
		// Without clearance a browser is served the page it can pass;
		// WebView2 identifies itself as Edge, our client sends a plain
		// Chrome agent.
		if strings.Contains(r.Header.Get("User-Agent"), "Edg/") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(challengePage))
			return
		}
		w.Header().Set("Server", "cloudflare")
		w.Header().Set("Cf-Mitigated", "challenge")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(blocked))
	})
	return start(mux)
}

func start(mux *http.ServeMux) (string, func(), error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	return ln.Addr().String(), func() { _ = srv.Close() }, nil
}

func quote(s string) string { return `"` + s + `"` }
