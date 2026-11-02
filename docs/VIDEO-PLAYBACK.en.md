# Video playback: what each option actually costs

[简体中文](VIDEO-PLAYBACK.md) | English

Updated 2026-11-02. The roadmap now implements route A (WebView and media proxy). Other options remain as tradeoff background; the verification record below states actual progress and limits.

## Implementation progress (2026-11-02)

The client media transport and `internal/player.MediaProxy` are implemented: they use the Core provider's search/detail/play declarations, share source cookie and redirect policy, stream media and use Core HLS rewriting to route child playlists, keys, initialization fragments and segments through proxy resources. Loopback cases verify relative paths after redirects, Referer, Range/206, data before response completion and upstream HTTP cancellation on session close.

The main window's 视频 entry reaches search, details, lines/episodes and playback. An actual Windows WebView now verifies native MP4 (`640×360`) and bundled HLS.js 1.7.4 (`320×180`): frame readback, an advancing clock, Referer/Cookie, HLS master/child playlists and fMP4 byte ranges, the page's pause event writing progress, and real resume after reopening. The HLS fixture and commands are in [testdata](../internal/player/testdata/README.en.md); both a video-only and an AAC rendition are verified, the latter reading `18537` decoded audio bytes through HLS.js and `17802` through native MP4 (`E:/Temp/desktop-audio-decoded-counter.log`, `E:/Temp/desktop-mp4-audio-native.log`), while the video-only controls report 0 and exit nonzero. HLS log: `E:/Temp/desktop-hls-browser-playback.log`; native MP4: `E:/Temp/desktop-mp4-auth-real-resume.log`.

The probe's leftover synthetic `3s` POST is removed and failures return nonzero status. Save and resume share a completion rule and preserve fractional seconds. Normal source management now has explicit kind selection, file/URL installation and persistence. Actual Windows native UI installation survives close/restart, and restored sources pass HLS playback. Search/pagination, details and play preparation now use separate cancellable generations. Race cases verify real slow-HTTP cancellation and a successful next read; returned play URLs are rechecked on the main thread. HLS sessions now prefer the bundled HLS.js because native HLS drops `EXT-X-MEDIA` subtitle groups (measured: no subtitle playlist request); the native fallback remains and has no subtitles. On the default path, subtitle selection, active text and Chinese text over the actual window are verified. The selector supports language/Off, while manual operation remains unverified after the desktop tool returned `foreground_unavailable`. PiP, macOS and live online sites remain unverified; audio is verified only down to the decoded-byte counter, not audible output. Sniffing, DRM and DASH explicitly return unsupported errors.

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
