# 内置播放器资源

[English](NOTICE.en.md) | 简体中文

`hls.min.js` 是未经修改的 HLS.js v1.7.4,取自
`https://github.com/video-dev/hls.js/releases/download/v1.7.4/release.zip`
(`dist/hls.min.js`,623915 字节,SHA-256
`cfc70cc9e9fc63d071919e087670abc760c8fd1bc889b88588026e4d22193bbe`)。

- HLS.js © 2017 Dailymotion 及其贡献者,Apache License 2.0,全文见同目录
  `HLS-LICENSE.txt`。
- 采用内置而不是运行时从 CDN 拉取,这样播放器离线可用,远端脚本也无法改变页面
  执行的内容。升级时需同时更新文件、此说明与 SHA-256。

`player.html` 是本仓库自己的页面,授权与其他部分一致。
