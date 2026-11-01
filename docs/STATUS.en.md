# Nekophage status

English | [简体中文](STATUS.md)

Updated 2026-10-31. Every result below comes from **Windows** on this machine, with real comic sources and the real network. That day's additions are three: a real bug in redirect cookies on the transport (it read the wrong field), Chinese error text under the same rule the iOS client uses, and `live-sources.sh` gaining repeat runs and a `FLAKY` class.


The four CI jobs of Core's `6e07429` are green (three Go platforms plus Apple's xcframework with two Swift smoke tests), checked on 2026-11-01.

**Pin moved to `6e07429` (2026-11-01)**: Core gained the three standard AI entry points and the optional chunked transport (`HostStream`), and it fixed a deadlock in the package bridge — the package pushed to `__magpieChatPush` from inside its own hook while the reader stopped at the end marker, so the writer waited for a reader that had gone. The desktop client **does not implement `HostStream` yet**, so replies still arrive whole; `scripts/fetch-core.sh` has aligned `build/core` to that commit and `scripts/check-core-pin.sh` passes.

## Implemented

| Part | Contents |
|---|---|
| Wiring | Core is compiled into the same process; `internal/core` provides the runtime, the registry and the comic data API. On 2026-10-31 it was rebuilt against the **latest Core** (including `magpieplugin`, `host/sse` and the Magpie mobile API) and rechecked whole: `gofmt` clean, `go vet ./...` and `go test ./...` all pass — the desktop does no AI work, but Core's new packages must not break it. `build/core` is identical to Core's working tree (`diff -rq` finds nothing, excluding `.git`/`.github`/`apple`/`build`, the tests and `testdata`; `build/bin` is the gomobile toolchain, 9 MB, and copying it would only be waste). That tree is Core `519623d`, so `diff` plus the commit id answer "which Core is this" instead of memory. After wiring it in, `go test -count=1 ./...` and `go test -race -count=1 ./...` pass in all ten packages that have tests, with no data race (re-run on 2026-11-01: **eleven packages with tests**, all passing under `-race`; `internal/sourceapi` has no test files) (the race run used to cover only `./internal/...`, which was not the same set of packages as `go test`; they now agree). `-version` now prints the program's own commit and **Core's** commit: the latter comes from a `corestamp.go` written by `scripts/fetch-core.sh`, and without a stamp it says `Core unknown` rather than passing off its own commit as Core's |
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
| Magpie gateway | `internal/magpie` has the connection layer and an AI window (address, connection test, catalog, send, cancel; redraws throttled to 80ms; a tier picker maps simple/normal/complex to models or groups and shows where this request goes): `/v1/models` catalog with model/group IDs preserved, native bodies for three protocols, SSE streaming, `ctx` cancellation and Chinese diagnostics; ten fake-gateway contract tests pass under `-race`. **Wired into the interface on 2026-11-01 and run against the real installation on this machine**: a 53-model catalogue, one real stream through each of `chat`, `responses` and `messages`, and a real answer in the window. Still unverified: platforms other than Windows, cancellation timing, and what happens when Magpie has no route configured |
| Verification | A window of its own loads the challenge, checks Core's done rule, and hands back cookies and the User-Agent before the app retries |

## Verified with real sources and the real network

**One command reruns every real source**: `NK_LIVE_DIR=/path/to/scripts scripts/live-sources.sh` runs `TestLiveSource` for each `.js` in that directory (`NK_LIVE_REPEAT=3` repeats each source three times and lists one that passes some runs and fails others as `FLAKY` instead of folding it into PASS or FAIL), prints PASS or FAIL per source, and leaves logs in `build/live/<source>.log`. A failing source is a result, not a crash: a site that changed has to be distinguishable from a client that broke. On 2026-10-31 all five scripts present (comick, copy_manga, komiic, komiic2, manga_dex) passed.

**Four sources re-run on 2026-10-31**: with Core at `519623d`, `TestLiveSource` passes for MangaDex, Komiic, Comick and Copy Manga. A second run the same day sent Copy Manga back to HTTP 210 while the other four still passed — so its state is intermittent, not recovered; `live-sources.sh` reports the two kinds of result separately for exactly this case. Comick exposed an assumption in the test: its chapter had a single image while the case asked for `page=3`, the reader clamped it to 1 as it should, and the assertion failed — the test was wrong, not the code; it now asserts `min(3, len(images))`. The HTTP 210 that Copy Manga used to return no longer reproduces: that was temporary site behaviour, not a client regression.
**The reader was re-run after the pin change**: with Core at `c1db2c3`, `readercheck` read one Komiic chapter (53 pages) and reported `page title=TEMPEST figures=53 images=53 decoded=4` — 53 images in the DOM and 4 decoded in the current viewport. The capture does not contain the web layer, so the conclusion comes from the DOM and decode state, not the screenshot.

**One command runs the whole local gate**: `sh scripts/check.sh` runs `gofmt`, `go vet ./...`, `go test ./...`, `check-docs.sh` and `check-partitions.sh`, prints PASS or FAIL for each and exits non-zero if any failed. `check-partitions.sh` comes from the restore bug: the partition name `"comic:<key>"` may appear only in `core.ComicPartition`, and a second spelling fails the build — because that bug was a test written from the same wrong assumption as the code, and only a rule against the second spelling catches it.

**Run again on 2026-10-31** (`NK_LIVE_REPEAT=3`, three runs each): all five sources passed, Copy Manga 3/3. That strengthens but does not prove "stable" — one green 3/3 says this half hour did not reproduce it, and the earlier failure still happened.

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

**Cookie restore**: the official app has one jar; this client has one partition per source. The import screen has a "restore sign-ins (cookies)" checkbox, off by default — two `uicheck -settings -backup-cookies -settings-out` captures show both states rendering correctly — the ticked one is produced by `DevOptions.BackupCookies` setting the state directly, which proves the state-to-UI link; **the full import after a mouse click is still not covered by an automated check**. When ticked, Core's plan carries `cookieTargets` — sources present in the backup *and* installed here come first, falling back to every installed partition — and the client writes the cookies into those partitions without recreating a shared jar. A domain that is not a host (such as a junk row holding `https://…`) and a row that has **already expired** are dropped — Core has no clock, so the expiry decision belongs to the client; otherwise the summary would claim a session that cannot sign in; each cookie is only sent to a matching domain, so a copy in a partition that never visits that host is inert.

Backup import was checked with three `.venera` files: `backup-test` (1 source, 2 data keys, 2 search entries), `backup-shelf` (1 history entry, 1 folder holding 1 comic, 1 search entry) and `backup-pages` (1 image favourite that lands as 2 kept pages, since the favourite and the episode row each carry one). It was first checked with a `.venera` carrying the real `copy_manga.js`: the preview reported 1 source, 2 with data and 2 search-history entries; the import installed `copy_manga` 1.4.3 through the normal flow and wrote back 4 data keys, keeping `12345678901234567890` intact.

Browsing works on all three sources: MangaDex showed its Popular part as a list with covers, Copy Manga showed its 推荐 and 热门 parts, and Komiic's paging browse changed content between page 1 and page 2. The reader window really runs in the WebView: the page's title was the comic's, it built 53 figures, images decoded (the first at 757×1200), and `currentSrc` pointed at the `nkreader://` image URL (`http://nkreader.localhost` on Windows). Reopening the same chapter returns to where reading stopped: after scrolling to 1200 and reloading, `scrollY` was still 1200. A separate Comick check recorded 51 figures, 51 images, and 4 decoded images, the first at 750×1194, with `currentSrc` pointing at `http://nkreader.localhost/img?source=comick&…`.

### Verification (2026-10-30)

- The window alone: a local page that drops its marker and sets `cf_clearance` by script was recognised as passed in 3.5 s; the window's User-Agent came back and the cookie reached the partition's jar.
- The whole flow: a local site that answers a plain request with 403 (`cf-mitigated: challenge` plus challenge markers) made Core open the window; once the page passed, the cookie and User-Agent went back, and the retry with `cf_clearance=clear` got `200:PASSED-OK`, with the cookie in that source's own `comic:c` partition. It took 3.4 s.

## Problems found and fixed this round

**Cookies set on a redirect hop were never stored (2026-10-31)**: the `CheckRedirect` in `internal/core/transport.go` read `via[len(via)-1].Response`, but Go attaches the redirect response to the **new** request's `Response` field; entries in `via` keep it `nil`. So the code that was supposed to store the previous hop's `Set-Cookie` and send this partition's cookies onward quietly did nothing, while its comment said otherwise. The new test "a cookie set on a redirect hop must be stored" failed on the first run and exposed it; the fix reads `req.Response` and its `Request.URL`. On a real site this looks like being signed out after a redirect, which is hard to attribute.

**A scheme-relative image address could not be fetched (2026-11-01)**: a source wrote the image address of a chapter page as `//images.example.com/1.jpg` — the standard HTML form, with no scheme. RFC 3986 says such an address inherits the scheme of the page, and our client refused it with "image URL is not http(s)". The fix has two layers: the rule lives in Core (`host.NormalizeURL`, exported to gomobile as `Core.NormalizeURL`), and the client only decides *which* scheme to inherit — the desktop `Transport` remembers the scheme each **partition** last used, so a source on http does not lose its images because another source is on https. **This does not weaken the rules**: `file://`, `javascript:` and `ftp://` are never rewritten, the filled URL still has to be http(s), and a URL that is not scheme-relative is not touched. `internal/core/image_url_test.go` covers both points; **checked in reverse**: putting the fixed https back makes the test fail with `http: server gave HTTP response to HTTPS client`.

1. **The version number shut real sources out**: the app reported 0.1.0, while sources ask for `minAppVersion` 1.3.4 or 1.6.0, so they refused to install. The app's own version and the source API version are separate now; the latter reports 1.6.0.
2. **Core had no search options**: `search.load(keyword, options, page)` takes one value per option group, and without them a source pastes `undefined` into its URL. Core now reads `search.optionList` (`js.Runtime.SearchOptions`, `mobile.SearchOptions`) and the desktop app passes the defaults. MangaDex went from a protocol error to 20 comics.
3. **Spaces in URLs**: the MangaDex source pastes the keyword into `?title=one piece`; Go wrote the raw space and the server answered `PROTOCOL_ERROR`. Bytes a request line cannot carry are escaped now, as Dart's `Uri` does.
4. **Covers stayed grey**: the first frame ran before a cover arrived, and the decode cache remembered "no image" for good. Only decoded bitmaps and failed downloads are remembered now.
5. **A stale source list**: `Library.Sources()` read its own cache, so a source installed another way never showed. It asks Core every time now.

