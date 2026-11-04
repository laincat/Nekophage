# MyGo examples and desktop rewrite candidates

[简体中文](MYGO-EXAMPLES.md) | English

On 2026-11-04, following the user's [official Showcase](https://mygo.egoist.dev/showcase) link, we checked local MyGo 0.7.3 examples, GitHub READMEs, module manifests and licenses. Start with native MyGo controls and evaluate Onevra's sidebar/themes and the built-in code editor; keep isolated WebViews for reading and playback. Product dependencies and UI are unchanged.

## Official examples: use existing capabilities first

The pinned local version contains 33 examples. The website has 17 application cards plus an “Add your app” submission heading. The earlier count of 18 applications, based on headings, is corrected here.

| Example | Value for Nekophage | Verified boundary |
|---|---|---|
| [gallery](https://github.com/egoist/mygo/tree/v0.7.3/examples/gallery) | Sidebar, routing, grids, virtual lists, search, overlays, themes and state updates | Windows compilation passed; not a complete media application |
| [embed-webview](https://github.com/egoist/mygo/tree/v0.7.3/examples/embed-webview) | Native sidebar/toolbars host multiple WebViews, with native controls above pages | Windows compilation passed; closest to the existing reader/player structure |
| [code-editor](https://github.com/egoist/mygo/tree/v0.7.3/examples/code-editor) | Native editing, completion, diagnostics and split/unified diffs | Windows compilation passed; ui/editor includes JS, MJS, CJS, JSX, TS and JSON configurations; large files and system IME untested |
| [notes](https://github.com/egoist/mygo/tree/v0.7.3/examples/notes) | Sidebar, rich text, toolbar, Markdown shortcuts and persistence | Source reviewed; not executed |
| [drag-drop](https://github.com/egoist/mygo/tree/v0.7.3/examples/drag-drop) | Files, URLs and text across windows and other applications | Relevant to source/theme/plugin imports; not executed |
| [video](https://github.com/egoist/mygo/tree/v0.7.3/examples/video) | Native animated backgrounds and overlays | GIF decoding only; not evidence of HLS playback |

The official examples are MIT-licensed and may be reused under those terms. Prefer existing `ui`, `ui/editor` and MyGo APIs over new controls or editors. Retain CodeMirror 6 as an editing alternative: the earlier Node highlighting prototype does not rule out the native editor.

## GitHub candidates

| Project | Most useful patterns or reusable parts | Actual findings |
|---|---|---|
| [Onevra](https://github.com/oneo-me/onevra) | Collapsible/resizable sidebar, header actions, themes, Lucide icons, chat and command palette | MIT; public module `github.com/oneo-me/onevra/source`; isolated v0.2026.17 minimal application compiled |
| [MixMusic](https://github.com/fzn0x/mixmusic) | Cover library, search, queue, separation of themes/layouts and community theme/plugin catalogue | README and LICENSE-MIT confirm MIT or GPL-3.0 at the user's option; go.mod uses MyGo 0.7.3 |
| [Video-Downloader](https://github.com/fazi-gondal/video-downloader-mygo) | Download queue, task states and subtitle/audio/format selection | Repository metadata reports MIT; not built or executed |
| [NuxtUI Go](https://github.com/josexy/nuxt-ui-go) | Forms, tables, trees, themes and sidebar | MIT; requires MyGo 0.7.4; upstream documents IME/platform limitations; not compiled here |
| [MyGo Cloud Music](https://github.com/egoist/mygo-cloudmusic) | Native library, lyrics and mini-player interactions | AGPL-3.0; observe behavior without copying source under this project's policy |

GitHub's automatic license field only recognized MixMusic's GPL file; its actual MIT dual license permits reuse. Nekophage has no blanket exclusion of GPL code. Retain applicable copyright and third-party notices when reusing specific parts; the AGPL exclusion remains. MixMusic's application UI and plugin host are internal packages, not an importable UI SDK. Its plugin host uses Sobek, which does not justify replacing the project's pinned MoeJS runtime.

## Actual Onevra integration conditions

Onevra currently uses MyGo's CLI for packaging: its README explicitly says it adds no packager. An older repository description is not evidence of current behavior. The module lives under `source/`; run examples by entering `source/`, then using `go run ./examples/gallery` or `go run ./examples/desktop`.

An isolated program using public `components`, `shells/plain` and `theme` APIs compiled with Onevra v0.2026.17, MyGo v0.7.4 and Go 1.27.3. The first build exceeded a 50-second limit; the subsequent incremental build passed, and both logs are retained. No window, theme/IME interaction or macOS validation ran. Nekophage remains pinned to MyGo 0.7.3.

NuxtUI Go's [compatibility notes](https://github.com/josexy/nuxt-ui-go/blob/main/docs/compatibility.md) describe possible loss of selected text when composing and then cancelling IME input; macOS has only cross-compilation evidence, without desktop execution. These are upstream statements, not reproductions from this round. Verify that exact input sequence before adopting a native source editor or upgrading MyGo.

## Later execution order

1. Reuse Gallery/Embed WebView navigation and window organization while retaining Go services, storage and media proxies.
2. Evaluate a small integration through Onevra's public sidebar/theme interfaces rather than copying a music application.
3. Compare the native editor and CodeMirror 6 with real source files, Chinese input and composition cancellation before choosing.
4. Use MixMusic's library and layout/theme separation to inform product information structure; its plugin SDK is not the unified source protocol.
5. Rewrite UI/UX after the foundational work. This round adds no visual matrix for the old interface.

## Evidence from this round

- Official 0.7.3 Gallery, Code Editor and Embed WebView: Windows `go build` passed; log `E:/Temp/nk-mygo-examples/official-build.log`.
- Onevra minimal program: `E:/Temp/nk-mygo-examples/onevra-probe/`; final build log `onevra-build-retry.log`, initial timeout log `onevra-build.log`.
- Captured official page, GitHub metadata, READMEs, modules and license text: `E:/Temp/nk-mygo-examples/`; GitHub API receipts: `E:/Code/build/receipts/mygo-*.json`.

These are source-review and compilation results, not product integration, UI interaction or performance acceptance.
