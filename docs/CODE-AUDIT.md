# 三项目代码体检

[English](CODE-AUDIT.en.md) | 简体中文

[打开交互式架构分析](analysis/architecture-audit.html)。单文件 HTML,不依赖 CDN、外部字体或网络,支持 320px 手机、桌面、深浅色和打印;可筛选项目并切换架构、依赖、修复、性能与证据页。

## 范围与基线

基线:Core `469edfc`、Nekularity `b294f04`、Nekophage `2e5b32f`。图中行数是本轮开始时 git tracked 文件的静态统计,排除 Vendor/build;测试行数占比不是覆盖率。Lead 负责 Core/桌面/修复/报告,Orca Worker 负责 iOS 审查与 Mac 验证。不存在的第二位 worker 未执行任何审查。

## 桌面本轮修复

| 级别 | 问题 | 处理 |
|---|---|---|
| P1 | `Cookies=false` 但重定向仍用分区 jar | 只有启用 Cookie 才传分区到 redirect callback;负向用例已复现后转绿 |
| P1 | `/private` 的 Cookie 被旧头带到 `/public` | 每跳先清旧 Cookie 再为目标 URL 选择;不存在匹配项就保持空头 |
| P1 | 自定义 `X-Api-Key` 等在跨 origin 跳转泄露 | 相对初始 origin 检查每跳,清 Authorization、X-Api-Key、Proxy-Authorization 与 Cookie;连续两跳回归 |
| P1 | `data:null` 写入 panic;损坏文件静默覆盖 | null 初始化为空 map;坏根/字段/读取错误不作为空数据;保留原件 |
| P2 | 写入/删除失败后缓存先行改变 | 副本写盘成功后提交缓存;真实阻塞 tmp 路径模拟写盘失败 |
| P2 | Cookie 文件正文为 null 时后续写入 panic | Open 后保证 jar map 非 nil |
| P2 | NKDEBUG 输出 jar 的真实 Cookie 值 | 移除该输出;不把秘密用于诊断 |
| P2 | gate 后续步骤覆盖失败日志,gofmt 进程失败被忽略 | 每项独立日志并检查退出码;外部坏 gofmt/vet harness 反向测试 |
| P1 | pin 写着 `469edfc`,实际链接 `build/core` 还是 `7662e7b` | 已 fetch 对齐;新增门禁读取 Go 实际模块目录并检查 HEAD、干净树与生成戳记,错误 pin 的反向用例通过 |
| P3 | 图片压缩原件和解码位图永久双份保留 | 解码成功后释放压缩原件 |
| P2 | 图片缓存无容量上限,浏览越久占用越多 | 抽出 `imageCache`:下载 400 条、位图 160 条上限,最旧优先淘汰;失败与损坏项也计入,不再每帧重试;5 条用例覆盖去重、解码复用、损坏记忆与两个上限 |
| P2 | 过期搜索结果覆盖新页面 | `generation` 票据:新搜索/详情/浏览递增,回包票据不是最新就丢弃;翻页沿用当前票据而不递增,避免把仍在前一页的请求判成过期;3 条用例含并发,错误实现的反向验证通过 |
| P3 | 请求 body 从 bytes 转 string 再变 reader | 改 bytes.NewReader,避免该次完整转换 |
| P2 | 源的设置回调没有返回值时,面板把 JSON 字面量 `null` 当成源说的话显示 | `CallSettingCallback` 把 `null` 归一化为空串,面板已有的「已完成」随之生效;`LoginAccount` 同一类问题一并修;单测钉住,去掉归一化会报 `void callback = "null"` |
| P2 | 源声明的每个回调共用一条结果文本,按其中一个按钮后所有回调行都显示它的答案 | `manage` 改为按设置项保存答案(`callbackMsgs map[string]string`),只在自己那一行显示;截图对比确认只按 `quiet` 时 `Run` 行不再跟着显示「已完成」 |
| P2 | Core 与桌面各有一份「把源的回答当文本」的实现,只改一边不会生效 | **结构性风险,本轮实证**:把修复只放进 Core 后桌面仍然显示 `null`,因为 `internal/core` 有自己的同类实现。现在两端规则一致,并把 `internal/core` 里其余 `json.Unmarshal(raw…)` 站点逐个看过:其余都解到结构体或布尔,没有再把协议字面量直接交给界面的路径 |

