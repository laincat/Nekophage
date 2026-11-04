# Nekophage 架构

[English](ARCHITECTURE.en.md) | 简体中文

日期:2026-10-30。本文记录框架选型与分层。Wails 已弃用:旧版本用它把网页包成桌面窗口,现在整个桌面端是一个 Go 程序。

本轮清理与待优化项见 [代码体检](CODE-AUDIT.md),可交互的三项目架构/依赖图见 [HTML 报告](analysis/architecture-audit.html)。

## 选型

| 选择 | 原因 |
|---|---|
| Go + [MyGo](https://github.com/egoist/mygo) | 纯 Go、无 cgo,交叉编译到三平台;原生 UI 由 MyGo 自绘,不必用 WebView 承载整个界面;需要网页时可以把 WebView 嵌进原生界面 |
| [VeneraCat Core](https://github.com/laincat/VeneraCat-Core) 编译进同一进程 | Core 是 Go 库,`js.Runtime` 和 `plugin.Registry` 直接用;不需要 gomobile 或 FFI,也不需要跨进程协议 |
| 当前使用MyGo,后续UI/UX重新选型 | 这是现有实现,不以Magpie为全原生界面作依据。2026-11-04核对上游确认Magpie使用Wails v3.0.0-beta.24与HTML/CSS/JS主内容;此前描述不准确。新壳方案见下方生态调研,尚未迁移 |

参考 [Magpie](https://github.com/yetone/magpie)(MIT):借鉴它的分层与模块思路(单一 Library 作为数据源、变更串行化、显式来源归属、内置模块开关),不复制代码。

2026-11-04生态调研与选型见[统一源项目方案](../../VeneraCat%20Sources/docs/ECOSYSTEM.md)。按用户要求先处理底层、架构与基建,后续整体重写UI/UX;直接复用Wails/成熟组件库是候选,不是本轮已经换壳。

## 分层

```
main.go                装配:数据目录、Core、Library、阅读器、验证器、主窗口
internal/core          Core 的装配与漫画数据
  transport.go         host.Transport:net/http,按分区取用 Cookie,重定向逐跳处理
  core.go              js.Runtime + plugin.Registry;搜索、详情、章节、图片请求、解析
internal/cookies       每个源一个 Cookie 罐(RFC 6265 匹配,持久 Cookie 写文件)
internal/store         源数据、设置、插件记录;值是 Core 给的原文,不经重新编码
internal/shelf         阅读历史与收藏(SQLite 之外的 shelf.json,崩溃时只会留下旧文件或新文件);收藏夹的新建/重命名/删除与换夹都在这里做,界面只读结果
internal/library       安装/替换/删除源,读取 index.json 并计算每项状态
internal/reader        nkreader:// 方案:阅读器网页 + /chapter + /img + /progress(阅读位置回写)
  assets/reader.html   自包含单文件网页(样式与脚本内联)
internal/browser       人机验证窗口(WKWebView / WebView2 / WebKitGTK)
internal/ui            主窗口:源、搜索(含来源筛选、多源合并搜索)、浏览、书架、详情,以及阅读器窗口的章节按钮
```

## 数据流

```
用户点“安装” → library.InstallURL → core.Transport.Fetch(不带 Cookie)
             → Core 编译并校验脚本 → plugin.Record → store/plugins.json

搜索 → Core.Call("search.load") → 脚本 Network.get → Transport.Do
     → 分区 Cookie → 响应 → 脚本解析 HTML → JSON → core 解析成 Comic

阅读 → reader 网页 fetch app://chapter → Core.ChapterImages
     → <img src="nkreader://img?u=..."> → Go 按 onImageLoad 取图 → 流式返回
```

图片不经过 base64 或内存里的数据 URI:浏览器按需请求,Go 逐张取回,滚动到哪取到哪。

## 隔离

- 每个源的 Cookie、登录状态、源数据都在自己的分区里;`comic:<key>` 是分区名。
- 不使用系统共享 Cookie 存储:HTTP 请求的 Cookie 头由应用自己拼。
- 验证窗口共用浏览器的默认数据存储,所以同一时间只允许一个验证窗口,打开前和关闭后都清空,页面之间不互相污染。
- 源不可信:脚本在独立引擎里运行(Core 保证),网页只在阅读器和验证窗口里出现,不加载源的任意 HTML 到主界面。

## 挑战完成规则(2026-11-01 已实现)`

`host.DoneRule` 有三组条件:某些标记从 HTML 里消失、某些 Cookie 必须存在、**URL 必须带上指名的查询参数**。**有内容的组全部满足**才算通过;一组都没有时永远不通过——空组表示「没有这个条件」,不是「自动成立」。

三组并存是因为两类挑战的凭据形状不同。Cloudflare 拿到的是 `cf_clearance` Cookie。MangaDex 实测(2026-11-01):`GET api.mangadex.org/manga` 返回 `302` 到 `/edge/challenge?src=<原路径>`,挑战页做纯 JS 的 proof-of-work,解出后 `location.replace` 回原地址并带上 `powblock=<nonce>&pbchal=<token>`——**凭据在 URL 参数里,不在 Cookie 里**,所以只有 Cookie 的规则对它只能等超时。

Core 的 `challenge.DetectEdge` 认出这一类(302 的 body 或挑战页自身都带标记),`challenge.EdgeDone()` 给出完成规则;客户端只是读当前 URL 的 query 与上面三组比对,判断逻辑在桌面的 `internal/browser` 与 iOS 的 `ChallengeRule` 各一份,语义由 Core 的 `DoneRule` 注释与单测固定。

## 制作台(后续模块)

制作台是第二阶段:编辑主题、漫画源与功能插件,打包成可发布目录,生成 `index.json` 与安装包。设计参照 Magpie 的内置模块:

- 制作台是一个可开关的内置模块,关闭时不占启动时间和内存。
- 数据仍走同一个 Library:安装记录、来源归属、变更串行化只有一套。
- 预览与校验在桌面完成,效果以能在 Nekularity 复现为准。
