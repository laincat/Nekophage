# Nekophage architecture

English | [简体中文](ARCHITECTURE.md)

Date: 2026-10-30. This records the framework choice and the layers. Wails is gone: the old version used it to wrap a web page in a desktop window, and now the whole desktop app is one Go program.

See the [code audit](CODE-AUDIT.en.md) for fixes and remaining work, and the [interactive HTML report](analysis/architecture-audit.html) for the three-project architecture/dependency map.

## Choices

| Choice | Why |
|---|---|
| Go + [MyGo](https://github.com/egoist/mygo) | Pure Go, no cgo, cross-compiles to three platforms; native UI is drawn by MyGo, so the interface is not a web page; where a page is needed, a WebView sits inside native UI |
| [VeneraCat Core](https://github.com/laincat/VeneraCat-Core) in the same process | Core is a Go library: `js.Runtime` and `plugin.Registry` are used directly, with no gomobile, FFI or cross-process protocol |
| Wails dropped | It gives the whole interface to a web page, while Magpie (the reference) builds on one Library with native windows and panels; Nekophage's main window should be native, and only the reader needs a page |

Reference: [Magpie](https://github.com/yetone/magpie) (MIT). Its layering and modules are the model (one Library as the source of truth, serialised changes, explicit provenance, switches for built-in modules); no code is copied.

## Layers

```
main.go                wiring: data directory, Core, Library, reader, verifier, main window
internal/core          Core assembly and comic data
  transport.go         host.Transport: net/http, cookies by partition, redirect hops
  core.go              js.Runtime + plugin.Registry; search, details, chapters, image requests, parsing
internal/cookies       one cookie jar per source (RFC 6265 matching, persistent cookies on disk)
internal/store         source data, settings and plugin records; values stay as Core wrote them
internal/shelf         reading history and favourites in shelf.json; a crash leaves the old file or the new one, and folder create/rename/delete/move live here rather than in the window
internal/library       install, replace, remove sources; read index.json and compute each entry's status
internal/reader        the nkreader:// scheme: the reader page, /chapter, /img and /progress
  assets/reader.html   one self-contained page (styles and script inline)
internal/browser       the verification window (WKWebView / WebView2 / WebKitGTK)
internal/ui            the main window: sources, search, details, and the reader window's chapter buttons
```

## Data flow

```
Install → library.InstallURL → core.Transport.Fetch (no cookies)
        → Core compiles and validates the script → plugin.Record → store/plugins.json

Search  → Core.Call("search.load") → script's Network.get → Transport.Do
        → partition cookies → response → script parses HTML → JSON → core parses Comics

Read    → the reader page fetches nkreader://chapter → Core.ChapterImages
        → <img src="nkreader://img?u=..."> → Go fetches through onImageLoad → streamed
```

Images never pass through base64 or a data URI in memory: the web engine asks for what it scrolls to, and Go fetches each one.

## Isolation

- Every source's cookies, login state and data live in its own partition; the partition name is `comic:<key>`.
- The system cookie store is not used: the app builds the Cookie header itself.
- Verification windows share the browser's default data store, so only one runs at a time and the store is cleared before a page loads and after the window closes.
- Sources are untrusted: their scripts run in their own engine (Core guarantees that), and a source's HTML only ever appears in the reader and verification windows, never in the main window.

## Workbench (a later module)

The workbench is phase two: edit themes, comic sources and feature plugins, package a publishable directory, and produce `index.json` and an installable package. It follows Magpie's built-in modules:

- The workbench is a built-in module with a switch; off, it costs no start-up time and no memory.
- Its data still goes through the same Library: there is one set of install records, provenance and serialised changes.
- Preview and validation happen on the desktop, but the acceptance test is that Nekularity reproduces the result.
