# Nekophage status

English | [简体中文](STATUS.md)

Updated 2026-11-03. Sections distinguish Windows live sites, loopback media fixtures, native windows, unit tests and macOS. Recent progress: the macOS reading mainline and an ad-hoc bundle launch are verified, and the desktop window suite grew from twelve to **12 with no external variables and 16 with every script** (local video playback, subtitles across a reopen, the native search and detail pages, progress reporting and the reading place), all folded into `scripts/desktop-probes.sh`. Local fixtures do not establish real-account or live-site behaviour.


The four CI jobs of Core's `6e07429` are green (three Go platforms plus Apple's xcframework with two Swift smoke tests), checked on 2026-11-01.

**Pin moved to `6e07429` (2026-11-01)**: Core gained the three standard AI entry points and the optional chunked transport (`HostStream`), and it fixed a deadlock in the package bridge — the package pushed to `__magpieChatPush` from inside its own hook while the reader stopped at the end marker, so the writer waited for a reader that had gone. The desktop client **does not implement `HostStream` yet**, so replies still arrive whole; `scripts/fetch-core.sh` has aligned `build/core` to that commit and `scripts/check-core-pin.sh` passes.

## Late writes after media-session closure (2026-11-03)

A synchronization barrier reproduced a lifecycle defect: an HTTP request resolved its token to a session but was still receiving its body; closing the session or server did not stop the handler returning 204 and invoking the progress/subtitle persistence callback. Old state could overwrite later saved state. `TestClosedSessionDiscardsPendingReports` uses `io.Pipe` to control the sequence of body consumption, closure and body EOF. All four combinations failed before the fix (`status=204 callbacks=1`; `E:/Temp/nk-session-late-reports-before.log`).

Reports now recheck the media proxy's existing cancellation context after receiving the body and before acceptance. Late reports return 410 without invoking persistence; server and session closure share the signal, with no additional lifecycle state. Race regression covers the four combinations and normal live-session progress/subtitle paths (`E:/Temp/nk-session-late-reports-after.log`). This is a deterministic HTTP-handler timing test, not UI or TCP-disconnection acceptance. Operations accepted by a callback before cancellation may finish; the fix does not roll back accepted saves. The local 0.2.5 package at `e1c2da4` does not include this subsequent fix; the 0.2.6 package (`3359910`) does.

## macOS App packaging and launch (2026-11-03, commit `580f9fd`)

With the new `scripts/package-macos.sh`, a real build ran on macOS 27.0.1 arm64 with Go 1.27.3: `go build -trimpath -ldflags "-s -w"` produced `build/macos/Nekophage.app`, the script generated and `plutil -lint`-checked the Info.plist, and the version comes from the binary's own `0.2.7` with Core at pin `214e7782`. The bundle is 25,487,983 bytes; the executable is 25,449,712 bytes with SHA-256 `0d8ae671d4b55473790b29068085f829f0a44af6b6ee5f29e45d89e638384d58`.

The signature is **ad-hoc** (`codesign --force --sign -`): `codesign --verify --verbose=2` reports `valid on disk` and `satisfies its Designated Requirement`, which is what lets a Go binary run on Apple Silicon, but it is **not a Developer ID signature and not notarised**; another machine's Gatekeeper still refuses it without credentials this repository does not have.

Launch check: after `open build/macos/Nekophage.app` the process exists, and CoreGraphics reports a window owned by `Nekophage`, titled `Nekophage`, **1100 × 760** at `(410, 78)`; the capture `/tmp/nk-mac-window.png` is 81,740 bytes (1146×806) and shows the toolbar and the empty-library hint. The full-screen capture is black because the terminal lacks screen-recording permission, which is recorded as permission evidence only. `osascript quit` left no process or PID behind. Logs are under `/tmp/nk-mac-pkg-evidence/`.

The round also found and fixed an environment defect: pointing `fetch-core.sh` at an external Core checkout still hit go.mod's `./build/core` replacement and failed to build. The workspace now carries a `go work edit -replace`, and external mode skips `go mod tidy` with the reason stated; both modes were rechecked (external fetch/build/test pass, and the default path still fetches the pin and passes `check.sh`), commit `16025c4`. Limits: this is a development build, not a release artifact; playback, subtitles and live sites were not exercised inside this `.app`.

## The Windows package after the pin move (2026-11-03)

Rebuilt on `daf55ac` (Core `8e14aad`) and installed once more:

| artefact | bytes | SHA-256 |
|---|---|---|
| `…-windows-amd64-daf55ac.zip` | 21,561,961 | `d7dc173b…4d79bf` |
| `…-windows-amd64-daf55ac-setup.exe` | 18,716,090 | `b70d7799…fec015` |

Unpacked into an arbitrary directory and run from there, `Nekophage.exe -version` reports the same `0.2.7` / `8e14aad…` / `daf55ac`, so the portable ZIP does not depend on the install location. `Start-Process setup.exe -ArgumentList '/S' -Wait` exits 0; the installed binary reports `Nekophage 0.2.7`, `Core 8e14aade75f563a95f50d4ec80be27bae0d02c63` and commit `daf55ac`, matching the file name; `Uninstall.exe /S` exits 0 and the install directory is gone three seconds later while `APPDATA/Nekophage` stays. The earlier package (`5c5ce67`) recorded Core `214e7782`; both are real artefacts of their own commits and neither stands in for the other. Log `E:/Temp/nk-package-pin-build.log`.

## The Core pin moved to `8e14aad` (2026-11-03)

Nekophage's pin goes from `214e7782` to `8e14aad`, the Core commit that carries `b2610a5` (async rejections keep their error name). The rule was to roll production changes into the next client build, and this is that moment.

Only one production file changed in Core between the two pins (`comic/js/engine.go`), and nothing under `apple/` moved, so the linked difference is an error message rather than an API change.

Verified here: `scripts/set-core-pin.sh` plus `scripts/fetch-core.sh` put `build/core` on that commit, `go build ./...` succeeds, `go test ./...` passes in **all 16 packages**, `scripts/check.sh` passes, the window suite is **14 PASS / 0 FAIL / 0 SKIP**, and `-version` reports `Core 8e14aade75f563a95f50d4ec80be27bae0d02c63`. Logs `E:/Temp/nk-pin-build.log`, `E:/Temp/nk-pin-test.log`, `E:/Temp/nk-pin-probes.log`.

## The Windows package was rebuilt at HEAD and installed (2026-11-03)

Rebuilt at `5c5ce67` and, rather than trusting the build, installed it:

| artefact | bytes | SHA-256 |
|---|---|---|
| `Nekophage-0.2.7-windows-amd64-5c5ce67.zip` | 21,561,030 | `5ba010a7…d8d3af3` |
| `Nekophage-0.2.7-windows-amd64-5c5ce67-setup.exe` | 18,714,941 | `847cda97…d8576f8` |

Install: `Start-Process setup.exe -ArgumentList '/S' -Wait` exits 0, `LOCALAPPDATA/Programs/Nekophage/Nekophage.exe` exists, `-version` reports `0.2.7`, Core `214e7782…` and commit `5c5ce67` (matching the file name), and `Uninstall.exe` is there. Uninstall: `/S` exits 0 and three seconds later the install directory is gone (`dir_left=False`) while `APPDATA/Nekophage` remains, so the `SetOutPath "$TEMP"` fix still holds in a freshly built package.

One operational trap: `./setup.exe /S` from bash exits silently without installing because the shell mangles the arguments; it has to go through PowerShell's `Start-Process -ArgumentList '/S'`. Logs in `E:/Temp/nk-package-head-build.log`.

## The macOS reading mainline passes at the current pin (2026-11-03, `56c9f0c` / Core `8e14aad`)

The repository has no remote, so the code travelled as a private gist: `base64` 3,255,720 bytes, SHA-256 `c6bd87de…1181fa`, a 2,441,789-byte bundle that `git bundle verify` accepted (`contains this ref`, `records a complete history`), HEAD `56c9f0c`, and `scripts/core-version` matching the Core `fetch-core.sh` fetched, `8e14aade75f563a95f50d4ec80be27bae0d02c63`. `go build ./...` exits 0 and `-version` reports that pin (the client commit reads `unknown` because only Core is stamped; the client commit is injected when packaging).

All three reading checks ran on the repository fixture and exited 0:

- **progress**: `pctKeys` contains this chapter's `nkreader.at:loopback_pages/one/1:pct` with `pct=55`;
- **double-page layout**: two columns of `440.5px`, page 1 at `x=443` on the right and page 2 at `x=0` on the left, `dirLabel` `右→左`;
- **the reading place**: scrolled to 1200, reopened, `scrollY=3207`.

The third one deserves a question mark: macOS landed on **3207**, while the same operation on Windows returns exactly 1200 under the stricter rule described above. I tightened that rule from "at least 80%" to "within 20% of the saved value" (floor 80 px); Windows still reports `scrollY=1200 saved=1200` twice, but the macOS run predates the change, so whether 3207 is a defect is not settled — the Mac side has been asked to re-run with the new rule. The 20,756-byte captures hold only the native layer, not WebView pixels; the rendering evidence is `decoded=2 / images=2`.

## The reading place drifted because the restore ran before the images sized the page (2026-11-03)