**A merged search reported a broken source as one with no results (2026-11-01)**: `searchEverySource` recorded each source's failure in `failed[i]` and **never read it**; the summary only counted `len(found[i]) == 0`, so a source that errored and a source that found nothing landed in the same bucket. The interface could only say "其中 N 个源没有结果", leaving the user unable to tell a bad keyword from a broken source. `quietSources` separates them now: running `goda` (working) together with `ykmh` (Cloudflare) for 火影 prints **"30 部,其中 1 个源出错"**, with all 30 cards shown and each carrying its source name; before the fix the same line read "30 部,其中 1 个源没有结果". Each branch is pinned by `TestQuietSourcesSeparatesErrorsFromNoResults`.

**The source settings panel verified in the real interface (2026-11-01)**: with `goda`, which declares three input settings (`domains`, `api`, `image`), the sources window listed "[goda] 的设置" with 域名 `godamh.com`, API域名 `v2.apikk.top` and 图片域名 `c-nd3-1.6wm.top`, each with a 修改 button; the same page showed "已安装 1 个源 · GoDa漫画 goda · v1.2.1" with its enable toggle, 设置 and 删除. Changing the domain to `godamh.example` produced a "已保存设置" notice, the panel showed the new value, and it was stored as `"settings":{"domains":"godamh.example"}` — only the changed key, with the other two continuing to use the source's declared defaults. That a stored setting **reaches the source** was verified separately at the Core level (changing `hot_manga`'s `image_quality` changes what `loadSetting` returns), so the claim here is limited to the interface reading, changing and storing.

**The browse view said "这个源没有分类浏览" for sources that declare explore (2026-11-01)**: both `goda` and `manhuagui` declare `explore` in their scripts, and both showed "这个源没有分类浏览". The cause is that `a.win.Update`'s callback **may run asynchronously**: the old code started `loadExplore` before the list had been stored, and that call advanced the `exploreGen` generation, so the list update was judged stale and dropped — leaving the page saying there were no browses. Storing the list and loading its first page now happen in the same callback, so the generation only advances once the list is committed. Checked against real sources afterwards: `goda` shows its tab and a 近期更新 list, `manhuagui` shows 热门漫画最新更新 with entries — so this was not one source's problem.

**Shelf, favourites and reading progress verified in the real interface (2026-11-01)**: with `goda`'s real data over the real network on Windows, each action was checked against both `shelf.json` and a capture:

- **Next chapter**: opening chapter 1 and pressing the reader's 下一话 recorded `chapterIndex: 1` and `chapter: "第2话 木叶丸！"` in `history` — the index moved with the title, not just the label.
- **Favourite + progress + keep this page**: pressing 收藏, then reading, then 收藏本页 left three records at once: the comic under `folders["收藏"].items`, `page: 27, percent: 47` in `history`, and the real image address for page 27 (`https://c-nd3-1.6wm.top/scomic/.../27.webp`) with its chapter name in `pictures`. The interface matched: a "已收藏第 27 页" note, the button reading 已收藏, "共 162 章 · 上次读到 第 27 页 · 47%" with 继续阅读, and a 读到 marker on the chapter row.
- **Shelf page**: all three sections rendered — 继续阅读 (cover, 第1话 · 第27页 · 47% · 1 分钟前), 图片收藏 · 1 张 (thumbnail and 回到这一话), and 收藏 · 1 部 (打开, 取消收藏, 重命名, 删除).

Captures: `ui-fav.png` for the detail view's three states and `ui-shelf-real.png` for the shelf. **Boundary**: this covers Windows and this machine's real network only; nothing here covers macOS or Linux, and touch gestures are not covered.

**A merged search clipped the source name off its cards (2026-11-01)**: in that same capture the first row of cards drew their source name and the rows below did not. A subtitle pushing the label out was ruled out first — all 12 results have an empty `subtitle` — and raising the grid cell from 240 to 260 then to 275 brought the names back, so the cause is that **the cell could not hold cover 180 + title + source name** and the taller row clipped the name. A result then looked like it came from nowhere. The value is 260; 275 only shows fewer rows.

**The AI window's 默认模型 field was dead, so every send was a 400 (2026-11-01)**: pressing send in the real interface answered "Magpie 拒绝了这个请求(HTTP 400)". Two things stacked. First, `a.ai.model` — the 默认模型 field — was only ever assigned when settings were read and **never written back**, so a typed model was worth nothing. Second, `ModelFor` documented an empty model as "the gateway's default route", while this Magpie answers `invalid request: model must be a nonempty string`: the assumption itself was wrong, and a request built on it can only be refused. Now the field is written back for both 测试连接 and 发送 (`applyAIFields`, trimmed), and `Session.Ask` returns `ErrNoModel` when nothing is chosen, which the window shows as "先点读取模型，把这次要用的模型或组填进档位或默认模型" instead of sending a request that cannot succeed. **Checked in reverse**: removing the write-back line makes `TestTheTypedModelReachesTheRequest` report `model = "", want what was typed`.

**Interface evidence for the AI window against the real Magpie (2026-11-01)**: `cmd/uicheck` gained `-ai` / `-ai-test` / `-ai-model` / `-ai-prompt` / `-ai-out`, which open the AI window — a window of its own, so it needs its own capture — and drive the real flow. Result: address `http://127.0.0.1:3425`, `Magpie 可用,共 53 个模型或路由组`, and the catalogue listing `group/laincat`, `group/laincat-subagent`, `auto/gpt-6-luna`, `auto/deepseek-v4.1-flash`, `auto/minimax-m3`, `auto/deepseek-v4-pro`. Sending through `group/laincat` produced a real streamed answer in the window: "我收到了你的消息。". Captures `ui-ai-magpie.png` and `ui-ai-answer2.png` are kept with the evidence.

**The checking tools reported a different source API level than the app (2026-11-01)**: after `main.go`'s `sourceVersion` moved to 1.6.3, `cmd/uicheck`, `cmd/readercheck`, `cmd/importcheck` and `cmd/verifycheck` each still hardcoded `AppVersion: "1.6.0"`. The consequence was concrete: the two sources gated on 1.6.3 (`cxc`, `mangabz`) install in the app and were refused by the version gate in the tools — **so a tool could report the wrong verdict about the product**, which is exactly the trap this round hit. The number now lives in one place, `internal/sourceapi.Level`; the app and all four tools use it, and `main.go`'s `sourceVersion` is an alias for it.

**A real Magpie installation now answers (2026-11-01)**: "send a request to a real Magpie" had been an open item. `magpie-windows-amd64.exe` runs on this machine, and the gateway listens on `127.0.0.1:3425`, which is what `internal/magpie.DefaultAddr` names (the same process serves a sign-in-gated web UI on 3430). The new `internal/magpie/live_test.go` only runs with `NK_MAGPIE_LIVE=1` and does the whole path against the real gateway: **53 models** (first is `group/laincat`), a real completion, and one stream through each of `chat`, `responses` and `messages`, with all three replies exactly `ready`. That also shows the catalogue read, the protocol choice, the SSE parsing and the end-of-stream decision working on this machine.

**The `responses` protocol appended the answer twice (2026-11-01)**: the verification above first came back as **`readyready`**. Reading the real event stream explained it: the gateway sends `response.output_text.delta` (field `delta:"ready"`) and then `response.output_text.done` (field `text:"ready"`, the **finished text**). The parser read both `delta` and `text`, so the same text entered twice. Only `delta` is read now; `text` in a finishing event is the finished answer, and taking it again copies the reply. **Checked in reverse**: putting the `text` fallback back makes the unit test produce `text = "readyready", want "ready"` — exactly what the real gateway showed.

**Desktop verified in the real interface (2026-11-01)**: until now there were only headless tests, and the interface itself had never been shown to complete a run against a real source. This round ran two of them on **Windows**, over the real network, with `cmd/uicheck` and `cmd/readercheck`:

- `goda`: searched 火影, the detail view showed the cover, the author 集英社, the tags 冒险/战争 and **162 chapters**; `readercheck` opened chapter 1 and found **57 pages**, the reader rendered 57 `figure`/`img` nodes and decoded 4 (the first at 623×960). Captures `ui-real-goda.png` and `ui-real-reader.png` are kept with the evidence. After a chapter had been opened, the interface showed 读到 and 继续阅读.
- `ehentai` (a gallery source with no chapter list): searched `naruto`, the detail view showed the cover and tags. Before this it said "共 0 章 / 没有匹配的章节" — **misleading wording for a work that is a single reading unit**; it now says "整部作品是一个阅读单元" and the meaningless chapter filter is gone. `-chapter 0` drives `openReader(0)`, and the shelf's `shelf.json` recorded `chapterIndex: 0`.

That also fixed a blind spot in `cmd/uicheck`: its `-chapter` was gated on `n < len(d.Chapters)`, so a gallery work could never reach the reader and those sources could not be checked in the interface at all. With an empty chapter list `n == 0` is now allowed, on the same path the 开始阅读 button uses.

**What this evidence does not cover**: the captures show that the interface draws, lists chapters, renders real images in the reader and records a position; they do **not** cover the feel of scrolling through a chapter, and nothing here covers macOS or Linux.

**`kuaikan` hits the engine's register limit (not fixed; it belongs to the MoeJS compiler)**: the source hands the site's inline decoder to `Function(...)`, and MoeJS's bytecode compiler has a hard limit of **256 registers** per function; past it the error is `SyntaxError: function needs more than 256 registers is not supported yet`. That number is a constant in `compiler/compiler.go` with no switch, so lifting it means adding register spilling to the compiler — engine work, outside what the client can fix. The same sweep applies to `manhuaren`, `one_kkk`, `webtoons_zh_hant` and `cosplaytele`: all four reported "search found nothing" in our first patrol, which used 火影, and all four pass with the fixed samples vensrc patrols with (斗破苍穹 / love / girl), so those were **keyword mismatches**, not defects.

