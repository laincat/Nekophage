# Nekophage status

English | [简体中文](STATUS.md)

## Architecture

The desktop uses MyGo for its main window and separate WebViews for reading, sign-in, and verification. VeneraCat Core runs source scripts and owns the registry, index parsing and backup format. Its optional gohost package provides the Go host. The desktop supplies the profile, platform, source API level, WebView and logging.

| Component | Responsibility |
|---|---|
| internal/core and internal/sourceapi | Assemble the shared Go host. The app and its commands share source API level `1.6.3`. |
| Core gohost | HTTP, source-partitioned cookies, JSON data and settings, typed comic calls, source installation and records. |
| internal/browser | Serialized desktop WebView sessions, reading cookies and verification handoff. |
| internal/collections | One collection manager per profile session. It owns snapshots, refresh and compatibility checks. |
| internal/archive, internal/shelf and internal/searches | Headless backup apply and export, the shelf and search history. |
| internal/player and internal/reader | Fetch through the host, bridge reading, proxy media, cancel per session, handle ranges and keep local playback state. |
| internal/services and internal/magpie | Desktop service credentials and the local Magpie connection. Core stores ordinary service values outside the package. |
| internal/portable and scripts | Portable layout, existing profile migration, Core checkout, verification and release packaging. |

## Features

- Comic sources install from URL, text, files and comic index entries. Source declarations determine which search, category, browse, details, chapter, image callback, tag, comment and favorite actions are available.
- The reader and native image entry points share the host fetch path. It accepts scheme-relative addresses and preserves signed URLs while validating the scheme. The profile shelf skips repeated reading positions.
- Sign-in and verification read cookies from the active WebView before committing them to the source partition. HTTP matches cookies on each hop. Changes to persistent cookies are written to disk, and session cookies stay in memory.
- Collections support comic sources and service packages. Installation verifies the digest and declared identity first. Unfollowing keeps previously installed entries. See [Source collections](SOURCE-COLLECTIONS.en.md).
- Backup preview can restore sources, source data, reading history, shelf entries, kept pages, search history and explicitly selected cookies. Export carries comic sources, source data, settings and search history. Cookie restoration targets installed source partitions. Export excludes cookies.
- The local media proxy supplies request headers and partitioned cookies, processes HLS playlists and cancels work when playback closes. See [Video playback](VIDEO-PLAYBACK.en.md) for playback and subtitles.
- Windows portable builds store data beside the executable. Packaging and updates retain standalone release entry points. See the [project README](../README.en.md).

## Verification

| Command | Coverage |
|---|---|
| `sh scripts/check.sh` | Formatting, the linked Core checkout, vet, race tests and bilingual docs. |
| `go test github.com/laincat/veneracat-core/gohost/...` | Cookie partitions and writes, transport and escaping, install identity and recovery, comic contracts. |
| `sh scripts/desktop-probes.sh` | WebView login, verification and closure, real decode, shelf persistence, playback and subtitles. |
| `sh scripts/reader-probe.sh` and `--layout` | A loopback chapter by default, checked by decoded page count and spread coordinates. |
| `sh scripts/login-probe.sh` | Success, timeout, failed callback, delayed cookie and window closure. |
| `go run ./cmd/verifycheck browser -mode clear` or `-mode dismiss` | Checks the verification window's cookie, user agent and closure behavior. |
| `go run ./cmd/videocheck subtitles` | Active cues and restoring the off selection after reopening. |
| `go run ./cmd/importcheck -file ARCHIVE -dir PROFILE -apply` | Applies a backup through the same path as the UI. Add `-cookies` to restore cookies. |
| `sh scripts/live-contracts.sh` | Uses scripts from `NK_OFFICIAL_SRC` to check the version floor, cookie handoff, logout, callbacks and tags. |
| `sh scripts/live-sources.sh` or `sh scripts/live-sources.sh browse` | Uses `NK_LIVE_DIR` for live reading or browsing. Set `NK_LIVE_REPEAT` for multiple samples. |

Release entry points are `scripts/package-windows.py`, `scripts/installer-windows.nsi` and `scripts/package-macos.sh`. Run `scripts/fetch-core.sh` to sync Core or pass a local checkout path. Run `Nekophage -version` to confirm the generated revision stamp.

## Follow-up and historical baselines

- Carry window and source-switch cancellation, background I/O, WebView isolation, real decode, source partitions and media closure into the UI rewrite.
- See [Video playback](VIDEO-PLAYBACK.en.md) for platform and media items still to verify. Consult the corresponding release tests for installers, signing and native windows on each platform.
- See the [code audit](CODE-AUDIT.en.md) and [architecture audit](analysis/architecture-audit.html) for historical baselines. The old status log is available with `git show 341cb9c:docs/STATUS.en.md`.
