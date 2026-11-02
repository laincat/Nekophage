# Nekophage

<img src="Assets/brand/nekophage-logo.png" alt="nekophage" width="720">

English | [简体中文](README.md)

The desktop companion of Nekularity: read comics from Venera-compatible sources, with video playback and authoring or publishing of extensions as later modules. Written in Go, with an interface [MyGo](https://github.com/egoist/mygo) draws; licensed GPL-3.0-or-later.

Logic comes from [VeneraCat Core](https://github.com/laincat/VeneraCat-Core), compiled into the same executable: no FFI, no cgo, no separate Core process.

## Shape

- **Main window**: native UI, written in Go, drawn by MyGo on the GPU, with no WebView: it opens at once and takes little memory. Only the reader and the verification window use a WebView.
- **Reader**: a web page in a WebView. Go fetches each page through the source's own rules and streams it, so a source that needs a Referer or a login still shows its pages.
- **Verification**: a Cloudflare challenge opens in a window of its own and the app continues once it is passed.
- **Search every source**: one keyword searches all installed sources, results are merged with their source on each card, and opening another source's comic switches to it.
- **Source settings**: a source's `settings` declaration becomes selects, switches and inputs — MangaDex's image quality, Copy Manga's CDN line.
- **Search filters**: the filters a source offers — sorting, content rating, status and the rest — instead of quietly searching with the defaults.
- **Search history**: remembers recent keywords and offers them on the empty results screen; an official backup's search history comes along too.
- **Kept pages**: keep the page you are reading with one click, see them as thumbnails on the shelf, and jump back to that chapter.
- **Shelf**: reading history and favourites; where you stopped is kept with the page and a percentage. Opening a chapter remembers where you are, the details screen keeps a comic in the folder you pick, and 继续 goes straight back to that chapter; folders can be created, renamed and deleted, and comics moved between them.
- **Backup import**: reads an official `.venera` backup, previews it, then imports what is supported — source scripts, installed through the normal flow so nothing from the backup runs unchecked, and their data, kept exactly as written.
- **Magpie plugins and routing (planned)**: initially connect to the locally installed Magpie and reuse its provider/other plugins and model-group routing. Accounts and rules stay in Magpie; Nekophage handles connections, task mappings and calls. Windows/macOS first, with Linux/Web interfaces reserved. Not implemented; see [docs/MAGPIE-DESKTOP.en.md](docs/MAGPIE-DESKTOP.en.md) for scope and acceptance.
- **Workbench**: a later module, modelled on Magpie's built-in module switches; see [docs/ARCHITECTURE.en.md](docs/ARCHITECTURE.en.md).

## Status

The first phase is the reading path: add a comic source → search → details → read. It builds, passes its unit and end-to-end tests, and opens its window on Windows; real comic sources, macOS, Linux and the verification window are not verified yet. See [docs/STATUS.en.md](docs/STATUS.en.md).

## Building

Needs Go 1.27 or later.

```sh
scripts/fetch-core.sh      # clones Core into build/core (version in scripts/core-version)
go build .                 # builds the Nekophage executable
./Nekophage -version        # version, Core commit, source API level, own commit and Go runtime, no window
go test ./...              # unit and end-to-end tests
```

`go.mod` replaces Core with `build/core`; once Core has release tags it becomes a plain dependency. Linux needs GTK 3 (`libgtk-3-0`) and, for web pages, WebKitGTK 4.1 (`libwebkit2gtk-4.1-0`); Windows needs the WebView2 runtime (Windows 11 has it).

## Layout

| Directory | Contents |
|---|---|
| `internal/core` | Puts Core in the app: runtime, plugin registry, HTTP and images, comic data parsing |
| `internal/cookies` | Cookie jars isolated per source |
| `internal/store` | Source data, settings and plugin records on disk |
| `internal/library` | Installs, updates and removes sources; reads indexes |
| `internal/archive` | reads official `.venera` backups and turns them into a local import plan |
| `internal/reader` | The reader page and the image service |
| `internal/browser` | The verification window |
| `internal/ui` | The main window: sources, search, details, the reader window |
| `cmd/*` | verification tools: window screenshots (`uicheck`), the reader (`readercheck`), sources and verification (`verifycheck`), backup import (`importcheck`) |
| `scripts/live-sources.sh` | one command that runs the live end-to-end test for every source script in a directory |

How the real-source and manual checks are run is in [docs/STATUS.en.md](docs/STATUS.en.md); `go run ./cmd/verifycheck --help` lists its flags.

## Licence

GPL-3.0-or-later; see [LICENSE](LICENSE).
