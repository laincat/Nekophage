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
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/probesite"
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
	failFirst := flag.Int("fail", 0, "with -serve, answer this many requests with a server error first")
	retryCheck := flag.Bool("retry", false, "check that a page which failed says so and recovers when retried")
	flag.Parse()
	if *script == "" || *query == "" {
		flag.Usage()
		os.Exit(2)
	}
	if *failFirst > 0 && *servePort == 0 {
		log.Fatal("-fail needs -serve: it fails requests on the loopback server")
	}

	// A source can hand back an image address on loopback, which is how a chapter
	// can have real pages with no site: the iOS interface tests use this, and their
	// fixture points at 127.0.0.1:18731. Serving here makes the same trick available
	// on the desktop side.
	if *servePort > 0 {
		stop, err := probesite.ServeOnePageFailing(*servePort, *failFirst)
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
		// The page is read from a goroutine after a wait. If the app loop ends
		// first — a window that never opened, a startup that quit early — the
		// goroutine never runs and the tool used to print "done" anyway, which
		// reads as a pass. That is how one run of the suite reported nothing and
		// exited 0.
		checked atomic.Bool
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
		view, err := win.NewWebView(mygo.WebViewOptions{
			URL: target, Page: mygo.PageOptions{Session: mygo.NewEphemeralSession()},
		})
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
			checked.Store(true)
			// The checks below move and reload the page, which only means anything
			// once the reader itself is there. The window opens before the source
			// answers, so a scroll at a fixed wait can land on the empty document
			// that is about to be replaced — the reader then never sees it and the
			// place is never stored.
			ready := false
			for i := 0; i < 120 && !ready; i++ {
				v, err := view.Page().Eval("document.readyState === 'complete' && document.querySelectorAll('#pages figure').length")
				if err == nil {
					if n, ok := v.(float64); ok && n > 0 {
						ready = true
						break
					}
				}
				time.Sleep(250 * time.Millisecond)
			}
			if !ready {
				checkErr = fmt.Errorf("the reader page never reached a loaded state")
			}
			checkErr = nil
			if *retryCheck {
				// The decode assertion below reads the page before the retry, which is
				// empty by design here; the retry block reads it again afterwards.
			} else if n, ok := loaded.(float64); ok && n > 0 {
				log.Printf("the reader rendered %v of %v pages", loaded, total)
			} else {
				checkErr = fmt.Errorf("no image decoded: images=%v decoded=%v", total, loaded)
			}
			if *retryCheck {
				// The fixture failed on purpose: the first look has to show the reader
				// saying so, and the retry it offers has to bring the picture back.
				failed := report("document.querySelectorAll('.retry').length")
				log.Printf("pages reporting a failure: %v", failed)
				if n, ok := failed.(float64); !ok || n == 0 {
					checkErr = fmt.Errorf("a page that failed did not say so: .retry=%v", failed)
				} else {
					_, _ = view.Page().Eval("Array.from(document.querySelectorAll('.retry button')).forEach(b => b.click()); 0")
					time.Sleep(4 * time.Second)
					left := report("document.querySelectorAll('.retry').length")
					decoded := report("Array.from(document.images).filter(i => i.complete && i.naturalWidth > 0).length")
					log.Printf("after retrying: failures=%v decoded=%v", left, decoded)
					cleared := true
					if n, ok := left.(float64); !ok || n != 0 {
						checkErr = fmt.Errorf("the failure notice stayed after retrying: .retry=%v", left)
						cleared = false
					}
					if n, ok := decoded.(float64); !ok || n == 0 {
						checkErr = fmt.Errorf("the retry decoded nothing: decoded=%v", decoded)
					} else if cleared {
						log.Printf("the reader recovered the page it could not load")
						checkErr = nil
					}
				}
			}
			if *progressCheck && checkErr == nil {
				// Scroll, then ask the page what it reported.
				_, _ = view.Page().Eval("scrollTo(0, (document.body.scrollHeight - innerHeight) * 0.55); 0")
				time.Sleep(2 * time.Second)
				// Only read what the page itself wrote. Posting here would prove that
				// the endpoint answers, which is not the question: the page has to
				// notice the scroll and report it on its own.
				// Read this chapter's key in the probe's private session. The default
				// session used to expose earlier probes' chapter positions here.
				own := fmt.Sprintf("nkreader.at:%s/%s/%s", rec.Key, pick.ID, chapterID)
				raw, err := view.Page().Eval(fmt.Sprintf(`(() => {
					const place = %q;
					return JSON.stringify({scrollY: Math.round(scrollY), h: document.body.scrollHeight,
						key: place, pct: localStorage.getItem(place + ':pct'),
						allKeys: Object.keys(localStorage).filter(k => k.endsWith(':pct'))});
				})()`, own))
				if err != nil {
					checkErr = fmt.Errorf("progress: %w", err)
				} else {
					log.Printf("progress: %v", raw)
					text, _ := raw.(string)
					// The key is the evidence, not the number: a one-page chapter
					// reports 0% however far it is scrolled, because the percentage
					// follows the page index. What matters is that the page wrote a
					// report for *this* chapter on its own.
					if !strings.Contains(text, ":"+rec.Key+"/") {
						checkErr = fmt.Errorf("the page reported no position for %s: %s", rec.Key, text)
					} else if !strings.Contains(text, `"pct":"`) && !strings.Contains(text, `"pct":null`) {
						checkErr = fmt.Errorf("progress: no percentage in the report: %s", text)
					} else if strings.Contains(text, `"pct":null`) {
						checkErr = fmt.Errorf("the page stored no percentage for this chapter: %s", text)
					}
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
					// The direction is a toggle: clicking it once lands on whichever
					// way the page was not. Asserting a specific word made the check
					// depend on the starting state; what matters is two columns and
					// which page is on the right.
					text, _ := raw.(string)
					var shape struct {
						Columns       string
						DirLabel      string
						First, Second struct{ X, W int }
					}
					if json.Unmarshal([]byte(text), &shape) != nil || len(strings.Fields(shape.Columns)) != 2 {
						checkErr = fmt.Errorf("two columns did not apply: %s", text)
					} else if strings.HasPrefix(strings.TrimSpace(shape.DirLabel), "右") ||
						strings.Contains(strings.ToLower(shape.DirLabel), "rtl") {
						if shape.First.X <= shape.Second.X {
							checkErr = fmt.Errorf("right-to-left should put the first page on the right: %s", text)
						}
					} else if shape.First.X >= shape.Second.X {
						checkErr = fmt.Errorf("left-to-right should put the first page on the left: %s", text)
					}
				}
				_, _ = view.Page().Eval("document.getElementById('spread').click(); document.getElementById('dir').click(); 0")
			}
			if *scrollCheck && checkErr == nil {
				// Scrolling is remembered per chapter: move, reopen, look. The
				// distance comes from the page, not from a fixed number: a fixture
				// page shorter than the target would fail here for being short, and a
				// page that cannot scroll at all would pass on a threshold of zero.
				// Wait until the reader has booted before touching it: its own
				// report at 900 ms proves the setup got that far, and a scroll
				// delivered before that has nothing to deliver to. On macOS the
				// probe scrolled into a page whose storage held three layout keys
				// and nothing else, and no save ever happened.
				booted := false
				for i := 0; i < 100 && !booted; i++ {
					v, err := view.Page().Eval("localStorage.getItem('nkreader.at:loopback_pages/one/1:pct') !== null || Object.keys(localStorage).some(k => k.endsWith(':pct'))")
					if err == nil {
						if ready, ok := v.(bool); ok && ready {
							booted = true
							break
						}
					}
					time.Sleep(100 * time.Millisecond)
				}
				if !booted && checkErr == nil {
					// Without its own boot the page has nothing to save to, and a
					// scroll here measures the probe rather than the reader.
					checkErr = fmt.Errorf("the reader never reported its own progress; storage never showed a position key")
				}
				// Clear the stored place before moving: the profile survives between
				// runs, and a key left by an earlier one would be what the reader
				// restores, so the check would measure the store instead of the
				// restore. Clearing first also means the value written below is this
				// run's.
				raw, err := view.Page().Eval(`(() => {
					localStorage.removeItem('nkreader.at:loopback_pages/one/1');
					localStorage.removeItem('nkreader.at:loopback_pages/one/1:pct');
					const maxY = Math.max(0, document.body.scrollHeight - innerHeight);
					let t = Math.min(1200, Math.max(0, maxY - 40));
					// A page restored from a stale key can already sit exactly here,
					// and a scroll to where it already is fires no event — so nothing
					// is written and the reopen has nothing to restore. Move anyway.
					if (Math.abs(scrollY - t) < 2) t = Math.max(0, Math.min(maxY, t + 40));
					scrollTo(0, t);
					return Math.round(scrollY);
				})()`)
				moved, _ := raw.(float64)
				if err != nil || moved < 100 {
					checkErr = fmt.Errorf("the page cannot be scrolled enough to check: %v (%v)", raw, err)
				}
				if checkErr == nil {
					// The page writes its place 400 ms after the scroll stops, and
					// that debounce is only armed once the reader has booted. Waiting a
					// fixed second therefore raced the boot: one macOS run scrolled
					// before the listener existed, nothing was written, and the reopen
					// had nothing to restore. Poll for the key and say plainly when it
					// never appears, rather than reporting the later symptom.
					const placeKey = "nkreader.at:loopback_pages/one/1"
					var stored any
					var seen time.Duration
					for i := 0; i < 50; i++ {
						v, serr := view.Page().Eval("localStorage.getItem('" + placeKey + "')")
						if serr == nil {
							stored = v
							if v != nil {
								break
							}
						}
						seen += 100 * time.Millisecond
						time.Sleep(100 * time.Millisecond)
					}
					if stored == nil && checkErr == nil {
						checkErr = fmt.Errorf("the page stored no place for %s after %v", placeKey, seen)
						log.Printf("no place key after %v; scrollY=%v", seen, func() any {
							y, _ := view.Page().Eval("scrollY")
							return y
						}())
					}
					if saved, serr := view.Page().Eval("JSON.stringify([scrollY, localStorage.length, localStorage.getItem('" + placeKey + "')])"); serr == nil {
						log.Printf("settled at %v after moving to %v", saved, moved)
					}
					if checkErr != nil {
						mygo.App.Quit()
						return
					}
					if err := view.Page().LoadURL(target); err != nil {
						checkErr = fmt.Errorf("reload: %w", err)
					} else {
						time.Sleep(*wait)
						y, err := view.Page().Eval("scrollY")
						saved, _ := view.Page().Eval("localStorage.getItem('nkreader.at:loopback_pages/one/1')")
						log.Printf("after reopening the chapter, scrollY=%v saved=%v (%v)", y, saved, err)
						// Landing anywhere below the old place is not restoring it: the
						// point is that the reader comes back to where it was, not that
						// it scrolled somewhere. Allow a fifth of the distance for a
						// page whose height changed between visits, and floor it so a
						// small move is not compared proportionally.
						// The target is the place this probe moved to, not whatever the
						// store says: trusting the store would let a stale value agree
						// with itself.
						want, _ := saved.(string)
						if stored, convErr := strconv.ParseFloat(want, 64); convErr != nil || stored != moved {
							checkErr = fmt.Errorf("the store says %v, but the probe scrolled to %v", saved, moved)
						}
						target := moved
						slack := target * 0.2
						if slack < 80 {
							slack = 80
						}
						if v, ok := y.(float64); !ok || v < target-slack || v > target+slack {
							checkErr = fmt.Errorf("the place was not restored: scrollY=%v, saved %v", y, saved)
						}
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
	if !checked.Load() {
		log.Fatal("the window ended before the page could be checked")
	}
	fmt.Println("done")
}
