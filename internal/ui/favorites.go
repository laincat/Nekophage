package ui

import (
	"context"
	"sort"
	"time"

	"github.com/egoist/mygo/ui"
)

// loadFavorites reads the capabilities and, if needed, the folder list before
// loading comics. Source errors remain visible so users can distinguish an
// empty list from an expired session.
// FavoritesForCheck reports what the source's favourites view holds: how many
// comics, the first one's title, and any error it is showing.
//
// A capture cannot tell a listing from an empty page — that is the lesson the
// comment check already paid for — so a check asks the view instead.
func (a *App) FavoritesForCheck() (count int, first string, errText string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	count = len(a.favComics)
	if count > 0 {
		first = a.favComics[0].Title
	}
	return count, first, a.favErr
}

func (a *App) loadFavorites(key string) {
	if key == "" {
		return
	}
	a.mu.Lock()
	a.favBusy, a.favErr = true, ""
	ticket := a.favGen.next()
	a.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		sup := a.core.FavoritesSupport(ctx, key)
		var folders map[string]string
		var err error
		if sup.CanLoad && sup.MultiFolder && sup.Folders {
			folders, _, err = a.core.FavoriteFolders(ctx, key, "")
		}
		a.win.Update(func() {
			a.mu.Lock()
			if !a.favGen.current(ticket) {
				a.mu.Unlock()
				return
			}
			a.favLoaded, a.favBusy, a.favSup = true, false, sup
			a.favFolders, a.favComics, a.favErr = folders, nil, ""
			if err != nil {
				a.favErr = explain(err)
				a.mu.Unlock()
				return
			}
			folder := ""
			if sup.MultiFolder {
				keys := make([]string, 0, len(folders))
				for k := range folders {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				if len(keys) > 0 {
					folder = keys[0]
				}
			}
			a.sourceFavFolder = folder
			shouldLoad := sup.CanLoad && (!sup.MultiFolder || folder != "")
			a.mu.Unlock()
			if shouldLoad {
				a.loadFavoritePage(key, folder, 1)
			}
		})
	}()
}

func (a *App) loadFavoritePage(key, folder string, page int) {
	a.mu.Lock()
	a.favBusy, a.favErr = true, ""
	a.sourceFavFolder, a.favPage = folder, page
	ticket := a.favGen.next()
	a.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		result, err := a.core.FavoriteComics(ctx, key, folder, page)
		a.win.Update(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			if !a.favGen.current(ticket) {
				return
			}
			a.favBusy = false
			if err != nil {
				a.favErr = explain(err)
				a.favComics = nil
				return
			}
			a.favComics, a.favMax = result.Comics, result.MaxPage
		})
	}()
}

