# Three-project code audit

English | [简体中文](CODE-AUDIT.md)

[Open the interactive architecture report](analysis/architecture-audit.html). This standalone HTML needs no CDN, external fonts or network. It supports 320px mobile screens, desktop, light/dark mode and printing, with project filters and architecture, dependencies, fixes, performance and evidence views.

## Scope and baseline

Baselines: Core `469edfc`, Nekularity `b294f04`, Nekophage `2e5b32f`. Line counts are static measurements of git-tracked files at audit start, excluding Vendor/build. Test-line share is not coverage. Lead owns Core/desktop review, fixes and reporting; an Orca Worker owns iOS review and Mac validation. The unavailable second worker performed no review.

## Desktop fixes in this pass

| Priority | Problem | Change |
|---|---|---|
| P1 | Redirects used the partition jar despite `Cookies=false` | Carry a jar partition only when cookies are enabled; reproduced before the fix |
| P1 | A stale header sent `/private` cookies to `/public` | Clear old cookies before selecting for the next URL; no match means no header |
| P1 | Custom `X-Api-Key` and other credentials crossed origins | Compare each hop with the initial origin; clear Authorization, X-Api-Key, Proxy-Authorization and Cookie; regression includes two hops |
| P1 | `data:null` panicked on write; damaged files were replaced silently | Initialize null maps; reject bad roots/fields/read errors and preserve the original |
| P2 | Failed writes/deletes changed cached state first | Persist a copy before committing cache state; regression blocks the temporary path |
| P2 | A null cookie file caused a later map-write panic | Ensure the jar map is initialized after loading |
| P2 | NKDEBUG printed actual jar cookies | Remove the output; secrets are not diagnostic metadata |
| P2 | Later gates overwrote failure logs and ignored gofmt process errors | Per-step logs and real exit-code handling; negative harness injects broken gofmt/vet |
| P1 | Pin said `469edfc` but the linked `build/core` was still `7662e7b` | Fetch aligned the checkout; a new gate resolves the actual Go module and checks HEAD, clean tree and generated stamp; a wrong-pin negative case passes |
| P3 | Compressed originals and decoded bitmaps were retained together | Release compressed data after decoding |
| P2 | The image cache had no capacity bound, growing for the whole session | Extracted `imageCache`: 400 downloads and 160 bitmaps, oldest evicted first; failures and corrupt images count too and are not retried per frame; five cases cover de-duplication, decode reuse, failure memory and both bounds |
| P2 | A stale search reply could replace a newer screen | `generation` tickets: a new search/detail/browse advances it, and a reply whose ticket is stale is dropped; pagination joins the current ticket without advancing, so an in-flight page is not discarded; three cases include concurrency, and a deliberately broken implementation fails them |
| P3 | Request bytes became a string and then a reader | Use bytes.NewReader to avoid the full conversion |
| P2 | A source's setting callback that returns nothing made the panel print the JSON literal `null` as if the source had said it | `CallSettingCallback` normalises `null` to an empty answer, so the panel's existing 已完成 wording applies; `LoginAccount` had the same problem and was fixed with it; a unit test pins it and removing the normalisation reports `void callback = "null"` |
| P2 | Every callback a source declares shared one result line, so pressing one button showed its answer beside all of them | `manage` keeps answers per setting (`callbackMsgs map[string]string`) and each row shows only its own; comparing captures confirms that pressing `quiet` alone no longer puts 已完成 beside `Run` |
| P2 | Core and the desktop each carry a copy of "read a source's answer as text", so fixing one does nothing | **Structural risk, demonstrated this round**: with the fix only in Core the desktop still printed `null`, because `internal/core` has its own implementation of the same path. Both now apply one rule, and every other `json.Unmarshal(raw…)` site in `internal/core` was read: they decode into structs or bools, so no other path hands a protocol literal to the interface |

**How far the desktop's duplication of Core actually reaches (2026-11-01)**: `internal/core` exports 57 methods and Core's `mobile` exports 56; **24 names are shared**. Reading them one by one, the drift risk sits only in the few places that *parse Core's output as text* — the rest either delegate to the same JS runtime (`WebviewStatus`, `WebviewLoginDone`, `ClickTag` and friends) or decode JSON into structs (`parseComic`, `parseDetails`), and those cannot drift apart. There are four text paths: `CallSettingCallback` and `LoginAccount` (both changed this round so that `null` means "no message"), `CategoryComics`/`ComicValue` (a JSON contract where `null` is meaningful and must stay), and the interface layer's `rawText`. The last was already correct — Go leaves a string untouched when it decodes JSON `null` into it, so it returns an empty string; I added an explicit check, a negative test showed the check was redundant, and it was withdrawn in favour of a test that pins the behaviour. Recording the boundary here because it decides what else has to change whenever Core's output semantics move.

## Architecture conclusions

Retain shared Go Core with native clients. Compatibility and business logic live once; platform UI, networking, sign-in and storage belong to each client. Desktop uses MyGo rather than Wails: the main window is native, with WebViews for reading and verification. iOS uses SwiftUI, Swift 6 and a gomobile binaryTarget, with no additional Swift package dependency.

Core uses MoeJS for per-source runtimes, cascadia for CSS selectors, htmlquery/xpath for XPath, and modernc SQLite for official backup databases. SQLite is a runtime dependency even though desktop state is mostly JSON. Desktop directly uses MyGo 0.7.3 and indirectly uses purego, font shaping and x/image. The HTML lists exact versions.

## Improvement order

1. **Cancel, not only drop**: stale replies are dropped by ticket, but the underlying request still runs to completion. Cancel superseded requests through their context and give cover downloads their own concurrency limit.
2. **Image capacity**: desktop now bounds entries, but entries differ widely in size (a tall page versus a thumbnail), so a byte budget would be more accurate. iOS already has an NSCache cost limit and downsampling but needs an explicit shared inFlight cancellation policy.
3. **Real streaming**: Core/mobile and iOS still wait for entire bodies. Accept only with first chunks before completion and proof that cancellation stops transport work.
4. **Responsibility boundaries**: baseline `internal/ui/app.go` has 1530 lines and `internal/core/core.go` 834. Split around search, browsing and reader lifetimes rather than reducing line counts alone.
5. **Version truth**: desktop now checks actual module HEAD, clean tree and stamp. iOS still needs artifact-level framework version evidence. Generated strings do not verify linked artifacts.

## Verification and limits

No throughput or percentage memory reduction is claimed. No real AI account or vulnerability database scan was used. Passing race/vet/tests does not prove absence of other defects. Cookie/storage/AI/cancellation inputs failed before fixes and pass as regressions. Full logs live in `E:/Temp/tri-audit`. Desktop gates, race checks and Windows builds must pass again after correcting the stale linked dependency; earlier green results against old Core do not verify the new pin.

HTML script rendering, five tabs, four filters and offline-resource checks pass. Headless CDP with an isolated temporary Chrome profile checks every view at 1440/390/320px without horizontal overflow; no user login profile is used. Screenshots live beside the logs. Host-browser login copying failed with `APP_BOUND_ENCRYPTION_UNSUPPORTED`; the global browser mode was not changed.

Mac validation must match patched-file hashes and whole-tree shape rather than use the old 173-test routing prototype. Final delivery lists Swift 6 tests and the App build separately.
