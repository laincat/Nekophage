// Command explorecheck runs every source's browse pages against the real
// network and says which ones answer.
//
// Every official source declares `explore`, and the reading path never touched
// it: `TestLiveSource` searches, opens details and reads a chapter. So the browse
// tab — the first thing a user sees for a source with no search in mind — had no
// live evidence at all.
//
//	go run ./cmd/explorecheck -dir /path/to/scripts
//
// One source failing is a result, not a crash: the summary separates "the site
// refused" from "the client could not read what came back".
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/library"
	"github.com/laincat/nekophage/internal/sourceapi"
)

func main() {
	dir := flag.String("dir", "", "directory of source scripts")
	per := flag.Duration("timeout", 90*time.Second, "timeout per source")
	only := flag.String("only", "", "comma-separated file names to run")
	verbose := flag.Bool("v", false, "log every page, not just failures")
	flag.Parse()
	if *dir == "" {
		flag.Usage()
		os.Exit(2)
	}
	paths, err := filepath.Glob(filepath.Join(*dir, "*.js"))
	if err != nil || len(paths) == 0 {
		log.Fatalf("no scripts in %s", *dir)
	}
	sort.Strings(paths)
	if *only != "" {
		want := map[string]bool{}
		for _, name := range strings.Split(*only, ",") {
			want[strings.TrimSpace(name)] = true
		}
		kept := paths[:0]
		for _, p := range paths {
			if want[filepath.Base(p)] {
				kept = append(kept, p)
			}
		}
		paths = kept
	}

	// A source that needs a server the user runs cannot be judged here; saying
	// "failed" would mix a missing server with a broken client.
	needsServer := map[string]bool{"kavita": true, "komga": true, "lanraragi": true}
	var pass, fail, skipped, server []string
	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), ".js")
		if needsServer[name] && os.Getenv("NK_LIVE_SERVER_SOURCES") == "" {
			server = append(server, name)
			fmt.Printf("%-24s SKIP  needs a server the user runs\n", name)
			continue
		}
		result, err := run(path, *per, *verbose)
		switch {
		case err != nil:
			fail = append(fail, name)
			fmt.Printf("%-24s FAIL  %v\n", name, short(err))
		case result == "":
			skipped = append(skipped, name)
			fmt.Printf("%-24s SKIP  the source declares no browses\n", name)
		default:
			pass = append(pass, name)
			fmt.Printf("%-24s PASS  %s\n", name, result)
		}
	}
	fmt.Printf("\nexplorecheck: %d source(s); passed: %s\n", len(paths), strings.Join(pass, " "))
	if len(skipped) > 0 {
		fmt.Printf("no browses declared: %s\n", strings.Join(skipped, " "))
	}
	if len(fail) > 0 {
		fmt.Printf("failed: %s\n", strings.Join(fail, " "))
		os.Exit(1)
	}
}

// run installs one source, lists its browses and loads the first page of each
// one. It returns a one-line description of what came back.
func run(path string, timeout time.Duration, verbose bool) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp("", "explore-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	c, err := core.Open(core.Options{DataDir: dir, AppVersion: sourceapi.Level, Locale: "zh_CN"})
	if err != nil {
		return "", err
	}
	defer c.Close()
	lib := library.New(c)
	record, err := lib.InstallText(ctx, string(body), path)
	if err != nil {
		return "", fmt.Errorf("install: %w", err)
	}
	pages, err := c.ExplorePages(ctx, record.Key)
	if err != nil {
		return "", fmt.Errorf("list: %w", err)
	}
	if len(pages) == 0 {
		return "", nil
	}
	// Only the first page of each browse: the point is the contract, not the crawl.
	total, sections := 0, 0
	var failed []string
	for i, page := range pages {
		res, err := c.Explore(ctx, record.Key, i, 1, "")
		if err != nil {
			failed = append(failed, fmt.Sprintf("[%d] %s: %v", i, page.Title, short(err)))
			continue
		}
		n := 0
		for _, s := range res.Sections {
			n += len(s.Comics)
		}
		total += n
		sections += len(res.Sections)
		if verbose {
			fmt.Printf("    [%d] %-12s %-32s sections=%d comics=%d maxPage=%d\n",
				i, page.Type, page.Title, len(res.Sections), n, res.MaxPage)
		}
	}
	if len(failed) > 0 {
		// A source with several browses can have one that needs a login and one
		// that works; saying so is the difference between a client bug and a site.
		return "", fmt.Errorf("%d of %d browses failed: %s", len(failed), len(pages), strings.Join(failed, "; "))
	}
	return fmt.Sprintf("%d browses, %d sections, %d comics", len(pages), sections, total), nil
}

// short trims a message for the one-line summary; a sweep of 34 sources needs
// rows that stay on one line.
func short(err error) string {
	text := strings.ReplaceAll(err.Error(), "\n", " ")
	if len(text) > 160 {
		text = text[:160] + "…"
	}
	return text
}
