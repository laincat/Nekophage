// readercheck opens the reader on a real chapter of a real source and saves a
// screenshot of the window, so that "the pages render" is checked rather than
// assumed. It is a development tool, not part of the app.
//
//	go run ./cmd/readercheck -script /path/to/copy_manga.js -query 海贼王 -out shot.png
//
// MyGo needs the main goroutine for its event loop, so this is a program and
// not a test.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/reader"
	"github.com/laincat/nekophage/internal/sourceapi"
)

func main() {
	script := flag.String("script", "", "source script to install")
	query := flag.String("query", "", "search keyword")
	out := flag.String("out", "reader.png", "where to write the screenshot")
	wait := flag.Duration("wait", 12*time.Second, "how long the page may take to load")
	scrollCheck := flag.Bool("scroll", false, "check that reopening the chapter returns to the same place")
	layoutCheck := flag.Bool("layout", false, "check that two pages side by side, right column first, works")
	scan := flag.Bool("scan", false, "list the first chapters with their page counts")
	progressCheck := flag.Bool("progress", false, "check that the page reports its reading position")
	servePort := flag.Int("serve", 0, "also serve one PNG on this port, for a source whose chapter points at loopback")
	flag.Parse()
	if *script == "" || *query == "" {
		flag.Usage()
		os.Exit(2)
	}

	// A source can hand back an image address on loopback, which is how a chapter
	// can have real pages with no site: the iOS interface tests use this, and their
	// fixture points at 127.0.0.1:18731. Serving here makes the same trick available
	// on the desktop side.
	if *servePort > 0 {
		stop, err := serveOnePage(*servePort)
		if err != nil {
			log.Fatalf("serve: %v", err)
		}
		defer stop()
	}

	body, err := os.ReadFile(*script)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	c, err := core.Open(core.Options{DataDir: filepath.Join(os.TempDir(), "readercheck"), AppVersion: sourceapi.Level, Locale: "zh_CN"})
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	rec, err := c.Install(ctx, string(body), *script, "")
	if err != nil {
		log.Fatalf("install: %v", err)
	}
	options, err := c.DefaultOptions(ctx, rec.Key)
	if err != nil {
		log.Fatalf("search options: %v", err)
	}
	page, err := c.Search(ctx, rec.Key, *query, options, 1)
	if err != nil || len(page.Comics) == 0 {
		log.Fatalf("search: %v, %d comics", err, len(page.Comics))
	}
	pick := page.Comics[0]
	d, err := c.Details(ctx, rec.Key, pick.ID)
	if err != nil {
		log.Fatalf("details: %v", err)
	}
	// Some chapters carry no pages; take the first that has some.
	var chapterID string
	var images []string
	for i, ch := range d.Chapters {
		if i >= 10 {
			break
		}
		got, err := c.ChapterImages(ctx, rec.Key, pick.ID, ch.ID)
		if *scan {
			log.Printf("scan %d %q: %d pages (%v)", i, ch.Title, len(got), err)
		}
		if err != nil || len(got) == 0 {
			continue
		}
		chapterID, images = ch.ID, got
		log.Printf("chapter %d %q: %d pages", i, ch.Title, len(got))
		break
	}
	if len(images) == 0 {
		log.Fatal("no chapter of the first comic had pages")
	}
	log.Printf("%s: %d pages", d.Title, len(images))

	if err := mygo.Protocol.Handle(reader.Scheme, reader.New(c)); err != nil {
		log.Fatal(err)
	}
	target := reader.ReaderURL(rec.Key, pick.ID, chapterID, d.Title)

	var (
		checkErr error
		web      *mygo.WebView
	)
	mygo.App.WhenReady(func() {
		// A web view shows inside native UI: the window's content places it.
		win := mygo.NewWindow(mygo.WindowOptions{Title: "reader", Width: 900, Height: 1000,
			Content: ui.View(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					if web != nil {
						ui.WebView(c, web).Grow(1)
					} else {
						ui.Text(c, "正在加载…").Padding(16)
					}
				})
			})})
		view, err := win.NewWebView(mygo.WebViewOptions{URL: target})
		if err != nil {
			checkErr = err
			mygo.App.Quit()
			return
		}
		web = view
		win.Invalidate()
		go func() {
			// The page reports its own state: a window capture shows only the
			// native layer, not what the web engine painted, so the page is
			// asked whether its images decoded.
			time.Sleep(*wait)
			report := func(js string) any {
				v, err := view.Page().Eval(js)
				if err != nil {
					log.Printf("eval %q: %v", js, err)
					return nil
				}
				return v
			}
			title := report("document.title")
			pages := report("document.getElementById('pages') ? document.getElementById('pages').children.length : -1")
			total := report("document.images.length")
			loaded := report("Array.from(document.images).filter(i => i.complete && i.naturalWidth > 0).length")
			first := report("document.images.length && document.images[0].naturalWidth ? document.images[0].naturalWidth + 'x' + document.images[0].naturalHeight + ' ' + document.images[0].currentSrc.slice(0, 60) : ''")
			log.Printf("page title=%v figures=%v images=%v decoded=%v first=%v", title, pages, total, loaded, first)
			if data, err := win.CapturePage(); err == nil {
				if err := os.WriteFile(*out, data, 0o644); err == nil {
					log.Printf("window capture: %s (%d bytes; the web layer is not in it)", *out, len(data))
				}
			}
			checkErr = nil
			if n, ok := loaded.(float64); ok && n > 0 {
				log.Printf("the reader rendered %v of %v pages", loaded, total)
			} else {
				checkErr = fmt.Errorf("no image decoded: images=%v decoded=%v", total, loaded)
			}
			if *progressCheck && checkErr == nil {
				// Scroll, then ask the page what it reported.
				_, _ = view.Page().Eval("scrollTo(0, (document.body.scrollHeight - innerHeight) * 0.55); 0")
				time.Sleep(2 * time.Second)
				raw, err := view.Page().Eval(`(() => {
					const xhr = new XMLHttpRequest();
					xhr.open('GET', 'progress?' + new URLSearchParams({
						source: 'probe', comic: 'probe', chapter: 'probe',
						page: '3', pages: '10', percent: '55'}), false);
					let status = -1, body = '';
					try { xhr.send(); status = xhr.status; body = xhr.responseText; } catch (e) { body = String(e); }
					const key = Object.keys(localStorage).filter(k => k.endsWith(':pct'));
					return JSON.stringify({y: Math.round(scrollY), h: document.body.scrollHeight,
						pctKeys: key, pct: key.length ? localStorage.getItem(key[0]) : null,
						status, body});
				})()`)
				if err != nil {
					checkErr = fmt.Errorf("progress: %w", err)
				} else {
					log.Printf("progress: %v", raw)
				}
			}
			if *layoutCheck && checkErr == nil {
				// Two pages side by side, right column first: the page's own
				// buttons are clicked, then the boxes are measured.
				_, _ = view.Page().Eval("document.getElementById('spread').click(); document.getElementById('dir').click(); 0")
				time.Sleep(600 * time.Millisecond)
				raw, err := view.Page().Eval(`(() => {
					const f = [...document.querySelectorAll('#pages figure')];
					if (f.length < 2) return 'few figures';
					const box = (e) => { const r = e.getBoundingClientRect();
						return {x: Math.round(r.x), y: Math.round(r.y), w: Math.round(r.width)}; };
					return JSON.stringify({
						cls: document.getElementById('pages').className,
						spreadLabel: document.getElementById('spread').textContent,
						dirLabel: document.getElementById('dir').textContent,
						columns: getComputedStyle(document.getElementById('pages')).gridTemplateColumns,
						first: box(f[0]), second: box(f[1]), wide: f[0].className,
					});
				})()`)
				if err != nil {
					checkErr = fmt.Errorf("layout: %w", err)
				} else {
					log.Printf("layout: %v", raw)
					if text, ok := raw.(string); !ok || !strings.Contains(text, `"spread rtl"`) {
						checkErr = fmt.Errorf("two columns did not apply: %v", raw)
					}
				}
				_, _ = view.Page().Eval("document.getElementById('spread').click(); document.getElementById('dir').click(); 0")
			}
			if *scrollCheck && checkErr == nil {
				// Scrolling is remembered per chapter: move, reopen, look.
				_, _ = view.Page().Eval("scrollTo(0, 1200); 0")
				time.Sleep(1 * time.Second)
				if err := view.Page().LoadURL(target); err != nil {
					checkErr = fmt.Errorf("reload: %w", err)
				} else {
					time.Sleep(*wait)
					y, err := view.Page().Eval("scrollY")
					log.Printf("after reopening the chapter, scrollY=%v (%v)", y, err)
					if v, ok := y.(float64); !ok || v < 800 {
						checkErr = fmt.Errorf("the place was not restored: scrollY=%v", y)
					}
				}
			}
			mygo.App.Quit()
		}()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
	if checkErr != nil {
		log.Fatal(checkErr)
	}
	fmt.Println("done")
}

// serveOnePage answers every request with a small PNG on the given loopback port.
//
// The picture is drawn rather than embedded as a blob: a hand-typed base64 that
// fails to decode looks exactly like a client that cannot render.
func serveOnePage(port int) (func(), error) {
	const side = 8
	img := image.NewRGBA(image.Rect(0, 0, side, side))
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			img.Set(x, y, color.RGBA{R: 0x20, G: 0x90, B: 0x90, A: 0xFF})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	pages := buf.Bytes()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", itoa(len(pages)))
		_, _ = w.Write(pages)
	})
	srv := &http.Server{Addr: "127.0.0.1:" + itoa(port), Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return nil, err
	}
	go func() { _ = srv.Serve(ln) }()
	log.Printf("serving one page on http://%s/page.png", srv.Addr)
	return func() { _ = srv.Close() }, nil
}

func itoa(n int) string { return strconv.Itoa(n) }