**A compressed response was not decoded when the source set `Accept-Encoding` itself (2026-11-01)**: the JM sources send `Accept-Encoding: gzip, deflate, br, zstd` explicitly. Go's `net/http` only decompresses transparently when it **added** gzip itself; once the caller sets that header, it hands the body back untouched. The source then parsed gzip bytes and `JSON.parse` failed on the first byte, `0x1f`, while the source wrapped that exception in its own "后台初始化失败: Error: HTTP 404" — **an error message with no relation to the actual cause**, the hardest kind to chase. `Transport.Do` now narrows the declared encodings to the two it can actually decode (`gzip`, `deflate`), decodes according to `Content-Encoding`, keeps the decoded body under `maxBody` (a compression bomb cannot get around the limit by being decompressed), and removes `Content-Encoding` and `Content-Length` afterwards so a caller does not read a decoded body as a compressed length. **Checked in reverse**: with decoding turned off, `jmcomic` returns to exactly that "HTTP 404" failure; with it on, the source passes.

**`rumanhua2` cannot reassign `atob` (engine differs from the specification; not fixed)**: the source's decoder takes the global object with `Function("return this")()`, supplies `atob` when it is missing, and writes the old value back in `finally`. Per the web specification `globalThis.atob` is **writable and configurable**; MoeJS makes it read-only and non-redefinable to keep its shared frozen builtin template, so that write-back throws `TypeError: Cannot assign to read only property 'atob'`. This was verified: changing that source's write-back to "restore only when it supplied one" makes search, details, 546 chapters, image fetch and the reader page all pass, so the source itself is fine. There are two ways to fix the engine — turn on `MutableIntrinsics` globally (measured at 3.03 ms per runtime against 2.60 ms, and the process-wide shared frozen template is lost), or make MoeJS set `atob`/`btoa` writable and configurable as the HTML standard says. Of all 105 sources only this one writes a frozen builtin global (`cxc` writes `URL`, which is not on its failing path), so **the cost does not match the benefit and the engine was left alone**; this is recorded as a known limitation.

**What a full sweep of the third-party sources actually found (2026-11-01)**: all 71 vensrc scripts, **59 pass and 12 fail**. This is not comparable with the earlier "34 passing", which counted two different things as client failures: "the sample word has no results on this site" (`cxc`: 0 for 海贼, 4 for 魔氣, the work being titled 航海王 there) and "the source offered callbacks the client dropped". Fixing those turned 14 and then 9 sources green. The 12 that remain are each placed in a category:

| Category | Count | Sources | Verdict |
| --- | --- | --- | --- |
| The image address itself errors | 7 | `311`, `baozi`, `baozi_fixed`, `haokan`, `manga51` (cover only), `mh03`, `mh1234_fixed` | site/CDN. **Checked directly**: `baozi`'s image URL is 404 to curl as well, and `manga.5um.net` (`311`, `mh03`) fails even curl's TLS handshake |
| Site search 404s, or its front page is empty too | 3 | `guazi`, `isamanhua`, `sunday_webry` | the site moved or is unavailable |
| Version gate | 1 | `recode-jm` (`minAppVersion 1.16.0`) | requires a newer app than the reference; refusing is right |
| Runtime locks a JS builtin as read-only | 1 | `rumanhua2` | dependency issue, below |

**The six "encrypted/scrambled image" sources and the one "mirror fallback" source all pass now**: with the desktop client wired to Core's image callbacks, `cxc` (659,568 bytes of ciphertext → a real 494,647-byte PNG), `ccc`, `mojoin`, `mojoin_fixed_v107`, `manwaba_fixed`, `manwaba_fixed_v113` and `4khd_cosplay` (a 1,243-byte HTML error page → 248,428 bytes of webp from the next mirror) all pass. The reader was checked separately: `cxc`'s work rendered all 3 pages with `decoded=3` and a first image of 690×1380 — what came out decodes as a picture, not just bytes that look plausible.

**Upstream has not fixed it either (re-checked 2026-11-01)**: the tags of `MoeForge/moejs` stop at `v0.1.0-alpha.11`, and `main` has a commit newer than the one pinned here (`v0.1.0-alpha.11.0.20261030083738-b9fa0469d1df`, 2026-10-30). A standalone probe — no repository touched — ran the same `Object.getOwnPropertyDescriptor(globalThis,'atob')` plus a strict-mode assignment against that newer version and got **`writable=false` and `Cannot assign to read only property 'atob'` again**. So this is not "the pinned version is too old": it is what the runtime does today, and either upstream changes or the engine does.

**`rumanhua2` exposes a fidelity problem in the runtime**: it restores `root.atob` in a `finally` block, and **moejs defines `atob`/`btoa`/`TextDecoder` and friends as non-writable and non-configurable**, so an assignment inside a class method (strict mode) throws `TypeError: Cannot assign to read only property 'atob'`. A probe (`Object.getOwnPropertyDescriptor(globalThis,'atob')`) reports `writable=false, configurable=false`, and the same holds for `atob`, `btoa`, `TextEncoder`, `TextDecoder`, `structuredClone` and `Intl` — while in browsers and Node these are ordinary writable, configurable properties. Being non-configurable, Core's JS layer **cannot** change them; the fix belongs to the runtime or a different implementation. Recorded rather than widening a dependency on the spot.

**`hot_manga`'s image-quality setting does nothing — its own regex is stale (2026-11-01, located)**: this one had been open for several rounds. Changing `image_quality` from the default to a lower value left the first image address of a chapter **byte for byte identical** (both `...jpg.h1500x.webp`). The source rewrites quality with `url.replace(/\.jpg\.h\d+x\.jpg$/, ".jpg.h{N}x.jpg")`, **and the site now returns `.webp`**, so the pattern never matches. Checked that our side is not the problem in the same breath: the same `Store.SaveSetting` → `loadSetting` path does take effect on `manga_dex` (`/data/` → `/data-saver/`). So this is a source fix — the pattern needs `.((jpg)|(webp))$`.

**Only one decoder was registered, so the script silently did not run (2026-11-01, fixed)**: after wiring `modifyImage`, a real run against `shonen_jump_plus` still returned `image/jpeg` — the script was there (548 characters in the config, `req.ModifyImage` non-empty). The build imported only `_ "image/png"`, so **no jpeg or gif decoder was registered** and `image.Decode` reported an unknown format for a JPEG, which took the "cannot decode" path. With both blank imports added, the same page turns from 232,750 bytes of JPEG into 879,725 bytes of PNG — that source uses a script to undo a page cut into shuffled blocks. `TestModifyImageRunsOnJPEGBytes` pins it with JPEG input and a rotate script: the result must be `image/png`, or the script did not run.

**A format this build cannot read must pass through, not fail the page (2026-11-01)**: `hitomi` serves avif, and only png/jpeg/gif/webp decoders are registered — so once a source pairs such a format with a `modifyImage` script, the decode failure took the whole page down. "This build cannot read that format" is now kept apart from "the script failed": the former returns the server's bytes and content type unchanged (the reader can still show the picture), the latter still reports. `TestModifyImagePassesThroughAnUnknownFormat` pins it with an avif header.