The Mac side settled what the 3207 was: not a broken restore but **a stored value that had drifted from 1200 to 3207** — the restore happened before the images took their height, the layout grew, the browser carried the scroll position along, and the debounced save wrote the drifted position down, so the reopen read 3207 and comparing 3207 with 3207 of course agreed. Their estimate fits too: `1200 × 6752/2522 ≈ 3212`.

The fix is about **timing**, not sampling:

- the place is applied again once the **last image** has its size, and nothing is written until the layout has stopped moving;
- the listeners attach to `pages.querySelectorAll('img')` rather than to `images`, which holds URL strings — the first version called `addEventListener` on a string, which threw and took the rest of the setup with it, so nothing was restored *or* saved (only three failing local runs showed it);
- a cached image can be complete before the listener exists, so `img.complete` counts immediately, otherwise the count never reaches zero.

Measured: three consecutive local `-scroll` runs pass (`1200 saved=1200`, `1240 saved=1240`), the window suite with retries disabled passes **12 PASS / 0 FAIL** twice, and `check.sh` plus the `internal/reader` tests pass. This changes page behaviour, so the verdict still comes from the Mac: the new probe compares against the position it actually scrolled to, so a drift fails loudly.

## The intermittent reading-place failure was "nothing moved", not "nothing remembered" (2026-11-03)

With the stricter rule (within 20% of the saved value), `-scroll` started failing about half the time: after reopening, `scrollY=0` and no place key in storage, even though `scrollY` was 1200 at the moment of the scroll.

The cause was the probe, not the app: **it cleared the store and then scrolled to 1200, while the page had already been restored to 1200 from the old key**. Scrolling to where the page already is fires no scroll event, the reader writes nothing, and the reopen has nothing to restore. That is also why failing runs had two fewer keys — neither the place nor the percentage was written.

The fix is not a sampling patch but making the step **never a no-op**: if the target is within 2 px of the current position, offset by 40 px first, and return the **actual** `scrollY` as the baseline. The check then also requires the stored value to equal that baseline, failing with "the store says X but the probe scrolled to Y" otherwise. Before moving, it waits for the page to be really loaded (`readyState === 'complete'` and `#pages figure` populated) so it never scrolls the empty document that is about to be replaced.

Measured: four consecutive runs pass (`scrollY=1240 saved=1240` or `scrollY=1200 saved=1200`, depending on whether the 40 px offset was needed), and the window suite with retries disabled passes **12 PASS / 0 FAIL** three times. Logs `E:/Temp/nk-nudge-{1..4}.log` and `E:/Temp/nk-probes-nudge-{1,2,3}.log`.

This also explains the macOS `scrollY=3207`: the old rule compared against "where the probe meant to go", while the reopened position may have come from a stale stored value. The new probe fails when the store disagrees with the baseline, so one macOS run settles it.

## The intermittent window failures, traced to their causes (2026-11-03)

Running the suite back to back occasionally produced "one entry FAILs while its log is full of PASSes" or "a dialog window was never opened". Three separate things, not one blanket retry:

- **The dialog probe** slept two seconds and then looked for the window; with other windows already running the dialog sometimes took longer. It now polls for up to ten seconds, and eight consecutive solo runs passed. The cause is fixed rather than hidden.
- **`readercheck`** read the page state in a goroutine after a wait, so a window that ended first still printed `done` and exited 0 — one suite run "passed" that way with no page state at all. A `checked` flag now makes that case fail with `the window ended before the page could be checked`.
- **The suite** starts a dozen window processes in a row and can still lose one. `run()` allows **one retry**, prints `RETRY <entry>` and keeps the first log as `.first`, so the failure stays visible instead of turning the suite into noise; `NK_NO_RETRY=1` disables it when measuring the real rate.

Measured after the fixes: **three consecutive runs of 12 PASS / 0 FAIL / 0 RETRY**, and with retries off an environment-level failure still appeared once (login and dialog each seen once), which is why the retry is a backstop rather than the explanation. Logs `E:/Temp/nk-probes-final-{1,2,3}.log` and `E:/Temp/nk-dialog-{1..8}.log`.

## The native search joins the default suite too (2026-11-03)

Last round I set `uicheck` aside as a check that could only ever be green: it captured a picture and asserted nothing, and a grid that found something captures to nearly the same size as one that found nothing. This round it got an assertion, which is what made it worth including:

- the app grew `SearchForCheck()`: the number of results, the first title and the failure sentence, in the same shape as `FavoritesForCheck` and `CommentsForCheck`;
- `cmd/uicheck` grew `-expect`: the first title has to contain the given text or it exits non-zero.

Both directions were measured: `-expect 'Loopback Comic'` gives `search: count=1 first="Loopback Comic" err=""` and exit 0; `-expect 'Nonexistent Title'` gives `first result "Loopback Comic" does not contain "Nonexistent Title"` and exit 1.

The same check now opens the first comic as well, and the detail page has to report its own title and chapter count: `detail: title="Loopback Comic" chapters=1 err=""`; `-expect-detail 'Wrong Comic'` exits non-zero with `detail title "Loopback Comic" does not contain "Wrong Comic"`. An empty detail page and a comic with no chapters capture to nearly the same picture, so the picture is not the evidence.

The window suite therefore stays at **12 PASS / 3 SKIP** with no external variables (the assertion went into the existing entry) and **16 PASS / 0 FAIL / 0 SKIP** with all of them. Logs `E:/Temp/nk-uicheck-ok.log`, `E:/Temp/nk-uicheck-neg.log`, `E:/Temp/nk-detail-ok.log`, `E:/Temp/nk-detail-neg.log` and `E:/Temp/nk-probes-detail.log`.

## The subtitle choice joins the default window suite (2026-11-03)

`cmd/subcheck` is in `scripts/desktop-probes.sh` now. The check is not "the name was stored" but "the cue rendered after reopening is the one that was chosen": `chosen: selected="English" active="Subtitle rendering proof" cues=[en:1]` → `reopened: selected="English" active="Subtitle rendering proof" cues=[en:1]`. This path has been wrong before: the choice lived only in the page, so closing the player and opening it again fell back to the manifest default.

The window suite is therefore **11 PASS / 3 SKIP** with no external variables and **16 PASS / 0 FAIL / 0 SKIP** with all of them (the three skips still need external scripts). Logs `E:/Temp/nk-sub-head.log` and `E:/Temp/nk-probes-with-subs.log`.

## Video playback joins the default window suite (2026-11-03)

`cmd/videocheck` existed but only ran by hand. It is now part of `scripts/desktop-probes.sh`: the repository clip `internal/player/testdata/hls/stream.mp4` plays through the media proxy, and three things have to hold — the position advances **by itself** (two reads of `currentTime` increase), the page shows no error text, and the player reports progress when paused. The window capture only holds the native layer, so the numbers the page reports are the evidence.

Measured: `readyState=4 videoWidth=320 duration=10.066667 time=2.0674 -> 4.0688 hls.js=1.7.4 err=""`, exit 0.

The same check covers **byte ranges**: it takes an address the page actually fetched under `/media/…` and asks again with `Range: bytes=0-1023`, requiring `206`, a `Content-Range` and 1024 bytes back. Measured `status=206 content-range="bytes 0-1023/178570" bytes=1024` — seeking depends on it, and a player that only ever returns whole files looks fine until someone drags the scrubber. The window suite is therefore **10 PASS / 3 SKIP** with no external variables and **16 PASS / 0 FAIL / 0 SKIP** with all of them. Logs `E:/Temp/nk-video-head.log` and `E:/Temp/nk-probes-with-video.log`.

## Progress reporting and the reading place join the default window suite (2026-11-03)

`-progress` and `-scroll` used to be manual runs. Both now use the repository fixture (readercheck serves the pages on 18731) and are part of `scripts/desktop-probes.sh`:

- **progress**: it only reads the report key the page itself writes; a request from the probe would prove the endpoint answers, not that the page reports;
- **the reading place**: scroll, reopen, and require at least 80% of the distance the page can actually move.

Measured: **9 PASS / 3 SKIP** with no external variables and **16 PASS / 0 FAIL / 0 SKIP** with all of them. Logs `E:/Temp/nk-probes-7plus.log` and `E:/Temp/nk-probes-7plus-full.log`.

## The failed-page notice and its retry now have an end-to-end check (2026-11-03)

The reader shows "page N failed to load / retry" when a picture does not arrive, and nothing had ever exercised that path. `probesite.ServeOnePageFailing(port, failures)` answers the first N requests with a server error and the rest with the page, and `readercheck` grew `-fail N` and `-retry`.

`-retry` asks two things: the failure notice must be there after the first load (a non-zero count of `.retry` elements), and after clicking retry there must be no notice left and at least one decoded picture. Both directions were checked:

- `-fail 2 -retry`: two notices, then `failures=0 decoded=2` (`E:/Temp/nk-retry-recover.log`);
- `-fail 0 -retry`: exits non-zero with `a page that failed did not say so: .retry=0` (`E:/Temp/nk-retry-negative.log`), so the check cannot pass when nothing failed.

The window suite carries it: **7 PASS / 3 SKIP** with no external variables and **14 PASS / 0 FAIL / 0 SKIP** with all of them (`E:/Temp/nk-probes-retry-default.log`, `E:/Temp/nk-probes-retry-full.log`).

## The scroll check now takes its target from the page (2026-11-03)

