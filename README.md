# Nekophage

[English](README.en.md) | 简体中文

Nekularity 的桌面端:用 Venera 兼容的漫画源阅读,并把视频播放与扩展的制作分发作为后续模块。Go 编写,界面由 [MyGo](https://github.com/egoist/mygo) 绘制,许可证 GPL-3.0-or-later。

业务逻辑来自 [VeneraCat Core](https://github.com/laincat/VeneraCat-Core),直接编译进同一个可执行文件:不需要 FFI,不需要 cgo,也不需要单独的 Core 进程。

## 形态

- **主界面**:原生 UI,Go 写,MyGo 用 GPU 绘制,没有 WebView,启动快、内存小;只有阅读器与人机验证窗口用 WebView。
- **阅读器**:章末与顶栏都能直接翻到下一话,也可以开「自动接续」一路读下去,单页或双页、左起或右起(漫画常用右起),横图自动占满一行;WebView 里的一个网页,图片由 Go 按源的规则取回再流式送出,所以需要 Referer 或登录的源也能正常显示。
- **人机验证**:Cloudflare 挑战在单独的窗口里打开,通过后自动继续。
- **多源搜索**:一次搜索所有已装的源,结果合并展示并标注来源,点开别家的漫画会自动切过去。
- **源设置**:源的 `settings` 声明会渲染成下拉、开关或输入框,例如 MangaDex 的画质、拷贝漫画的 CDN 线路。
- **搜索筛选**:按源提供的搜索选项筛选(排序、分级、状态等),不用再默默用默认值。
- **搜索历史**:记住最近搜过的关键词,空结果页一键再搜;官方备份里的搜索记录也能导进来。
- **图片收藏**:阅读器里一键收藏当前页,书架里看成排的缩略图,能跳回那一话。
- **书架**:阅读历史与收藏。读到哪儿会连同页码与百分比一起记下。打开章节就记下位置,详情页选收藏夹一键收藏,「继续」直接回到上次那一话;收藏夹可新建、重命名、删除,条目可换夹。
- **备份导入**:读官方 `.venera` 备份,先预览再选择;导入漫画源脚本、源数据、阅读历史与收藏(经正常安装流程校验,不执行备份里的脚本),值按原样保留。
- **制作台**:后续模块,参照 Magpie 的内置模块开关设计,见 [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)。

## 状态

第一阶段是阅读主线:添加漫画源 → 搜索 → 详情 → 阅读。已在 Windows 上编译、跑通单元与端到端测试并开出窗口;真实漫画源、macOS、Linux 与验证窗口尚未验收,详见 [docs/STATUS.md](docs/STATUS.md)。

## 编译

需要 Go 1.27 或更高版本。

```sh
scripts/fetch-core.sh      # 把 Core 拉到 build/core(版本见 scripts/core-version)
go build .                 # 生成 Nekophage 可执行文件
./Nekophage -version        # 版本、源 API 级别、提交号与 Go 运行时,不开窗
go test ./...              # 单元与端到端测试
```

`go.mod` 用 `replace` 指向 `build/core`;Core 有了发布标签后改成普通依赖即可。Linux 上需要 GTK 3(`libgtk-3-0`),WebView 页面还需要 WebKitGTK 4.1(`libwebkit2gtk-4.1-0`);Windows 需要 WebView2 运行时(Windows 11 自带)。

## 目录

| 目录 | 内容 |
|---|---|
| `internal/core` | 把 Core 装进应用:运行时、插件注册表、HTTP 与图片、漫画数据解析 |
| `internal/cookies` | 按源隔离的 Cookie 罐 |
| `internal/store` | 源数据、设置与插件记录的持久化 |
| `internal/library` | 安装、更新、删除源,读取订阅索引 |
| `internal/reader` | 阅读器网页与取图服务 |
| `internal/browser` | 人机验证窗口 |
| `internal/archive` | 读官方 `.venera` 备份,整理成本地可应用的导入计划 |
| `internal/ui` | 主窗口:源、搜索、详情、备份导入、阅读器窗口 |

## 许可证

GPL-3.0-or-later,见 [LICENSE](LICENSE)。
