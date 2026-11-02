# Video playback: what each option actually costs

[简体中文](VIDEO-PLAYBACK.md) | English

Updated 2026-10-31. This is **material for a decision**, not a chosen plan. Playback is a product
call (Core's `OPEN-QUESTIONS.md`, item 3); the page exists because the three routes differ by an
order of magnitude, and picking one from memory would mean rework.

## Implementation progress (2026-11-02)

The client media transport and `internal/player.MediaProxy` are implemented: they use the Core provider's search/detail/play declarations, share source cookie and redirect policy, stream media and use Core HLS rewriting to route child playlists, keys, initialization fragments and segments through proxy resources. Loopback cases verify relative paths after redirects, Referer, Range/206, data before response completion and upstream HTTP cancellation on session close.

This is not yet a usable player. The video UI entry, WebView page, Windows/Linux HLS decoding dependency and watch progress remain unconnected. Nothing silently falls back to the system player; sniffing and DRM are not implemented. Route A's transport is the first implemented step; the remaining boundaries and acceptance below still apply.

**Update, 2026-11-02 (after the paragraph above):** Route A is now connected to the interface: 视频 in the main window → search/details/line-episodes → player window. The page is served by the local `player.Server` and bundles HLS.js 1.7.4 (native HLS preferred where available); positions are stored in `shelf.json` and resume. Measured in a real window: a progressive MP4 decodes actual pixels (`readyState=4`, `640x360`, frame read back as PNG) and a paused position is persisted. Still unverified: **HLS decoded in the browser** (no local fMP4/TS fixture), subtitles, picture-in-picture and a run on macOS.

## What already exists

Core (implemented and tested):

- `video.Provider`: `Search`, `Details`, `Play`
- `Play` carries `URL`, `Format`, `Header` (headers the request needs), `Variants`, `HLS` (segment
  options), `Sniff` (find the stream inside a page first), and `Encrypted` — an explicit flag for
  media the client cannot play, rather than a black rectangle
- HLS parsing lives in `video/hls`

The desktop:

- The reader already owns a **WebView plus a custom-scheme proxy** (`internal/reader`); images are
  streamed through it today
- `browser.Verifier` already opens a WebView for 403 challenges
- `browser.Sniffer.Sniff` returns `ErrUnsupported` on purpose, and nothing calls it

## Option A: WebView with `<video>`

**Work**

1. A player page shaped like the reader: `internal/player` plus `internal/player/assets/player.html`
2. A scheme proxy for media that adds `Play.Header` and passes `Range` through
3. HLS needs `hls.js`: neither WebView2 on Windows nor WebKitGTK on Linux plays HLS natively; the
   library is about 500 KB and can be embedded in the binary
4. If `Sniff` is needed, the first media request has to be caught inside the WebView — the same
   machinery the verification window already uses

**Cost**: pure Go, no cgo, one implementation for all platforms — though WKWebView on macOS plays
HLS natively, so its behaviour will differ slightly and needs its own check.

**Risk**: a site that demands DRM or its own decryption still will not play; `Play.Encrypted` must
become an error the user can read.

## Option B: a native player

**Work**

Start with a fact: MyGo's `ui.Video` plays **decoded frames** (`VideoDecoder` → RGBA frames), not a
media element, and the only decoder MyGo ships is **GIF** (`ui.NewGIFDecoder`). So "native playback"
in this repository means:

1. Demuxing and decoding MP4/HLS yourself (an ffmpeg binding — that is **cgo**)
2. Feeding decoded frames to `ui.Video` on a timeline
3. Per-platform hardware decoding and a separate audio path

**Cost**: cgo breaks the current property that one command cross-compiles darwin, linux and
windows; the work is an order of magnitude larger; audio is another subsystem.

**Where it fits**: precise decoding control, or codecs a WebView cannot play. Not what this project
needs now.

## Option C: sniff a direct URL and hand it to the system player

**Work**

1. Actually implement `browser.Sniffer`: load the page in a WebView, catch the media request, keep
   the URL and its headers
2. Launch the system player with `os/exec` — `xdg-open` on Linux, `open` on macOS, and no single
   command on Windows

**Cost**: least work, but **the headers cannot travel with the URL**. Many sources require `Referer`
or `User-Agent`, which a system player will not add. Playback also leaves the app: no progress
reporting, no shelf integration.

## Side by side

| | Work | cgo | Headers | Progress in the app | Same on every platform |
|---|---|---|---|---|---|
| A. WebView with `<video>` | medium (reuses the reader) | no | yes | yes | mostly |
| B. Native player | large (decoders, audio, cgo) | yes | yes | yes | no |
| C. Sniff plus system player | small | no | **no** | no | no |

**One fact worth checking against**: this repository cross-compiles and vets cleanly for
`darwin/arm64` and `linux/amd64` today. Option B's cgo would end that; A and C would not.