`readercheck -scroll` used to scroll to a fixed 1200 and require at least 800 after reopening. That had two faults: a fixture page shorter than 1200 failed for being short, and a page that cannot scroll at all slipped past a threshold of zero. The target now comes from the page, `min(1200, max(0, scrollable height - 40))`; a target under 100 is reported as a page that cannot demonstrate scrolling, and the reopened page must come back within 80% of it.

One self-inflicted detour is worth recording: the first rewrite dropped the one-second settle after scrolling, and the position is written to localStorage 400 ms after the scroll stops, so reopening fell back to the top every time. That was the probe killing the behaviour it meant to observe, not a defect in the app. The wait is back and the log prints where the page settled.

Measured: `scrollY=1200` on two consecutive runs, and the window suite is still **13 PASS / 0 FAIL / 0 SKIP**. Logs `E:/Temp/nk-scroll10-1.log` and `E:/Temp/nk-probes-scroll-target.log`.

## The reader decode check no longer borrows another repository's fixture (2026-11-03)

`scripts/reader-probe.sh` used to ask for a source script and the repository had none, so runs borrowed `UITests/Fixtures/callback_probe.js` from Nekularity. With `internal/reader/testdata/loopback_source.js` in place, leaving `NK_READ_SRC` unset uses it (query `probe`, pages served by readercheck on 18731), and the entry in `scripts/desktop-probes.sh` now runs by default instead of only when an environment variable is set.

Measured: with no environment variables the window suite is **6 PASS / 3 SKIP** (the skips are source favourites, the comment thread and the official callbacks and tags, which still need external scripts); with everything set it is still **13 PASS / 0 FAIL / 0 SKIP**. Logs `E:/Temp/nk-probes-default-reader.log` and `E:/Temp/nk-probes-full-default-reader.log`.

## The double-page layout can be checked offline (2026-11-03)

`-layout` used to need a live site: the fixture chapter had one page and measuring a spread needs two. The repository now has `internal/reader/testdata/loopback_source.js` (a two-page chapter, both pages on `127.0.0.1:18731`, served by readercheck itself) and `scripts/layout-probe.sh`, wired into the window suite.

The check itself had two problems: it asserted the page class contains `spread rtl`, but the direction is a toggle, so clicking it lands on whichever side it was not and a fixed word makes the result depend on the starting state; and it looked for "右" with `contains`, which also matches "左→右" and reads left-to-right as right-to-left. It now reads the arrow direction from the label and checks that two columns exist with the first page on the right.

Measured in a real Windows window: two columns of `433.5px`, the first page at `x=436` and the second at `x=0` (right-to-left), capture `build/probes/layout.png`; a one-page chapter fails the opposite way with `few figures` and a non-zero exit. The window suite is now **13 PASS / 0 FAIL / 0 SKIP** (`E:/Temp/nk-desk-probes-layout.log`).

## The fixture page is now scrollable (2026-11-03)

`probesite.ServeOnePage` served an 8×8 PNG: the reader could decode it, but a page shorter than the window keeps the reading position at zero, so `-progress` and `-scroll` only proved the reporting path, not that a position is remembered. It now serves **320×1200** with a colour band every 100 rows, so a capture shows where the page is scrolled, and the unit test asserts the document's height and width.

The `-progress` criterion changed with it: the tool used to post its own `progress?...percent=55` XHR, which proves the endpoint answers, not that the page reports. It now reads only what the page wrote to `localStorage` and requires a report key for **this chapter** (a one-page chapter reports 0% by page index, so the key is the evidence). Local rerun (`callback_probe.js`, `-serve 18731`): `decoded=1 first=320x1200`; `-progress` exits 0 with `y=1335 h=3388` and a `callback_probe/1/1` key; `-scroll` reopens the chapter at **scrollY=1200**. Logs `E:/Temp/nk-reader-tall.log`, `E:/Temp/nk-reader-scroll.log` and `E:/Temp/nk-reader-progress4.log`.

## macOS reading mainline (2026-11-03, commit `3c42463`)

The repository has no remote, so the code travelled as a git bundle through a private gist (base64 3,190,624 bytes, SHA-256 `093cbfc4…c2e3a173`; bundle 2,392,967 bytes). `git bundle verify` passed, the clone's HEAD is `3c42463`, and `scripts/fetch-core.sh` fetched Core at pin `214e7782`. `go build ./...` passed on macOS 27.0.1 arm64 with Go 1.27.3.

The reading mainline ran through `cmd/readercheck` with an offline fixture (`callback_probe.js`, pages pointing at `127.0.0.1:18731`, served by readercheck itself): exit 0, **1 / 1** pages actually decoded out of a one-page chapter (`figures=1 images=1 decoded=1 first=8x8`), and the capture `/tmp/nk-mac-reader.png` is 20,756 bytes at 900×920. The reading-progress report answered HTTP 200 with `{"ok":true}` and wrote `nkreader.at:callback_probe/1/1:pct`. A single page is not tall enough to scroll, so the position is `0`; this proves the reporting path, not a reading position. Port 18731 had no listener afterwards.

Limits: the capture is a native window and holds no WebView pixels, so the decode conclusion comes from the tool's own count; the fixture is loopback with no live site or account; this is a development build, not a macOS package. Logs are under `/tmp/nk-mac-reader-evidence/`.

## Windows 0.2.7 local test package (2026-11-03, commit `f114dea`)

First package to carry the header fix: with no comic sources installed, the hint is no longer squeezed into one character per line. Portable ZIP **21,559,053 bytes**, SHA-256 `7d348877318ec1af88b5c2991933e2441b902c296887b23dd596065ea31e3bc5`; setup **18,712,668 bytes**, SHA-256 `c0ff8bfaa9fa89fbe4a4648bfc8443756bf01c514b229dc010a425b360b44da0`. `package-windows.py` passes the build, ZIP CRC, extracted-file hash and executable version checks; the binary reports 0.2.7, Core `214e7782`, commit `f114dea`.

Launch check: the executable unpacked from the ZIP started with isolated `APPDATA`/`LOCALAPPDATA`, drew a 1100×760 window titled `Nekophage` with only the title and controls in the header (`E:/Temp/nk-run-027/window.png`), and the process exited when the window closed. The install check is again not isolated (NSIS uses the shell folder API), installing into the real `%LOCALAPPDATA%\Programs\Nekophage`, where it reports 0.2.7. This round found that the uninstaller left an empty `Nekophage` directory: Windows cannot remove a directory that a process still uses as its working directory. Adding `SetOutPath "$TEMP"` at the start of the `Uninstall` section, rebuilding the setup and repeating install → silent uninstall removed the directory and kept the `%APPDATA%\Nekophage` profile. Playback and subtitles are still not verified inside a package.

## Windows 0.2.6 local test package (2026-11-03, commit `3359910`)

This package carries the media-session late-write fix that 0.2.5 (`e1c2da4`) does not. The version in `main.go` is now 0.2.6; portable ZIP **21,557,923 bytes**, SHA-256 `5b67cd240f2faf9b6097a5bafef8b2542eb9ab43d05bd2919b6c47dabe8cc335`; setup **18,712,671 bytes**, SHA-256 `0d2955eaf9e3881fdad4210122e761d76c8f9157ed284b4d68440c182b20ad30`. `package-windows.py` passes the build, ZIP CRC, extracted-file hash and executable version checks; `scripts/check.sh` passes at `7bd9965`.

The install check **was not isolated**: NSIS resolves `$LOCALAPPDATA` through the shell folder API, so redirecting the environment variable has no effect — `/S /NOREGISTER` installed into the real `%LOCALAPPDATA%\Programs\Nekophage`, replacing the previous test install. The installed program reports 0.2.6, Core `214e7782`, commit `3359910`, `go1.27.3 windows/amd64`. A silent `Uninstall.exe /S` then removed the directory and left the `%APPDATA%\Nekophage` profile in place.

Package 0.2.6 also had a launch check of its own: the executable unpacked from the portable ZIP (no installer) started with isolated `APPDATA`/`LOCALAPPDATA`, drew a window titled `Nekophage` at 1100×760 (`E:/Temp/nk-run-026/window.png`), and the process exited when the window was closed. That capture also shows a real layout defect: with no comic sources the header hint was squeezed to one character per line by the fixed-width controls and overlapped the title. It is fixed in the current development build (`e54944d` drops the hint; the empty state already explains it) and the repaired window is `E:/Temp/nk-run-dev-out/window.png`. **The 0.2.6 package does not contain that fix**, and playback and subtitles remain unverified inside the package. Package 0.2.6 has **no** window-level playback, subtitle or resume acceptance; that evidence still belongs to 0.2.5. There is no Git remote, so nothing was published.

## Windows 0.2.5 local test package (2026-11-03, commit `e1c2da4`)

Packages the independent subtitle-preference storage and migration fixes: portable ZIP **21,556,043 bytes**, SHA-256 `a43520be5753c7d983a151f681b4c5fbd287a47cf581950ea08740534bda2e6e`; setup **18,711,551 bytes**, SHA-256 `e7db5370ec9a3d948a08680fde552cc4b7fd08a4809df5edbc783384c887980e`. `package-windows.py` passes ZIP CRC, extracted-file hashes and executable version checks. The isolated `/S /NOREGISTER` installation reports 0.2.5, Core `214e7782`, commit `e1c2da4`. The actual installed window completed video search, details, Episode 1 and visible Chinese subtitles (`E:/Temp/desktop-package-025-dropdown.png`); local Range, subtitle playlist and VTT requests all logged auth=true.

