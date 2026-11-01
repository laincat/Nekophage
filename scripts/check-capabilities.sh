#!/bin/sh
# Fails when a capability the official comic sources declare is no longer read.
#
# Every one of these was found by hand, one round at a time: the category page,
# the random part, the ranking page, the three login styles, the source's own
# favorites, the translation table. Each had been declared by the sources and
# silently ignored by the client, so the only way to find the next one was to
# read the contracts again.
#
# This is a regression net, not a discovery tool: it checks that the client still
# has a path for what it claims to support, so deleting one of those paths fails
# here instead of showing up as an empty screen. Matching is by fixed string, so
# a renamed function cannot keep passing on a shared prefix.
set -e
root=$(dirname "$0")/..
cd "$root"

fail=0
need() {
	# need <capability> <file> <fixed text>
	if ! grep -qF -- "$3" "$2" 2>/dev/null; then
		printf 'check-capabilities: %s is not read any more (%s)\n' "$1" "$2"
		fail=1
	fi
}

# A source's category page: parts, their values, and the loader.
need "category page" internal/core/core.go 'func (c *Core) CategoryPage(ctx context.Context, key string) (Category, error)'
need "category values" internal/ui/app.go 'a.loadCategoryPage('
need "category options" internal/ui/filters.go 'optionChips('
# A part whose values are searched rather than loaded.
need "random part" internal/ui/app.go 'func (a *App) randomCategory(part core.CategoryPart)'
# The ranking page: a flat option list with its own loader.
need "ranking page" internal/core/core.go 'func (c *Core) RankingComics(ctx context.Context, key, option string, page int) (Page, error)'
need "ranking entry" internal/ui/app.go 'a.loadRankingPage('
# The three login styles, each with its own call.
need "password login" internal/core/core.go '"account.login"'
need "cookie login" internal/core/core.go '"account.loginWithCookies.validate"'
need "webview login" internal/core/core.go '"account.loginWithWebview"'
need "account panel" internal/ui/settings.go 'm.account.Cookies'
need "account login" internal/ui/settings.go 'a.loginAccount(key, user, pass)'
# The source's own favorites, in one folder or several.
need "favorites shape" internal/core/core.go 'func (c *Core) FavoritesSupport(ctx context.Context, key string) FavoritesSupport'
need "favorites list" internal/core/core.go '"favorites.loadComics"'
need "favorites folders" internal/core/core.go '"favorites.loadFolders"'
need "favorites view" internal/ui/favorites.go 'func (a *App) favoritesView(c *ui.Context, t *ui.Theme)'
need "favorites toggle" internal/ui/favorites.go 'func (a *App) toggleSourceFavorite()'
need "backup export" internal/ui/backup_export.go 'func (a *App) writeBackup(path string) error'
need "source log panel" internal/ui/sourcelogs.go 'func (a *App) OpenSourceLogs()'
need "source log buffer" internal/core/core.go 'func (c *Core) LogLines() []LogLine'
need "browse cursor" internal/core/core.go 'rt.Explore(ctx, key, index, page, cursor)'
need "browse cursor paging" internal/ui/app.go 'a.loadExploreAt(key, index, page+1, res.Next)'
need "comments" internal/core/comments.go 'func (c *Core) Comments(ctx context.Context'
need "link to id" internal/core/core.go 'func (c *Core) OpenLink(ctx context.Context, raw string)'
need "send comment" internal/core/comments.go 'func (c *Core) SendComment('
need "comment vote" internal/ui/comments.go 'func (a *App) likeComment(one core.Comment, isLike bool)'
need "comment window" internal/ui/comments.go 'func (a *App) openComments()'
need "favorite folders" internal/ui/favorites.go 'func folderPicker(folders map[string]string)'
# The UI strings a source declares for this locale.
need "translations" internal/core/core.go 'func (c *Core) Translations(ctx context.Context, key string) (map[string]string, error)'
need "translated titles" internal/ui/settings.go 'm.translations[title]'
need "setting callback" internal/core/core.go 'settings."+settingKey+".callback'
need "source UI handler" internal/ui/jsui.go 'func (h *JSHandler) ShowDialog(title, content string, actions []js.DialogAction) error'
# A declaration that is data rather than a function: favorites.multiFolder
# decides whether a folder id is expected at all, and Call cannot reach it.
need "data declarations" internal/core/core.go 'rt.Value(ctx, key, "favorites.multiFolder")'

if [ "$fail" -ne 0 ]; then
	printf 'check-capabilities: a declared capability lost its path\n'
	exit 1
fi
printf 'check-capabilities: ok (category, random, ranking, account, favorites, translations)\n'
