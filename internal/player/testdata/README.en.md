# Local playback fixtures

[简体中文](README.md) | English

`hls/` contains ten seconds of 320×180 H.264 fMP4 HLS without an audio track. A master playlist points to a media playlist sharing one `stream.mp4`; its initialization fragment and five media fragments use `BYTERANGE`, exercising Range propagation during real decoding. These files belong to the development probe and are not embedded in the production application.

The picture is **Big Buck Bunny**, © 2008 Blender Foundation, [peach.blender.org](https://peach.blender.org/), [CC BY 3.0](https://creativecommons.org/licenses/by/3.0/). The input is the ten-second sample used by Nekularity's `UITests/Fixtures/media/clip.mp4`, originally downloaded from `test-videos.co.uk`. This derivative scales, compresses and packages that clip as HLS without audio.

To regenerate, obtain that MP4 and run in this directory (FFmpeg is a temporary development tool, not an application dependency):

```sh
ffmpeg -y -i /path/to/clip.mp4 -an -vf scale=320:180 \
  -c:v libx264 -crf 28 -preset fast -g 60 -sc_threshold 0 \
  -hls_time 2 -hls_playlist_type vod -hls_segment_type fmp4 \
  -hls_flags single_file -hls_segment_filename hls/stream.mp4 hls/main.m3u8
```

Run from the repository root in a Windows desktop session:

```sh
go run ./cmd/uicheck -dir /path/to/new-test-profile \
  -video-script internal/player/testdata/hls_probe.js \
  -video-media internal/player/testdata/hls -video-auth -video-hlsjs \
  -video clip -video-frame /path/to/hls-frame.png
```

The probe requires decoded dimensions and an advancing clock, reads back an actual video frame, saves progress using the page's own pause event, reopens the shelf from disk and restores that position through Continue Watching. Failures return a nonzero exit code. With `-video-auth`, every media request must carry the declared Referer and the Cookie obtained at `/login` during search; otherwise it returns 403. Logs contain request paths, Range and authentication results, without printing Cookie values.

`hls-audio/` is another rendition of the same clip: 320×180 H.264 plus stereo AAC at 48 kHz, still one shared fMP4 file addressed with `BYTERANGE`. Regenerate it by replacing `-an` in the command above with `-f lavfi -i sine=frequency=440:sample_rate=48000:duration=10 -shortest -c:a aac -b:a 64k -ac 2` and writing to `hls-audio/` with the same `hls_time` and segment settings. The probe's `-video-audio` requires a positive `webkitAudioDecodedByteCount` after playback; `data-audioDeclared` only records manifest metadata and does not establish decoding. Video-only media still decodes zero audio bytes even if the manifest declares AAC, and exits nonzero. This establishes audio decoding, not audible speaker output.

`hls-subs/` has two subtitle tracks: English first with `DEFAULT=NO`, Chinese second with `DEFAULT=YES`. The five WebVTT segments under Chinese `subs.m3u8` carry 10 cues, including `字幕渲染检查` at 0–10 s; English `subs-en.m3u8` uses `sub-en.vtt`. An implementation that forces the first track therefore selects the wrong language. `hls_subs_probe.js -video-subs-text 字幕渲染检查` requires matching active cue text on the track currently in `showing` mode; cached cues on a disabled track cannot pass. With both declarations changed to `DEFAULT=NO`, subtitles should remain off and the probe should exit nonzero. Chinese text over the video was separately captured from the actual window, not from canvas video-frame readback.

`mp4-audio/` uses the same audio-carrying media without segmentation: concatenate the `EXT-X-MAP` init segment and every `EXT-X-BYTERANGE` media segment of `hls-audio/main.m3u8` in order into one `clip.mp4` (262,284 bytes, SHA-256 `ed70a8687bbdaac74ef1342edc4bedf44c7c9ab67c15d7a56501dc5a332d7e59`); no re-encode is needed. It lets the native MP4 path be checked too: `mp4_probe.js` with `-video-audio` must report `decoder=native` and a positive decoded-byte count, while the video-only file built the same way from `hls/` reports 0 and exits nonzero.

`mp4_probe.js` uses the same login and header requirements. Set `-video-media` to the original MP4 file and omit `-video-hlsjs` to check native MP4 decoding and resume. Use a fresh test profile for each run so old history cannot conceal a missing progress write.
