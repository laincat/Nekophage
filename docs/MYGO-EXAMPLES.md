# MyGo 示例与桌面重写候选

[English](MYGO-EXAMPLES.en.md) | 简体中文

2026-11-04按用户提供的[官方 Showcase](https://mygo.egoist.dev/showcase)检索，并核对本机 MyGo 0.7.3 示例、GitHub README、go.mod 与许可证。建议先以 MyGo 原生组件为基础，评估 Onevra 的侧栏/主题和内置代码编辑器；阅读与播放继续使用隔离 WebView。当前产品依赖和界面均未改变。

## 官方示例：优先直接使用现有能力

本机固定版本有33个示例。网页有17个应用卡片，另一个“Add your app”标题是投稿入口。此前按标题统计为18个应用的说法已纠正。

| 示例 | 对 Nekophage 的价值 | 已核对边界 |
|---|---|---|
| [gallery](https://github.com/egoist/mygo/tree/v0.7.3/examples/gallery) | 侧栏、路由、网格、虚拟列表、搜索、弹层、主题与状态更新 | 已在 Windows 编译；不是成品媒体应用 |
| [embed-webview](https://github.com/egoist/mygo/tree/v0.7.3/examples/embed-webview) | 原生侧栏/工具栏承载多个 WebView，原生控件覆盖网页 | 已在 Windows 编译，最贴近当前阅读器/播放器结构 |
| [code-editor](https://github.com/egoist/mygo/tree/v0.7.3/examples/code-editor) | 原生代码编辑、补全、诊断、分栏/统一 diff | 已在 Windows 编译；ui/editor 内置 JS、MJS、CJS、JSX、TS、JSON 语言配置；大文件与系统输入法未验 |
| [notes](https://github.com/egoist/mygo/tree/v0.7.3/examples/notes) | 侧栏、富文本、工具栏、Markdown 快捷键与保存 | 已读源码；未运行 |
| [drag-drop](https://github.com/egoist/mygo/tree/v0.7.3/examples/drag-drop) | 跨窗口和系统文件/URL/文本拖放 | 可用于源、主题和插件文件导入；未运行 |
| [video](https://github.com/egoist/mygo/tree/v0.7.3/examples/video) | 原生动画背景与叠加控件 | 示例只解码 GIF，不能当作 HLS 播放能力证明 |

官方示例为 MIT，可按许可复用。优先调用 `ui`、`ui/editor` 和 MyGo 的现有接口，不另造控件或编辑器。CodeMirror 6 保留为编辑能力备选，不能仅因先前做过 Node 高亮原型就排除原生编辑器。

## GitHub 候选

| 项目 | 最值得参考或复用的部分 | 实际核对结果 |
|---|---|---|
| [Onevra](https://github.com/oneo-me/onevra) | 可折叠/调宽侧栏、页头动作、主题、Lucide 图标、聊天和命令面板 | MIT；公共模块为 `github.com/oneo-me/onevra/source`，v0.2026.17 最小应用已隔离编译通过 |
| [MixMusic](https://github.com/fzn0x/mixmusic) | 封面资料库、搜索、队列、主题与布局分离、社区主题/插件目录 | README 与 LICENSE-MIT 确认可选 MIT 或 GPL-3.0；go.mod 使用 MyGo 0.7.3 |
| [Video-Downloader](https://github.com/fazi-gondal/video-downloader-mygo) | 下载队列、任务状态、字幕/音轨/格式选择 | 仓库元数据为 MIT；未编译或运行 |
| [NuxtUI Go](https://github.com/josexy/nuxt-ui-go) | 表单、表格、树、主题、侧栏 | MIT；要求 MyGo 0.7.4；上游明确列出输入法和平台验收限制，当前未试编译 |
| [MyGo 云音乐](https://github.com/egoist/mygo-cloudmusic) | 成品原生资料库、歌词和迷你播放器交互 | AGPL-3.0，按本项目约定只观察交互，不复制源码 |

MixMusic 的 GitHub 自动许可证字段只识别 GPL 文件，不能据此推断禁止复用：实际存在 MIT 双授权。Nekophage 也没有“一切 GPL 代码都不能用”的规则。复用具体部分时保留其版权和第三方通知即可；AGPL 排除约定仍有效。MixMusic 的应用界面和插件宿主位于 internal 目录，不是可直接 import 的 UI SDK；它的插件宿主使用 Sobek，不能据此替换本项目已固定的 MoeJS。

## Onevra 的实际接入条件

Onevra 当前采用 MyGo 自带 CLI 打包，README 明确说它不再另带打包器；仓库简介中的旧描述不是当前实现依据。模块位于 `source/`，示例命令是先 `cd source` 再 `go run ./examples/gallery` 或 `go run ./examples/desktop`。

已在独立模块用公开的 `components`、`shells/plain`、`theme` API 编译小程序，解析到 Onevra v0.2026.17、MyGo v0.7.4、Go 1.27.3。首轮编译超出50秒限时；后续增量编译通过，两份日志都保留。没有运行窗口、测试主题/IME，或验证 macOS；Nekophage 仍固定 MyGo 0.7.3。

NuxtUI Go 的[兼容性说明](https://github.com/josexy/nuxt-ui-go/blob/main/docs/compatibility.md)记录了取消组合时可能丢失选区文字，以及 macOS 仅交叉构建、未桌面运行的范围。Lead随后在Windows使用MyGo公开的 `ui.Tester` 和 `ui/editor` 复现了模型层行为：在0.7.3和0.7.4中，选中JS字符串里的 `原文😀`，Compose("拼", 1)后Compose("", 0)，结果均变成空字符串。升级0.7.4并未消除该行为。

两个版本的只读模式均保持原字节，选择复制仍返回 `原文😀`，组合/键入/剪切/删除不能修改内容；正常提交中文后一次撤销恢复原文的对照也通过。接入候选先限定为只读源码查看，不开放原地编辑保存。上述为公开测试接口的输入模型结果，不是Windows系统IME或macOS原生窗口验收。

证据 `E:/Temp/nk-mygo-ime/`：`preedit.log`、`mygo-074.log`保留选区恢复失败；`readonly-copy-073.log`、`readonly-copy-074.log`为只读复制/防修改与提交撤销通过。原始最小复现为 `editor_input_test.go`，在独立模块执行，未修改MyGo依赖缓存或产品版本。

## 原生Liquid Glass补充核查（2026-11-04）

固定的MyGo0.7.3已带 `github.com/egoist/mygo/plugins/glass`，支持Glass的Regular/Clear、Tint、Interactive、Clarity，以及ScrollEdge/渐进模糊。它实现ui.Material，可直接在现有控件上调用；无需先升级MyGo或移植CSS/SVG光学代码。窗口VibrancyGlass在macOS使用系统玻璃，Windows映射Acrylic，与插件内的同窗口背景采样分开。Windows的23项上游测试通过，包含CPU渲染与Direct3D11离屏对照；亮/暗和实色对照样张已由公开ui.Tester渲染。日志 `E:/Temp/nk-liquid-glass/mygo-glass-tests.log`。尚未接入产品或验真实视频合成/性能，配方的自定义RGB色散不是此公开API的现成参数。

## 后续执行顺序

1. 复用 Gallery/Embed WebView 的导航和窗口组织，保留 Go 服务、存储及媒体代理。
2. 以 Onevra 的公共侧栏/主题接口做小规模集成评估，不复制整套音乐应用。
3. 比较 MyGo 原生编辑器与 CodeMirror 6 的真实源文件、中文输入和取消组合行为，再决定编辑器。
4. 用 MixMusic 的资料库、布局与主题分离模式整理产品信息结构；其插件 SDK 不能直接当作统一源协议。
5. 底层工作完成后再整体重写 UI/UX；本轮不追加旧界面视觉矩阵。

## 本轮证据

- 官方0.7.3的 Gallery、Code Editor、Embed WebView：Windows `go build` 通过，日志 `E:/Temp/nk-mygo-examples/official-build.log`。
- Onevra 最小程序：`E:/Temp/nk-mygo-examples/onevra-probe/`，最终构建日志 `onevra-build-retry.log`；首次限时日志 `onevra-build.log`。
- 抓取的官方页面、GitHub 元数据、README、模块与许可证原文位于 `E:/Temp/nk-mygo-examples/`；GitHub API回执在 `E:/Code/build/receipts/mygo-*.json`。

以上为源码核对和编译结果，不是产品集成、界面交互或性能验收。
