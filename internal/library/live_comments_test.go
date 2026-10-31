package library

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/laincat/nekophage/internal/core"
)

// TestLiveComments reads a real comic's comment thread.
//
//	NK_LIVE_JS=/path/to/komiic.js NK_LIVE_QUERY=test go test -run TestLiveComments -v ./internal/library
//
// Twenty of the official sources declare `comic.loadComments`, and nothing here
// called it until the thread window existed: the thread of those sources was
// unreachable, which no other test would have noticed.
func TestLiveComments(t *testing.T) {
	script, query := os.Getenv("NK_LIVE_JS"), os.Getenv("NK_LIVE_QUERY")
	if script == "" || query == "" {
		t.Skip("set NK_LIVE_JS and NK_LIVE_QUERY to run against a real source")
	}
	body, err := os.ReadFile(script)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	c, err := core.Open(core.Options{DataDir: filepath.Join(t.TempDir(), "d"), AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	lib := New(c)
	record, err := lib.InstallText(ctx, string(body), script)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("declares comments: %v  canSend: %v  canLike: %v",
		c.CommentsSupported(ctx, record.Key), c.CanSendComment(ctx, record.Key), c.CanLikeComment(ctx, record.Key))
	res, err := c.Search(ctx, record.Key, query, nil, 1)
	if err != nil || len(res.Comics) == 0 {
		t.Fatalf("search: %v (%d comics)", err, len(res.Comics))
	}
	id := res.Comics[0].ID
	details, err := c.Details(ctx, record.Key, id)
	if err != nil {
		t.Fatalf("details: %v", err)
	}
	t.Logf("comic %q subId=%q", details.Title, details.SubID)
	page, err := c.Comments(ctx, record.Key, id, details.SubID, 1, "")
	if err != nil {
		t.Fatalf("comments: %v", err)
	}
	t.Logf("comments: %d, maxPage %d", len(page.Comments), page.MaxPage)
	for i, one := range page.Comments {
		if i >= 3 {
			break
		}
		t.Logf("  [%d] %s · %s · %s", i, one.UserName, one.Time, trim(one.Content, 60))
	}
}

func trim(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
