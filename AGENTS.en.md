# Nekophage development notes

English | [简体中文](AGENTS.md)

For people and AI assistants working on the code. Read before changing it.

## Principles

- Logic belongs in VeneraCat Core. The desktop app provides the interface, networking, WebViews, the player and storage; new logic goes into Core first.
- Core is a dependency, not a copy: `build/core` is fetched by `scripts/fetch-core.sh`, at the version in `scripts/core-version`. Never edit code under `build/`.
- The main window is native UI (the `ui` package). Only the reader, verification and, later, playback and sniffing use a WebView, and always inside native UI.
- Slow work never runs on the interface's thread: finish it in a goroutine, then come back through `Window.Update`.
- Every source's cookies, login state and data are isolated: the partition name is `comic:<key>`, and the system cookie store is not used.
- Sources are untrusted: their HTML only ever loads into the reader and verification windows, never into the main window.
- JSON stays as written: source data values are the bytes Core produced and are not re-encoded; objects whose order matters (chapters) are parsed in document order and never sorted.
- Do not copy code from AGPL projects (Animeko, AnimeFlow) or unlicensed ones (AniCh). When reading Magpie (MIT) or MyGo (MIT), rewrite the behaviour.

## Repository conventions

- The root holds only files that tools or conventions require there.
- Every `.md` exists in Chinese (`X.md`) and English (`X.en.md`), linked to each other at the top. Change both together. `scripts/check-docs.sh` checks the pairs and the root.
- Not every directory needs a README; only the root and `docs/` hold documents, and those come in pairs.

## Checks

From the repository root:

```sh
gofmt -l .          # should print nothing
go vet ./...
go test ./...       # includes end-to-end tests; no internet, they use httptest
./scripts/check-docs.sh
```

Checking against a real source and the interface (needs the network and a desktop session, optional):

```sh
NK_LIVE_JS=/path/to/manga_dex.js NK_LIVE_QUERY="one piece" go test -run TestLiveSource -v ./internal/library
go run ./cmd/readercheck -script ... -query ... -out reader.png
go run ./cmd/uicheck -script ... -query ... -open -out window.png
go run ./cmd/verifycheck -e2e
NK_LIVE_JS=... NK_LIVE_QUERY=... go test -run TestLiveSource -v ./internal/library   # the whole line   # verification: 403, window, retry, on a local fake site
```

Cross-compilation:

```sh
CGO_ENABLED=0 GOOS=linux   go build ./...
CGO_ENABLED=0 GOOS=windows go build ./...
CGO_ENABLED=0 GOOS=darwin  go build ./...
```

Set `GOTMPDIR` and `GOCACHE` when Go's temporary files and cache belong elsewhere.