**「桌面重复实现 Core」的实际范围(2026-11-01)**:桌面 `internal/core` 有 57 个导出方法,Core 的 `mobile` 有 56 个,其中 **24 个同名**。逐个看过之后,分叉风险只落在**自己解析 Core 输出文本**的那几处 —— 其余要么直接委托同一个 JS 运行时(`WebviewStatus`、`WebviewLoginDone`、`ClickTag` 等),要么把 JSON 解到结构体(`parseComic`、`parseDetails`),这些不会各自演化。文本解析一共四处:`CallSettingCallback` 与 `LoginAccount`(本轮都改成「`null` 视为没有消息」)、`CategoryComics`/`ComicValue`(返回的是 JSON 契约本身,`null` 有意义,不能改),以及界面层的 `rawText`。最后这个本来就没问题——Go 把 JSON `null` 解到 `string` 时不会改动目标值,所以它返回空串;我一度加了显式判断,负向验证证明那段是多余的,已撤回,改为用一条测试把这个行为钉住。这条边界写在这里,是因为它决定了以后改 Core 的输出语义时,哪些地方必须一起改。

## 三项目框架结论

共享 Go Core + 原生客户端的边界应保留。核心业务与兼容逻辑一份,系统 UI/网络/登录/存储归各平台。桌面已使用 MyGo,不是 Wails;主窗口原生绘制,阅读器与验证才使用 WebView。iOS 是 SwiftUI + Swift 6 + gomobile binaryTarget,没有额外 Swift package 依赖。

Core 的 MoeJS 提供每源独立运行时,cascadia 负责 CSS selector,htmlquery/xpath 负责 XPath,modernc SQLite 负责官方备份数据库兼容。SQLite 是运行依赖,不能因为桌面业务状态主要写 JSON 就删除。桌面直接引入 MyGo 0.7.3,间接依赖 purego、字体排版与 x/image;版本逐项见 HTML。

## 待优化顺序

1. **取消而非只丢弃**:搜索已接入代次级 `context`(2026-11-02):新搜索、切源、离开搜索页都会取消旧搜索,包括多源排队请求、分页与粘贴链接;同代的单页结束不取消其他页。真实 loopback HTTP 验证旧连接断开且后续搜索成功。浏览与详情仍需接入同一机制。封面与收藏页图片已共用 4 个下载槽位,排队任务可取消(2026-11-02)。
2. **图片容量**:桌面已有下载 32 MiB、位图 128 MiB 的预算,同时保留 400/160 条目上限(2026-11-02)。位图费用包含 MyGo 实际保留的压缩字节与完整 RGBA 缩小图链;超出单项预算时保留失败标记、显示占位,不每帧重试。缓存预算不是 GPU 或进程总内存硬上限。iOS 已有 NSCache 成本上限、降采样和共享等待者计数,不再按“未实现”排队。
3. **真流式**:Core/mobile 与 iOS 当前等待完整正文。要以请求结束前首块到达及底层取消证据验收。
4. **职责收敛**:`internal/ui/app.go` 基线 1530 行,`internal/core/core.go` 834 行;后续按搜索、浏览、阅读器生命周期拆分。不要仅为降低行数搬文件。
5. **版本真实性**:桌面已增加真实模块 HEAD/干净树/戳记检查;iOS 后续仍需补 framework 产物级版本证据。生成版本字符串不能替代产物证据。

## 已知边界:源可以访问 loopback 与内网(2026-11-01)

查了一遍:客户端与 Core 都**没有**对目标地址做限制——源脚本写 `http://127.0.0.1:···` 或 `http://169.254.169.254/` 都能发出去。这属于 SSRF 面:一个恶意源可以扫本机端口或云元数据服务,把响应当作漫画数据回传给界面。

**没有加拦截,是有意的**:自建漫画服务(Komga、LANraragi 之类)通常就跑在局域网地址上,一刀切会让真实用法失效;而 iOS 的界面测试现在也依赖 `http://127.0.0.1:18731` 取图。信任边界定在「用户自己装的源脚本」,与脚本本身能执行任意 JS 这件事一致。

要收紧的话,正确的做法不是黑名单,而是**按源授权**:安装或首次访问时把非公网目标列出来让用户确认。这需要产品决定,所以只记录现状,不改行为。

## 验证与局限

本轮未声称吞吐量或内存提升百分比,未调用真实 AI 账号,未执行依赖漏洞数据库扫描。Go race/vet/tests 通过不表示没有其他缺陷。已对 Cookie/存储/AI/取消先做坏输入复现,再运行回归。全量日志位于 `E:/Temp/tri-audit`。发现并纠正实际依赖落后后,桌面门禁、race 与 Windows build 必须重新通过;先前旧 Core 上的绿结果不作为新 pin 证据。

HTML 脚本渲染、5 标签、4 筛选与离线资源检查通过。独立临时 Chrome 配置的无头 CDP 验证在 1440/390/320px 检查全部页面无横向溢出,不使用用户登录资料;截图保存在同日志目录。宿主浏览器的登录拷贝因 `APP_BOUND_ENCRYPTION_UNSUPPORTED` 启动失败,未改全局浏览器模式。

Mac 修复验证必须使用修复文件 md5 与整树 shape,不能把旧 173 条路由原型视为当前主线。最终报告将单列 Swift 6 测试与 App 构建状态。