func (a *App) favoritesView(c *ui.Context, t *ui.Theme) {
	a.mu.Lock()
	sup, loaded, busy := a.favSup, a.favLoaded, a.favBusy
	folders, folder := a.favFolders, a.sourceFavFolder
	comics, page, maxPage, errText := a.favComics, a.favPage, a.favMax, a.favErr
	key := a.sourceKey
	a.mu.Unlock()
	if busy || !loaded {
		ui.Text(c, "正在加载…").Padding(16, 16)
		return
	}
	if errText != "" {
		ui.Column(c).Padding(16, 16).Gap(8).Children(func() {
			ui.Text(c, errText).Selectable()
			if ui.Button(c, "重试").Clicked() {
				a.loadFavorites(key)
			}
		})
		return
	}
	if !sup.CanLoad {
		ui.Text(c, "这个源没有提供收藏列表。").Padding(16, 16)
		return
	}
	if sup.MultiFolder {
		keys := make([]string, 0, len(folders))
		for k := range folders {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		ui.Row(c).Padding(8, 16).Gap(6).Wrap().Children(func() {
			for _, id := range keys {
				id := id
				if id == folder {
					if ui.PrimaryButton(c, folders[id]).Clicked() {
						a.loadFavoritePage(key, id, 1)
					}
				} else if ui.Button(c, folders[id]).Clicked() {
					a.loadFavoritePage(key, id, 1)
				}
			}
		})
	}
	if len(comics) == 0 {
		ui.Text(c, "这个收藏夹没有内容。").Padding(16, 16)
		return
	}
	ui.Scroll(c).Grow(1).Padding(12, 16).Children(func() {
		for _, comic := range comics {
			a.browseRow(c, t, key, comic)
		}
		if maxPage > 0 {
			ui.Row(c).Padding(8, 0).Gap(10).Children(func() {
				ui.Textf(c, "第 %d / %d 页", page, maxPage)
				if page > 1 && ui.Button(c, "上一页").Clicked() {
					a.loadFavoritePage(key, folder, page-1)
				}
				if page < maxPage && ui.Button(c, "下一页").Clicked() {
					a.loadFavoritePage(key, folder, page+1)
				}
			})
		}
	})
}

// loadSourceFavorites reads what the source offers for a comic that is already
// open: whether it keeps favorites, its folders, and which of them hold this
// comic. A source that keeps none is left alone, and its button never appears.
func (a *App) loadSourceFavorites(key, comicID string) {
	if key == "" || comicID == "" {
		return
	}
	a.mu.Lock()
	ticket := a.favGen.next()
	a.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		sup := a.core.FavoritesSupport(ctx, key)
		if !sup.Supported {
			a.win.Update(func() {
				a.mu.Lock()
				defer a.mu.Unlock()
				if !a.favGen.current(ticket) {
					return
				}
				a.srcFav, a.srcFavLoaded = sup, true
			})
			return
		}
		var folders map[string]string
		var held []string
		var err error
		if sup.Folders {
			folders, held, err = a.core.FavoriteFolders(ctx, key, comicID)
		}
		a.win.Update(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			if !a.favGen.current(ticket) {
				return
			}
			a.srcFav, a.srcFavLoaded = sup, true
			a.srcFavFolders, a.srcFavHeld = folders, held
			if err != nil {
				// A source that needs a session says so; the button still works
				// and reports the same words when pressed.
				a.srcFavMsg = ""
				return
			}
			if sup.Folders && len(folders) > 0 {
				names := make([]string, 0, len(folders))
				for id := range folders {
					names = append(names, id)
				}
				sort.Strings(names)
				if len(held) > 0 {
					a.srcFavFolder = held[0]
				} else {
					a.srcFavFolder = names[0]
				}
			}
		})
	}()
}

// toggleSourceFavorite adds this comic to the source's favorites, or removes it
// when the source reports it is already there.
func (a *App) toggleSourceFavorite() {
	a.mu.Lock()
	key, comicID := a.currentKey(), a.detailComic.ID
	sup, folder, held, favoriteID := a.srcFav, a.srcFavFolder, a.srcFavHeld, a.srcFavID
	// The page that listed the comic and the details page may each carry the
	// handle; either one is enough.
	if favoriteID == "" {
		favoriteID = a.detailComic.FavoriteID
	}
	if favoriteID == "" {
		favoriteID = a.detail.FavoriteID
	}
	if a.srcFavBusy || comicID == "" {
		a.mu.Unlock()
		return
	}
	a.srcFavBusy, a.srcFavMsg = true, ""
	ticket := a.favGen.next()
	a.mu.Unlock()

	adding := true
	for _, id := range held {
		if id == folder || len(held) > 0 {
			adding = false
			break
		}
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		var err error
		if sup.Supported {
			err = a.core.SetFavorite(ctx, key, comicID, folder, favoriteID, adding)
		}
		a.win.Update(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			if !a.favGen.current(ticket) {
				return
			}
			a.srcFavBusy = false
			switch {
			case err != nil:
				// The source's own words: "Login expired", "Failed to add
				// favorite", or whatever it chose.
				a.srcFavMsg = explain(err)
			case adding:
				a.srcFavMsg = "已加入源收藏"
				a.srcFavHeld = append(a.srcFavHeld, folder)
			default:
				a.srcFavMsg = "已从源收藏移除"
				a.srcFavHeld = nil
			}
		})
	}()
}

// folderPicker builds the labels and lookup a Select needs for a source's
// folders. The label is the name the source gave; the id is what its loader
// expects.
func folderPicker(folders map[string]string) (labels []string, byLabel map[string]string, labelOf map[string]string) {
	ids := make([]string, 0, len(folders))
	for id := range folders {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	byLabel = make(map[string]string, len(ids))
	labelOf = make(map[string]string, len(ids))
	for _, id := range ids {
		name := folders[id]
		if name == "" {
			name = id
		}
		// Two folders may share a name; the id disambiguates the label rather
		// than dropping one of them.
		if _, taken := byLabel[name]; taken {
			name = name + " (" + id + ")"
		}
		labels = append(labels, name)
		byLabel[name] = id
		labelOf[id] = name
	}
	return labels, byLabel, labelOf
}
