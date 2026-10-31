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
