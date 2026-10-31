package library

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/laincat/nekophage/internal/core"
)

// A source's category page is a second way into its comics, and it pages
// separately from search: page two must not be page one again, and a source
// with several parts must load each of them.
//
// Gated by NK_LIVE_JS like the other live tests, so the normal gate stays
// offline.
func TestLiveCategoryPages(t *testing.T) {
	script := os.Getenv("NK_LIVE_JS")
	if script == "" {
		t.Skip("set NK_LIVE_JS to run against a real source")
	}
	body, err := os.ReadFile(script)
	if err != nil {
		t.Fatal(err)
	}
	c, err := core.Open(core.Options{DataDir: filepath.Join(t.TempDir(), "data"), AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	rec, err := c.Install(ctx, string(body), script, "")
	if err != nil {
		t.Fatal(err)
	}
	cat, err := c.CategoryPage(ctx, rec.Key)
	if err != nil {
		t.Fatalf("category: %v", err)
	}
	if len(cat.Parts) == 0 {
		t.Skipf("%s declares no category page", rec.Key)
	}
	t.Logf("category %q: %d part(s), %d option group(s)", cat.Title, len(cat.Parts), len(cat.Options))
	options := make([]string, 0, len(cat.Options))
	for _, g := range cat.Options {
		options = append(options, g.Default)
	}
	// Each part loads its own first entry; a part that has no values is one
	// the source declared empty.
	for i, part := range cat.Parts {
		// A random part is searched, not loaded: the interface picks some of its
		// values and runs a search, so there is no category page to read.
		if part.Type == "random" {
			t.Logf("  part %d %q is random (%d values, take %d)", i, part.Name, len(part.Categories), part.RandomNumber)
			continue
		}
		if len(part.Categories) == 0 {
			continue
		}
		// The same choice the interface makes: a per-value parameter, else the
		// part's groupParam, else the value itself.
		param := part.Categories[0]
		if len(part.Params) == len(part.Categories) {
			param = part.Params[0]
		} else if part.GroupParam != "" {
			param = part.GroupParam
		}
		page, err := c.CategoryComics(ctx, rec.Key, part.Categories[0], param, options, 1)
		if err != nil {
			t.Errorf("part %d %q: %v", i, part.Name, err)
			continue
		}
		t.Logf("  part %d %q (first value %q): %d comics, maxPage %d", i, part.Name, part.Categories[0], len(page.Comics), page.MaxPage)
		if len(page.Comics) == 0 {
			t.Errorf("part %d returned nothing", i)
			continue
		}
		if page.MaxPage < 2 {
			continue
		}
		// The second page must move on rather than repeat what was shown.
		//
		// A pause first: some sites answer an empty page to a request that
		// follows too closely, and an empty answer would read as a paging bug in
		// the client.
		time.Sleep(time.Second)
		next, err := c.CategoryComics(ctx, rec.Key, part.Categories[0], param, options, 2)
		if err != nil {
			t.Errorf("part %d page 2: %v", i, err)
			continue
		}
		seen := make(map[string]bool, len(page.Comics))
		for _, comic := range page.Comics {
			seen[comic.ID] = true
		}
		shared := 0
		for _, comic := range next.Comics {
			if seen[comic.ID] {
				shared++
			}
		}
		if len(next.Comics) == 0 {
			t.Errorf("part %d page 2 is empty", i)
			continue
		}
		if shared == len(next.Comics) {
			t.Errorf("part %d page 2 repeats page 1", i)
		}
		t.Logf("    page 2: %d comics, %d shared", len(next.Comics), shared)
	}
}

// A random part is not a category: its declared meaning is "take this many
// values and search for them", so the values have to work as a keyword.
func TestLiveCategoryRandomSearch(t *testing.T) {
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
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	rec, err := c.Install(ctx, string(body), script, "")
	if err != nil {
		t.Fatal(err)
	}
	cat, err := c.CategoryPage(ctx, rec.Key)
	if err != nil {
		t.Fatal(err)
	}
	var part *core.CategoryPart
	for i := range cat.Parts {
		if cat.Parts[i].Type == "random" && len(cat.Parts[i].Categories) > 0 {
			part = &cat.Parts[i]
			break
		}
	}
	if part == nil {
		t.Skip("SKIP: source declares no random part")
	}
	// A search takes the search's own options, not the category page's: nhentai
	// declares "/-Recent" for the category loader and "date" for search.load.
	groups, err := c.SearchOptions(ctx, rec.Key)
	if err != nil {
		t.Fatal(err)
	}
	options := []string{}
	for _, g := range groups {
		options = append(options, g.Default)
	}
	// What the interface sends: one value, which the site accepts. Joining the
	// declared randomNumber of values asks for works carrying all of them —
	// measured on nhentai, eight tags still answered, ten answered nothing and
	// twenty were rejected — so the count is a display size, not a join width.
	keyword := part.Categories[0]
	page, err := c.Search(ctx, rec.Key, keyword, options, 1)
	if err != nil {
		t.Fatalf("%q: %v", keyword, err)
	}
	t.Logf("  random part %q: %q -> %d comics, maxPage %d", part.Name, keyword, len(page.Comics), page.MaxPage)
	if len(page.Comics) == 0 {
		t.Fatalf("%q returned nothing, so the random part cannot be searched", keyword)
	}
}
