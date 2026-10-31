package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestZZNhentaiJoin(t *testing.T) {
	script := os.Getenv("NK_LIVE_JS")
	if script == "" {
		t.Skip("")
	}
	body, _ := os.ReadFile(script)
	c, err := Open(Options{DataDir: filepath.Join(t.TempDir(), "d"), AppVersion: "1.6.3", Locale: "zh_CN"})
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
	groups, _ := c.SearchOptions(ctx, rec.Key)
	opts := []string{}
	for _, g := range groups {
		opts = append(opts, g.Default)
	}
	tags := []string{"maid", "stockings", "sole female", "nakadashi", "big breasts", "glasses",
		"school uniform", "twintails", "blonde", "femdom", "bondage", "tentacles",
		"full color", "english", "chinese", "catgirl", "yuri", "incest", "ahegao", "dark skin"}
	for _, n := range []int{1, 2, 3, 5, 8, 10, 12, 20} {
		kw := strings.Join(tags[:n], " ")
		res, err := c.Search(ctx, rec.Key, kw, opts, 1)
		t.Logf("tags=%2d len=%3d -> %d comics, err=%v", n, len(kw), len(res.Comics), err)
		time.Sleep(1200 * time.Millisecond)
	}
}