**`modifyImage` is wired up (2026-11-01, pin `6aa9356`)**: Core adds `Runtime.ModifyImage` (decoded RGBA into a throwaway bitmap store, the source's script runs in its own runtime, the result bitmap comes back; decoding and encoding stay with the client), and the desktop client uses it on the image path: fetch → `onResponse` first if present → `modifyImage` decodes to RGBA, runs the script and re-encodes as PNG, with the content type changed to `image/png` to match. In the config `modifyImage` is a **string of JavaScript, not a function**, so it travels as `ImageConfig.ModifyImage` rather than as a callback handle — a mistake made and pinned down with a test. Deterministic check: a source returns a 4×2 red/blue picture plus a clockwise-rotate script, and what arrives is 2×4 with red on top; a script that returns no image reports the failure instead of falling back to the original. Against real sources `hitomi` and `manhuagui` pass (their images are mostly avif/webp, which need not trigger the script).

**Removal was failing silently: a missing fourth argument (2026-11-01)**: six official sources' `addOrDelFavorite` takes a fourth `favoriteId` — the source's own handle on the favorite entry. `wnacg` builds its delete URL from it:

```js
Network.get(`${this.baseUrl}/users-fav_del-id-${favoriteId}.html?ajax=true...`)
```

Passing only three arguments sent the delete to `del-id-undefined`, which still answered 200 and the source still reported success — **it looked removed and was not**. That kind of "succeeds but does nothing" is harder to notice than an error.

Core now parses `Comic.favoriteId` and `details.favoriteId` (either may carry it), and `SetFavorite` always passes the fourth argument: the handle when known, `null` otherwise, which the sources that do not use it ignore. A recording fake source pins both shapes: `["c1",null,false,"fav-77"]` and `["c1","f2",true,null]`.

**Boundary**: a logged-out source returns no `favoriteId`, so a real `wnacg` removal cannot be verified on this machine — the unit tests are the evidence.

**Favorite to the source: from read-only to add and remove (2026-11-01)**: the previous round only *listed* a source's favorites; the details page had no way in. It now carries 收藏到源 beside 收藏 (the local shelf): it shows whether the source supports them, offers a folder when there are several, and turns into 已在源收藏 — pressing again removes — when the comic is already in the chosen folder. Whatever the source throws is shown in its own words.

Three deliberate choices:

- **Two lists, two controls**: the local shelf and the source's favorites are different data, so they keep separate buttons instead of one merged 收藏 key.
- **A single-folder source gets no folder**: measured on the live sources, `wnacg` answers success for both add and remove without a session (the source accepts it), while `nhentai` answers `Authentication required. Please login again or provide API Key in source settings.` — the client repeats what it is told rather than deciding which is "right".
- **A source with only a folder toggle** (`ehentai`) gets no add control on the details page, because it has no list to read and nothing would show the comic afterwards.

The unit test pins the argument shape: a single-folder source must receive `null`, not an empty string; add and remove are both covered. The capture confirms the button and the folder picker render.

**Another full reading sweep (2026-11-01, pin `cf9eb9b`)**

All 34 sources again: **20 passed / 11 failed / 3 need a server**. Two things moved, neither a client regression:

- **`jm` now passes consistently** — the BOM tolerance from the previous round did it; before, it could not even finish `init`, and now search, details, chapters and image fetch all complete.
- **`manga_dex` failed this time**: `api.mangadex.org` answers **302 → "Redirecting to bot challenge page"** today, and the source hands that HTML to `JSON.parse`. That is bot protection on the site, not a broken transport — our challenge verifier only fires on a Cloudflare-style 403, so **a 302 to a challenge page is not covered**, and the honest conclusion is "the site changed", not "the client works".
- **`ikmmh` returned zero results and an empty front page** this round after passing the last one — the clearest example yet of why one round is not a conclusion.

`manga_dex` deserves its own evidence — the response measured today is:

```
HTTP/1.1 302 Found
location: https://mangadex.org/edge/challenge?src=…
content-type: text/plain
server: MangaDex

Redirecting to bot challenge page... (if this isn't a browser, don't send a browser UA)
```

So it is neither a 403 nor a `cf-mitigated`: it is MangaDex's own edge challenge, a 302 to a text notice. Our detection is deliberately conservative (only 403/429/503 + `Server: cloudflare` + page markers) and **does not cover this shape**; treating a 302 as a challenge would need a new challenge kind and completion rule (today only cloudflare and captcha exist, and neither can say "left this path"), which is not a one-line judgement. Recorded as **a new mechanism on the site that the client does not cover**, rather than half-built.

The rest are unchanged in kind: `picacg` needs a login, `mh18`/`mycomic` are Cloudflare, `baozi` is 403, `comic_walker`/`shonen_jump_plus` return HTML, `happy`'s search answers EOF, `jcomic` moved its image addresses into a `data-locked` attribute that needs JavaScript to decrypt, and `mh1234` times out.

**"2 replies" was a number that did nothing when clicked (2026-11-01)**

`replyCount` in a thread was text only. Reading the replies means **calling `comic.loadComments` again** with `replyTo` set to that comment's id — both clients only ever read the first level.

- The desktop turns "N replies" into a button that loads that level, with "回复 X 的评论" and 返回全部评论 above it; going back to the thread passes `replyTo` as `null`, not an empty string, because a source tests for null.
- A test pins it: null reads the thread, an id reads the replies, and the content tells them apart.
- The capture `E:/Temp/nkprobe/ui-comments-thread.png` shows the count as a button.

**Not done / not verified**: the same wiring just went into iOS, with syntax and key checks only, and a real source's reply level needs a login to read — so none of this is claimed as verified against a live thread.

**Comments could be written but not voted on (2026-11-01)**

Four sources declare `comic.likeComment` (`ccc`, `lanraragi`, `picacg`, `zaimanhua`) with the contract `likeComment(comicId, subId, commentId, isLike)` — **the same call with `false` takes the vote back**. The core layer had it all along; the window had no button.

Each comment now carries 赞 / 取消赞, shown only when the source declares `likeComment`, and the page is read again afterwards because a source may only report the new score on the next read.

**Verified live**: `ccc` and `zaimanhua` both declare the vote (`canLike: true`). Their comment lists need a login to read, so **voting end to end still needs a real account** and is not counted as verified.

**A pasted link of a source's own was searched as a keyword (2026-11-01)**

Eight sources declare `comic.link` (domains plus `linkToId`) to turn an address like `https://nhentai.net/g/691529/` into a comic id. Neither client used it: the search box sent the whole URL as a keyword and of course found nothing.

- The core layer gained `LinkDomains` (read with `Runtime.Value`, because `comic.link` is an object and calling it only answers "source does not define this function"), `LinkToID`, and `OpenLink`, which matches the host and asks each source that claims it — the first one that answers wins.
- The desktop sends an address-shaped query through `OpenLink` first and **opens the comic** when a source recognises it; when none does it says so instead of silently searching for a URL.
- "Looks like an address" is not just "has a dot": `12.5` and `v1.2` must stay keywords, or every search spends a round of source calls for nothing.

**Verified live**: an `nhentai` link resolved to id `691529` and the details opened (title, cover and tags all from the source); the capture is `E:/Temp/nkprobe/ui-link-open2.png`. A comic id that does not exist answers 404 honestly rather than pretending to open.

**Not done**: iOS has no `linkToId` wiring yet, and `enableTagsTranslate` is still unwired.

**A tapped tag stopped halfway when it pointed at a category (2026-11-01, pin `0c721f3`)**

Six sources declare `comic.onClickTag` and the desktop has used it for a while, but **the "category" branch never worked** — a tap only said "this version cannot open category parameters yet". On iOS the tags are not even tappable.

The desktop now works: when the source maps a tag to a category, the category tab opens **with the parameter the source named selected**; a parameter that is not in the current declaration is stated plainly rather than silently falling back to the first value. A fixture pins both cases (value 0 is selected; an unknown parameter produces a message), and the exercise recorded two real contract details: the declared array is `categoryParams`, not `params`, and the declaration object is `category` while `categoryComics` only loads it.

Core gained `ClickTag(callID, key, namespace, tag)` so the phone gets the same semantics — including an empty action when the source has no opinion (not an error) and the source's own words in the log when it throws.

**Not done**: tapping a tag on iOS is still unwired (`LinkToId` and `enableTagsTranslate` are unwired too).

**Twenty sources have comments and the client had no way in (2026-11-01)**

**Twenty official sources declare `comic.loadComments` and thirteen declare `comic.sendComment`**, and neither client ever called either — for those sources the thread did not exist.

- **The path was wrong at the start**: the handlers hang off the source's `comic` object (`comic.loadComments`), not off the source itself. Looking for a top-level `loadComments` reports "no comments" for every source that has them.
- The core layer gained `CommentsSupported` / `CanSendComment` / `CanLikeComment` / `Comments` / `SendComment` / `LikeComment`, and `Details` carries `subId`, which some threads are keyed by; an empty id is passed as `null`, not an empty string.
- The details view has a 评论 button (hidden when the source declares none) and the thread opens in its own window: list, pages, reply to someone, send, refresh. `canSend` is read once when the window opens — putting it in the render path would run a source call on every repaint.

**Verified live**: `komiic`, search then details then comments, returned three real comments; the capture is `E:/Temp/nkprobe/ui-comments-komiic.png`. `nhentai` declares support but returned nothing while logged out — the account dimension, not a client fault.

**Not done**: iOS has no comment interface yet (this round is the desktop), and there is no Mac build.

**The browse sweep is one command now (2026-11-01)**: `NK_LIVE_DIR=/path/to/scripts scripts/live-explore.sh` runs `cmd/explorecheck`, which lists each source's browses and loads the first page of each; failures are reported per source, sources needing a server are listed separately, and `NK_LIVE_REPEAT=2` re-runs to tell an intermittent site from a stable failure.

The desktop's `internal/core` also lost `exploreType`, `parseParts` and `parseOrderedParts`, which nothing called any more — leaving them would invite the next reader to copy a second implementation out of them, and "one implementation per client" is exactly why the iOS browse page was broken.

Latest full sweep (34 sources): **23 passed / 8 failed / 3 need a server**.

**Browse normalisation moved into Core so the two clients cannot drift (2026-11-01, pin `c855fd3`)**: the iOS gap recorded above (building `explore[i].load` itself) came from the same logic existing twice. The four shapes and the `load`/`loadNext` fallback now live in Core's `comic/js`, and the desktop only maps each comic into the shape its interface draws.

A second full browse sweep afterwards: **23 passed / 8 failed / 3 need a server**; `ehentai`'s three pages work and `mh1234` passed this time (the earlier run was a site timeout, which is why one round is not a conclusion).

**The browse tab had never been tested live, and for one class of source it could not work at all (2026-11-01, pin `110b0d8`)**

All 34 official sources declare `explore`, and the live test only walked search → details → chapter: the browse tab was never looked at. `cmd/explorecheck` lists each source's browses and loads the first page of each; the first run gave **24 passed / 10 failed**, and `ehentai` failed all three of its browses with `source does not define this function`.

The cause is a **shape in the contract**: the official API lets a browse use `load(page)`, or `loadNext(next) → {comics, next}`, which pages by cursor. `ehentai` declares only the second, and the client only ever called `load` — the browse exists, and the client asked for the wrong function. Five sources use `loadNext`: ehentai, hitomi, jcomic, manhuagui, manhuaren.

- The desktop's core layer gained cursor support: `Explore` falls back to `loadNext` when `load` is missing, `ExploreResult.Next` carries the next token, and 下一页 pages by cursor without pretending to know a page count.
- Along the way, one behaviour that multiplied the damage: **without a UI handler, a UI call that returns nothing (showMessage / launchUrl / cancelLoading) no longer fails.** `ehentai`'s logged-out "watched" browse shows a message and then returns an empty page; that used to make the whole call fail and now returns what the source meant. A dialog that needs an answer still fails — its answer decides what the source does next, and it cannot be invented.

After the fix all three `ehentai` browses work (25 + 77 + 0 galleries, the login-dependent one empty as the source intends). `kavita`/`komga`/`lanraragi` are SKIP in the probe (they need a server the user runs), no longer mixed into the failures.

**The same gap existed on iOS (2026-11-01, pin `7019844`)**: mobile's `CallComic` also recorded nothing on failure; it now writes the same shape the desktop does (source key, call path, cause). Verified negatively: turning that check into `if false` makes the test fail. The desktop only moves its pin.

**The panel was empty exactly when it was needed (2026-11-01)**: pointed at a real failing source (`jcomic`), the capture showed **0 lines** — because that source logs nothing and the failure is Core's (`image URL is not http(s)`). The panel only helped when a source used `console.log`, which is the opposite of the moment a user opens it.

- A failed `Core.Call` now records a line: the source key, the call path and the cause. "The search failed" is no longer only an alert that the next screen covers.
- An image address that is not http(s) records **the string it got**, quoted, so an empty one is visibly empty. The reader used to say only "not http(s)" with no hint what the source had returned.

Two tests pin it: after a source throws, the last line carries the key, `search` and the cause; and a rejected empty image address leaves that source and message in the buffer.

**What the source itself said was never visible (2026-11-01)**: during the live round I could read `Failed to convert base64 ...` because I ran the app from a terminal. A user who launches it from a launcher **never sees that line**. All they get is the message the source threw next — "Cannot read properties of null" — with the real cause hidden, which is exactly the answer to "is the client broken or did the site change".

- Core now tags each source log line (a source's `console.log` and Core's failures on its behalf) **with the source key**, and keeps the last 400 in memory. `Services.LogSource` is optional; a host that only sets `Log` behaves as before.
- The desktop gains a 源日志 panel: time, source key, level and text; errors red, warnings amber; 复制全部 for a bug report, and 清空.
- The buffer is bounded: a source logging in a loop cannot grow the process (a test logs 500 lines and confirms 400 are kept and that the **newest** survive).

Visual evidence: `E:\Temp\nkprobe\ui-source-logs.png` (5 lines, including a base64 failure correctly attributed to its source), and `E:\Temp\nkprobe\ui-source-logs-baozi.png` for a real site — `Core: search.load: js: Invalid status code: 403`, attributed to `baozi`.

**Live source round N: one BOM took a whole source down (2026-11-01, pin `a93ac09`)**

Re-ran all 34 official sources: **20 passed**, 11 failed, 3 need a server the user runs (counted separately as SKIP). Reading each log turned up one real problem:

`jm`'s init fetches a base64 blob from its domain-list endpoint. That response now begins with a **UTF-8 BOM**, so decoding failed at **byte 0** → `Convert` logged and returned null, as the official client does → the source received null, `JSON.parse(null)` produced null, and it died on `null.length`. The error the user saw was "Cannot read properties of null (reading 'length')", and the source's own **fallback domain list never ran**.

The fix drops only what no base64 alphabet contains: a BOM and ASCII whitespace. Valid input is unaffected and garbage still fails. Afterwards `jm`: install ✓, browse ✓, details with 62 chapters ✓, chapter 1 with 118 images ✓, first image really fetched at 10,098 bytes of webp ✓; only the paging endpoint still answers 500 from the site.

The rest are classified by dimension, and none of them points at the client:

| Source | Category |
| --- | --- |
| `picacg` | needs a login |
| `mh18` `mycomic` | Cloudflare challenge |
| `baozi` | 403 |
| `comic_walker` `shonen_jump_plus` | HTML instead of JSON |
| `happy` | the search endpoint answers EOF |
| `jcomic` | **site change**: page has `src=""` and the real address moved into `data-locked`, which needs JS to decrypt (confirmed by fetching the page) |
| `mh1234` | connection times out here, and the source assumes `#total-page` always exists |
| `jm` | paging endpoint 500 (the init crash is fixed) |

**The export could write a file it cannot itself read (2026-11-01, pin `9013ca5`)**: the reader caps every file under `comic_source/` at 16 MiB; the writer checked the script and **never the `.data`**. So a source holding a lot of data exported successfully into a file Core's own reader refuses to open — the worst kind of failure, because the user believes the backup is good until the day they need it.

- `MaxWriteDataBytes` matches the reader's `MaxSourceBytes`; going over is an error that **names the source**.
- The desktop turns that into one sentence: 这个漫画源存的数据太大,写进备份后会读不回来,所以没有导出.
- Another test **builds an over-cap member by hand** and confirms the reader refuses it — evidence that the cap is a real boundary rather than a guess.

**The source's settings did not travel either — now they do, and land back in their own bucket (2026-11-01)**: the previous entry fixed `.data` (what a source stores for itself), while each source's **settings** (what the user picked in the settings panel) still stayed behind.

The root of the mismatch is shape: the archive has **one object per source**, and this client splits that into `data` and `settings`. Merging on export is easy; the hard part is knowing, on the way back in, which key belongs to which bucket.

The answer is the source's own word: **a key it declares under `settings` is a setting, everything else is data.** Import is now `Library.ImportSourceValues`, which reads `core.Settings(key)` for the declared keys and routes each value accordingly; a source that fails to load sends everything to data, which is exactly what the old importer did, so it cannot be worse. Export merges the two buckets into one object, with data winning a name clash because data is what the source wrote for itself.

The verification: a source that declares `settings = {domain: ...}` and also stored a `token` is exported and imported into an empty client; the `token` lands in the data bucket, `domain` in the settings bucket, **and a reverse assertion confirms `domain` was not filed as data**. This also fixes the older problem of the official app's settings being dropped into our data bucket.

**The export carried the code but not the source's state (2026-11-01, pin `e46c8f9`)**: the previous export wrote only `comic_source/<key>.js`, while the format also holds `comic_source/<key>.data` — what a source keeps between runs (tokens, the domain it settled on). The reader always parsed it and the import always wrote it back; **only the writer never produced it**. The result was "the restore worked, and the source woke up logged out".

- Core's `WriteSource` gains `Data`, written as `<key>.data` so the reader's pairing by key still holds.
- The desktop takes each source's whole data object from the store.
- On iOS Core cannot enumerate the host's store, so `ExportBackup` receives `{"<key>": {"<name>": <value>}}` from the client, and the kit collects it through `SourceStore.allData(key:)`.

The verification is **two closed loops**: a client that has a source exports, an empty client imports, and the source is there; and a source that stored `token` exports, is imported elsewhere, and `LoadData` returns the original value.

**Still not carried**: each source's **settings** (a different bucket from its data in this client). Merging them into `.data` would make the importer write settings back as data — that is not a fix, it is the same mistake in a new place — so it stays open for its own round.

**A backup could come in but not go out; now it can (2026-11-01, pin `ffdfbe7`)**: Core could always **read** a `.venera` backup and never write one, so a user could move into this client and not out of it.

- **Core** gains `backup.Write`, which follows the format the reader already defines: `comic_source/<key>.js` plus `appdata.json`, with a fixed key order so two exports of the same data are **byte-identical** — which is what makes "did anything change" answerable.
- **Deliberately absent**: `cookie.db` (per-source credentials, and an export is a file people mail to themselves) and `history.db` / `local_favorite.db` (the official app's SQLite schemas; this client's shelf is a different format, and half a database of someone else's schema reads as corruption).
- **The desktop** puts 导出 .venera 备份 beside the import button, writing sources and search history, with "不含 Cookie" stated on the button's own row.

The verification is a **round trip**: the file is written, then read back through Core's own reader, checking sources, script text, settings and history — and that the archive contains **no** `cookie.db`; another test pins that two exports are byte-identical.

**Boundary**: a Venera-compatible client can read the sources and settings out of this file, but it is **not a clone** of the official app's data (shelf and image favorites do not travel), and whether the official app reads our export **has not been tested**, because it is not on this machine.

**The image a source hands over (a captcha, a QR code) was dropped (2026-11-01)**: the second argument of `UI.showInputDialog(title, image, validate)` is a **picture** — a source uses it to show a captcha or a QR code. The desktop ignored it, so the field was missing something the source needed.

The picture is fetched through the **source's own cover rules** (`onThumbnailLoad`), so the source's headers and any image rewriting apply here too. The bytes are kept and decoded by the dialog itself rather than in the main window's image cache: that cache is read by the main window's thread and written here, so sharing it would be a race.

Three traps on the way, all worth recording:

1. The first version wrote the bytes into the main window's cache and read them back there → a race, and the window did not repaint;
2. With the bytes kept by the handler the picture still did not appear → the log showed the **paint did rerun and the bytes did arrive** (`picture=true`), so the problem was layout: a bare `Height(96)` inside a `Fill()` column can work out to no width, and the image vanished;
3. An explicit `Width(200).Height(96)` fixed it, and the picture is visible in the capture.

The picture is httpbin's test image rather than a real source's captcha, but fetching, authenticating, decoding and showing it are the same code a source drives.

**The dialog window's layout has been looked at too (2026-11-01)**: the section above only showed that a callback completes, not **what the window looks like** — those windows in `jsui.go` were written from scratch with no picture behind them.

The capture tool gained `-dialog`: it opens a real dialog through the same handler (two buttons), with auto-confirm off, photographs it and closes it. The first capture had the buttons jammed into the bottom-left corner with the content area unused; they now sit at the trailing edge, the way a desktop dialog does, and the second capture confirms the layout. The dialog is synthetic rather than one a source opened, but the window, the buttons and the callback path are the same code.

**A panic in a dialog callback no longer hangs the script (2026-11-01)**: the buttons in a dialog run **the source's own code**. If `Invoke` panicked, the goroutine that owed a result was gone while the caller stayed blocked inside `ShowDialog` — the source was wedged, and the window just sat there. `invokeSafely` now turns a panic into an error handed back to the source, the same path as the source's own `throw`; four tests cover a panic, the source's own error, a missing callback, and a panicking validator.

`ShowLoading` also became visible: it only kept a count before, with nothing in the window. While a source holds a loading id the main window shows 源正在处理… with a spinner, and it disappears on cancel (cancelling twice cannot drive the count negative — also tested). The handler also guards `app == nil`: it can exist before the window and outlive it, and a source's call should not crash because of that.

**Re-checked inside the real process, and one potential deadlock fixed (2026-11-01)**: the table above used a test handler, which does not prove `jsui.go` itself works. The capture tool grew `-callback <key>`: it runs `settings.<key>.callback` against the same Core **inside the real App process**, with the real windows, threads and handler.

```
$ go run ./cmd/uicheck -script wnacg.js -callback refreshDomains
nekophage: callback refreshDomains -> "null" err=<nil>
$ go run ./cmd/uicheck -script happy.js  -callback wipeCache
nekophage: callback wipeCache -> "null" err=<nil>
```

Writing that found a **potential deadlock in my own code**: the auto-confirm path called `actions[0].Invoke()` on the JS thread, which is the caller already blocked inside `ShowDialog` — the callback would re-enter the engine from inside itself. It now runs on another goroutine and waits, the same shape as a real button press.

**The desktop can finally give a source a UI (2026-11-01)**: Core has always defined a `UIHandler` (message, dialog, input, select, loading), but **the desktop never provided one**, so any source calling `UI.showDialog` or `UI.showMessage` got `No JavaScript UI handler provided`. That is not cosmetic: several sources do their work from inside a dialog callback.

`internal/ui/jsui.go` supplies it now. `ShowDialog` builds a window from the source's own buttons, pressing one runs **the source's callback**, and closing the window settles the promise with null instead of leaving the script hanging. Input and select dialogs wire the source's own validator and show its message under the field. Messages land in the main window's notice and links go to the system browser. The capture tool turns on `SetAutoConfirm(true)` explicitly, since it cannot press a button in the window it is photographing; nothing in a normal run sets it.

**Measured on live sources**, with a test handler that presses the first button:

| Source | Callback | Before | Now |
| --- | --- | --- | --- |
| `wnacg` | `refreshDomains` | `No JavaScript UI handler provided` | completes, returns null |
| `happy` | `wipeCache` | same | completes, returns null |
| `ccc` | account login | same | **gets past the UI block** into a real request, where the site now answers HTML |

That last row is the point: the client-side blockage is gone, and what fails now is the site's response.

`callback` settings (nine sources declare one) also stopped saying "not supported by the client": Core runs `settings.<key>.callback` in the source's own context, and the panel shows the answer or the source's error.

**Patrol baseline (2026-11-01)**: `TestLiveSource` (search, details, chapters, images) ran over 34 official sources: **18 pass, 16 fail**. Every failure was run a second time, and `hcomic` and `ykmh` passed on the retry — a single round really is not a conclusion.

The remaining 14 were each attributed, and **none is a client regression**:

| Cause | Sources | Evidence |
| --- | --- | --- |
| Site blocks with Cloudflare / 403 / HTML | `manwaba`, `mh18`, `baozi`, `comic_walker`, `shonen_jump_plus` | `CloudflareException`, `Invalid status code: 403`, `Unexpected token < in JSON` |
| Site times out or drops the connection | `baihehui`, `happy`, `hcomic` (first round) | `context deadline exceeded`, `Post … EOF` |
| Needs an account | `picacg` | `Not logged in` |
| Needs a server the user runs | `kavita`, `komga`, `lanraragi` | talks to `demo.kavita.org`, `Setting not found: default_username`, API 400 |
| Site or source drift | `jcomic`, `jm` | below |

`jcomic` is worth recording: search already returns a comic whose `id` is a **title slug** (`"Scarlet Prisoner"`) and whose chapter images are all empty strings — the site moved comic URLs to slugs and images to lazy loading, so a source reading `img.attributes["src"]` finds nothing. The client passes the `id` through unchanged and reports the empty string as `core: image URL is not http(s)`, which is right; what broke is between the source and the site. `jm` has the same shape: its `init` fetches a domain list, that file changed format, and `Cannot read properties of null (reading 'length')` comes from the source's own parse.

So **no client behaviour changed this round**; the finding was recorded instead — the patrol now prints sources that need a user-run server on their own `SKIP (needs a server the user runs)` line instead of mixing them with real failures (`NK_LIVE_SERVER_SOURCES`, defaulting to `kavita komga lanraragi`).

**A gate for "declared but not wired up" (2026-11-01)**: the `category`, `ranking`, `account`, `favorites` and `translation` gaps were all the same shape — the source declares it, the client cannot read it, and the only symptom is an empty screen or an English key, found by reading the contracts again by eye. `scripts/check-capabilities.sh` now lists, per capability, the Core entry point and the interface entry point, matched as **fixed strings** (a rename cannot keep passing on a prefix), and fails when one disappears; `scripts/check.sh` runs it.

Checked negatively: renaming `favoritesView` and renaming `"favorites.loadComics"` each failed with the capability named; restoring returned `ok`. It guarantees the **path still exists**, not that the behaviour is right.

**A source's own favorites: eleven declare one, and none was read (2026-11-01, pin `0bcdaef`)**: a `favorites` block holds the favorites the source keeps **on its own side**, which is a different thing from the client's local shelf. The three shapes measured differ:

| Source | multiFolder | loadFolders | loadComics | What actually happens |
| --- | --- | --- | --- | --- |
| `nhentai` | no | none | yes | Without a session the source answers `Authentication required. Please login again or provide API Key in source settings.` |
| `ehentai` | yes | yes | **none** | It lists one folder but has **no list function** — calling one gave `source does not define this function` |
| `wnacg` | yes | yes | yes | Both folders and a list |

So the client has to tell the capabilities apart: a source with only a folder toggle and no list (`ehentai`) must not be asked for one, and a single-folder source gets no folder id. The desktop grows a 源收藏 entry that shows what the source can actually do — a list and pages where there is one, a plain "这个源没有提供收藏列表" where there is not, and the **source's own words** plus a retry button when a session is missing.

Core gained `Value` for this: `favorites.multiFolder` is data, and `Call` invokes functions.

**The interface text a source declares was showing keys (2026-11-01, pin `239221b`)**: `ehentai`'s settings say `title: "domain"` and keep the readable text in its own `translation` table — seven sources ship one. Without reading it the panel showed `domain`, `ehevent` and `hvevent`, which is exactly what the account capture in the previous round displayed. Core now picks the table by the client's locale: the exact locale first, then the same language (a table written as `zh_CN` still answers a `zh_TW` client), then English and Chinese. The desktop uses it for setting titles and option text, and the capture confirms 域名, 触发黎明事件 and 提示HV遭遇战.

Cookie field names (`ipb_member_id` and the like) are **not** translated: those are cookie names, not interface text.

**Source accounts: three ways to log in, none of them wired up (2026-11-01, pin `896f647`)**: fifteen official sources declare an `account`, but in **three** different styles, so reading only `account.login` missed half of them:

| Style | Declaration | Measured |
| --- | --- | --- |
| Password | `account.login(user, pwd)` | `picacg` (answers its own `Failed to login`), `ccc` |
| In-page | `account.loginWithWebview {url, checkStatus, onLoginSuccess}` | `nhentai` offers only this; `ccc`/`mycomic` offer both |
| Cookie fields | `account.loginWithCookies {fields, validate(values)}` | `ehentai` `[ipb_member_id ipb_pass_hash igneous star]`, `mycomic` `[cf_clearance]` |

The desktop panel now draws what the source declares: a password style gets account, password, 登录 and 登出; a cookie style gets one field per declared name plus 校验并保存, and the source's own `validate` decides and writes the cookies (empty values are refused by the source, never accepted silently); a source that **only** offers in-page login (`nhentai`) says **桌面端暂不支持** rather than showing a form that cannot work. `ehentai` is exactly that case — it has no password login at all.

Errors are shown in the source's own words: `picacg` answers `Failed to login`, `ccc` answers `No JavaScript UI handler provided`, because its login wants a JS UI session from the host — a current boundary, not a client bug. `TestLiveAccountStyles` covers all three styles.

**In-page login: five official sources log in on their own site and the client could only say "not supported" (2026-11-01, pin `466b26f`)**: `nhentai`, `ehentai`, `ccc`, `mycomic` and `manga_dex` all declare `account.loginWithWebview {url, checkStatus, onLoginSuccess}`. None of the three parts crosses the language boundary as it stands: `checkStatus` and `onLoginSuccess` are functions, so Core names them with a handle and the client passes that handle back. The desktop panel offers 在网页里登录; the window is the same WebView the challenge flow uses, the source's own `checkStatus(url, title)` decides when the login is finished, and `onLoginSuccess` runs after.

Two things that are easy to get wrong are pinned separately:

1. **Handover order**: `ehentai`'s `onLoginSuccess` reads its forum cookies back through `Network.getCookies("https://forums.e-hentai.org")` and writes them to `.exhentai.org`, so the client must store the cookies before running the callback. Both directions are tested — stored, and nothing stored.
2. **Which cookies belong to the source**: a login page also loads third parties, and copying the whole browser store into the source's partition would make their cookies part of a login the source never asked for. Only the login site's cookies are kept, by the browser's own rule (leading dot: the host and its subdomains; no dot: that exact host), and **both the starting and the finishing host count** (`manga_dex` opens `forums.mangadex.org` and lands on `mangadex.org`), with a cookie that matches either stored once.

`mycomic`'s login page counts as finished immediately, which is not a defect: its "login" is passing the Cloudflare challenge, so a real page is success.

Both clients also stopped hiding logout and the login answer behind the password style: a cookie-only source like `ehentai` computed "these cookies are not usable" and never showed it, and had no way to log out.

Desktop visual evidence (2026-11-01): `ui-webview-login.png` shows 在网页里登录 and 登出 together in `nhentai`'s settings; `ui-webview-ehentai.png` shows in-page login, four cookie fields, 校验并保存 and 登出 together in `ehentai`'s; `ui-callback-wnacg.png` shows `Refresh` from `wnacg`'s own `buttonText`. All three live under `E:/Temp/nkprobe/`, captured by `cmd/uicheck -settings` against the real sources.

The whole login chain is now exercised, not only photographed (2026-11-01, `455d0ec`, `487b15a`): `cmd/loginprobe` serves a local page whose `/login` is titled `Sign in` and which moves itself to `/done`, titled `Logged in`, setting a cookie on the way. That drives the window, the WebView, the polling, the source's own `checkStatus`, the cookie import into the source's partition and `onLoginSuccess` with nobody at the keyboard. Three modes each pin one behaviour: `ok` keeps one cookie and `Network.getCookies` sees it inside the callback; `timeout` ends with an error when the source never agrees and stores **no half login**; `failcallback` reports the source's own reason **immediately**.

The third one found a real defect: a throwing callback used to keep polling until the ninety-second deadline, so the user saw `context deadline exceeded` instead of what the source said. Only "the page agreed but its cookies have not settled" waits now; every other error surfaces at once, and the probe asserts the message mentions `onLoginSuccess` so a return to the deadline behaviour fails the check.

One trap worth recording: the first version of the probe opened its own jar and handed that to the login window, so the cookies landed where no source ever looks and it read like a product bug. The real path is `Library.Jar() → core.Jar()`, the same instance (`settings.go`).

**A ranking page: nine official sources declare one, and it was unreachable (2026-11-01, pin `c22a944`)**: `enableRankingPage: true` with `categoryComics.ranking = {options: ["date-Recent", …], load: (option, page) => …}` is a separate entry a source offers, not a category dimension. Core's `Category` now reports it as `ranking.options` (value and label split on the same `-`), and `RankingComics` calls `categoryComics.ranking.load`; the desktop's category screen grows a 排行 row that reuses the same list and pager, with `catRank` deciding which loader a page belongs to.

Verified on the live source: all five `nhentai` options answer 25 comics — `date` over 26307 pages, `today`/`week`/`month` over 20 each, `popular` over 26307 — and the capture shows the list switching to the real weekly ranking once Popular Week is chosen. `TestLiveCategoryRanking` covers it.

**A random part is not "join twenty tags" (2026-11-01)**: the previous round read `randomNumber` as "how many tags to join", which is wrong. The source's own note is explicit — `randomNumber` is **how many comics to show at the same time**, not how wide a keyword is. The measurement agrees: joined into one query, eight `nhentai` tags still answered 20 comics, ten answered none, and twelve or twenty were refused by the site (429; 400 when the wrong option was sent). A comic search reads several tags as **all of them**, and no work carries twenty. A random part now searches for **one** value per tap, and `nhentai`'s `maid` answers 25 comics over 845 pages.

Worth recording as a trap: **a search takes the search's own options, not the category page's**. `nhentai` declares `/-Recent` for its category loader, which strips the slash, while `search.load` wants `date` — sending the former answers 400.

**Thirteen official sources with a category page, run one by one (2026-11-01)**: **11 pass, 2 do not**, and neither is the client's fault. Three real problems were fixed on the way:

1. **A value may be an object**: `hcomic`'s 热门TAG lists objects, and `String(v)` gave `"[object Object]"` — a parameter no loader recognises. The entry's `label`/`name`/`text`/`value` and its own parameter are used now.
2. **An option without a default takes its first value**: `komiic` answered `Variable "orderBy" has invalid value .` because an empty string was sent as an option; search options already fell back this way, category options did not.
3. **The test itself passed the wrong first argument** (the dimension's name instead of the chosen value) — the same mistake the interface had made, which is why `nhentai` showed an empty page.

The two that remain: `copy_manga`'s 排行 dimension answers **210** every time (a site policy for that endpoint; its 主题 dimension is fine), and `baihehui`'s 作品类型（需要登陆） needs an account while its 更多推荐 returns nothing from the site. **So the patrol concludes per dimension, not per source**: one source can have a dimension that works and one the site blocks, and that is the true state.

**A random part, and a parameter I had wrong myself (2026-11-01, pin `c146110`)**: `nhentai`'s Tags part is `type:"random"` with `itemType:"search"`, which means **pick 20 tags at random and search for them** rather than loading a category — the interface now shows a 随机 20 个 button that picks that many values and searches. The same round fixed a real bug: **the loader's first argument is the chosen value, not the dimension's name**. `nhentai` therefore showed "这个分类没有内容" every time (it looks the value up as an id but was handed "Language"), while `manhuagui` and `copy_manga` only worked by luck, because they read `param` alone. Sources like `nhentai` also need `groupParam` as the parameter — its loader switches on `"language"` — so Core now reports it and the client chooses in the order per-value parameter, then groupParam, then the value itself. Captured afterwards: `nhentai`'s Language category lists real works, and the live checks for `manhuagui` and `copy_manga` still pass.

**`dynamic` parts can be read now (2026-11-01, pin `d90b7ed`)**: twelve category parts in the official sources are declared `type:"dynamic"` — they **list no values and give only a `loader`**, which returns them at runtime, sometimes asynchronously (`manga_dex` computes its Tags from a tag map). This client read only the static `categories`, so such a part came out empty. Core now treats every part as a promise: it calls the loader when one is declared and normalises the entries — a string, or `{label, target:{param}}` with the parameter taken from the target. Against `manga_dex` the Tags part now lists values (e.g. `Oneshot`) and loads 20 comics over 688 pages, page two sharing nothing. **The `random` kind is not handled yet**: `nhentai`'s Tags is `type:"random"` with `itemType:"search"`, which means "pick some tags at random and search", not a category load.

**A live check of category paging and multiple parts (2026-11-01)**: `internal/library/live_category_test.go` (gated by `NK_LIVE_JS`, skipped offline) reads the declaration, loads one entry per part, then reads page two when the source reports one and requires it to be non-empty and not a repeat of page one. Two sources measured: `manhuagui`'s 漫画柜 has one dimension 类型 with 42 comics over 1479 pages, page two 42 comics and 0 shared; `copy_manga`'s two dimensions (排行, 主题) both load, 主题 has 2675 pages, and page two again shares nothing. One fact worth recording came out of it: `copy_manga`'s **details** answer 210 every time while its **category listing** is fine — one site, two policies, and a patrol has to look at them separately.

**The category page is in the interface (2026-11-01)**: the header gained a 分类 button, and it opens the source's own category page — one row per dimension with each value as a button, the chosen one filled in, then the comics themselves (cover and 打开) with paging. **Each value is its own entry**, not one choice per dimension combined: the loader takes a name and the parameter it maps to, and `copy_manga`'s two dimensions (排行, 主题) each feed the loader directly. Verified with a capture: `manhuagui`'s 漫画柜 lists all 39 类型 values with 全部 chosen and the list below carrying covers and open buttons. **The category options are adjustable now**: a source's `categoryComics.optionList` is drawn by the same `optionChips` as a search's filters, and a tap reloads the entry that is open. Sharing the drawing means a change to the chip style cannot land in one place and miss the other. Captured against `manhuagui`: all four options (地区/年龄/状态/排序) with their defaults chosen.

**A source's category page can be read now (2026-11-01, pin `722d8d2`)**: **32 of the 34** official sources declare `category`, and it is the official client's main way of browsing — a title, several dimensions (类型/地区/状态) and the loader's own options. This client had no such path at all and could not even read the declaration: those values are **data, not functions**, and `Call` only calls functions. Core adds `Runtime.Category`, which evaluates the object once and reduces it to JSON — each dimension's values, the parameter each value maps to, and `categoryComics.optionList` in the same shape as search options so a client reuses its filter code; the desktop client wraps it as `CategoryPage` and `CategoryComics`. Checked against a real source: `manhuagui`'s 漫画柜 has one dimension 类型 with 39 values and 39 parameters, four options, and `categoryComics.load` answers 42 comics over 4 pages. **The interface is not wired yet**: the browse view still offers only a source's own `explore`, and the category entry is the next step.

**Tags can be tapped now (2026-11-01)**: 24 official sources implement `comic.onClickTag(namespace, tag)` and the desktop client never called it — the detail view drew tags as one joined line of text that did nothing. Each tag value is a button now, and tapping it asks the source what the tag means: `action:"search"` runs the search with the keyword the source supplied (`copy_manga` answers `作者:<name>`); `action:"category"` would open a category page, **which is not implemented**, because this browse path cannot pass the source's own category parameter — so it says exactly that instead of searching for a word the source never asked for. A source that refuses the namespace throws in its own words ("未支持此类Tag检索"), and those words are shown. Core's `ClickTag` has four assertions (a search keyword, a category parameter, a source refusing with its reason, and a source without the hook answering `ErrNoTagAction`); the interface side puts "what a tag turns into" in `tagOutcome` with one case, and a `goda` detail capture shows the tags rendered as buttons.

**A cover does not run the pixel script (2026-11-01)**: the official client documents in `onThumbnailLoad` that `modifyImage` and `onLoadFailed` are **ignored** for thumbnails. This client used to run it for covers too; now only pages get the pass — a cover is small and decoding and re-encoding it buys nothing. `TestModifyImageIsIgnoredForCovers` has one source declare both hooks and checks that a `thumbnail=true` request carries no script while a page request does. Checked against `manwaba_fixed` afterwards: **a scrambled cover still decrypts** (40,111 bytes came back with the content type cleared, which means `onResponse` ran) — only the pixel script and the mirror fallback are skipped (`onLoadFailed` is documented as ignored for covers too).

**What actually uses `modifyImage`, and a corrected count (2026-11-01)**: only **six files** attach a script in `onImageLoad` — `jm` among the official ones, and `hanabimanga`, `jmcomic`, `recode-jm`, `shonen_jump_plus` and `sunday_webry` in vensrc. The earlier "11 sources" was a counting mistake: it also included `hitomi`, `manhuagui`, `manhuaren` and `nhentai`, which only **mention** `modifyImage` in a comment (saying the field is ignored for thumbnails). The script is used two ways: `jm` cuts a long strip into `num` blocks and reorders them, and `shonen_jump_plus` puts back a page cut into a shuffled 4×4 grid. **Both run on the desktop client now**; `shonen_jump_plus` measured 232,750 bytes of JPEG turning into 879,725 bytes of PNG.

**The desktop client is wired up, and six sources stopped failing (2026-11-01)**: `ImageRequest` now carries Core's handle, and `doImage` uses it three ways — an `onResponse` gets the body to rewrite (decrypt); a fetch that succeeds but returns **an HTML page** counts as a failure when the source declared `onLoadFailed`, because the official client falls back on a decode failure and here decoding happens in the WebView; a real failure asks for the next mirror too. At most four hops, each re-checked for http(s) and size. `4khd_cosplay` is exactly the "200 with an error page" case. Measured: all six sources that were filed under "client gap" now pass — `cxc` (659,568 bytes of ciphertext → a real 494,647-byte PNG), `4khd_cosplay` (a 1,243-byte HTML page → 248,428 bytes of webp from the next mirror), `ccc`, `mojoin`, `mojoin_fixed_v107`, `manwaba_fixed`, `manwaba_fixed_v113`.

**Core now offers the callbacks (2026-11-01, pin `69ed1f9`)**: Core's `comic/js` gained `ImageConfig`/`ApplyImageResponse`/`ImageFallback`, which keep `onResponse`/`modifyImage`/`onLoadFailed` inside the source's runtime under a handle so a host can hand the body to `onResponse` or ask for the next mirror; a host that ignores handles behaves exactly as before. **This repository moved its pin to `69ed1f9` and re-fetched the vendored Core, but the client does not call the three methods yet**, so the symptom below is unchanged — the difference is that only the wiring is left.

**None of the three callbacks in an image config is wired up (2026-11-01, confirmed, not fixed)**: besides `url`/`method`/`headers`/`data`, an `ImageLoadingConfig` may carry `onResponse` (decrypt or rewrite the body before use), `modifyImage` (a pixel-buffer script) and `onLoadFailed` (try the next mirror when this address does not work). **`ImageRequest` drops all three**, so the same class of source breaks in three different places: `cxc`/`mojoin`/`manwaba` hand ciphertext to the reader (next section), while `4khd_cosplay`'s preferred mirror `img.uuss.uk` answers with a 1,243-byte HTML error page — the source already says "if this host fails, use 4khd or wordpress", but with `onLoadFailed` never called that HTML page becomes the picture. One mechanism fixes both: Core keeps the config's functions addressable and lets the host invoke one by handle with the response body.: a source like `cxc` returns not an image address but "address + `#cxc-v1=key:IV`", and `comic.onImageLoad` answers with an `ImageLoadingConfig` whose `onResponse` callback AES-CBC decrypts the body into the real image. **The client's `ImageRequest` keeps only `url`/`method`/`headers`/`data` and drops `onResponse` and `modifyImage`**, so `/img` hands back ciphertext: for `cxc`, 659,568 bytes with `Content-Type: image/png` whose first bytes are no known image format, which the reader can only draw as a broken picture. `comic/js/init.js` does define both fields on `ImageLoadingConfig`, so the source is not at fault — the host does not implement them. Fixing it means handing the response body back to a JS callback and taking the returned bytes: Core would need an entry point that applies `onResponse` for a config id. That is a Core interface change (it moves the pin and both clients), so it is recorded here rather than widened on the spot.

**A source is asked for the search entry point it declares (2026-11-01)**: the official client has two. `search.load(keyword, options, page)` takes a page number and `search.loadNext(keyword, options, next)` takes whatever cursor the previous page returned. This client only knew the first, so a source that declares only `loadNext` — `ehentai` does — answered "source does not define this function" on every search, while working in the official client. The client now asks for `load` when the source has it and otherwise drives `loadNext`, remembering the cursor per source and page and clearing it when a new search starts. The video path already handled both; the comic path never did.

**A live check that a cursor reaches page two (2026-11-01)**: for a source that declares only `loadNext`, a first page proves little — the second fails when the token is dropped or has the wrong shape, while page one still looks fine. `internal/core/live_cursor_test.go` (gated by `NK_LIVE_JS` / `NK_LIVE_QUERY`, skipped offline) reads two pages and requires the second to be non-empty and not a repeat of the first. Against `ehentai.js`: page 1 25 comics, page 2 25 comics, 0 shared. The iOS client now makes the same call, so the Core side already has evidence for that path.

**A multi-select filter group is a JSON array (2026-11-01)**: the value of a `multi-select` group is a JSON array of the chosen keys as a string — that is what a source's `JSON.parse(options[i])` reads. We sent a single key, so `ehentai` failed in `JSON.parse("0")` with `Unexpected token ,`. `SearchOptionGroup` now carries `DefaultValues` (the array a source declares), the desktop app encodes a multi-select group as a JSON array, and a single-select group still sends one key. The two shapes are never mixed.

**An option with an empty value was read as a literal (2026-11-01)**: the option format is "`value-text`" and the left side may be empty, so `-<none>` means an empty value with the text `<none>`. Reading any leading `-` as part of the value turned "no filter chosen" into the string `-<none>`; a source that checks `if (stars)` sees a truthy value and sends a filter that does not exist. The parse now splits at the first `-` and allows an empty left side.

**A source with no chapter list could not be read (2026-11-01)**: gallery sources — `ehentai`, `hitomi`, `nhentai`, `wnacg` — return no `chapters` from `comic.loadInfo`. The gallery is the single reading unit and their `loadEp` takes an empty chapter id. Three places assumed a chapter list: the detail view's 开始阅读 emptied the list and `readerURLFor` then returned nothing, so the button did nothing; the reader's `/img` took the page reference (a gallery source gives `"0"`, `"1"`) for an address and answered 502; `/switch` and `switchChapter` required a non-empty chapter id and answered 404. All three now treat an empty chapter id as a legitimate reading unit: the reader address carries an empty `chapter=`, `/img` asks the source's `onImageLoad` for the real address **and then** checks http(s) (the safety rule is unchanged, it now checks the address actually fetched), and `/switch` leaves the decision to the window. **This is not a patch for one source**: four of the 34 official sources have this shape and all four were being reported as failures.

**The source API version moved from 1.6.0 to 1.6.3 (2026-11-01)**: the official client's current release is v1.6.3 and only `cxc` and `mangabz` ask for `minAppVersion` 1.6.3; every other source asks for 1.6.0 or less. Both sources were run end to end — search, details, chapters, images — before the number was raised, so it is backed by a test rather than copied from the official release. A source that needs an interface this client lacks still fails at the call that needs it, instead of being turned away by a version gate.

### Two real patrols of both source sets (2026-11-01, two rounds)

**Official sources re-checked (2026-11-01, with a caveat about rate limiting)**: running all 34 in one sweep gives **16 passing**, but **that number is not a verdict** — re-run on their own, `ykmh` (403) and `mycomic` (a Cloudflare challenge), both of which failed in the sweep, pass immediately, so a back-to-back full sweep trips site rate limiting and anti-bot checks. The rest were looked at one by one: official `manwaba` does not even declare `onImageLoad`, while vensrc's `manwaba_fixed` (which supplies `onResponse` to undo the site's scrambling) passes against the same client — the official source has not kept up; `copy_manga` **searches fine and answers 210 on details every time** (a different keyword returned 30 comics, details still 210), and the same headers and signature are accepted for search, so the site changed its policy for that endpoint rather than the client blocking itself. Whether a source should count as failing is decided by the classification above, not by a pass count.

The official set (`venera-app/venera-configs`, 34 sources) passed 5 in the first round and **10 after the fixes above**: `copy_manga`, `copy_manga_multi_accounts`, `ehentai`, `goda`, `hcomic`, `hitomi`, `komiic`, `manhuagui`, `nhentai`, `wnacg`. The third-party set (`laincat/vensrc`, 24 in `index.json` plus 47 in `extended.json`, 71 after de-duplication) went from 31 to **34**.

Every remaining failure was read, and none is this client's parsing:

| Failure | Count (examples) | Verdict |
| --- | --- | --- |
| Site redesign / API change | `manga_dex`, `hot_manga`, `jm`, `jmcomic`, `recode-jm`, `damotu` | source or site |
| Login or account required | `baihehui`, `cxc`, `hanabimanga`, `kanman`, `kiutaku`, `kuaikan`, `xiutaku`, `myhl` | source or site |
| Anti-bot / Cloudflare | `ykmh`, `311`, `haokan`, `zymk` | site, unstable between rounds |
| Search found nothing | `baihehui`, `dongman`, `one_kkk`, `mojoin` | keyword mismatch, not a defect |
| A bug in the source script | `copy_manga_multi_accounts` (`res is not defined`), `hot_manga` (replaces only `.jpg.hNNNx.jpg` while the site now answers `.webp`) | source side |

Three deserve their own note. **`manga_dex`**: the `at-home` endpoint returns an empty `data` array and an empty `hash` for every chapter of a comic — five chapters tried, all empty — so this is the site. **`hot_manga`**: the setting does reach the source (a separate case proves `SaveSetting` → `loadSetting` works); the source's own replace pattern does not match `.webp`. **`cxc`**: its images live under `/private/`, and without an account the 191,184 bytes returned are not any image format even though `Content-Type` says `image/jpeg`.

## A source changing, not a regression

Re-running the live sources on 2026-10-31, Copy Manga began answering **HTTP 210** for details, and
that source accepts only 200 (`copy_manga.js` throws ``Invalid status code: ${res.status}``), so the
run reports `js: Invalid status code: 210`. It is not a client defect: on the same day MangaDex (6.4 s), Komiic (1.7 s) and Comick (56 s) all
went end to end, and the text is thrown by the source itself.
A 210 is an off-spec answer from the site — plausibly a challenge or a rate limit — and this E2E
stays red until the site recovers.


### One patrol of third-party sources (2026-11-01)

**A methodological mistake first**: I ran one round with `NK_LIVE_REPEAT=1` and drew conclusions. A second round with `NK_LIVE_REPEAT=2` contradicted them — `copy_manga` and `copy_manga_multi_accounts` passed the first time and failed the second (the site answered 210), while `ykmh` failed the first time (Cloudflare) and passed the second. **One round is not a conclusion**, which is exactly why `live-sources.sh` has a `FLAKY` class.

The **34 comic sources** that ship with `venera-miuix` (a Miuix-styled rewrite of Venera, for the author's own use), keyword "火影":

**Passed both rounds (6)**: `goda`, `hcomic`, `hitomi`, `komiic`, `manhuagui`.
**Passed one round and failed the other (FLAKY)**: `copy_manga`, `copy_manga_multi_accounts`, `ykmh`.

The rest failed, and **almost none of that is a client defect** — the scripts no longer match the sites, in three kinds:
- **A sign-in or a credential is needed**: `baozi`, `comick`, `komga`, `mh18`, `picacg`.
- **The site changed what it returns** (HTML instead of JSON, or a Cloudflare challenge): `ccc`, `comic_walker`, `shonen_jump_plus`.
- **The script and the site no longer match**: the details page has no chapters (`nhentai`, `mxs`, `mycomic`, `manhuaren`, `lanraragi`, `gelbooru`), the chapter page lazy-loads so the `src` attribute is empty (`jcomic`), and the cover address returns something that is not an image (`manwaba` — that site answers with encrypted bytes, with or without a Referer).

**The one failure the client caused, now fixed**: see the scheme-relative URL entry above.

That repository is **not a theme plugin** — it is a complete Android client whose theme is compiled into the code (`VeneraTheme.kt`, `MiuixMotion.kt`, `VeneraLiquidGlassNavBar.kt`). What it ships is the 34 standard `.js` sources above, which our Core reads directly.


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
