# Nekophage

English | [简体中文](README.md)

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/assets/brand/nekophage-logo-dark.png">
  <img src="docs/assets/brand/nekophage-logo.png" alt="Nekophage" width="720">
</picture>

The desktop companion of Nekularity: read comics and play videos from Venera-compatible sources. Extension authoring and publishing come later. Written in Go, with an interface [MyGo](https://github.com/egoist/mygo) draws; licensed GPL-3.0-or-later.

Logic comes from [VeneraCat Core](https://github.com/laincat/VeneraCat-Core), compiled into the same executable: no FFI, no cgo, no separate Core process.

## Shape

- **Main window**: native UI, written in Go, drawn by MyGo on the GPU, with no WebView: it opens at once and takes little memory. The reader, verification and player windows use WebViews.
- **Reader**: a web page in a WebView. Go fetches each page through the source's own rules and streams it, so a source that needs a Referer or a login still shows its pages.
- **Verification**: a Cloudflare challenge opens in a window of its own and the app continues once it is passed.
- **Search every source**: one keyword searches all installed sources, results are merged with their source on each card, and opening another source's comic switches to it.
- **Source management**: select a comic source, video source or Kazumi rule and install it from a URL or local file. Kind, script and enabled state persist across restarts. The comic subscription entry still handles comic indexes only.
- **Video**: choose a provider, search, select a line/episode and play. The local media proxy retains source headers and cookies; the player uses native MP4 or HLS.js and persists resume positions. Windows fixtures verify actual pixels, Range, authentication and resume; this does not establish live sites, DRM, subtitles or PiP.
- **Source settings**: a source's `settings` declaration becomes selects, switches and inputs — MangaDex's image quality, Copy Manga's CDN line.
- **Search filters**: the filters a source offers — sorting, content rating, status and the rest — instead of quietly searching with the defaults.
- **Search history**: remembers recent keywords and offers them on the empty results screen; an official backup's search history comes along too.
- **Kept pages**: keep the page you are reading with one click, see them as thumbnails on the shelf, and jump back to that chapter.
- **Shelf**: reading history and favourites; where you stopped is kept with the page and a percentage. Opening a chapter remembers where you are, the details screen keeps a comic in the folder you pick, and 继续 goes straight back to that chapter; folders can be created, renamed and deleted, and comics moved between them.
- **Backup import**: reads an official `.venera` backup, previews it, then imports what is supported — source scripts, installed through the normal flow so nothing from the backup runs unchecked, and their data, kept exactly as written.
- **Local Magpie**: connection, model discovery, streaming through three standard protocols and a cancellation entry are implemented, with evidence against the installed gateway on Windows. Magpie continues to own accounts, provider plugins and routing; the desktop client does not duplicate them. macOS and other environments require their own verification. See [docs/MAGPIE-DESKTOP.en.md](docs/MAGPIE-DESKTOP.en.md).
- **Workbench**: a later module, modelled on Magpie's built-in module switches; see [docs/ARCHITECTURE.en.md](docs/ARCHITECTURE.en.md).

## Status

The reading path (add a source → search → details → read) has real-source and offline-fixture evidence on Windows. The current tree has rerun native details, actual reader decoding and persisted shelf progress; request cancellation and image budgets have race tests. Evidence remains specific to its commit and platform: Windows success does not validate current macOS/Linux behavior. Local MP4/HLS, installation through normal source management and restoration after restart have Windows execution evidence. Real Core/HTTP race cases verify video replacement/closure cancellation, retained pagination queries and rejection of late results; the studio comes later. See [docs/STATUS.en.md](docs/STATUS.en.md).

## Building

Needs Go 1.27 or later.

```sh
scripts/fetch-core.sh      # clones Core into build/core (version in scripts/core-version)
go build -o build/Nekophage.exe .  # Windows; keep the output under build/
./build/Nekophage.exe -version     # version, Core commit, source API level, own commit and Go runtime, no window
go test ./...                     # unit and end-to-end tests
python scripts/package-windows.py # Windows + Python 3.11+: build and verify a portable ZIP
python scripts/package-windows.py --nsis <makensis.exe>  # same, plus a per-user setup.exe
```

On macOS, build the native program with `go build -o build/Nekophage .`. The Windows packaging script writes a ZIP under `build/releases/` containing the executable, license, bilingual instructions, build information and SHA-256 checksums. It actually extracts the archive, verifies all file hashes and runs the extracted executable with `-version`. It does not install the app, alter user data or publish remotely. With `--nsis <makensis>` it also builds setup.exe, which installs under `%LOCALAPPDATA%\Programs\Nekophage` and writes a current-user uninstall entry and Start menu shortcut; uninstall keeps the `%APPDATA%\Nekophage` profile, and `/NOREGISTER` exists for isolated checks.

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
| `internal/player` | Player page, local media proxy and session lifetime |
| `internal/browser` | The verification window |
| `internal/ui` | The main window: sources, search, details, the reader window |
| `cmd/*` | verification tools: window screenshots (`uicheck`), the reader (`readercheck`), sources and verification (`verifycheck`), backup import (`importcheck`) |
| `scripts/live-sources.sh` | one command that runs the live end-to-end test for every source script in a directory |

How the real-source and manual checks are run is in [docs/STATUS.en.md](docs/STATUS.en.md); `go run ./cmd/verifycheck --help` lists its flags.

## Licence

GPL-3.0-or-later; see [LICENSE](LICENSE).