Further interface operation stopped when the user prioritized underlying work. Packaged language switching and preference restoration remain unverified and do not inherit development-tool results. Task windows closed, the app exited, the fixture stopped, uninstall removed the install directory and retained the isolated APPDATA marker (`E:/Temp/nk-package-025-cleanup.log`); receipt `E:/Temp/nk-package-025-acceptance/receipt.json`. The unsigned package is delivered locally only, with no remote publication; SmartScreen, missing WebView2, macOS and iOS are outside this round.

## Windows 0.2.4 package (2026-11-02, commit `29bab49`)

Portable ZIP `21,547,137` bytes, SHA-256 `da5098c58b519c91544e6d5decee79a690cbf0a501d518f8347df0c4f04f336c`; per-user setup `18,705,377` bytes, SHA-256 `f089f62aa5429ffee6a8adc345fe044f3f048d11f8dce5d04661c9dafb55e105`. Receipt: `E:/Temp/desktop-package-0.2.4-receipt.json`.

Packaging caught a real defect: playing the same fixture in the 0.2.3 package produced **no subtitle track and no `subs.m3u8` request**. Default playback used the browser's native HLS, which on WebView2 ignores `EXT-X-MEDIA` subtitle groups; `cmd/uicheck` without `-video-hlsjs` reproduces it (`decoder=native`, `activeSubtitleCues=0`). `851884d` makes HLS sessions prefer the bundled HLS.js with native only as a fallback, keeping `-video-native` to exercise that path.

Measured: after an isolated silent install, `-version` reports `0.2.4`/Core `214e7782`/commit `29bab49`; in the **packaged GUI** the path Video → search → details → Episode 1 → Play shows the subtitle selector on 中文 and the picture carries 第0段字幕 / 字幕渲染检查, the fixture server logs `subs.m3u8` and `sub0.vtt`, screenshot `E:/Temp/desktop-package-0.2.4-subtitle.png`, and `Uninstall.exe /S` removes the directory. Subsequent acceptance of subtitle language switching/Off in this package is recorded next. **Unverified**: macOS/iOS subtitles, signing and a machine without WebView2.

## Subtitle language and Off controls (2026-11-02, packaged `29bab49`)

Using Cindy desktop controls, the Lead operated the unchanged installed 0.2.4 build at the same paused `0:01` frame: Chinese → English → Off → Chinese. English shows `Subtitle rendering proof`; Off removes it; Chinese restores `字幕渲染检查`. Actual dropdown options were selected without injected DOM events. Comparison captures: `E:/Temp/desktop-subtitle-024-english-paused.png`, `E:/Temp/desktop-subtitle-024-off-paused.png`, `E:/Temp/desktop-subtitle-024-chinese-restored.png`; full receipt: `E:/Temp/desktop-subtitle-language-024-receipt.json`.

The final fixture uses Go `http.FileServer`, login Cookies and Referer validation matching `cmd/uicheck`, with real Range support. Chinese/English playlists, VTT and byte-range requests all record `auth=true` in `E:/Temp/nk-subtitle-language-024/range-fixture.stderr.log`. Initial observations using Python's server without Range support are excluded. All task windows were closed through the UI, the process exited, uninstall completed and the fixture stopped (`E:/Temp/desktop-subtitle-language-024-cleanup.log`). This verifies language/Off controls in this Windows package. Selection persistence across a player reopen is covered separately in the next section; the choice after a variant-group refresh, other subtitle styling and macOS/iOS remain outside these results.

## Active subtitles and Off after reopening (2026-11-03)

`cmd/subcheck` now pauses at the same 1.000s position, requires a decoded video width and a completed seek, and reads only `activeCues` from showing tracks. The old probe read `cues`, including loaded text for future times, so its records alone do not establish active subtitles. A new session reads preferences from disk instead of relying only on the original shelf's memory; the stored preference must also match this selection rather than any old record. The probe now supports `-choose 关闭` and `-text`.

Three separate real Windows WebView2 / HLS.js 1.7.4 runs pass: English, Chinese and Off restore respectively English active text, Chinese active text and `value=-1` with no showing tracks or active text. All report decoded width 320 at paused position 1.000s. Logs: `E:/Temp/nk-subtitle-active-english.log`, `E:/Temp/nk-subtitle-active-chinese.log` and `E:/Temp/nk-subtitle-active-off.log`. A negative fixture moves the English cue to 100–110s: English is still selected at 1s but reports `en:0` and empty active text, and the probe exits nonzero as expected (`E:/Temp/nk-subtitle-future-cue-negative.log`). Future subtitles can no longer produce a pass.

This development tool operates the real WebView select and dispatches change; it is separate from human control operation in the packaged GUI or pixel evidence. Coverage is the local fixture, persisted preference reads and a new session/page, not a full app-process restart, variant-group switching, macOS or iOS. The delivered package remains 0.2.4; no new package was built this round.

## Subtitle choice survives reopening the player (2026-11-02)

The choice used to live only in the page: `subtitleChoice` was a page variable, so closing the player and reopening the same episode fell back to the manifest default. The subtitle choice is stored independently in the shelf file, including an explicit Off. The page reads it on load and reports changes. A choice made before the first progress report is retained without creating viewing history; clearing finished playback progress also preserves the choice. Choices embedded in older progress records are migrated when loaded. These records are deliberately not pruned by the history limit, so they carry their own cap: past 2000 records the oldest are dropped. `TestVideoSubtitlesAreCapped` covers the cap and that the newest record survives a reopen (`go test ./internal/shelf` passes).

Correction to the earlier mapping-defect claim: the initial attempt to select English chose Chinese because the development probe confused option index and value; this did not establish a product defect. The page now maps options through the track list and uses the track's own id; other track arrangements on live sites remain unverified.

The evidence comes from a new development tool, `cmd/subcheck`: it opens an HLS session in a real WebView2 (repository fixture `internal/player/testdata/hls-subs`, English `DEFAULT=NO`, Chinese `DEFAULT=YES`), stores the preference in the real shelf, then **opens a new session for the same episode**. The first page shows the default Chinese track; after choosing English the shelf record is `{Name:English, Language:en}` and the showing track's loaded text is `Subtitle rendering proof`; the reopened page reports `selected="English"`, `value=0`, `cues=[en:1]` and still contains the loaded English cue (that earlier count was not an active-cue count) (`E:/Temp/nk-subcheck-run3.log`). The window capture `build/probes/subtitle-reopen/reopened.png` contains only the native layer, not the WebView pixels.

On 2026-11-03 the three cases were rerun at commit `3c42463`: English (reopened `selected="English"`, `value=0`, active text `Subtitle rendering proof`), Chinese (`value=1`, active text `字幕渲染检查 第0段字幕`) and Off (`value=-1`, no active cue or text). All three decoded at width 320 while paused at 1.000s (`E:/Temp/nk-subcheck-3c42463-{en,zh,off}.log`). A subsequent first-play fix passed `cmd/subcheck` with no pre-existing history (`E:/Temp/nk-subtitle-first-play-webview.log`); separate unit tests cover clearing progress and migrating older data. Limits: this is a real window and a real HLS.js session opened by a development tool, not a click in the packaged 0.2.4 GUI; the fixture is in-repository and no live site is involved; subtitle size/placement styling, picture-in-picture, the choice after a variant-group refresh, macOS and iOS remain unverified. `sh scripts/check.sh` passes (`E:/Temp/nk-subtitle-persist-full-check.log`).

## Subtitle delivery (2026-11-02)

Previously the default was the browser's own HLS: the same fixture in an actual WebView **requested no `subs.m3u8` and had no subtitle track at all** (`decoder=native`, `activeSubtitleCues=0`), while forcing HLS.js worked. That is a product defect, not a fixture problem — WebView2's native HLS ignores `EXT-X-MEDIA` subtitle groups. HLS sessions now prefer the bundled HLS.js and fall back to native only when HLS.js cannot run; `-video-native` still exercises the fallback, where missing subtitles are an accepted, recorded limitation. The default path measures `decoder=hls.js`, `activeSubtitleCues=2` and a `2.006s` resume (`E:/Temp/desktop-subtitle-prefer-hls.log`).

Subtitle groups load through the media proxy with source Referer/Cookie headers. Early `41a89da` forced the first track; this round respects `DEFAULT=YES`, keeps subtitles off when there is no default, and adds a subtitle language/Off selector that retains an explicit choice within the page across track-list updates.

The fixture puts English first (`DEFAULT=NO`) and Chinese second (`DEFAULT=YES`). The actual Windows WebView defaults to Chinese: `activeSubtitleCues=2`, text `字幕渲染检查`, modes `disabled,showing`, and paused `2.103s` resumes at `2.103s` (`E:/Temp/desktop-subtitle-choice-default.log`). With both tracks changed to `DEFAULT=NO`, the negative control reports `cues=0`, modes `disabled,disabled`, makes no subtitle playlist/VTT request and exits 1 (`E:/Temp/desktop-subtitle-no-default.log`). The probe now accepts only active cues on showing tracks; `-video-subs-text` can check the language, so cached cues on disabled tracks cannot pass.

