# 本地视频播放夹具

[English](README.en.md) | 简体中文

`hls/` 是 10 秒、320×180、无音轨的 H.264 fMP4 HLS。它包含主清单、子清单和一个共享的 `stream.mp4`;初始化片段与五个媒体分片都用 `BYTERANGE` 读取,可在真实解码时检验 Range 传递。文件只用于开发探针,不内置到正式程序。

画面来自 **Big Buck Bunny**,© 2008 Blender Foundation,[peach.blender.org](https://peach.blender.org/),[CC BY 3.0](https://creativecommons.org/licenses/by/3.0/)。输入是 Nekularity `UITests/Fixtures/media/clip.mp4` 使用的 10 秒样本,原下载自 `test-videos.co.uk`。本夹具做了缩放、压缩和无音轨 HLS 封装。

重建时先准备该 MP4,在本目录执行(需要临时开发工具 FFmpeg,正式应用不依赖它):

```sh
ffmpeg -y -i /path/to/clip.mp4 -an -vf scale=320:180 \
  -c:v libx264 -crf 28 -preset fast -g 60 -sc_threshold 0 \
  -hls_time 2 -hls_playlist_type vod -hls_segment_type fmp4 \
  -hls_flags single_file -hls_segment_filename hls/stream.mp4 hls/main.m3u8
```

Windows 桌面中,从仓库根目录执行:

```sh
go run ./cmd/uicheck -dir /path/to/new-test-profile \
  -video-script internal/player/testdata/hls_probe.js \
  -video-media internal/player/testdata/hls -video-auth -video-hlsjs \
  -video clip -video-frame /path/to/hls-frame.png
```

探针要求真实解码的尺寸和自行前进的时钟,回读实际视频帧,用页面自己的暂停事件保存进度,重新读取磁盘书架,再通过「继续观看」恢复该位置。失败会以非零退出码结束。`-video-auth` 要求所有媒体请求带上声明的 Referer 和搜索时从 `/login` 获得的 Cookie;不满足会返回 403。每个媒体请求的路径、Range 与鉴权结果写入日志,不输出 Cookie 内容。

`hls-audio/` 是同一片段的另一版:320×180、H.264 + AAC 立体声 48 kHz,仍是 fMP4 共享文件加 `BYTERANGE`。重建时把上一条命令里的 `-an` 换成 `-f lavfi -i sine=frequency=440:sample_rate=48000:duration=10 -shortest -c:a aac -b:a 64k -ac 2` 并把输出目录改为 `hls-audio/`(`hls_time`、分片参数保持一致)。它用于验证播放器确实处理了音轨:页面把 HLS.js 解析到的音轨/编解码器写进 `data-audio`,探针加 `-video-audio` 时要求为 `yes`,而纯视频夹具会得到 `no` 并以非零退出。这证明的是**解码管线看到并处理了音轨**,不是扬声器实际发声。

`hls-subs/` 给同一视频加了字幕组:主清单用 `EXT-X-MEDIA:TYPE=SUBTITLES` 指向 `subs.m3u8`,五个 WebVTT 分片合计 10 条字幕,其中 `sub0.vtt` 另有一条 0→10 秒的长字幕,便于任何取样时刻都有文本。它验证的是字幕链路而不是排版:Core 会重写 `EXT-X-MEDIA` 的 URI,播放器按代理地址取到 `subs.m3u8` 与五个 `.vtt`(日志里可见 `auth=true`),HLS.js 把轨设为 `showing` 并把首条文本读回为 `字幕渲染检查`。用 `hls_subs_probe.js` 加 `-video-subs` 时要求字幕条数与文本都非空;没有字幕组的素材得到 0 并退出非零。

`mp4-audio/` 是同一带音轨素材的非分片用法:把 `hls-audio/main.m3u8` 里的 `EXT-X-MAP` 初始化片段与全部 `EXT-X-BYTERANGE` 媒体分片按顺序拼成一个 `clip.mp4`(262,284 字节,SHA-256 `ed70a8687bbdaac74ef1342edc4bedf44c7c9ab67c15d7a56501dc5a332d7e59`),不需要重新编码。它让原生 MP4 解码路径也能被检查:用 `mp4_probe.js` 加 `-video-audio` 应得到 `decoder=native` 且解码字节数大于零;把 `hls/` 同样拼出的无音轨文件用作反例应得到 0 并以非零退出。

`mp4_probe.js` 用同一登录与请求头规则,把 `-video-media` 换成原始 MP4 文件并去掉 `-video-hlsjs`,即可验证原生 MP4 解码和续播。每次使用新的测试 profile,避免旧历史掩盖进度是否真的落盘。
