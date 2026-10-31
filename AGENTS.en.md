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
sh scripts/check.sh   # gofmt, go vet, go test, docs, partitions — one line per step
```

The steps individually:

```sh
gofmt -l .          # no output expected
go vet ./...
go test ./...       # includes end-to-end tests; no network, httptest
sh scripts/check-docs.sh
sh scripts/check-partitions.sh
```

`check-partitions.sh` guards a partition name: `"comic:<key>"` may appear only in `core.ComicPartition`. It comes from a real bug — a backup restore wrote cookies under the bare source key while every request reads `comic:<key>`, so they landed where nothing would read them and the summary still claimed "N sign-ins restored". **The test asserted the same wrong assumption**, so it passed. That is not uncovered code: the case was confirming a misunderstanding. The only reliable check is a fact source outside the code (the key the transport really uses), plus making the wrong call impossible (take a source key, add the prefix inside) rather than relying on "remember the helper".
## Run the checks before committing — do not pipe them

`bash scripts/check-docs.sh | tail -1 && git commit ...` looks like a gate, but the exit
status is `tail`'s: a failing check commits anyway. That is how a stray file once reached
the repository root. Run the script directly, use `set -o pipefail`, or run it and commit
as two separate steps.