The Lead navigated the actual Video/search/details/episode/player windows and saw Chinese text over the video. Screenshot: `E:/Temp/desktop-subtitle-visible-window.png`; receipt: `E:/Temp/desktop-subtitle-choice-receipt.json`. This supplies pixel evidence for Chinese in that window. The earlier round stopped after `foreground_unavailable`; subsequent language switching/Off acceptance in the installed 0.2.4 build is recorded above. **Still unverified**: font/position styling, macOS, iOS and live subtitles. The older `bcbe602` 0.2.2 package does not include this selector; canvas video-frame readback still excludes subtitle overlays.

## Audio decode pipeline (2026-11-02)

After playing the H.264+AAC fMP4 HLS fixture in an actual Windows WebView, `webkitAudioDecodedByteCount` reads **18,537 bytes**, the picture is `320×180`, authentication/Range work, and paused `1.975s` resumes at `1.975s`. Log: `E:/Temp/desktop-audio-decoded-counter.log`. The probe's `-video-audio` now requires a positive actual decoded-byte counter; browsers without that counter remain unobserved.

Native MP4 audio passes as well: concatenating the init segment and media segments of the same audio-carrying media into one `clip.mp4` (`internal/player/testdata/mp4-audio`) makes the native path report `audioDecodedBytes=17802`, `decoder=native`, and a paused `2.054s` reopens at `2.054s` (`E:/Temp/desktop-mp4-audio-native.log`); the video-only file built the same way reports `audioDecodedBytes=0` and exits 1 (`E:/Temp/desktop-mp4-video-only-negative.log`).

A stronger negative control falsely declares AAC in the manifest but contains video only: `audioDeclared=yes audioDecodedBytes=0`, exit status 1, log `E:/Temp/desktop-audio-false-manifest-negative.log`. The earlier `8f1d812` criterion used manifest/codec metadata alone and did not establish decoding; it is replaced. Actual decoded bytes still do **not** establish audible speaker output; native MP4 audio is now verified down to the decoded-byte counter with the fixture above.

## Video read cancellation (2026-11-02)

Video lists, details and play-URL resolution use separate cancellable generations. A new search stops the previous search/pagination and old detail preparation; switching providers stops all three kinds of reads. Closing a list stops its HTTP; closing details stops details and unresolved playback preparation. Lists and independent detail windows have separate loading/error state, so finishing details cannot clear an active search. An already running media session stays independent of the search window.

Pagination retains the provider and keyword that produced its list, rather than later unsubmitted input. Changing episodes or lines cancels older play preparation; a returned URL must still own the current generation on the main thread before a player opens. Closed list/detail windows can be recreated, and empty-line details no longer index an empty array.

`go test -race` uses real Core, JS video sources and a slow loopback HTTP server for replacement searches/pages, provider changes, list/detail closure, canceled history resume and replaced playback preparation. The server must observe cancellation, followed by an actual successful next read. Additional cases reject late same-provider/same-ID completions, keep list/detail state independent and prevent pagination from using an edited keyword. Log: `E:/Temp/desktop-video-cancel-targeted.log`. These are controller/HTTP checks. A separate actual Windows WebView rerun verifies HLS.js decoding, authentication/Range, pause persistence and `2.051s → 2.051s` resume: `E:/Temp/desktop-video-cancel-hls-window.log`. Network cases are not native-window gesture validation.

## Video-source installation and restart restoration (2026-11-02)

源管理 now selects comic sources, video sources or Kazumi rules, with URL installation and a native file picker. The library handles explicitly typed installs, same-kind replacement and persistence; the comic default remains, while shared Core rejects cross-type key collisions. Files and URL downloads share the 4 MiB bound. Settings are shown for JS kinds; the video picker includes video JS, Kazumi and existing built-in video records, while the comic picker lists only comics.

**Actual Windows native UI passed**: start the production program with isolated APPDATA, choose video source in 源管理, install `hls_probe.js` through the native file dialog, then enter a loopback URL and click Install for `mp4_probe.js`. The window lists two video sources and `plugins.json` persists their `video-js` kinds, origins and enabled state. Close through the windows and restart the same program: both remain in the video picker. Receipt: `E:/Temp/desktop-video-source-manager-ui-receipt.json`; capture: `E:/Temp/desktop-source-installed-both.png`. Running the playback probe against this GUI-installed profile without reinstalling verifies actual HLS.js decoding and `2.042s → 2.042s` resume: `E:/Temp/desktop-restored-video-source-playback.log`.

`go test -race ./internal/library ./internal/ui` covers file installation, URL replacement, retained disabled state, rejected kind collisions, failed input preserving existing records, disk restoration and actual execution of the updated provider after enabling. Core stages disabled JS bodies without executing them to refresh metadata; enabling refreshes the version. The client retains that contract. The test corrected an assumption that disabled replacement refreshed versions immediately; shared Core was not changed. Kazumi type selection is implemented; this receipt does not prove playback from a live Kazumi site.

## Desktop video player (2026-11-02)

The main window now has a 视频 entry: provider search → details (lines/episodes) → player window. `player.Server` serves the page on `127.0.0.1`; session URLs carry a random token and the page only ever sees proxy resource IDs. The page bundles HLS.js 1.7.4 (Apache-2.0, `internal/player/assets`); MP4 uses native decoding, while HLS prefers native support and otherwise uses HLS.js. Positions are stored in `shelf.json` under `videos`, separate from comic history so a shared key cannot collide; the details list shows watched time and reopening resumes; reaching the last two percent within ten seconds of the end counts as finished and the next open starts from zero.

Latest real-window evidence (`cmd/uicheck -video …`, fresh isolated profiles): HLS.js decodes `320×180`, the clock advances, and an actual frame is read back to `build/probes/hls-playback/frame.png`. The page's pause event writes `2.0381s` to disk; reopening the shelf and using Continue Watching really restores `2.0381s`. The master playlist, child playlist, initialization fragment and five fMP4 fragments all carry the declared Referer and the Cookie acquired during search; upstream logs record actual byte ranges such as `bytes=0-1122`. Log: `E:/Temp/desktop-hls-browser-playback.log`. Native MP4 also passes authentication, `640×360` decoding and actual `2.0921s → 2.0921s` resume: `E:/Temp/desktop-mp4-auth-real-resume.log`.

The earlier `desktop-player-ui-probe7.log` establishes the MP4 picture and clock only: a leftover synthetic `3s` progress POST contaminated the save criterion, and failures still exited with status 0. That injection is removed. Acceptance now requires the page's pause event, a fresh disk read and the actual reopened position. The negative case omits the login Cookie, gets 403 and exits with status 1: `E:/Temp/desktop-video-negative-check.log`.

Fixed here: MP4 decoder selection no longer depends on native HLS capability. Saving and opening an episode share one completion rule, allowing a two-second position in a ten-second clip to resume. Fractional seconds survive; HLS seeks after metadata, and Continue Watching uses the saved provider rather than the current search picker. Fixtures and rerun commands: [testdata](../internal/player/testdata/README.en.md). The production application does not require FFmpeg.

Ordinary video-source file/URL installation, restart restoration and video read cancellation have the separate records above; audio decoding, active subtitle text and Chinese window rendering have their own evidence. Language switching/Off is now also exercised in the installed 0.2.4 build. **Not verified**: picture-in-picture, macOS execution, live online sites and speaker output. The older 0.2.2 (`bcbe602`) package is historical; the latest 0.2.4 (`29bab49`) includes the verified subtitle selector.

## Desktop video media transport (2026-11-02)

`internal/core` now delegates VideoSearch/VideoDetails/VideoPlay directly to the shared Core provider, without duplicating parsing. Media reads reuse the existing transport's cookie and redirect policy and return a closable stream rather than buffering a whole film into the 64 MiB source-response limit. Video JS Network uses the existing `comic:<key>` partition; media uses the same one, with an actual cookie test demonstrating that sign-in state reaches playback requests.

`internal/player.MediaProxy` is a per-play media proxy. The player accesses generated resource IDs; HLS parsing and rewriting stay in Core. Loopback end-to-end cases cover relative references after a master-playlist redirect, child playlists, AES keys, initialization fragments, Referer, Range/206/Content-Range, first data before response completion and upstream HTTP cancellation when the session closes. Media marked `Encrypted` (such as unsupported DRM), sniff-required and DASH playback return explicit unsupported errors. Proxying HLS AES-128 keys does not establish DRM support.

The historical transport-test log is `E:/Temp/desktop-media-proxy-tests.log`; current UI, HLS decoding and resume evidence are listed separately above. Network unit tests are not browser-decoding evidence. Existing comic requests retain their bounded whole-body behavior; media does not use that whole-response timeout and is controlled by the caller context and session close.

## Windows installer and portable package 0.2.2 (2026-11-02, commit `bcbe602`)

Includes subtitle-track selection and the audio/subtitle criteria. Portable ZIP `21,544,794` bytes, SHA-256 `c69db3d7762366784f891dd1cc23e99c97f89378621fcae3d90a7a754f6cf554`; per-user setup `18,701,040` bytes, SHA-256 `461e4fb02d046f22ffe72014d68a04dfd6899b1f4718e517bd1ca8247e145f74`.

