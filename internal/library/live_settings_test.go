package library

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/laincat/nekophage/internal/core"
)

// Does a stored setting actually reach the source? MangaDex picks its image
// URLs by image_quality, so the two values must give different addresses.
func TestLiveSourceSettingsTakeEffect(t *testing.T) {
	script := os.Getenv("NK_LIVE_JS")
	query := os.Getenv("NK_LIVE_QUERY")
	if script == "" || query == "" {
		t.Skip("set NK_LIVE_JS and NK_LIVE_QUERY to run against a real source")
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
	options, err := c.DefaultOptions(ctx, rec.Key)
	if err != nil {
		t.Fatal(err)
	}
	// Only a source that declares image_quality can show it reaching the
	// source. Running the suite against another source used to fail here for
	// the wrong reason, so it skips instead.
	settings, err := c.Settings(ctx, rec.Key)
	if err != nil {
		t.Fatal(err)
	}
	hasQuality := false
	for _, s := range settings {
		if s.Key == "image_quality" {
			hasQuality = true
			break
		}
	}
	if !hasQuality {
		t.Skipf("%s declares no image_quality setting", rec.Key)
	}
	page, err := c.Search(ctx, rec.Key, query, options, 1)
	if err != nil || len(page.Comics) == 0 {
		t.Fatalf("search: %v, %d comics", err, len(page.Comics))
	}
	pick := page.Comics[0]
	d, err := c.Details(ctx, rec.Key, pick.ID)
	if err != nil {
		t.Fatal(err)
	}
	first := ""
	for i, ch := range d.Chapters {
		if i >= 10 {
			break
		}
		got, err := c.ChapterImages(ctx, rec.Key, pick.ID, ch.ID)
		if err == nil && len(got) > 0 {
			first = got[0]
			break
		}
	}
	if first == "" {
		t.Skip("no chapter with pages")
	}
	t.Logf("default  : %s", first)

	// Each source spells its qualities differently (MangaDex says
	// "Compressed", Copy Manga says "800"), so the new value is taken from the
	// source's own options rather than hard-coded.
	var wanted json.RawMessage
	for _, s := range settings {
		if s.Key != "image_quality" || len(s.Options) < 2 {
			continue
		}
		for _, opt := range s.Options {
			if string(opt.Value) != string(s.Default) {
				wanted = opt.Value
				break
			}
		}
	}
	if wanted == nil {
		t.Skipf("%s declares no second image_quality option", rec.Key)
	}
	t.Logf("setting   : %s", wanted)
	if err := c.Store().SaveSetting(rec.Key, "image_quality", wanted); err != nil {
		t.Fatal(err)
	}
	compressed := ""
	for i, ch := range d.Chapters {
		if i >= 10 {
			break
		}
		got, err := c.ChapterImages(ctx, rec.Key, pick.ID, ch.ID)
		if err == nil && len(got) > 0 {
			compressed = got[0]
			break
		}
	}
	t.Logf("changed   : %s", compressed)
	if compressed == "" {
		t.Fatal("no chapter with pages after the setting")
	}
	// The check is that the setting reached the source, not one source's
	// naming: the address it now returns must differ from the default one.
	if compressed == first {
		t.Errorf("the setting did not reach the source: still %s", compressed)
	}
}
