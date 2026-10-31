package core

import (
	"context"
	"encoding/json"
	"testing"
)

// Adding and removing reach the source with what it expects: the comic, the
// folder its loader wants, and whether this is an add. A single-folder source
// takes no folder at all.
func TestSetFavoritePassesWhatTheSourceExpects(t *testing.T) {
	c := openTest(t)
	ctx := context.Background()
	if _, err := c.Install(ctx, favoritesComic, "file_fav2", ""); err != nil {
		t.Fatal(err)
	}
	// A token is required by the source, so the first attempt is refused and
	// proves the arguments reached it.
	if err := c.SetFavorite(ctx, "fav", "c1", "f1", true); err == nil {
		t.Fatal("expected the source to refuse")
	}

	const recorder = `class Rec extends ComicSource {
  name = 'Rec'; key = 'rec'; version = '1'
  favorites = {
    multiFolder: false,
    addOrDelFavorite: async (comicId, folderId, isAdding) => {
      this.saveData('last', JSON.stringify([comicId, folderId, isAdding]))
      return 'ok'
    },
    loadComics: async () => ({comics: [], maxPage: 1}),
  }
}`
	if _, err := c.Install(ctx, recorder, "file_rec", ""); err != nil {
		t.Fatal(err)
	}
	if err := c.SetFavorite(ctx, "rec", "comic-9", "", true); err != nil {
		t.Fatalf("add: %v", err)
	}
	// The single-folder source must see null, not an empty string.
	if got := recordedArguments(t, c, "rec"); got != `["comic-9",null,true]` {
		t.Errorf("arguments = %s", got)
	}
	if err := c.SetFavorite(ctx, "rec", "comic-9", "", false); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if got := recordedArguments(t, c, "rec"); got != `["comic-9",null,false]` {
		t.Errorf("arguments = %s", got)
	}
}

// recordedArguments is what the recorder source saved, unquoted: saveData stores
// a JSON value, and a string arrives wrapped in another pair of quotes.
func recordedArguments(t *testing.T, c *Core, key string) string {
	t.Helper()
	raw, ok, err := c.Store().LoadData(key, "last")
	if err != nil || !ok {
		t.Fatalf("recorded: %v %v", ok, err)
	}
	var inner string
	if err := json.Unmarshal(raw, &inner); err == nil {
		return inner
	}
	return string(raw)
}
