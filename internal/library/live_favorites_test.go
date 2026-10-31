package library

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/laincat/nekophage/internal/core"
)

// A source keeps favorites on its own side, and how it keeps them differs:
// nhentai has one folder and no folder list, ehentai has folders but no list to
// read, wnacg has both. Without reading the declaration the client could only
// guess, and a guess meant calling a function the source does not define.
//
// Gated by NK_LIVE_JS like the other live tests.
func TestLiveFavoritesSupport(t *testing.T) {
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	rec, err := c.Install(ctx, string(body), script, "")
	if err != nil {
		t.Fatal(err)
	}
	sup := c.FavoritesSupport(ctx, rec.Key)
	if !sup.Supported {
		t.Skip("SKIP: source declares no favorites")
	}
	t.Logf("  support: multiFolder=%v folders=%v canLoad=%v", sup.MultiFolder, sup.Folders, sup.CanLoad)

	if sup.Folders {
		folders, _, err := c.FavoriteFolders(ctx, rec.Key, "")
		// A source that needs a session may refuse; that is the source's word.
		if err != nil {
			t.Logf("  folders refused: %v", err)
		} else {
			t.Logf("  folders: %d", len(folders))
		}
	}
	if !sup.CanLoad {
		// A source with folders but no list must not be asked for one.
		return
	}
	page, err := c.FavoriteComics(ctx, rec.Key, "", 1)
	if err != nil {
		// Without a session the source says so; that is not a client failure.
		t.Logf("  favorites refused: %v", err)
		return
	}
	t.Logf("  favorites: %d comics, maxPage %d", len(page.Comics), page.MaxPage)
}
