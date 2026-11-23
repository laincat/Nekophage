# Nekophage 当前状态

[English](STATUS.en.md) | 简体中文

## 架构

Nekophage 用 MyGo 绘制桌面主窗口，在单独的 WebView 中显示阅读器、网页登录和人机验证。应用把源运行、插件注册、索引解析和备份格式交给 VeneraCat Core。Go 宿主由 Core 的可选包 gohost 提供，桌面端为它配置资料目录、平台、源 API 级别、WebView 和日志。

| 组件 | 职责 |
|---|---|
| internal/core 与 internal/sourceapi | 装配共享 Go 宿主。源 API 级别为 `1.6.3`，由应用与命令行共用。 |
| Core 的 gohost | HTTP、源分区 Cookie、JSON 数据与设置、漫画调用、源安装和记录持久化。 |
| internal/browser | 桌面 WebView 的串行会话、Cookie 读取与验证交接。 |
| internal/collections | 同一资料会话共用一个合集管理器。缓存、订阅刷新和兼容性判断保留于桌面端。 |
| internal/archive、internal/shelf 与 internal/searches | 备份的无界面应用与导出、书架和搜索历史。 |
| internal/player 与 internal/reader | 通过宿主取图片和内容，提供阅读器桥、媒体代理、逐会话取消、Range 和本地播放状态。 |
| internal/services 与 internal/magpie | 桌面服务凭据和 Magpie 本机接入。服务普通值由 Core 的包外存储管理。 |
| internal/portable 与 scripts | 便携资料位置、旧资料迁移、Core 检出、检查和发行打包。 |

## 已接入的功能

- 漫画源支持 URL、文本与文件安装和漫画索引更新。搜索、分类、浏览、详情、章节、图片回调、标签、评论与收藏按源声明开放。
- 阅读器与原生图片入口共用宿主取图，兼容协议相对地址和签名 URL，限制可访问的协议。阅读进度沿用资料会话的书架文件，重复进度跳过写入。
- 登录与验证先从当前 WebView 取得 Cookie，再提交到源分区。HTTP 按每一跳重新匹配 Cookie。持久 Cookie 发生变化才写盘，会话 Cookie 留在内存。
- 合集支持漫画源和服务包，安装前验证摘要及声明身份。退订只移除订阅，保留已安装条目。见 [统一源合集](SOURCE-COLLECTIONS.md)。
- 备份导入预览后可恢复源、源数据、历史、书架、图片收藏、搜索历史和明确选择的 Cookie。导出携带漫画源、源数据、设置与搜索历史。Cookie 恢复只写入对应已安装源的分区。导出不含 Cookie。
- 本地媒体代理为视频提供逐请求头与分区 Cookie、HLS 清单处理和关闭取消。字幕与播放的范围见 [视频播放](VIDEO-PLAYBACK.md)。
- Windows 便携版在程序目录存放资料，打包和更新保留可独立运行的发行入口。见 [项目说明](../README.md)。

## 验证入口

| 命令 | 覆盖场景 |
|---|---|
| `sh scripts/check.sh` | 代码格式、实际链接的 Core 检出、vet、race 与双语文档。 |
| `go test github.com/laincat/veneracat-core/gohost/...` | Cookie 分区与写盘、传输与转义、源安装识别及失败恢复、漫画契约。 |
| `sh scripts/desktop-probes.sh` | WebView 登录、验证与关闭、真实解码、读写书架、播放器与字幕。 |
| `sh scripts/reader-probe.sh` 与 `--layout` | 默认使用回环章节。通过解码页数和双页位置验收。 |
| `sh scripts/login-probe.sh` | 正常、超时、回调失败、迟到 Cookie 和关窗。 |
| `go run ./cmd/verifycheck browser -mode clear` 或 `-mode dismiss` | 验证窗口的 Cookie、User-Agent 和关闭行为。 |
| `go run ./cmd/videocheck subtitles` | 活动字幕及关闭选择的保存和重开。 |
| `go run ./cmd/importcheck -file ARCHIVE -dir PROFILE -apply` | 在指定资料目录运行与 UI 相同的备份应用；恢复 Cookie 另加 `-cookies`。 |
| `sh scripts/live-contracts.sh` | 使用 `NK_OFFICIAL_SRC` 提供外部源，覆盖版本门槛、Cookie 交接、登出、回调与标签。 |
| `sh scripts/live-sources.sh` 或 `sh scripts/live-sources.sh browse` | 使用 `NK_LIVE_DIR` 跑实站阅读或浏览。用 `NK_LIVE_REPEAT` 增加采样。 |

发行入口为 `scripts/package-windows.py`、`scripts/installer-windows.nsi` 和 `scripts/package-macos.sh`。用 `scripts/fetch-core.sh` 同步 Core，或传本地检出路径。生成的版本戳可通过 `Nekophage -version` 回读。

## 待办和历史基线

- UI 重写要继续验证关闭和切源取消、后台 I/O、WebView 隔离、真实解码、源分区名与媒体关闭。
- 平台与媒体的待验证项见 [视频播放](VIDEO-PLAYBACK.md)。安装器、签名和各平台真实窗口的结论以对应发行验证为准。
- 历史能力盘点见 [代码审核](CODE-AUDIT.md) 与 [架构审核](analysis/architecture-audit.html)。旧状态流水账可从 `git show 341cb9c:docs/STATUS.md` 找到。