Measured: after a silent install into an isolated directory (`skipRegistration=1`), `-version` reported `0.2.2`/Core `214e7782`/commit `bcbe6025`; the installed program opened a real window titled `Nekophage`, responded normally, and `Uninstall.exe /S` removed the directory after it closed. The packaging script itself passes ZIP CRC, per-file hashes after extraction and the extracted `-version`. **Unverified**: clicking through to subtitle playback inside the packaged GUI (the subtitle evidence comes from `cmd/uicheck` on the same commit), plus signing and a machine without WebView2.

## Windows installer and portable package 0.2.1 (2026-11-02, commit `e27d5f7`)

`python scripts/package-windows.py --nsis <makensis>` now emits a portable ZIP (`21,544,447` bytes, SHA-256 `a5e9f549927a2e937ba196e0fb862140d4c2fa39a3c9278ed05747d7b8f5c3b0`) and a per-user setup (`18,697,739` bytes, SHA-256 `58e980bacf3d6337abc8935ddfab950c392d4cf0ed6b2512b2bdc3d8d1e90025`). The script verifies the ZIP CRC, per-file hashes after extraction and the extracted `-version`; installation behaviour is measured separately.

Measured (receipt `E:/Temp/desktop-installer-0.2.1-receipt.json`): a silent install into an isolated directory produced all 12 files, every file hash matched `SHA256SUMS.txt`, and `-version` reported `0.2.1`/Core `214e7782`/commit `e27d5f79`. A normal install wrote the current-user uninstall entry and Start menu shortcut, the installed program opened a real window (title `Nekophage`; toolbar with search/browse/shelf/category/source favorites/source manager/AI/video; screenshot `E:/Temp/desktop-installed-0.2.1-window.png`), and `Uninstall.exe /S` removed the directory, entry and shortcut. A `/NOREGISTER` install recorded `skipRegistration=1` and wrote no registry key or shortcut, nor on uninstall; a sentinel file placed in `%APPDATA%\Nekophage` survived uninstall, so the profile is preserved.

Unverified: signature and SmartScreen reputation, a machine without WebView2, other Windows versions. This build includes `0d8ce15`, `30d460f`, `8f1d812` and the decoded-audio-byte criterion, superseding the `0.2.0` package below.

## Windows portable package 0.2.0 (2026-11-02, commit `7dc3051`)

Rebuilt after the video window landed: `build/releases/Nekophage-0.2.0-windows-amd64-7dc3051.zip`, **21,520,864 bytes**, SHA-256 `18250017734e129ab00ff9573392909aea847b72cc6e6f7c719bebd9ac332322`. ZIP CRC, per-file SHA-256 after extraction and the extracted `-version` (reporting commit `7dc3051`) all pass. The extracted program was then started with an isolated `APPDATA`/`LOCALAPPDATA`: window title `Nekophage`, 1100×760, capture `build/probes/package-0.2.0/packaged-window.png`; the window's close button ended it and a follow-up check confirmed the process was gone. **The user's real configuration directory was not touched.**

Packaging caught a real interface defect: the new 视频 button pushed the toolbar past the 1100px default width and was cut off at the right edge. After narrowing the source picker, the mode switch and the search field, captures of both the development and the packaged build show the last control fully. Logs `E:/Temp/windows-package-build-0.2.0b.log` and `E:/Temp/nk-package-window-0.2.0b.log`.

This is still an unsigned portable package, not an installer (this machine has no Inno Setup or NSIS and no code-signing certificate), and there is no remote to publish to.

## Windows portable-package acceptance (2026-11-02, commit `dbbaf0a`)

`python scripts/package-windows.py` produced `build/releases/Nekophage-0.1.0-windows-amd64-dbbaf0a.zip` (20,755,698 bytes). ZIP CRC, per-file SHA-256 and extracted `-version` checks all passed, reporting Core `214e7782190d` and commit `dbbaf0a`. The extracted production executable was also launched with isolated APPDATA/LOCALAPPDATA in that child process only. The host window receipt confirmed the Nekophage title and search/browse/shelf/source-manager/AI controls; using its close button exited and the postcondition confirmed the window disappeared.

This is an unsigned portable build, not an installer; nothing was published remotely. Window capture: `build/probes/package-dbbaf0a/packaged-window.png`; packaging log: `E:/Temp/windows-package-build.log`; UI receipt: `E:/Temp/windows-package-ui-receipt.json`. Evidence on this Windows host does not verify macOS or another Windows machine.

## Current Windows window rerun (2026-11-02, code `00a1055`)

Using an independent test profile and loopback image fixture, `cmd/uicheck` displayed Probe Comic, its cover, tags and Chapter 1 in the native detail window and opened the reader. The on-disk `shelf.json` recorded `callback_probe / Chapter 1 / page 1`. An independent `cmd/readercheck` then verified `figures=1 images=1 decoded=1 first=8x8`. This is actual Windows-window and decoding evidence after the cancellation/cache changes; screenshot size is not used to infer reader contents, and a fixture does not establish current live-site availability.

Logs: `E:/Temp/desktop-delivery-ui.log` and `E:/Temp/desktop-delivery-reader.log`; native detail capture: `build/probes/delivery-00a1055/detail.png`. The new `scripts/package-windows.py` produces an unsigned portable ZIP and verifies extracted file hashes and `-version`. Window validation and packaging checks remain separate; a command-line launch is not a GUI pass.

## Cancelling page reads (2026-11-02)

Generation cancellation now covers browsing, categories/rankings, details, tags and source-favorite reads inside details. Changing mode or source, closing details or opening another comic retires stale reads and prevents late results from reaching the new page. The comment-capability check runs in the background under the detail context instead of waiting for Core while holding the UI lock. Source changes clear old category declarations; opening a comic from merged search preserves the grid for returning to it.

Completed lists survive navigation; interrupted lists reload on re-entry instead of remaining permanently busy. Source-favorite mutations are not cancelled when leaving details. Real Core + JS + loopback HTTP cases independently verify server-side connection cancellation, old-call completion and a successful subsequent read for details, browsing and categories. State regressions cover source-switch cleanup and the distinction between completed and interrupted lists.

## Image cache and download budgets (2026-11-02)

Downloads are bounded at 32 MiB and 400 entries; bitmaps at 128 MiB and 160 entries. The first reached bound evicts the oldest entry. Accounting uses download slice capacity and reserves RGBA pixels and the complete mip pyramid. MyGo's `DecodeBitmap` closure retains encoded data for re-decoding, so that data is included in the bitmap budget: clearing the download map never proved it was freed. Individually over-budget images use a placeholder with a failure marker instead of repeatedly downloading or decoding; dimensions are checked before allocating over-budget pixels. GPU textures and other process overhead are outside this cache budget; no process peak-memory bound is claimed.

Covers and saved-page images share four download slots. Queued cancellation does not start a download; completion releases the slot. A loopback HTTP case verifies peak concurrency and subsequent progress, with race testing of queued cancellation. Cache cases cover byte eviction, replacement accounting, retained encoded data, odd-dimension mips, corrupt-data release, huge header sizes and revisiting keys. The entry-eviction case also fixes its ineffective `a-0` lookup: the actual first key was `a/0`.

## Search cancellation (2026-11-02)

Search generations now own a cancellable context. Starting a new search, changing source or leaving search mode stops the old search's HTTP work. Pasted links, single-source pages and multi-source searches all use that context; multi-source jobs can also leave while waiting for a concurrency slot. Pages retain individual timeouts, so completing one does not cancel another. A stale ticket starting late receives an already-cancelled context rather than joining the new search.

`TestSearchReplacementStopsHTTPAndLeavesNewRequestUsable` uses the real Core, a JS fixture and loopback HTTP to observe the old server connection being cancelled, the old Core call failing and the next search on the same source returning new results. It does not sleep to infer cancellation. Generation cases also cover late starts, page isolation and leaving search mode. This paragraph records the search fix; the later browse/detail and cover validation appears above. It does not establish cancellation for arbitrary mutations.

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

**Interface evidence for the AI window against the real Magpie (2026-11-01)**: `cmd/uicheck` gained `-ai` / `-ai-test` / `-ai-model` / `-ai-prompt` / `-ai-out`, which open the AI window — a window of its own, so it needs its own capture — and drive the real flow. Result: address `http://127.0.0.1:3425`, `Magpie 可用,共 53 个模型或路由组`, and the catalogue listing `group/laincat`, `group/laincat-subagent`, `auto/gpt-6-luna`, `auto/deepseek-v4.1-flash`, `auto/minimax-m3`, `auto/deepseek-v4-pro`. Sending through `group/laincat` produced a real streamed answer in the window: "我收到了你的消息。". Captures `ui-ai-magpie.png` and `ui-ai-answer2.png` are kept with the evidence. **Re-checked on Core `d81445f` (2026-11-01)**: the same window connected again (`53 个模型或路由组`) and a different prompt came back as a real streamed answer — "我在验证结果是否符合你的要求，以及关键行为是否按预期工作。" — so the AI path still works on the current pin.

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

