# Bundled player assets

[简体中文](NOTICE.md) | English

`hls.min.js` is HLS.js v1.7.4, unmodified, from
`https://github.com/video-dev/hls.js/releases/download/v1.7.4/release.zip`
(`dist/hls.min.js`, 623915 bytes, SHA-256
`cfc70cc9e9fc63d071919e087670abc760c8fd1bc889b88588026e4d22193bbe`).

- HLS.js © 2017 Dailymotion and contributors, Apache License 2.0 — the full text
  is in `HLS-LICENSE.txt` next to this file.
- It is bundled rather than fetched at run time so the player works offline and
  a remote script cannot change what the page executes. Do not replace it with
  a CDN import; update this note, the file and the hash together.

`player.html` is this repository's own page, licensed like the rest of it.
