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

`mp4_probe.js` uses the same login and header requirements. Set `-video-media` to the original MP4 file and omit `-video-hlsjs` to check native MP4 decoding and resume. Use a fresh test profile for each run so old history cannot conceal a missing progress write.