Desktop visual evidence (2026-11-01): `ui-webview-login.png` shows 在网页里登录 and 登出 together in `nhentai`'s settings; `ui-webview-ehentai.png` shows in-page login, four cookie fields, 校验并保存 and 登出 together in `ehentai`'s; `ui-callback-wnacg.png` shows `Refresh` from `wnacg`'s own `buttonText`. All three live under `E:/Temp/nkprobe/`, captured by `cmd/uicheck -settings` against the real sources. **Two more, as a before and after**: `ui-cb-quiet2-settings.png` is before the fix, where pressing only `Quiet` still put 已完成 beside the `Run` row because `callbackMsg` was one string for the whole panel; `ui-cb-quiet3-settings.png` is after it, with only the pressed row showing a result. `ui-wnacg-cb-settings.png` is a real source: pressing `Refresh` puts 已完成 on the 刷新域名列表 row and leaves the other settings alone. `-callback` now goes through the same entry the button uses, so the capture shows what a person would see.

The whole login chain is now exercised, not only photographed (2026-11-01, `455d0ec`, `487b15a`): `cmd/loginprobe` serves a local page whose `/login` is titled `Sign in` and which moves itself to `/done`, titled `Logged in`, setting a cookie on the way. That drives the window, the WebView, the polling, the source's own `checkStatus`, the cookie import into the source's partition and `onLoginSuccess` with nobody at the keyboard. Five modes each pin one behaviour: `ok` keeps one cookie and `Network.getCookies` sees it inside the callback; `timeout` ends with an error when the source never agrees and stores **no half login**; `failcallback` reports the source's own reason **immediately**; `latecookie` keeps polling when the page agrees before its cookies exist; `close` **ends the wait as soon as the window is closed**.

The last one is the defect this whole thread started from, and it now has end-to-end evidence: `LoginWindow` gained an `OnWindow` hook (tools only), so the probe closes the window for real three seconds after the page loads. The wait ends within **3.4 s** with "the window was closed before the source agreed" and stores nothing. **Checked the other way**: removing the `OnClosed` wiring makes the same probe report `the wait outlived the window by 1m0s — it kept polling a closed page`.

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


## A live sweep of the sources (2026-11-01, 35 official scripts)

`NK_LIVE_DIR=… NK_LIVE_QUERY=love sh scripts/live-sources.sh` reported **18 passed, 3 that need a server the user runs, 13 failed**. Every failure is attributed below, and none of them is a regression in the client:

| Source | What happened | Why |
|---|---|---|
| `manga_dex` | `Unexpected token <` | The site answered with HTML (its 302 edge challenge); the core does not support that kind |
| `mycomic`, `mh18`, `ykmh` | `CloudflareException` | A site challenge that needs clearance, again a kind the core does not support |
| `baozi`, `ikmmh` | 403 | The site refuses |
| `comic_walker`, `shonen_jump_plus` | `Unexpected token <` | The site answered with HTML |
| `happy` | `Post …: EOF` | The site closed the connection |
| `picacg` | `Not logged in` | A real account is required |
| `zaimanhua` | `漫画不存在或已被删除` | The sample is gone, in the source's own words |
| `jcomic` | image URLs are empty | The source returns empty strings; the client refuses them correctly (`image URL is not http(s)`) |
| `manwaba` | the cover "is not an image" | **Measured**: the 13328-byte cover comes back with `Content-Type: image/jpeg` but starts with `aa254a…` rather than `ffd8`, so the site serves encrypted bytes. Refusing to treat that as a picture is right |
| `mh1234` | `Cannot read properties of null` | The script expects an element the site no longer has, which is the source being brittle |

## Twelve window probes rerun after the header and installer fixes (2026-11-03, commit `580f9fd`)

After dropping the header hint, adding `SetOutPath` to the uninstaller, the install rollback and the subtitle cap, all four environment variables ran again: exit=0, **PASS=12, FAIL=0, SKIP=0**, log `E:/Temp/nk-desk-probes-6f120fe.log`. The coverage is the same as the previous round (login, challenge, dialog, page reads after close, favourites, reader decoding, comments, browse chips, shelf, category page, official callbacks and tags), so those changes did not break window behaviour. Still development-build windows with local fixtures, not acceptance inside a package. After the fixture page became 320×1200 and `-progress` started reading only the page's own state, the suite ran again at `fdfa83f`: still 12 PASS / 0 FAIL / 0 SKIP (`E:/Temp/nk-desk-probes-fdfa83f.log`).

## Twelve window probes rerun on the subtitle-cap commit (2026-11-03, commit `cc85ad9`)

After the subtitle-record cap landed, the real-window probes ran with all four environment variables (`NK_OFFICIAL_SRC` + `NK_PROBE_SRC` + `NK_FAV_SRC` + `NK_READ_SRC`): **exit=0, PASS=12, FAIL=0, SKIP=0**, log `E:/Temp/nk-desk-probes-full-cc85ad9.log`. They cover the login window (success, timeout, failing callback, late cookie, close), the challenge page, closing a source dialog releasing its caller, a closed window still taking updates and page reads, the source's own favourites, the reader decoding a loopback page, a comment thread, browse chips following the declaration, reading landing on the shelf, the category page drawing and loading, and the callbacks and tags of 34 official scripts. `scripts/check.sh` passes at the same commit (`E:/Temp/nk-desk-check-shelf.log`).

These are real WebView2 windows from a development build, not the executable inside the 0.2.6 package; the fixtures and official scripts are local files, with no live site or account involved.

## All twelve pass, none skipped (2026-11-01)

One full run with all four environment variables set (`NK_OFFICIAL_SRC` + `NK_PROBE_SRC` + `NK_FAV_SRC` + `NK_READ_SRC`):

```
exit=0   PASS=12   FAIL=0   SKIP=0
```

That says more than "each check passes on its own": **the new checks do not interfere with each other** — declaring a category did not change the browse chip count, the runner-style loopback page did not disturb the comment thread, and the real-source reader and the served-page reader each ran their own way. Running them one at a time could not show that.

## Twelve window checks (2026-11-01, Core `d81445f`)

`sh scripts/desktop-probes.sh` runs every check that needs a real window; all eight passed in one run, with three of them taking part only when the environment says so:

```
PASS  login window: ok, timeout, failing callback, late cookie, close
PASS  challenge window: cleared and dismissed
PASS  closing a source dialog releases its caller
PASS  a closed window still takes updates and page reads
PASS  the favourites the source keeps are listed (NK_FAV_SRC=the test source)
PASS  the reader decodes a real chapter          (NK_READ_SRC=manhuagui)
PASS  the reader decodes a served page           (the offline form, when only NK_PROBE_SRC is set)
PASS  a comment thread the source answers        (NK_PROBE_SRC=the test source)
PASS  the browse chips follow the sections       (NK_PROBE_SRC=the test source)
PASS  reading lands on the shelf                 (NK_PROBE_SRC=the test source, offline)
PASS  the category page draws and loads          (NK_PROBE_SRC=the test source)
PASS  every callback the official sources declare  (NK_OFFICIAL_SRC)
PASS  what every source's own tags mean            (NK_OFFICIAL_SRC)
```

The favourites and browse checks **do not look at captures**: one asks the window for `count/first/err`, the other for `sections/chips/comics` and asserts `chips == (sections>1 ? sections : 0)`. The browse check exists to guard the "a single section draws no chip row" change, and its negative sample is a source with two sections and an empty list — that fails with `the current section holds no comics`.

The reader is the one that needs a real **decoder**: a source can return plausible byte counts that are not a picture, so it requires `decoded>0` and runs on a real `manhuagui` chapter (210 pages, 4 decoded at first, first image 1520×1300). The negative run used `copy_manga`, which now answers 210 while loading details; the probe fails on it and prints the site's answer rather than "a command failed".

The comment check was first written as "the capture has content, so it passed" — and it passed against a source that **declares no comments**, because an empty thread and a real one capture to within a few hundred bytes of each other. Asking the window for `count / author / body / err` instead made the same source report `count=0 err="源返回错误:core: wnacg keeps no comments"` immediately. That kind of false positive only shows up if the check is run against a case that should fail.

## The reader's decode can run offline now (2026-11-01)

The real-source run needs the network and a site that cooperates. `cmd/readercheck` gained `-serve PORT`: it answers on loopback with a PNG drawn at runtime, and the fixture's chapter points at `http://127.0.0.1:18731/page.png`, so **an image really decoding into a picture can be shown without a network**:

```
go run ./cmd/readercheck -script UITests/Fixtures/callback_probe.js -query probe -serve 18731
→ page title=Probe Comic figures=1 images=1 decoded=1 first=8x8
→ the reader rendered 1 of 1 pages
```

In the probe suite use `NK_READ_SERVE=18731 sh scripts/reader-probe.sh`. It is the same port and the same fixture the iOS interface tests use, so both clients can be checked on one input.

## The reader, re-run on the new pin (2026-11-01, Core `d81445f`)

`go run ./cmd/readercheck` against two real sources, with the attribution kept separate:

| Source | Result | Attribution |
|---|---|---|
| `manhuagui` | details list 210 chapters; reader reports `figures=210 images=210 decoded=4 first=1520x1300`, with images fetched through the reader's own proxy | passes |
| `komiic` | the same numbers (`figures=210 images=210 decoded=4 first=1520x1300`) — one HTML-scraping source and one API source agreeing | passes |
| `copy_manga` | `Invalid status code: 210` while loading details | **the site** (it has answered 210 before; not a client or source-script regression) |

`decoded=4` does not mean only four pages rendered: the reader decodes as it goes, so that is how many were decoded at that moment. The window capture does not include the web layer, so the evidence is that log line rather than the image.

## Not verified

