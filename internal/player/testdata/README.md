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

`mp4_probe.js` 用同一登录与请求头规则,把 `-video-media` 换成原始 MP4 文件并去掉 `-video-hlsjs`,即可验证原生 MP4 解码和续播。每次使用新的测试 profile,避免旧历史掩盖进度是否真的落盘。
