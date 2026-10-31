# Nekophage status

English | [简体中文](STATUS.md)

Updated 2026-10-30. Every result below comes from **Windows** on this machine, with real comic sources and the real network.

## Implemented

| Part | Contents |
|---|---|
| Wiring | Core is compiled into the same process; `internal/core` provides the runtime, the registry and the comic data API. On 2026-10-31 it was rebuilt against the **latest Core** (including `magpieplugin`, `host/sse` and the Magpie mobile API) and rechecked whole: `gofmt` clean, `go vet ./...` and `go test ./...` all pass — the desktop does no AI work, but Core's new packages must not break it. `build/core` is identical to Core's working tree (`diff -rq` finds nothing, excluding `.git`/`.github`/`apple`/`build`, the tests and `testdata`; `build/bin` is the gomobile toolchain, 9 MB, and copying it would only be waste). That tree is Core `e828111`, so `diff` plus the commit id answer "which Core is this" instead of memory. After wiring it in, `go test -count=1 ./...` and `go test -race -count=1 ./...` pass in all ten packages that have tests, with no data race (the race run used to cover only `./internal/...`, which was not the same set of packages as `go test`; they now agree). `-version` now prints the program's own commit and **Core's** commit: the latter comes from a `corestamp.go` written by `scripts/fetch-core.sh`, and without a stamp it says `Core unknown` rather than passing off its own commit as Core's |
| Networking | `internal/core/transport.go`: net/http, cookies by partition, cookies stored and sent on every redirect hop, 64 MiB per response; bytes a request line cannot carry (spaces, control characters) are escaped |
| Cookies | `internal/cookies`: host-only and domain cookies, paths, Secure, expiry; persistent cookies go to `cookies.json` |
| Storage | `internal/store`: source data, settings and plugin records; values stay as Core wrote them, large integers and key order included |
| Live end to end | `TestLiveSource` runs the whole line against a real source: install → search → details → chapters → image → `/chapter` → `/img` → shelf record → `/progress` write-back → `/keep` a page → `/switch` the next chapter → reopen the profile and read it back; needs `NK_LIVE_JS` and `NK_LIVE_QUERY` |
| Source settings | reads the `settings` declaration Core reports (select, switch, input, callback) and draws each by type in the sources window; a chosen value is stored as JSON in that source's settings, so numbers and booleans stay as written; an input's `validator` runs as a regular expression; a callback is a function inside the source and cannot cross the boundary, so it is shown with a note |
| Sources | `internal/library`: install or update from a script URL, script text or `index.json`; removing a source also drops its data and cookies; search options go to `search.load` at the source's defaults |
| Indexes | Compared with Core's version rules: not installed, updatable, installed, from another index, or needs a newer app |
| Language | Chinese throughout; MyGo's own control strings (search placeholder, clear) follow through `mygo/locales/zh-Hans` and `App.SetLanguages("zh-Hans")` |
| Searching every source | the header switches between 当前源 and 全部源; with every source, each enabled source's first page is searched concurrently (three at a time) and the results are merged in installation order, each card labelled with its source; opening a comic from another source switches the chosen source |
| Search filters | reads the source's `search.optionList` and shows each group as a scrolling row of chips; single-choice groups swap their value, multi-choice groups send a JSON array as one string, which is what the official client does, and changing source resets to that source's defaults |
| Main window | MyGo native UI: source picker, search, a results grid whose covers are downloaded and decoded asynchronously and whose cards say 「已在书架」 when the comic is already kept, browsing (a source's `explore`, all four kinds), details (cover, tag groups, a Markdown description, grouped chapters with a filter, and a 「读到」 marker on the chapter the shelf remembers, and a 「上次读到 第 N 页 · P%」 line with a 「继续阅读」 button), paging; it remembers the last source, query and chosen favourite folder in `ui.json` |
| Reader | A self-contained page in a WebView: vertical scroll, lazy images, progress, width control, keyboard scrolling, per-page retry, and the place in a chapter remembered; Go fetches each image through the source's rules and streams it |
| Verification | A window of its own loads the challenge, checks Core's done rule, and hands back cookies and the User-Agent before the app retries |

## Verified with real sources and the real network

**One command reruns every real source**: `NK_LIVE_DIR=/path/to/scripts scripts/live-sources.sh` runs `TestLiveSource` for each `.js` in that directory, prints PASS or FAIL per source, and leaves logs in `build/live/<source>.log`. A failing source is a result, not a crash: a site that changed has to be distinguishable from a client that broke. On 2026-10-31 all five scripts present (comick, copy_manga, komiic, komiic2, manga_dex) passed.

**Four sources re-run on 2026-10-31**: with Core at `e828111`, `TestLiveSource` passes for MangaDex, Komiic, Comick and Copy Manga. Comick exposed an assumption in the test: its chapter had a single image while the case asked for `page=3`, the reader clamped it to 1 as it should, and the assertion failed — the test was wrong, not the code; it now asserts `min(3, len(images))`. The HTTP 210 that Copy Manga used to return no longer reproduces: that was temporary site behaviour, not a client regression.

| Source | Result |
|---|---|
| Copy Manga `copy_manga` v1.4.3 | searched 海贼王: 21 comics over 2 pages; details: 401 chapters, 4 tag groups; chapter: 209 images; image: 339,596 bytes of `image/webp`; cover: 108,565 bytes of `image/jpeg` |
| MangaDex `manga_dex` v1.2.0 | searched "one piece": 20 comics over 2 pages; details: 962 chapters, 3 tag groups; chapter: 53 images; image: 122,856 bytes of `image/jpeg`; cover: 56,745 bytes of `image/jpeg` |
| Comick `comick` v1.x | searched "one piece": 51 images in a chapter, a 37,912-byte image and a 44,668-byte cover, both WebP; the shelf recorded `page 3, 42%` |
| Komiic `komiic` v1.x | the paging browse returns different comics on page 1 and page 2, with covers and authors, so paging works; end to end: 67 images in a chapter, a 248,508-byte image, a 27,252-byte cover, and the shelf recording `page 3, 42%` |

Two real sources were run again after Core was synced to `af4f1a0`: MangaDex ("one piece" — 20 comics,
962 chapters, 53 images in chapter 1, an 8-image sweep and a 56,745-byte cover) and Copy Manga (海贼王 —
209 images, a 339,596-byte WebP, a 108,565-byte cover). Both `TestLiveSource` runs passed, and the shelf
still recorded "page 3, 42%" plus the kept page's page number.

The human-verification path runs too: `go run ./cmd/verifycheck --local` returns `ok` — a local fake site answers 403, the WebView opens, the challenge is solved, and `cf_clearance=local-ok` is written for that site's domain 2.8 seconds later. Rerun after Core was synced to `af4f1a0`.

Comick also exposed a defect in the test itself: it built `...&chapter=<chapter id>` by hand, and that source's chapter ids contain slashes, so the request line became an invalid HTTP version and `httptest.NewRequest` panicked. All four reader requests (`/chapter`, `/progress`, `/keep`, `/switch`) now encode with `url.Values`, and `/switch` parses the returned JSON and compares the `chapter` parameter instead of looking for a substring in the raw body.

Komiic also exposed two labelling problems on the server side: page images are WebP while the header says `image/jpeg`, and covers come back as `application/octet-stream`. The live test therefore classifies images by their header bytes (JPEG/PNG/GIF/WebP/HEIF/AVIF/BMP) instead of trusting `Content-Type` — the old check called a perfectly displayable WebP "not an image". WebP covers decode through MyGo's `ui.DecodeBitmap`, which already pulls in `x/image/webp`; page images go to the WebView, so both paths are unaffected.

The reader's two endpoints were called the way the page calls them: Copy Manga `/chapter` returned 209 images and `/img` 339,596 bytes; MangaDex 53 and 122,856.

The interface was checked with screenshots (Comick, `one piece`): the native window showed 18+ real covers with titles, `comick` in the source selector and "第 1 页 / 下一页", and the WebP covers decoded fine because MyGo's `ui.DecodeBitmap` pulls in `x/image/webp`. Earlier screenshots showed real covers, titles and authors; the details page showed the cover, tag groups, the description rendered as Markdown (`**Note:**` became bold, `---` a rule, links clickable) and 962 chapter rows, with the chapter filter in place. Source settings were checked on MangaDex: the panel offered Image Quality with its two options, choosing Compressed reported 已保存设置, and that source's data file became `{"data":{},"settings":{"image_quality":"Compressed"}}`. The choice also reaches the source: `TestLiveSourceSettingsTakeEffect` shows the default page address under `/data/` turning into `/data-saver/…` with a different file hash. That test is now source-agnostic: it takes a value the source itself declares that differs from the default and requires the address to change, so it also holds on Copy Manga (`…c1500x.webp` becoming `…c800x.webp`) instead of pinning MangaDex's `data-saver` wording. **Why** a stored value matters: it is read back through `getSetting`, so a wrong round trip would silently keep the old quality.

The live end-to-end test ran against MangaDex (one piece) in about three seconds: 20 search results, 962 chapters, six page-less chapters skipped, a 53-page chapter, a 122,856-byte image, `/chapter` listing 53 pages, `/img` returning the same bytes, the shelf recording `page 3, 42%`, a kept page, `/switch` answering with that chapter's address, and a reopened shelf holding one history row and one page.

Source settings were checked on MangaDex: the panel offered Image Quality with its two options, choosing Compressed reported 已保存设置, and that source's data file became `{"data":{},"settings":{"image_quality":"Compressed"}}`. The choice also reaches the source: `TestLiveSourceSettingsTakeEffect` shows the default page address under `/data/` turning into `/data-saver/…` with a different file hash. **Why** a stored value matters: it is read back through `getSetting`, so a wrong round trip would silently keep the old quality.

The live end-to-end test ran against MangaDex (one piece) in about three seconds: 20 search results, 962 chapters, six page-less chapters skipped, a 53-page chapter, a 122,856-byte image, `/chapter` listing 53 pages, `/img` returning the same bytes, the shelf recording page 3 at 42%, a kept page, `/switch` answering with that chapter's address, and a reopened shelf holding one history row and one page.

Importing kept pages was checked: a backup with an `image_favorites` row produced two pages, stored with the chapter name 第03话, chapter index 3, page numbers and image addresses, with the time converted from milliseconds to seconds.

Kept pages were checked on MangaDex chapter 7: 收藏本页 wrote the comic, chapter name, page number and image address to `shelf.json`, and the shelf's 图片收藏 · 1 张 showed the real thumbnail with 回到这一话.

Searching every source was checked with MangaDex and Komiic installed and one piece as the keyword: the merged grid held Komiic's six cards first and MangaDex's after, with the right labels and covers, and opening the last card (MangaDex's) switched the header to MangaDex with details and chapters loading.

Search filters were checked on MangaDex: the default Any, Sort By=Rating and Status=Completed each returned a different first screen, with the chosen chip highlighted.

Automatic continuation was checked on Komiic: with it on, reading 卷01 to its end moved the shelf on to 卷03 through 卷02 by itself, the bar showed 自动接续:开, and a short chapter that was never scrolled did not jump on its own.

Cross-chapter reading was checked on Komiic (卷01, 162 pages): clicking 下一话 moved the shelf from `chapterIndex 0` to `1 / 卷02` and the new chapter reported its own progress; the check also confirmed a late report from the replaced page no longer overwrites the new chapter (MangaDex keeps 7 rather than falling back to 6).

Progress write-back was checked on MangaDex chapter 7 of the list (53 pages): after scrolling to about 55%, the shelf file held `page:26, percent:55` and the shelf showed 1: Romance dawn · 55% · 刚刚.

The two-page layout was checked on MangaDex chapter 1 (53 pages) by asking the page: `cls="spread rtl"`, two columns of 433.5px, page 1 at x=436, page 2 at x=0 and on the same row.

Search history was checked after an import: the empty results screen offered 海賊王; searching one piece then left `["one piece","海賊王"]` in the file and both chips on screen.

Folder editing was checked with `-shelf-demo`, which runs the window's own code: it created 以后看, moved the kept comic into it and renamed it to 以后再看, leaving `shelf.json` with an empty 默认收藏 and 以后再看 holding one comic, and the capture showed 「（空）」 and the moved row.

The shelf was checked on a real source: opening chapter 3 of 我的女友老師 wrote the source, id, title, author, cover, chapter and time into `shelf.json`, and the shelf screen showed the cover with 第03话 · 刚刚.

Backup import was checked with three `.venera` files: `backup-test` (1 source, 2 data keys, 2 search entries), `backup-shelf` (1 history entry, 1 folder holding 1 comic, 1 search entry) and `backup-pages` (1 image favourite that lands as 2 kept pages, since the favourite and the episode row each carry one). It was first checked with a `.venera` carrying the real `copy_manga.js`: the preview reported 1 source, 2 with data and 2 search-history entries; the import installed `copy_manga` 1.4.3 through the normal flow and wrote back 4 data keys, keeping `12345678901234567890` intact.

Browsing works on all three sources: MangaDex showed its Popular part as a list with covers, Copy Manga showed its 推荐 and 热门 parts, and Komiic's paging browse changed content between page 1 and page 2. The reader window really runs in the WebView: the page's title was the comic's, it built 53 figures, images decoded (the first at 757×1200), and `currentSrc` pointed at the `nkreader://` image URL (`http://nkreader.localhost` on Windows). Reopening the same chapter returns to where reading stopped: after scrolling to 1200 and reloading, `scrollY` was still 1200. A separate Comick check recorded 51 figures, 51 images, and 4 decoded images, the first at 750×1194, with `currentSrc` pointing at `http://nkreader.localhost/img?source=comick&…`.

### Verification (2026-10-30)

- The window alone: a local page that drops its marker and sets `cf_clearance` by script was recognised as passed in 3.5 s; the window's User-Agent came back and the cookie reached the partition's jar.
- The whole flow: a local site that answers a plain request with 403 (`cf-mitigated: challenge` plus challenge markers) made Core open the window; once the page passed, the cookie and User-Agent went back, and the retry with `cf_clearance=clear` got `200:PASSED-OK`, with the cookie in that source's own `comic:c` partition. It took 3.4 s.

## Problems found and fixed this round

**Cookies set on a redirect hop were never stored (2026-10-31)**: the `CheckRedirect` in `internal/core/transport.go` read `via[len(via)-1].Response`, but Go attaches the redirect response to the **new** request's `Response` field; entries in `via` keep it `nil`. So the code that was supposed to store the previous hop's `Set-Cookie` and send this partition's cookies onward quietly did nothing, while its comment said otherwise. The new test "a cookie set on a redirect hop must be stored" failed on the first run and exposed it; the fix reads `req.Response` and its `Request.URL`. On a real site this looks like being signed out after a redirect, which is hard to attribute.

1. **The version number shut real sources out**: the app reported 0.1.0, while sources ask for `minAppVersion` 1.3.4 or 1.6.0, so they refused to install. The app's own version and the source API version are separate now; the latter reports 1.6.0.
2. **Core had no search options**: `search.load(keyword, options, page)` takes one value per option group, and without them a source pastes `undefined` into its URL. Core now reads `search.optionList` (`js.Runtime.SearchOptions`, `mobile.SearchOptions`) and the desktop app passes the defaults. MangaDex went from a protocol error to 20 comics.
3. **Spaces in URLs**: the MangaDex source pastes the keyword into `?title=one piece`; Go wrote the raw space and the server answered `PROTOCOL_ERROR`. Bytes a request line cannot carry are escaped now, as Dart's `Uri` does.
4. **Covers stayed grey**: the first frame ran before a cover arrived, and the decode cache remembered "no image" for good. Only decoded bitmaps and failed downloads are remembered now.
5. **A stale source list**: `Library.Sources()` read its own cache, so a source installed another way never showed. It asks Core every time now.

## A source changing, not a regression

Re-running the live sources on 2026-10-31, Copy Manga began answering **HTTP 210** for details, and
that source accepts only 200 (`copy_manga.js` throws ``Invalid status code: ${res.status}``), so the
run reports `js: Invalid status code: 210`. It is not a client defect: on the same day MangaDex (6.4 s), Komiic (1.7 s) and Comick (56 s) all
went end to end, and the text is thrown by the source itself.
A 210 is an off-spec answer from the site — plausibly a challenge or a rate limit — and this E2E
stays red until the site recovers.

## Not verified

- **A live Cloudflare challenge**: none could be reproduced. 18comic.vip answers 403 to a command-line client but opens in WebView2 without setting `cf_clearance`, so the current rule cannot call it passed; whether such a site lets the client in after verification needs another sample.
- **macOS and Linux**: cross-compile and `go vet` both pass (`GOOS=darwin GOARCH=arm64`, `GOOS=linux GOARCH=amd64`, rechecked 2026-10-31), but neither is **run**. Compiling is not opening a window: that layer needs each platform's own system libraries, so a real run still has to happen on those machines.
- **Video sources**: Core supports them, the desktop app does not wire them yet; `browser.Sniffer` returns "unsupported" for now.
- **The workbench**: not started; see [ARCHITECTURE.en.md](ARCHITECTURE.en.md).
- **A source's own decay**: vensrc's `wnacg` 1.4.1 fails to install because both release pages it reads now only offer an e-mail address instead of listing comic domains, so its `init` throws. The official client treats a failing `init` the same way, so this is not a client deviation.

## Checking tools

Two development programs, not in a release build:

```sh
# a real source end to end: install, search, details, chapter, image, and the
# reader page's own two endpoints
NK_LIVE_JS=/path/to/manga_dex.js NK_LIVE_QUERY="one piece" \
  go test -run TestLiveSource -v ./internal/library

# open the reader window, read the page's own state (did the images decode),
# and save a capture of the window
go run ./cmd/readercheck -script /path/to/manga_dex.js -query "one piece" -out reader.png
go run ./cmd/readercheck -script ... -query ... -layout   # two pages, right column first
go run ./cmd/readercheck -script ... -query ... -scan     # page counts per chapter

# open the main window, run a real search and save a capture; -open opens the
# first result first
go run ./cmd/uicheck -script /path/to/manga_dex.js -query "one piece" -open -out window.png

# verification: the window alone, and Core's whole challenge flow (a local
# site that answers like Cloudflare)
go run ./cmd/verifycheck -local
go run ./cmd/verifycheck -e2e
```

`NKDEBUG=1` prints cover download and decode lines.

## Next

1. Find a reproducible live Cloudflare challenge (a site that answers a command-line client with 403 and does hand out `cf_clearance`) and check the done rule against it. Sites like 18comic.vip, which open in WebView2 but never set `cf_clearance`, cannot be judged passed today; relaxing the rule is a design decision.
2. Build and run on macOS and Linux.
3. Wire video sources and sniffing — the three options, with the work and cost of each, are measured in [VIDEO-PLAYBACK.en.md](VIDEO-PLAYBACK.en.md): WebView with `<video>` (pure Go, reusing the reader's proxy), a native player (needs cgo and its own decoders; MyGo ships only GIF), or sniffing a direct URL for the system player (headers cannot travel with it).
4. The workbench module.
