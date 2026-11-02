# 视频播放:三个选项各自要动什么

[English](VIDEO-PLAYBACK.en.md) | 简体中文

更新:2026-11-02。当前按路线图实现方案 A(WebView + 媒体代理),其余选项保留为取舍背景;实际完成度与限制以下面的验证记录为准。

## 当前实现进度(2026-11-02)

已实现客户端的媒体传输层与 `internal/player.MediaProxy`:使用 Core provider 的搜索/详情/播放声明,共享源 Cookie 与重定向策略,流式读取媒体,通过 Core 的 HLS 改写把子清单、密钥、初始化片段和分片转为代理资源。loopback 用例已验证重定向后的相对路径、Referer、Range/206、首块提前到达与关闭会话取消 HTTP。

主窗口「视频」已接到搜索/详情/线路分集/播放。Windows 真实 WebView 已验证原生 MP4(`640×360`)与内置 HLS.js 1.7.4(`320×180`):实际视频帧回读、时钟前进、Referer/Cookie、HLS 主/子清单和 fMP4 字节范围、页面暂停落盘与重新打开续播。HLS 夹具和命令在 [testdata](../internal/player/testdata/README.md);无音轨与带 AAC 音轨两份都已验证,后者在 HLS.js 与原生 MP4 两条路径上分别实读 `18537` 与 `17802` 个解码音频字节(`E:/Temp/desktop-audio-decoded-counter.log`、`E:/Temp/desktop-mp4-audio-native.log`),无音轨反例均为 0 且探针非零退出。HLS 日志 `E:/Temp/desktop-hls-browser-playback.log`,原生 MP4 日志 `E:/Temp/desktop-mp4-auth-real-resume.log`。

已移除探针残留的手工 `3s` 进度 POST,失败改为非零退出;保存和续播使用同一个完成判断,保留小数秒。普通源管理已接类型选择、文件/URL 安装与持久化,真实 Windows 原生 UI 安装后关闭/重启仍能选择视频源,恢复后的 HLS 播放通过。视频搜索/分页、详情和播放准备已分代取消,慢 HTTP 的真实断连与下一次读取成功有竞态测试,迟到播放地址在主线程再次判定归属;HLS 会话改为优先内置 HLS.js(原生 HLS 会丢掉 `EXT-X-MEDIA` 字幕组,实测不请求字幕清单),回退原生路径仅作兜底且无字幕;默认路径的字幕选择、活动文本与中文窗口叠加已验证;0.2.4 安装包中的字幕选择框已完成真实操作验收:在同一暂停画面上选 English 后英文可见、选关闭后消失、再选中文后恢复,回执 `E:/Temp/desktop-subtitle-language-024-receipt.json`。字幕选择已独立存入书架并跨播放器重开恢复(`cmd/subcheck`,真实 WebView2),见 STATUS;PiP、字幕样式、变体组刷新后的选择、macOS 与真实在线站点仍待验证;音轨只验证到解码字节计数,未验证扬声器实际输出。嗅探、DRM 和 DASH 明确不支持。

## 现在已有的东西

Core 侧(已实现并有测试):

- `video.Provider`: `Search` / `Details` / `Play`
- `Play` 给出 `URL`、`Format`、`Header`(播放时要带的请求头)、`Variants`、`HLS`(分片选项)、`Sniff`(要先去页面里找流)、`Encrypted`(客户端无法直接播时**明确标记**,而不是黑屏)
- HLS 解析在 `video/hls`

桌面端侧:

- 阅读器已经有一套 **WebView + 自定义 scheme 代理**(`internal/reader`),图片就是经它流式转发的
- `browser.Verifier` 已经能开 WebView 做 403 人机验证
- `browser.Sniffer.Sniff` 目前**明确返回 `ErrUnsupported`**,没有任何调用方

## 选项 A:WebView + `<video>`

**要做的**

1. 一个播放页(与阅读器同构):`internal/player` + `internal/player/assets/player.html`
2. 自定义 scheme 代理媒体请求,把 `Play.Header` 带上并透传 `Range` 头
3. HLS 需要 `hls.js`:WebView2(Windows)与 WebKitGTK(Linux)都**不原生支持 HLS**;`hls.js` 约 500 KB,可以随二进制内嵌
4. `Sniff` 若被用到,需要在 WebView 里拦截首个媒体请求(这与现有验证窗口同一套机制)

**代价**:纯 Go,不引入 cgo;跨平台一致;Windows 与 Linux 走同一份代码 —— 但 macOS 上 WKWebView 原生支持 HLS,行为会与另外两个平台略有差异,需要分别验证。

**风险**:站点若要求特定 DRM/加密,WebView 也放不了;`Play.Encrypted` 为真时应直接报错。

## 选项 B:原生播放器

**要做的**

先确认一件事:MyGo 的 `ui.Video` 播放的是**解码后的帧**(`VideoDecoder` → RGBA 帧),并不是媒体元素;而 MyGo 目前只提供 **GIF** 解码器(`ui.NewGIFDecoder`)。也就是说,“原生播放”在本仓库里等于:

1. 自己拿到 MP4/HLS 的解复用与解码(ffmpeg 系绑定 = **cgo**)
2. 把解码帧按时间轴交给 `ui.Video`
3. 各平台再各自处理硬件解码与音频输出

**代价**:引入 cgo 会打破现在“一条命令交叉编译 darwin/linux/windows”的性质;工作量大一个量级;音频还要另一条通路。

**适合的场景**:需要精确控制解码、或要支持 WebView 放不了的编码。**不适合**当下的目标。

## 选项 C:只嗅探直链,交给系统播放器

**要做的**

1. 真正实现 `browser.Sniffer`:开 WebView 加载页面,拦截媒体请求得到直链与所需请求头
2. 用 `os/exec` 调系统默认播放器(Windows 无统一命令行播放器;Linux 有 `xdg-open`;macOS 有 `open`)

**代价**:实现最小,但**请求头带不过去** —— 很多源要求 `Referer`/`User-Agent`,系统播放器不会替我们加。而且播放体验脱离 App(无进度回传、无书架联动)。

## 放在一起看

| | 工作量 | 引入 cgo | 请求头 | 进度联动 | 跨平台一致 |
|---|---|---|---|---|---|
| A. WebView + `<video>` | 中(复用阅读器那套) | 否 | 可以 | 可以 | 基本一致 |
| B. 原生播放器 | 大(解码器 + 音频 + cgo) | 是 | 可以 | 可以 | 否 |
| C. 嗅探 + 系统播放器 | 小 | 否 | **不行** | 不行 | 否 |

**一个可核对的事实**:本仓库现在 `GOOS=darwin GOARCH=arm64` 与 `GOOS=linux GOARCH=amd64` 都能干净交叉编译并 `go vet` 通过;选项 B 的 cgo 会打破这一点,选项 A 与 C 不会。