- **A live Cloudflare challenge**: none could be reproduced. On 2026-11-01 `cmd/challengeprobe -mode inspect` (new) looked at what the window actually holds, so the two sites are no longer guesses:
  - `18comic.vip`: the window holds the **ordinary site** (title 免費A漫 - 禁漫天堂, cookies `yuo1`/`AVS`/`ipcountry`/`theme` and others), **no challenge page at all**, therefore no `cf_clearance`. So "cannot be judged passed" does not mean the client fails to see a cookie: this kind of site has **no challenge to pass**. The done rule wants `cf_clearance`, the page never needs one, and the client can only wait out the deadline.
  - `mangadex.org`: **the whole chain was captured on 2026-11-01.** `GET https://api.mangadex.org/manga?title=love` answers `302` with `location: /edge/challenge?src=<base64 of the original path>`, and a body of just `Redirecting to bot challenge page... (if this isn't a browser, don't send a browser UA)`. That challenge page is `200` plus `challenge-assets/challenge.js`, which reads `#c`/`#d`/`#h` and runs a **proof of work** (`SOLVER_LIMIT_MS = 400000`, difficulty 17 by default), then `location.replace`s back to the original address carrying `powblock=<nonce>&pbchal=<token>`. In other words **its clearance lives in URL parameters, not in a cookie**. The current `DoneRule` can only say "these markers are absent and these cookies must be present", so it cannot describe this; supporting it needs a completion condition of the form "the URL carries these parameters". **That is a product decision, and the rule was not loosened on my own.** One observation worth keeping: because the challenge solves itself in page JavaScript, a person opening it in the WebView passes automatically — only the rule to recognise that is missing.
  - Both ended `-mode live` with `context deadline exceeded` and zero cookies, matching what inspect saw. The **locally reproducible loop** (a page writes `cf_clearance`, the window calls it passed, the cookie reaches the partition, the page's real User-Agent comes back) runs as `clear` in `scripts/challenge-probe.sh`; what is still missing is a successful sample on a **real site**.
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
go run ./cmd/uicheck -script /path/to/manga_dex.js -query "one piece" -open -out build/uicheck.png

# verification: the window alone, and Core's whole challenge flow (a local
# site that answers like Cloudflare)
go run ./cmd/verifycheck -local
go run ./cmd/verifycheck -e2e

# the browser login's four endings with nobody at the keyboard (a local page),
# and the challenge window
go run ./cmd/loginprobe -mode ok|timeout|failcallback|latecookie
sh scripts/login-probe.sh
sh scripts/challenge-probe.sh

# a source's setting callback without the interface: one, or all it declares
go run ./cmd/cbprobe /path/to/source.js - --all
NK_OFFICIAL_SRC=/path/to/official-srcs sh scripts/callback-probe.sh

# press one tag on the detail page and see what the source's onClickTag makes of it
go run ./cmd/uicheck -script /path/to/source.js -query keyword -open -tag 'Genre|probe' -out tag.png

# check a source script's reading contract: install, search, details, and every tag
# group's action — a source can map one namespace to a search and another to a
# category, so pressing only the first group says little

# the source the interface tests ship has to pass this first
go run ./cmd/comicprobe /path/to/source.js keyword

# closing a source's dialog releases its caller; a closed window still takes updates and page reads
go run ./cmd/dialogprobe
go run ./cmd/updateprobe

# ask every official source what its own tags mean (needs the network)
NK_OFFICIAL_SRC=/path/to/official-srcs sh scripts/tag-probe.sh

# open a comment thread in a real window and capture it (no network; pass a source
# script that answers loadComments)
sh scripts/comment-probe.sh /path/to/source.js

# the reader decodes real pages from a real chapter (needs the network; when a site
# refuses, the script says so rather than blaming the client)
NK_READ_SRC=/path/to/source.js sh scripts/reader-probe.sh

# all of the window-dependent checks at once
# an unset item prints SKIP rather than being quietly dropped:
#   NK_OFFICIAL_SRC for official callbacks and tags, NK_PROBE_SRC for the comment
#   thread and the browse chips, NK_FAV_SRC for source favourites, NK_READ_SRC for
#   the reader's decode
NK_OFFICIAL_SRC=/path/to/official-srcs NK_PROBE_SRC=/path/to/source.js NK_FAV_SRC=/path/to/source.js NK_READ_SRC=/path/to/source.js sh scripts/desktop-probes.sh
```

The comment thread is the one added this round: the client has driven real sources' threads for a while, but that path needs the network and a login. Given a source script that answers for itself (the iOS repository's `UITests/Fixtures/callback_probe.js` is one), the check runs on any machine, and what it checks is whether the comment screen draws the author, the body and the buttons at all.


The category check proves **both halves of the contract** at once: `category` supplies the choices, `categoryComics` supplies the loader. Three inputs were measured — a full declaration gives `parts=1 values=2 comics=1` and passes; parts with no loader give `comics=0` and fail with `source does not define this function`; nothing declared fails with "offers no category parts". A page that draws its choosers and loads nothing is exactly the "the site looks empty" fault this exists to name.

`NKDEBUG=1` prints cover download and decode lines.

## Source favourites have a repeatable check too (2026-11-01)

The comment thread's false positive taught the lesson, so this one asks the view for its contents from the start. `uicheck -favorites` now also prints

```
favorites: count=1 first="Probe Comic" err=""
```

`scripts/favorites-probe.sh` passes on `count>0` with a non-empty title and prints the source's own error when it fails. Three inputs:

| Source | Result | Attribution |
|---|---|---|
| the test source (declares `favorites`) | `count=1 first="Probe Comic"` | passes |
| `manhuagui` | `count=0`, error 请先登录漫画柜账号 | **needs a login** |
| `komiic` | `count=0`, error `Cannot read properties of null (reading '0')` | **the source script** |

Favourites are declared in a **top-level `favorites` block**, not inside `comic`, and putting them in the wrong place yields an empty page — which is exactly why the check has to read contents rather than a capture.

## The gate runs with the race detector by default (2026-11-01)

`go test` in `scripts/check.sh` was plain mode, and this client loads comments, browses and images on background goroutines while the interface reads the same state — exactly what the detector exists for. It is now `-race` by default:

```
PASS  go test (race detector)      # 13 packages, exit 0
NK_NO_RACE=1 → PASS  go test (no race detector)
```

The mode is printed in the result, so **turning it off is visible**: "it ran" and "it ran with the detector" do not collapse into one thing. A machine with no C toolchain uses `NK_NO_RACE=1`.

## A single-browse source no longer draws a one-chip row (2026-11-01)

The same state used to look different on the two clients: iOS draws no section row when a source has one browse and no category (`BrowseView.swift:50`), while the desktop always drew one — a row holding a single chip that is permanently selected, which reads as noise.

The desktop now follows the same rule (`len(pages) > 1`, divider included). Both cases were checked by capture:

| Source | Result |
|---|---|
| one declared browse | no chip row, straight to `Solo Comic` and `第 1 / 1 页` |
| two declared browses | both chips, `Probe Shelf` selected, `Probe Second` switchable |

`go vet` and `go test ./internal/ui` pass. This kind of difference does not announce itself; it only shows up when both clients are given the same input.

## Seed generation is deterministic across platforms (2026-11-01)

`cmd/seedgen` put the file's bytes straight into a record's `body`, so **the same fixture** produced a seed with a different md5 on Windows (CRLF) and on macOS (LF). That is not theoretical: the verification machine received a different hash, so "the seed matches the tree" could not be answered with a hash — it had to be compared by eye, which is exactly how this went wrong.

It now turns `

` and lone `
` into `
` before installing:

```
the same command run twice → byte-identical output
of the four sources only callback_probe's sha256 changed (the others had no CRLF)
the iOS check-seed.sh still passes (it normalises line endings anyway)
```

## Cross-compilation: all three targets build (2026-11-01)

"compile and run on macOS and Linux" has been on the next list for a while with only macOS evidence from the Mac worker. This round settles the "does not compile" half:

```
GOOS=linux  GOARCH=amd64  go build ./...   exit 0
GOOS=linux  GOARCH=arm64  go build ./...   exit 0
GOOS=darwin GOARCH=arm64  go build ./...   exit 0
```

**This is portability, not execution**: the window layer needs GTK/WebKit runtime libraries, and nothing built here has been started on Linux. So "works on Linux" still has no evidence, while "does not compile on Linux" can now be ruled out.
This machine also has nothing that could run it (checked 2026-11-01): `wsl.exe` exists but **no distribution is installed**, and `docker` is absent. Closing that gap needs a Linux environment installed first, which is a system change and was not made unasked.

## Next

1. Find a reproducible live Cloudflare challenge (a site that answers a command-line client with 403 and does hand out `cf_clearance`) and check the done rule against it. Sites like 18comic.vip, which open in WebView2 but never set `cf_clearance`, cannot be judged passed today; relaxing the rule is a design decision.
2. Build and run on macOS and Linux.
3. Wire video sources and sniffing — the three options, with the work and cost of each, are measured in [VIDEO-PLAYBACK.en.md](VIDEO-PLAYBACK.en.md): WebView with `<video>` (pure Go, reusing the reader's proxy), a native player (needs cgo and its own decoders; MyGo ships only GIF), or sniffing a direct URL for the system player (headers cannot travel with it).
4. The workbench module.
