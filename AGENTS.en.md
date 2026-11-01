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

In the repository root:

```sh
sh scripts/check.sh   # gofmt, actual Core pin, go vet, go test (with the race detector), docs, partitions, capabilities — one line per step
```

The steps individually:

```sh
gofmt -l .          # no output expected
sh scripts/check-core-pin.sh  # actual module checkout, clean tree and generated stamp must match the pin
go vet ./...
go test ./...       # includes end-to-end tests; no network, httptest
sh scripts/check-docs.sh
sh scripts/check-partitions.sh
```

`check-partitions.sh` guards a partition name: `"comic:<key>"` may appear only in `core.ComicPartition`. It comes from a real bug — a backup restore wrote cookies under the bare source key while every request reads `comic:<key>`, so they landed where nothing would read them and the summary still claimed "N sign-ins restored". **The test asserted the same wrong assumption**, so it passed. That is not uncovered code: the case was confirming a misunderstanding. The only reliable check is a fact source outside the code (the key the transport really uses), plus making the wrong call impossible (take a source key, add the prefix inside) rather than relying on "remember the helper".
## Checking real sources and the interface (needs the network and a desktop session; optional)

```sh
NK_LIVE_JS=/path/to/manga_dex.js NK_LIVE_QUERY="one piece" go test -run TestLiveSource -v ./internal/library
go run ./cmd/readercheck -script ... -query ... -out reader.png        # the reader
go run ./cmd/readercheck -script ... -query ... -layout               # two pages, right to left
go run ./cmd/uicheck -script ... -query ... -open -out build/uicheck.png   # no -out means no file
go run ./cmd/verifycheck -e2e
```

Every check that needs a real window, in one run (three of them take part only when the environment says so):

```sh
NK_OFFICIAL_SRC=/path/to/official-srcs NK_PROBE_SRC=/path/to/source-with-comments.js NK_READ_SRC=/path/to/source.js sh scripts/desktop-probes.sh
```

On their own:

```sh
sh scripts/reader-probe.sh          # needs NK_READ_SRC: a real chapter, real decode, decoded>0
sh scripts/comment-probe.sh /path/to/source.js   # a comment thread; asks the window for count/author/body
sh scripts/favorites-probe.sh /path/to/source.js # source favourites; contents again, not a capture
```

Two of those criteria were learned the hard way and should not be changed back. **Comments** cannot be judged by capture size: an empty thread and a real one differ by a few hundred bytes, and that used to pass against a source that keeps no comments. **The reader** has to require `decoded>0`, because a source can return plausible byte counts that are not a picture. Both were validated against a case that must fail.

## Cross-compilation

```sh
CGO_ENABLED=0 GOOS=linux   go build ./...
CGO_ENABLED=0 GOOS=windows go build ./...
CGO_ENABLED=0 GOOS=darwin  go build ./...
```

Passing only says it **builds**: the window layer needs GTK/WebKit at runtime, and nothing built this way has been started on Linux, so it is not evidence that it works there.

## Run the checks before committing — do not pipe them

`bash scripts/check-docs.sh | tail -1 && git commit ...` looks like a gate, but the exit
status is `tail`'s: a failing check commits anyway. That is how a stray file once reached
the repository root. Run the script directly, use `set -o pipefail`, or run it and commit
as two separate steps.
