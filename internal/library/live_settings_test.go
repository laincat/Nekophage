package library

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	c, err := core.Open(core.Options{DataDir: filepath.Join(t.TempDir(), "data"), AppVersion: "1.6.0", Locale: "zh_CN"})
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

	if err := c.Store().SaveSetting(rec.Key, "image_quality", json.RawMessage(`"Compressed"`)); err != nil {
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
	t.Logf("compressed: %s", compressed)
	if compressed == "" {
		t.Fatal("no chapter with pages after the setting")
	}
	if !strings.Contains(strings.ToLower(compressed), "data-saver") {
		t.Errorf("the setting did not reach the source: %s", compressed)
	}
}
