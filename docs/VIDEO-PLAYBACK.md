# 视频播放:三个选项各自要动什么

[English](VIDEO-PLAYBACK.en.md) | 简体中文

更新:2026-10-31。这是一份**决策材料**,不是已定方案。视频播放属于产品取向(见 Core 的 `OPEN-QUESTIONS.md` 第 3 条);这里只把每个选项的实际工作量与代价摆出来,原因是这三条路的差距非常大,凭印象选会返工。

## 当前实现进度(2026-11-02)

已实现客户端的媒体传输层与 `internal/player.MediaProxy`:使用 Core provider 的搜索/详情/播放声明,共享源 Cookie 与重定向策略,流式读取媒体,通过 Core 的 HLS 改写把子清单、密钥、初始化片段和分片转为代理资源。loopback 用例已验证重定向后的相对路径、Referer、Range/206、首块提前到达与关闭会话取消 HTTP。

这还不是可使用的播放器:视频界面入口、WebView 播放页、Windows/Linux 的 HLS 解码依赖及观看进度仍未接。当前代码没有悄悄改用系统播放器,也没有实现嗅探或 DRM。方案 A 的传输层先落地,其余边界与验收仍按下文逐项推进。

**2026-11-02 更新(在上一段之后):** 方案 A 已接通到界面:主窗口「视频」→ 搜索/详情/线路分集 → 播放窗口。播放页由本机 `player.Server` 提供,自带 HLS.js 1.7.4(原生支持 HLS 时优先原生),观看位置存 `shelf.json` 并可续播。真实窗口实测:渐进 MP4 解码出真实画面(`readyState=4`、`640x360`、帧回读为 PNG)、暂停后位置落盘。仍未验证的是**浏览器内真正解码 HLS**(本地无 fMP4/TS 夹具),以及字幕、画中画、macOS 实机。

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
