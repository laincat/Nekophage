# Nekophage 状态

[English](STATUS.en.md) | 简体中文

更新:2026-10-31。下列结论来自 **Windows** 本机实测,包括真实漫画源与真实网络。当天新增的内容集中在三处:传输层重定向 Cookie 的真 bug(`CheckRoute` 读错了字段)、错误提示的中文化(与 iOS 同一规则:只翻认得的,其余保留 Core 原文)、以及 `live-sources.sh` 的重复跑与 `FLAKY` 分类。

## 已实现

| 部分 | 内容 |
|---|---|
| 装配 | Core 编译进同一进程;`internal/core` 提供运行时、注册表与漫画数据接口。2026-10-31 用**最新的 Core**(含 `magpieplugin`、`host/sse`、`mobile` 的 Magpie 接口)重新装配并全量跑过:`gofmt` 干净、`go vet ./...`、`go test ./...` 全过 —— 桌面端不接 AI,但 Core 新增的包不能把桌面端带坏。`build/core` 与 Core 工作树**逐文件一致**(`diff -rq` 无差异,不含 `.git`/`.github`/`apple`/`build`/测试与 `testdata`;`build/bin` 是 gomobile 工具链,9 MB,搬过来只是浪费)。当前工作树 = Core `519623d`,所以「用的是哪一版 Core」由 `diff` 与提交号回答,不由记忆回答。装配后跑 `go test -count=1 ./...` 与 `go test -race -count=1 ./...`,10 个有测试的包全过、无数据竞争(2026-11-01 复跑为 **11 个有测试的包**,`-race` 全过；`internal/sourceapi` 无测试文件)(此前 race 只覆盖 `./internal/...`,与 `go test` 覆盖的包不是同一批,现已对齐)。另有 `-version` 现在同时给出自身提交与 **Core 提交**:后者由 `scripts/fetch-core.sh` 生成的 `corestamp.go` 写入,没有戳记时如实打印 `Core unknown`,而不是拿自身提交冒充 |
| 网络 | `internal/core/transport.go`:net/http,按分区取用 Cookie,重定向逐跳保存与发送 Cookie,单响应 64 MiB 上限;请求行不能承载的字节(空格、控制字符)自动转义 |
| Cookie | `internal/cookies`:host-only 与域 Cookie、路径、Secure、过期;持久 Cookie 写入 `cookies.json` |
| 存储 | `internal/store`:源数据、设置、插件记录;值是 Core 给的原文,大整数与键顺序不变 |
| 源设置 | 读 Core 给出的 `settings` 声明(select/switch/input/callback),在「漫画源」窗口按类型渲染;值以 JSON 原样写进该源的 settings 段,数字与布尔不会被转成字符串;input 的 `validator` 按正则校验;callback 是源内函数,跨不过边界,只显示说明 |
| 源管理 | `internal/library`:从脚本链接、脚本文本、`index.json` 安装或更新;删除会清掉该源的数据与 Cookie;搜索选项按源的默认值传给 `search.load` |
| 订阅 | 用 Core 的版本规则比较,状态为未安装/可更新/已安装/来自其他订阅/需要更新 App |
| 界面语言 | 全部中文;MyGo 自带的控件文案(搜索框占位、清空等)通过 `mygo/locales/zh-Hans` 与 `App.SetLanguages("zh-Hans")` 一起切中文 |
| 多源搜索 | 顶栏「当前源/全部源」切换;全部源时并发(同时最多 3 个)搜索每个已启用源的第一页,结果按源的安装顺序合并,每张卡片标注来源;点开别家的漫画会自动切换当前源 |
| 真源端到端 | `TestLiveSource` 在真实源上跑完整条线:安装→搜索→详情→章节→取图→`/chapter`→`/img`→书架记录→`/progress` 回写→`/keep` 收藏本页→`/switch` 下一话→重开 profile 读回;需要 `NK_LIVE_JS` 与 `NK_LIVE_QUERY` |
| 搜索筛选 | 读源的 `search.optionList`,按组显示可横滑的选项;单选组直接换值,多选组按官方规则把 JSON 数组写成一个字符串传给 `search.load`;换源时回到该源默认值 |
| 主窗口 | MyGo 原生 UI:源选择、搜索、结果网格(封面异步下载并解码,已在书架的卡片标「已在书架」)、分类浏览(源的 `explore`,四种类型)、详情(封面、标签分组、Markdown 简介、分组章节与章节筛选,章节列表用「读到」标出书架里记下的那一话,章头显示「上次读到 第 N 页 · P%」与「继续阅读」)、分页;记住上次的源、关键词与所选的收藏夹(`ui.json`) |
| 阅读器 | WebView 里的自包含网页:竖向滚动、懒加载、进度、宽度调节、键盘滚动、单页重试,并按章节记住读到的位置;单页/双页与「左→右」「右→左」方向(`D`/`R` 键切换),横图自动独占一行;顶栏与章末都有「上一话/下一话」,点击由窗口(`/switch`)返回该话地址后页面自己跳转;另有「自动接续」开关(默认关,`A` 键),读到章末自动进下一话,且必须真的滚动过才触发;滚动时把页码与百分比回写给书架(`/progress`),图片由 Go 按源的规则取回流式送出 |
| Magpie 本机网关 | `internal/magpie` 已实现连接层与「AI」窗口(地址/测试连接/模型目录/发送/取消,回包 80ms 节流重绘;档位可选简单/普通/复杂并各自映射到模型或组,窗口显示这次会发到哪):`/v1/models` 目录(保留模型/组 ID)、三种协议的原生正文、SSE 流式、`ctx` 取消、中文诊断;10 条假网关契约测试 `-race` 通过。**2026-11-01 已接到界面上,并对本机真实安装版跑通**:目录 53 个模型或路由组,`chat`/`responses`/`messages` 三种协议各走一遍真实流式,窗口里拿到真实回答。三项未验证仍在:真机以外的平台、取消时机、以及 Magpie 里没配置路由时的表现 |
| 人机验证 | 独立窗口加载挑战页,按 Core 的完成规则判断,通过后把 Cookie 与 User-Agent 交回并自动重试 |

## 已验证(真实源、真实网络)

**一条命令复跑全部真实源**:`NK_LIVE_DIR=/path/to/scripts scripts/live-sources.sh` 会对目录里每个 `.js` 跑一遍 `TestLiveSource`(`NK_LIVE_REPEAT=3` 适用例连跑三次,**时好时坏的源单列为 `FLAKY`**,不混进 PASS 或 FAIL),逐个打印 PASS/FAIL,日志写到 `build/live/<源>.log`。一个源失败是一个结果,不是崩溃 —— 站点变了和客户端坏了要能分开。2026-10-31 实测 5 个脚本(comick、copy_manga、komiic、komiic2、manga_dex)全部 PASS。

**2026-10-31 复跑四源**:Core 同步到 `519623d` 后,MangaDex、Komiic、Comick、拷贝漫画的 `TestLiveSource` 全部 PASS。但**同一天稍后又复跑一次,拷贝漫画回到 HTTP 210 而其余四源仍 PASS** —— 所以它的状态是「时好时坏」,不是「已恢复」;`live-sources.sh` 会把两类结果分开打印,正是为了这种情形。Comick 这次暴露了测试里的一个假设:它取到的章节只有 1 张图,而用例固定要求 `page=3`,被阅读器正确地钳到 1,于是断言失败 —— 错的是用例不是代码,现在按 `min(3, len(images))` 断言。拷贝漫画此前报的 HTTP 210 已不复现,那是站点侧的临时行为,不是客户端回归。

Core 的 `6e07429` 那一版 CI 四个 job 全绿(Go 三个平台 + Apple 的 xcframework 与两份 Swift 冒烟测试),2026-11-01 实测。

**换 pin 到 `6e07429`(2026-11-01)**:Core 新增了标准 AI 的三个入口与可选的分段传输(`HostStream`),并修掉包桥的一处死锁 —— 包在自己的 hook 里往 `__magpieChatPush` 写,而读取侧读到结束标记就停了,写方永远等不到读者。桌面端目前**没有实现 `HostStream`**,所以回复仍整包到达;`scripts/fetch-core.sh` 已把 `build/core` 对齐到该提交,`scripts/check-core-pin.sh` 通过。

**换 pin 后重跑阅读器**:Core 同步到 `c1db2c3` 之后,`readercheck` 重跑 Komiic 一章(53 页):`page title=TEMPEST figures=53 images=53 decoded=4` —— DOM 里 53 张图都在,当前视口解码了 4 张。截图本身不含 WebView 内容,所以结论取自 DOM 与解码状态,不取自截图。

**一条命令跑完全部本地检查**:`sh scripts/check.sh` 依次跑 `gofmt`、`go vet ./...`、`go test ./...`、`check-docs.sh`、`check-partitions.sh`,逐项 PASS/FAIL 打印,任何一项失败则整体退出码非 0。`check-partitions.sh` 是这次恢复 bug 的产物:分区名 `"comic:<key>"` 只允许出现在 `core.ComicPartition` 一处,别处再拼一次就报错 —— 因为那个 bug 当初是**测试跟着代码一起写错**,只有「禁止第二种写法」这种检查才拦得住。

**2026-10-31 再次复跑**(`NK_LIVE_REPEAT=3`,五个脚本各跑三遍):五个源全部 PASS,拷贝漫画 3/3。这加强但不证明「稳定」—— 一次 3/3 只能说明这半小时内没复现,前面的失败仍然真实发生过。

| 源 | 结果 |
|---|---|
| 拷贝漫画 `copy_manga` v1.4.3 | 搜索「海贼王」21 部、2 页;详情 401 章、4 组标签;章节 209 张图;取图 339,596 字节 `image/webp`;封面 108,565 字节 `image/jpeg` |
| MangaDex `manga_dex` v1.2.0 | 搜索「one piece」20 部、2 页;详情 962 章、3 组标签;章节 53 张图;取图 122,856 字节 `image/jpeg`;封面 56,745 字节 `image/jpeg` |
| Comick `comick` v1.x | 搜索「one piece」：章节 51 张图、取图 37,912 字节、封面 44,668 字节(均为 WebP);书架记下 `page 3, 42%` |
| Komiic `komiic` v1.x | 分类浏览(分页型)第 1 页与第 2 页内容不同,封面与作者正常,翻页生效;端到端:章节 67 张图、取图 248,508 字节、封面 27,252 字节;书架记下 `page 3, 42%` |

Comick 顺带暴露了真实源测试自己的一个缺陷:测试直接拼 `...&chapter=<章节 id>`,而它的章节 id 本身带斜杠,拼出来的请求行变成非法 HTTP 版本,`httptest.NewRequest` 直接 panic。现在四个 reader 请求(`/chapter`、`/progress`、`/keep`、`/switch`)都改用 `url.Values` 编码,`/switch` 也改为解析返回的 JSON 再比对 `chapter` 参数,而不是在原始响应体里找子串。

Komiic 顺带暴露了两处服务端标注问题:页面图实际是 WebP,响应头却写 `image/jpeg`;封面响应头写 `application/octet-stream`。真实源测试因此改为按文件头判断图片类型(JPEG/PNG/GIF/WebP/HEIF/AVIF/BMP),不再信任 `Content-Type`——原来的写法会把一个能正常显示的 WebP 判成「不是图片」。WebP 的封面由 MyGo 的 `ui.DecodeBitmap` 解码(它已经引入 `x/image/webp`),阅读器里的页面图交给 WebView,两者都不受影响。

阅读器的两个接口按网页真实调用的方式各跑了一遍:拷贝漫画 `/chapter` 返回 209 张、`/img` 返回 339,596 字节;MangaDex 53 张、122,856 字节。

Core 同步到 `af4f1a0` 之后又跑了两条真实源:MangaDex(`one piece` —— 20 部、962 章、第 1 话 53 张图、8 张封面 56,745 字节)与拷贝漫画(`海贼王` —— 209 张图、339,596 字节 WebP、封面 108,565 字节),两条 `TestLiveSource` 都 PASS,书架进度"第 3 页 / 42%"与图片收藏的页码也照旧写入。

人机验证链路:`go run ./cmd/verifycheck --local` 通过 —— 本地伪造站点返回 403,WebView 打开、解开挑战,2.8 秒后拿到 `cf_clearance=local-ok` 并写入该站点域的 Cookie;这条链路在 Core 同步到 `af4f1a0` 之后重跑过。

界面用截图核对(Comick `one piece`):原生主窗口显示 18+ 张真实封面、标题、源选择器中的 `comick` 与「第 1 页 / 下一页」,封面是 WebP 也正常解码(MyGo 的 `ui.DecodeBitmap` 已引入 `x/image/webp`)。此前还核对过:原生主窗口显示了真实封面、标题与作者;详情页显示了封面、标签分组、Markdown 渲染的简介(`**Note:**` 成为粗体、`---` 成为分隔线、链接可点)和 962 行章节,章节筛选框就位。源设置验证(MangaDex):面板显示 `Image Quality` 两个选项,改成 Compressed 后提示「已保存设置」,该源的数据文件变为 `{"data":{},"settings":{"image_quality":"Compressed"}}`;并且该设置真的影响了请求——单测 `TestLiveSourceSettingsTakeEffect` 显示默认取图地址是 `/data/…`,改成 Compressed 后变成 `/data-saver/…` 且文件哈希不同。这个测试现在与源无关:它从该源自己声明的选项里取一个不同于默认值的值,断言图片地址发生变化,所以在拷贝漫画上也成立(`…c1500x.webp` → 改成 800 后变 `…c800x.webp`),不再写死 MangaDex 的 `data-saver` 字样。

真源端到端(MangaDex,one piece,3 秒内跑完):搜索 20 部→详情 962 章→遇到 6 个无页章节后找到 53 页章节→取图 122,856 字节→`/chapter` 53 张→`/img` 同样字节→书架记下进度 `page 3, 42%`→收藏第 1 页→`/switch` 返回该话地址→重开 shelf 读回 1 条历史 1 张图片。

图片收藏导入验证:带 `image_favorites` 的备份解析出 2 页,写入 `shelf.json` 时带上章节名「第03话」、章节序号 3、页码与图片地址,时间从毫秒转成秒。

图片收藏验证(MangaDex 第 7 项章节):点「收藏本页」后 `shelf.json` 记下漫画、章节名、页码与图片地址;书架「图片收藏 · 1 张」显示真实缩略图与「回到这一话」。

多源搜索验证(同时装 MangaDex 与 Komiic,关键词 one piece):合并结果里 6 张 Komiic 卡片在前、MangaDex 卡片在后,来源标注与封面都正确;点开最后一本(MangaDex 的)后顶栏源自动切到 MangaDex,详情与章节正常。

搜索筛选验证(MangaDex):默认 Any 与 `Sort By=Rating`、`Status=Completed` 三种设置返回的首屏结果各不相同,选中的选项在界面上高亮。

自动接续验证(Komiic):开启后从 `卷01` 一路读到章末,书架自动变为 `卷03`,页面上按钮显示「自动接续:开」;未滚动过的短章节不会自行跳转。

跨章连读验证(Komiic,開卷 卷01 162 页):点「下一话」后书架从 `chapterIndex 0` 变为 `1 / 卷02`,新章立即上报进度;同时验证了旧页面的迟到进度不会覆盖新章(MangaDex 第 7 项章节无图,仍正确记为 7 而不回退到 6)。

阅读进度回写验证(MangaDex 第 7 项章节,53 页):滚动到约 55% 后,书架文件记录 `page:26, percent:55`,书架页显示「1: Romance dawn · 第 26 页 · 55% · 刚刚」。

双页排版验证(MangaDex 第 1 话 53 页,页面自报):`cls="spread rtl"`、两列各 433.5px、第 1 页在 x=436、第 2 页在 x=0、同一行——右起双页生效。

搜索历史验证:从备份导入后空结果页出现「海賊王」标签;再搜「one piece」后文件变为 `["one piece","海賊王"]`,标签行同步显示两条。

收藏夹编辑用 `-shelf-demo` 走真实界面代码验证:新建「以后看」→ 把收藏的漫画移进去 → 重命名为「以后再看」,`shelf.json` 变为 `默认收藏(空)`、`以后再看(1 部)`,截图同步显示「（空）」与换夹后的下拉框。

书架用真实源验证:打开「我的女友老師」第 03 话后,`shelf.json` 记下源、id、标题、作者、封面、章节与时间;书架页显示封面、「第03话 · 刚刚」。

**登录状态(Cookie)导入**:官方 App 只有一个 cookie jar,这里每个源一个分区。导入界面上有「同时恢复登录状态(Cookie)」复选框(默认关闭;已用 `uicheck -settings -backup-cookies -settings-out` 两次截图确认:未勾选与勾选两种状态都渲染正确(勾选那张是 `DevOptions.BackupCookies` 直接把状态置真后重建的,证明的是「状态 → 界面」这条链);**鼠标点它之后的完整导入流程还没有自动化验证**),勾选后 Core 的 `PlanBackup`/`Plan` 会给出 `cookieTargets` —— 优先取「备份里有、本机也装了」的源,没有交集则退化为全部已装分区 —— 客户端据此把 cookie 写进各自分区,不重建共享 jar。域名不是主机(例如写进 `https://` 的脏行)、以及**已经过期**的行会被丢弃 —— Core 没有时钟,过期判断在客户端做,否则摘要会声称恢复了一条根本登不上的会话;每条 cookie 只在该源访问同名域名时送出,所以送给不访问该域的源等于没送。

备份导入用三个 `.venera` 各自验证:`backup-test`(源 1、数据 2、搜索历史 2)、`backup-shelf`(历史 1、收藏夹 1 个含 1 部、搜索历史 1)、`backup-pages`(图片收藏 1 条,落盘 2 张:图片收藏与集数行各带一页)。最初用含真实 `copy_manga.js` 的 `.venera` 验证:预览显示源 1 个、带数据 2 个、搜索历史 2 条;导入后 `copy_manga` 以正常流程装成 1.4.3,4 项源数据写回,大整数 `12345678901234567890` 原样保留。

分类浏览在三个源上都通过:MangaDex 显示 Popular 分区与带封面的列表,拷贝漫画显示「推荐」「热门」两个分区,Komiic 的分页型 browse 第 1、2 页内容不同,翻页生效。阅读器窗口确实在 WebView 里运行:页面标题为漫画名,生成 53 个 figure,图片解码成功(第一张 757×1200),`currentSrc` 指向 `nkreader://`(Windows 上是 `http://nkreader.localhost`)的取图地址。重开同一章会回到上次的位置:滚到 1200 后重新加载,`scrollY` 仍是 1200。 Comick 的另一次检查记录了 51 个 figure、51 张图、其中 4 张解码成功,首图 750×1194,`currentSrc` 指向 `http://nkreader.localhost/img?source=comick&…`。

### 人机验证(2026-10-30)

- 窗口本身:本地伪装挑战页(标记消失 + 脚本写入 `cf_clearance`)在 3.5 秒内被判定通过,拿到 WebView 的 User-Agent,`cf_clearance` 写进分区罐。
- 整条链路:本地伪造 Cloudflare 站点对普通请求回 403(`cf-mitigated: challenge` + 挑战标记),Core 检测后开窗口,页面通过后把 Cookie 与 User-Agent 交回,重试带 `cf_clearance=clear` 拿到 `200:PASSED-OK`,且 Cookie 落在该源自己的 `comic:c` 分区。全程 3.4 秒。

## 本轮发现并修掉的问题

**重定向跳上的 Cookie 从来没被存过(2026-10-31)**:`internal/core/transport.go` 的 `CheckRedirect` 里读的是 `via[len(via)-1].Response`,而 Go 只在**新请求**的 `Response` 字段上挂那个重定向响应,`via` 里的前一个请求始终是 `nil` —— 于是「保存上一跳的 Set-Cookie、并把本分区 Cookie 发给下一跳」这段代码**静默地什么也没做**,注释还写着它做了。新加的 5 条用例里「重定向跳上的 Cookie 要被存下」直接失败,才把它暴露出来。修法是用 `req.Response` 与它的 `Request.URL`。这类问题真实站点上表现为「登录后跳转一下就要重新登录」,不容易第一时间归因。

**协议相对的图片地址取不到(2026-11-01)**:一个源把章节页的图片地址写成 `//images.example.com/1.jpg` —— 这是 HTML 里的标准写法,少了 scheme。RFC 3986 说这种地址继承页面的 scheme,而我们的客户端直接报「image URL is not http(s)」。修法分两层:规则放在 Core(`host.NormalizeURL`,gomobile 导出 `Core.NormalizeURL`),客户端只负责决定「继承哪个 scheme」—— 桌面端的 `Transport` 按**分区**记住最近一次请求用的 scheme,所以一个用 http 的源不会因为另一个用 https 的源而丢图。**这不放宽安全规则**:`file://`/`javascript:`/`ftp://` 一律不改写,补全后仍要过 http(s) 校验,非协议相对的 URL 一个字都不动。`internal/core/image_url_test.go` 两条用例覆盖这两点;**反向验证**把归一化改回固定 https,测试立刻失败(`http: server gave HTTP response to HTTPS client`)。

1. **版本号把真实源挡在外面**:应用原来报 0.1.0,而源要求 `minAppVersion` 1.3.4 或 1.6.0,源会直接拒绝安装。现在区分「应用版本」与「源接口版本」,后者报 1.6.0。
2. **Core 没有搜索选项**:`search.load(keyword, options, page)` 需要每个选项组一个值,缺了会变成 `undefined` 拼进 URL。Core 新增 `search.optionList` 读取(`js.Runtime.SearchOptions`、`mobile.SearchOptions`),桌面端按默认值传参。MangaDex 因此从协议错误变成正常返回 20 部。
3. **URL 里的空格**:MangaDex 的源把关键词直接拼进 `?title=one piece`,Go 会原样写出,服务端回 `PROTOCOL_ERROR`。现在发送前转义非法字节(Dart 的 `Uri` 也这么做)。
4. **封面永远是灰块**:首帧渲染时封面还没下载完,解码缓存把「没有图片」记成了最终结果。现在只缓存解码成功的结果和失败的下载。
5. **源列表陈旧**:`Library.Sources()` 原来读自己的缓存,从其他路径安装的源不显示;现在每次都向 Core 要。

**合并搜索把「源出错」报成了「源没有结果」(2026-11-01)**:`searchEverySource` 里每个源的失败键 `failed[i]` 是被记下来的,但**从来没有被读过**;汇总时只数 `len(found[i]) == 0`,于是失败的源和「搜到 0 条」的源进了同一个篮子。界面上只会说「其中 N 个源没有结果」——用户没法区分是自己关键词的问题还是某个源坏了。现在 `quietSources` 把两者分开:实跑 `goda`(可用)+ `ykmh`(Cloudflare 拦)合并搜索「火影」,提示是 **「30 部,其中 1 个源出错」**,30 张卡片照常显示且各自带来源名;修之前这句会是「30 部,其中 1 个源没有结果」。反向:两条分支的措辞由 `TestQuietSourcesSeparatesErrorsFromNoResults` 分别锁住。

**源设置面板的真实界面验证(2026-11-01)**:用 `goda`(声明了 `domains`/`api`/`image` 三个 input 设置)打开「漫画源」窗口,面板如实列出「[goda] 的设置」:域名 `godamh.com`、API域名 `v2.apikk.top`、图片域名 `c-nd3-1.6wm.top`,每项带「修改」;同页显示「已安装 1 个源 · GoDa漫画 goda · v1.2.1」与启用开关、设置、删除。把域名改成 `godamh.example` 后,界面顶部出现「已保存设置」,面板里的值同步变成新值,落盘为 `"settings":{"domains":"godamh.example"}` —— 只写入被改的那一项,其余两项继续用源声明的默认值。「设置确实到达源」这一环由 Core 层单独验证过(改 `hot_manga` 的 `image_quality` 后 `loadSetting` 读到的值随之改变),所以这里只claim「界面读得出、改得动、存得下」。

**分类浏览页对声明了 explore 的源显示「没有分类浏览」(2026-11-01)**:`goda` 与 `manhuagui` 的脚本里都写了 `explore`,界面却都显示「这个源没有分类浏览」。根因是 `a.win.Update` 的回调**可能异步执行**:旧代码在列表写回去之前就把 `loadExplore` 启动了,后者推进了 `exploreGen` 代次号,于是列表那次更新被判定为过期而丢弃,页面就停在「没有分类」上。现在把「写入列表」与「加载第一页」放在同一个回调里,代次号只在列表已提交后才前进。修完用真实源复测:`goda` 显示分类标签与「近期更新」列表,`manhuagui` 显示「热门漫画最新更新」并列出条目 —— 两个源都正常,说明这不是个别源的问题。

**书架、收藏与阅读进度在真实界面上的验证(2026-11-01)**:用 `goda` 的真实数据、真实网络在 Windows 上跑了一遍界面动作,并在 `shelf.json` 与截图两边核对:

- **下一话**:打开第 1 话后点阅读器的「下一话」,`shelf.json` 的 `history` 记成 `chapterIndex: 1`、`chapter: "第2话 木叶丸！"` —— 不是只改了标题,索引也跟着走了。
- **收藏 + 阅读进度 + 收藏本页**:同一轮里点「收藏」「阅读」「收藏本页」,`shelf.json` 同时出现三处记录:`folders["收藏"].items` 里是这本书;`history` 里是 `page: 27, percent: 47`;`pictures` 里带着第 27 页的真实图片地址(`https://c-nd3-1.6wm.top/scomic/.../27.webp`)与章节名。界面上对应显示「已收藏第 27 页」提示、按钮变成「已收藏」、「共 162 章 · 上次读到 第 27 页 · 47%」与「继续阅读」,章节行上有「读到」标记。
- **书架页**:三种分区都画出来了 —— 「继续阅读」(封面 + 第1话 · 第27页 · 47% · 1 分钟前)、「图片收藏 · 1 张」(缩略图 + 回到这一话)、「收藏 · 1 部」(打开、取消收藏、重命名、删除)。

截图:`ui-fav.png`(详情页三处状态)、`ui-shelf-real.png`(书架)。**边界**:这些都只覆盖 Windows 与这一台机器的真实网络,不覆盖 macOS/Linux,也不覆盖触屏手势。

**合并搜索的卡片来源名被裁掉(2026-11-01)**:同一张合并搜索截图里,第一行卡片的来源名都画出来了,第二行起没有。先排除了「副标题把标签挤出去」——探针显示这 12 条结果的 `subtitle` 全为空;再把网格单元格高度从 240 试到 260、275,来源名就都回来了,所以原因是**单元格高度装不下「封面 180 + 标题 + 来源名」**,内容高的那一行把来源名裁掉了 —— 结果看起来像从没有来源的地方来的。定在 260(275 只会更少显示一行)。

**AI 窗口「默认模型」是死输入框,发送必然 400(2026-11-01)**:在真实界面上点发送,窗口回「Magpie 拒绝了这个请求(HTTP 400)」。逐层查下来是两件事叠在一起:第一,`a.ai.model`(默认模型输入框)只在读取设置时被赋值,**从来没有写回设置** —— 输进去的模型等于没输;第二,`ModelFor` 的注释写「空字符串表示交给 Magpie 的默认路由」,而这台 Magpie 的答案是 `invalid request: model must be a nonempty string`。也就是说「空模型 = 让网关自己路由」这个假设本身是错的,客户端据此发出的请求一定会被拒。现在:输入框的值在 `测试连接` 与 `发送` 时都写回设置(`applyAIFields`,同时去空格),`Session.Ask` 在模型为空时直接返回 `ErrNoModel` 并在窗口里给出「先点读取模型,把这次要用的模型或组填进档位或默认模型」,而不是把注定失败的请求发出去。反向验证:把写回那一行删掉,`TestTheTypedModelReachesTheRequest` 立刻得到 `model = "", want what was typed`。

**AI 窗口对真实 Magpie 的界面证据(2026-11-01)**:`cmd/uicheck` 新增 `-ai` / `-ai-test` / `-ai-model` / `-ai-prompt` / `-ai-out`,可以打开 AI 窗口(它是独立窗口,需要单独截屏)并走一遍真实流程。实跑结果:地址 `http://127.0.0.1:3425`、`Magpie 可用,共 53 个模型或路由组`、目录列出 `group/laincat`、`group/laincat-subagent`、`auto/gpt-6-luna`、`auto/deepseek-v4.1-flash`、`auto/minimax-m3`、`auto/deepseek-v4-pro` 等;选 `group/laincat` 后发送,窗口里出现真实流式回答「我收到了你的消息。」。截图 `ui-ai-magpie.png`(目录)与 `ui-ai-answer2.png`(回答)保留在证据目录。

**检查工具报的源接口版本和 App 不一致(2026-11-01)**:`main.go` 的 `sourceVersion` 提到 1.6.3 之后,`cmd/uicheck`、`cmd/readercheck`、`cmd/importcheck`、`cmd/verifycheck` 还各自写死 `AppVersion: "1.6.0"`。后果很具体:卡在 1.6.3 的两个源(`cxc`、`mangabz`)在 App 里能装,在检查工具里被版本门槛拒掉 —— **工具对产品下的结论是错的**,而这正是我这轮踩过的坑。现在这个数字只有一份:`internal/sourceapi.Level`,App 与四个工具的 `AppVersion` 都引用它,`main.go` 的 `sourceVersion` 也只是它的别名。

**真实 Magpie 安装版已连通(2026-11-01)**:此前「对真实 Magpie 发起请求」一直是待验证项。本机 `magpie-windows-amd64.exe` 正在运行,网关监听的 `127.0.0.1:3425` 与 `internal/magpie.DefaultAddr` 一致(同一进程另外在 3430 提供需要登录 cookie 的 Web UI)。新增的 `internal/magpie/live_test.go` 用 `NK_MAGPIE_LIVE=1` 才跑,对真实网关上做完整一条:**53 个模型**(首个 `group/laincat`)、真实补全、`chat` / `responses` / `messages` 三种协议各走一遍流式,三条回复都精确等于 `ready`。这条证据同时说明目录读取、协议选择、SSE 解析、流结束判定在这台机器上是通的。

**`responses` 协议把回答拼了两遍(2026-11-01)**:上一条验证一开始返回的是 **`readyready`**。抓真实事件流才看清原因:网关先发 `response.output_text.delta`(字段 `delta:"ready"`),再发 `response.output_text.done`(字段 `text:"ready"`,是**整段成品文本**)。解析器对 `delta` 和 `text` 都取了,于是同一段文本进了两次。现在只取 `delta`;`text` 是结束事件里的成品,再取一次就是把答案复制一遍。反向验证:把 `text` 回退加回去,单元测试立刻得到 `text = "readyready", want "ready"` —— 与真实网关上一模一样的症状。

**桌面端真实 UI 验证(2026-11-01)**:此前只有无头测试,界面本身没有证明过能用真实源走完。这轮用 `cmd/uicheck` 与 `cmd/readercheck` 在 **Windows 本机**、真实网络下跑了两条:

- `goda`:搜索「火影」→ 详情显示封面、作者「集英社」、标签「冒险/战争」、**共 162 章**;`readercheck` 打开第 1 话得到 **57 页**,阅读器渲染 57 个 `figure`/`img`,已解码 4 页(首图 623×960);截图 `ui-real-goda.png` 与 `ui-real-reader.png` 保留在证据目录。界面在打开过阅读后出现「读到」「继续阅读」。
- `ehentai`(画廊型、无章节列表):搜索 `naruto` → 详情显示封面与标签。修之前这里写的是「共 0 章 / 没有匹配的章节」——**对一个只有单个阅读单元的作品是误导性文案**,现在改成「整部作品是一个阅读单元」,并把无效的章节筛选框一并去掉。`-chapter 0` 走 `openReader(0)`,书架 `shelf.json` 落下 `chapterIndex: 0` 的记录。

顺带修了 `cmd/uicheck` 的一个盲点:它的 `-chapter` 原来被 `n < len(d.Chapters)` 挡住,画廊型作品永远进不去阅读器,这类源在 UI 上就没法验证。现在章节列表为空时允许 `n == 0`,与详情页「开始阅读」按钮走同一条路。

**这条证据的边界**:截图证明界面能画、章节能列、阅读器能渲染真实图片、书架能记位置;它**不**证明沉浸式阅读的滚动体验、也不覆盖 macOS/Linux。

**`kuaikan` 撞到引擎的寄存器上限(未修,属 MoeJS 编译器)**:该源把站点内联的解码函数整段交给 `Function(...)` 编译,而 MoeJS 的字节码编译器对单个函数有 **256 个寄存器**的硬上限,超了直接报 `SyntaxError: function needs more than 256 registers is not supported yet`。这个数字在 `compiler/compiler.go` 里是常量,没有开关;要放开就得给编译器加寄存器溢出(spilling),属引擎改造,不在客户端能修的范围。同一个扫描结论适用于 `manhuaren`、`one_kkk`、`webtoons_zh_hant`、`cosplaytele` —— 它们在我们第一轮用「火影」巡检时都报「搜索无结果」,换成 vensrc 巡检用的固定样本(斗破苍穹 / love / girl)后全部通过,说明那几条是**关键词不匹配**,不是缺陷。

**源自己写了 `Accept-Encoding` 时，压缩响应没有被解开(2026-11-01)**:JM 系源在请求头里显式写了 `Accept-Encoding: gzip, deflate, br, zstd`。Go 的 `net/http` 只在自己**插入** gzip 时才透明解压;一旦调用方自己设了这个头,它就把响应体原样交回。于是源拿到的是 gzip 字节,`JSON.parse` 在第一个字节 `0x1f` 上报 `Unexpected token`,而源把这个异常包装成自己的「后台初始化失败: Error: HTTP 404」——**一条与真实原因毫无关系的报错**,这种错误最难查。修法是在 `Transport.Do` 里:把源声明的编码收敛到本传输层真能解的两种(`gzip`、`deflate`),再按 `Content-Encoding` 解压,解压后仍受 `maxBody` 限制(压缩炸弹不会因为解压而绕开上限),解压完成就删掉 `Content-Encoding` 与 `Content-Length`,免得调用方再按压缩后长度理解。反过来验证:把解压关掉,`jmcomic` 立刻退回同一条「HTTP 404」失败;打开就通过。

**`rumanhua2` 的 `atob` 改不动(引擎与规范不一致,未修)**:该源在解码器里用 `Function("return this")()` 取全局对象,发现没有 `atob` 就自己补一个,并在 `finally` 里把原值写回。浏览器规范里 `globalThis.atob` 是**可写且可配置**的,而 MoeJS 为了共享冻结的内置模板把它设成只读且不可重定义,于是那次「写回」抛 `TypeError: Cannot assign to read only property 'atob'`。已验证:把该源的写回改成「只在真的补过时才还原」,搜索、详情、546 章、取图、阅读页全部通过,说明源本身没问题。修引擎有两条路 —— 全局打开 `MutableIntrinsics`(实测每个 runtime 构造 3.03ms 对 2.60ms,且失去进程级共享冻结模板),或让 MoeJS 按 HTML 规范把 `atob`/`btoa` 设为可写可配置。全部 105 个源里只有这 1 个会写冻结的内置全局(另一个 `cxc` 写的是 `URL`,不在失败路径上),**代价与收益不成比例,所以这一处没有动引擎**,记录在此作为已知限制。

**第三方源全量巡检的分类结果(2026-11-01)**:71 个 vensrc 脚本,**59 个通过、12 个失败**。这个数字和更早的「34 个通过」不能直接比 —— 老判定把两件事都算成客户端失败:一是「样本词在站点里就是没结果」(`cxc` 查 `海贼` 是 0 条、查 `魔氣` 是 4 条,因为该站作品叫《航海王》),二是「源给了回调、客户端没接」。两次修正分别让 14 个和 9 个源转通过,剩下 12 个逐项归类,没有一个被含糊带过:

| 类别 | 数量 | 源 | 结论 |
| --- | --- | --- | --- |
| 图片地址本身报错 | 7 | `311`、`baozi`、`baozi_fixed`、`haokan`、`manga51`(仅封面)、`mh03`、`mh1234_fixed` | 站点/CDN。**直连复核过**:`baozi` 的图片地址 curl 也是 404,`manga.5um.net`(`311`/`mh03`)连 TLS 握手都失败 |
| 站点搜索 404 或首页也空 | 3 | `guazi`、`isamanhua`、`sunday_webry` | 站点改了路径或不可用 |
| 版本门槛 | 1 | `recode-jm`(`minAppVersion 1.16.0`) | 源要求的版本高于参照版本,拒绝安装是对的 |
| 运行时把 JS 内建锁成只读 | 1 | `rumanhua2` | 依赖问题,见下 |

**原先 6 个「加密/乱序图片」和 1 个「镜像回退」全部转通过**:桌面端接上 Core 的图片回调后,`cxc`(659,568 字节密文 → 494,647 字节真实 PNG)、`ccc`、`mojoin`、`mojoin_fixed_v107`、`manwaba_fixed`、`manwaba_fixed_v113`、`4khd_cosplay`(1,243 字节 HTML 错误页 → 回退后 248,428 字节 webp)都过了。阅读器层面另外实测过一遍:`cxc` 的作品 3 页全部渲染、`decoded=3`、首图 690×1380 —— 解密出来的是能真正解码的图,不只是字节看起来对。

**上游也还没有修(2026-11-01 复核)**:翻过 `MoeForge/moejs` 的版本与分支 —— 标签只到 `v0.1.0-alpha.11`,而 `main` 上有一个比我们用更新的提交(`v0.1.0-alpha.11.0.20261030083738-b9fa0469d1df`,2026-10-30)。用独立探针(不碰仓库)在那个更新版本上跑同样的 `Object.getOwnPropertyDescriptor(globalThis,'atob')` + 严格模式赋值,**结果仍是 `writable=false`、`Cannot assign to read only property 'atob'`**。所以这不是「我们锁的版本太旧」,而是上游现在的行为;等它改,或另找实现。

**`rumanhua2` 暴露了运行时的一处保真度问题**:它在 `finally` 里把 `root.atob` 还原回去,而 **moejs 把 `atob`/`btoa`/`TextDecoder` 等内建都定义成不可写且不可配置**,于是类方法(严格模式)里的赋值直接抛 `TypeError: Cannot assign to read only property 'atob'`。探针实测(`Object.getOwnPropertyDescriptor(globalThis,'atob')`)返回 `writable=false, configurable=false`,并且列表里 `atob`、`btoa`、`TextEncoder`、`TextDecoder`、`structuredClone`、`Intl` 全部同样锁死;真实浏览器与 Node 里这些是可写可配置的普通属性。因为不可配置,Core 的 JS 层**改不动**它,只能在运行时侧修或换实现 —— 先记录,不擅自改依赖版本。

**`hot_manga` 的图片质量设置不生效 —— 是源自己的正则过期(2026-11-01,已定位)**:这一条挂了几轮。实测把 `image_quality` 从默认改成一个更低的值后,章节里第一张图的地址**一个字都没变**(都是 `...jpg.h1500x.webp`)。查源:`hot_manga` 用 `url.replace(/\.jpg\.h\d+x\.jpg$/, ".jpg.h{N}x.jpg")` 改质量,**而站点现在返回的后缀是 `.webp`** —— 正则根本匹配不上,自然不改。反向确认过我们的链路没坏:同一套 `Store.SaveSetting` → `loadSetting` 在 `manga_dex` 上确实生效(`/data/` → `/data-saver/`)。所以这条不是客户端问题,是源需要把正则改成 `.((jpg)|(webp))$`。

**只注册了一个解码器,导致脚本静默不生效(2026-11-01,已修)**:接上 `modifyImage` 之后,拿 `shonen_jump_plus` 实测却发现返回的还是 `image/jpeg` —— 脚本明明带上了(配置里 548 字符,`req.ModifyImage` 也非空)。原因是这个构建只 `import _ "image/png"`,**没注册 jpeg 与 gif 的解码器**,于是 `image.Decode` 对 JPEG 报「未知格式」,被当成「解不了」而走了放行分支。补上两个空导入后,同一张图从 232,750 字节 JPEG 变成 879,725 字节 PNG(该源用脚本还原被切块打散的页面)。`TestModifyImageRunsOnJPEGBytes` 用 JPEG 输入 + 旋转脚本钉住这条:结果必须是 `image/png`,否则说明脚本没跑。

**解不了的格式要放行,不能让整页失败(2026-11-01)**:`hitomi` 这类源给的是 avif,而我们只注册了 png/jpeg/gif/webp 解码器 —— 源一旦给它配上 `modifyImage` 脚本,解码失败就会让整页打不开。现在把「这个构建读不了这种格式」与「脚本本身出错」分开:前者原样返回服务器给的字节与内容类型(读者至少能看到图),后者仍然如实报错。`TestModifyImagePassesThroughAnUnknownFormat` 用一段 avif 头钉住这条。

**`modifyImage` 已接上(2026-11-01,pin `6aa9356`)**:Core 新增 `Runtime.ModifyImage`(把解码后的 RGBA 放进一次性位图仓库、在独立运行时里跑源给的脚本、取回结果位图;解码与编码仍由客户端做),桌面端在图片路径里接上:取回 → 有 `onResponse` 先改写 → 有 `modifyImage` 就解码成 RGBA、跑脚本、编码回 PNG,内容类型随之改成 `image/png`。`modifyImage` 在配置里是**字符串脚本**而不是函数,所以它走 `ImageConfig.ModifyImage` 而不是回调句柄 —— 这一点最初写错过,补了一条用例钉住。确定性验证:源返回 4×2 的红蓝图 + 顺时针旋转脚本,取回后是 2×4 且红在上;脚本返回非图像时如实报错、不回落到原图。真实源侧 `hitomi`、`manhuagui` 实测通过(它们的图多为 avif/webp,不一定触发脚本)。

**全量巡检基线(2026-11-01)**:34 个官方源跑 `TestLiveSource`(搜索 → 详情 → 章节 → 取图),**18 通过 / 16 失败**。失败的 16 个**全部复跑过一次**,其中 `hcomic`、`ykmh` 复跑即通过 —— 单轮结果确实不能当结论。

其余 14 个逐个归因,**没有一个是客户端回归**:

| 归因 | 源 | 证据 |
| --- | --- | --- |
| 站点 Cloudflare / 403 / 返回 HTML | `manwaba`、`mh18`、`baozi`、`comic_walker`、`shonen_jump_plus` | `CloudflareException`、`Invalid status code: 403`、`Unexpected token < in JSON` |
| 站点超时或断连 | `baihehui`、`happy`、`hcomic`(首轮) | `context deadline exceeded`、`Post … EOF` |
| 需要账号 | `picacg` | `Not logged in` |
| 需要自建服务 | `kavita`、`komga`、`lanraragi` | 连 `demo.kavita.org`、`Setting not found: default_username`、API 400 |
| 站点/源漂移 | `jcomic`、`jm` | 见下 |

`jcomic` 值得记一笔:搜索返回的作品 `id` 已经是**标题式 slug**(`"Scarlet Prisoner"`),章节图片全是空串 —— 站点把作品 URL 改成了 slug、图片改成了懒加载,源按旧结构取 `img.attributes["src"]` 自然取不到。客户端把 `id` 原样传给源、把源给的空串报成 `core: image URL is not http(s)`,行为是对的,坏的是源与站点之间。`jm` 是同一个形状:它的 `init` 会去拉一个域名列表文件,那份文件的格式变了,`Cannot read properties of null (reading 'length')` 是源在解析时炸的。

所以这轮**没有改客户端行为**,只把结论固化了:巡检里给需要自建服务的源单列一行 `SKIP (needs a server the user runs)`,不再和真失败混在一起(`NK_LIVE_SERVER_SOURCES` 覆盖,默认 `kavita komga lanraragi`)。

**门禁:源声明了却没接上的能力,现在会失败(2026-11-01)**:`category`、`ranking`、`account`、`favorites`、`translation` 这几处缺口都是同一类问题 —— 源在脚本里声明,客户端读不到,症状只是一个空屏或一个英文键名,只能靠人反复重读契约去找。现在 `scripts/check-capabilities.sh` 按能力列出「Core 入口 + 界面入口」,用**固定字符串**匹配(改名后不会靠前缀继续通过),任一路径消失就失败,已接进 `scripts/check.sh`。

负向验证:把 `favoritesView` 改名、把 `"favorites.loadComics"` 改名,脚本分别点名缺失的能力并非零退出;恢复后回到 `ok`。它只保证**路径还在**,不保证行为正确。

**源自己的收藏:11 个源声明了,之前完全没读(2026-11-01,pin `0bcdaef`)**:`favorites` 块是源在**它自己那边**保存的收藏,与客户端的本地书架是两回事。三种形状实测各不相同:

| 源 | multiFolder | loadFolders | loadComics | 实际表现 |
| --- | --- | --- | --- | --- |
| `nhentai` | 否 | 无 | 有 | 未登录时源回 `Authentication required. Please login again or provide API Key in source settings.` |
| `ehentai` | 是 | 有 | **无** | 能列出 1 个收藏夹,但**没有列表接口** —— 之前调用会得到 `source does not define this function` |
| `wnacg` | 是 | 有 | 有 | 收藏夹与列表都在 |

所以客户端必须**按声明区分能力**:只声明了收藏夹切换、没有列表的源(`ehentai`)不能去列表,否则就是上面那个错;只支持单收藏夹的源不传收藏夹 id。桌面端新增「源收藏」入口,按源实际能力显示:有列表的显示列表与分页,没有列表的明说「这个源没有提供收藏列表」,未登录的显示**源的原文**与重试按钮。

为此 Core 加了 `Value`(读**数据**而非调函数):`favorites.multiFolder` 是数据,`Call` 调不到。

**源声明的界面文字:之前一直显示键名(2026-11-01,pin `239221b`)**:`ehentai` 的设置写的是 `title: "domain"`,可读文字在它自己的 `translation` 表里(七成的源都有这张表)。不读这张表,面板上就是 `domain`、`ehevent`、`hvevent` 三个英文键 —— 这正是上一轮账号面板截图里能看到的东西。现在 Core 按客户端语言取表:先精确 locale,再同语言(表里写 `zh_CN`、客户端是 `zh_TW` 也能匹配),再英文、中文兜底;桌面端把它用在设置项的标题和选项文字上,截图确认变成「域名」「触发黎明事件」「提示HV遭遇战」。

Cookie 字段名(`ipb_member_id` 等)**不翻译**:那是 Cookie 名,不是界面文字。

**源账号登录:三种方式,之前一种都没接(2026-11-01,pin `896f647`)**:15 个官方源声明了 `account`,但有**三种**完全不同的登录方式,只认 `account.login` 会漏掉一半:

| 方式 | 声明 | 实测 |
| --- | --- | --- |
| 账号密码 | `account.login(user, pwd)` | `picacg`(返回源原文 `Failed to login`)、`ccc` |
| 网页内登录 | `account.loginWithWebview {url, checkStatus, onLoginSuccess}` | `nhentai` 只有这一种;`ccc`/`mycomic` 兼有 |
| Cookie 字段 | `account.loginWithCookies {fields, validate(values)}` | `ehentai` `[ipb_member_id ipb_pass_hash igneous star]`、`mycomic` `[cf_clearance]` |

桌面端现在在源设置面板里按声明画:密码方式给账号/密码/登录/登出;Cookie 方式按 `fields` 逐个给输入框 + 「校验并保存」,由源自己的 `validate` 判断并写入 Cookie(空值一律被源拒绝,不会静默通过);**只提供网页内登录的源**(`nhentai`)**明确显示「桌面端暂不支持」**,而不是给一个按不动的表单。`ehentai` 正是这种源 —— 它根本没有密码登录。

源抛出的错误按原文显示:`picacg` 是 `Failed to login`,`ccc` 是 `No JavaScript UI handler provided`(它的登录要宿主提供 JS UI 会话,这是当前边界,不是客户端 bug)。活体测试 `TestLiveAccountStyles` 覆盖三种方式。

**排行榜页:9 个官方源声明了它,之前一直没用(2026-11-01,pin `c22a944`)**:`enableRankingPage: true` 配 `categoryComics.ranking = {options: ["date-Recent", …], load: (option, page) => …}` 是源声明的**另一个独立入口**,不是分类维度。Core 的 `Category` 现在把它一起带出来(`ranking.options`,值/标签按同样的 `-` 拆分),并新增 `RankingComics` 调用 `categoryComics.ranking.load`;桌面端分类页多一行「排行」,选中后走同一套列表与分页(`catRank` 决定分页走排行还是走维度)。

真实源验证:`nhentai` 五个选项**全部**返回 25 部 —— `date` 26307 页、`today/week/month` 各 20 页、`popular` 26307 页;界面截图里 `Popular Week` 选中后列表换成了真实周榜。活体测试 `TestLiveCategoryRanking` 覆盖这一条。

**随机维度不是"拼 20 个标签"(2026-11-01)**:上一轮把 `randomNumber` 当成"要拼几个标签",这是错的。源里的注释写得很清楚 —— `randomNumber` 是**同时显示多少部**,不是拼关键词的数量。实测也说明拼不动:`nhentai` 把 8 个标签拼起来还能返回 20 部,10 个就是 0 部,12 个和 20 个直接被站点拒绝(429;用错的参数时是 400)。原因是漫画搜索把多个标签当**同时满足**,没有作品同时带 20 个标签。现在随机维度只取**一个**取值去搜索(点一次换一个),`nhentai` 的 `maid` 返回 25 部 / 845 页。

另外记一笔容易踩的坑:**搜索要用搜索自己的选项,不是分类页的选项**。`nhentai` 分类页声明的是 `/-Recent`(加载器会把 `/` 去掉),搜索要的是 `date` —— 拿前者去搜索会得到 400。

**13 个带分类的官方源逐个跑过(2026-11-01)**:结果 **11 通过 / 2 未通过**,而且两个都不是客户端的错。过程中修掉三处真问题:

1. **取值可以是对象**:`hcomic` 的「热门TAG」`categories` 是对象数组,`String(v)` 会得到 `"[object Object]"` —— 一个任何加载器都不认识的参数。现在取它的 `label`/`name`/`text`/`value` 与自带的参数。
2. **选项没有默认值时要取第一个**:`komiic` 报 `Variable "orderBy" has invalid value .`,因为空字符串被当成选项发出去;搜索选项本来就是这个回退,分类选项此前没有。
3. **测试自己传错了第一个参数**(用维度名而不是取值)—— 与界面曾经犯的是同一个错,`nhentai` 因此显示空页。

剩下两个:`copy_manga` 的「排行」维度稳定返回 **210**(站点对端点的策略,同源的「主题」维度正常);`baihehui` 的「作品类型(需要登陆)」维度需要账号、另一个「更多推荐」维度站点返回空。**巡检因此按维度而不是按源下结论**:一个源里有的维度能用、有的维度被站点挡住,这是真实状态。

**随机维度,以及一个我自己写错的参数(2026-11-01,pin `c146110`)**:`nhentai` 的「Tags」是 `type:"random"` + `itemType:"search"`,语义是**随机挑 20 个标签去搜索**,不经过分类加载 —— 界面上现在是一个「随机 20 个」按钮,点了随机取 20 个值当关键词搜索。同时修掉一个真 bug:传给加载器的**第一个参数应是选中的取值,而不是维度名**。`nhentai` 因此一直显示「这个分类没有内容」(它按取值查 id,拿到的却是「Language」);`manhuagui`、`copy_manga` 之前能出结果纯属巧合(它们只看 `param`)。另外 `nhentai` 这类源还要 `groupParam` 作为参数(它的加载器按 `"language"` 分支走),Core 现在把 `groupParam` 一并带出来,桌面端按「每个取值的参数 > groupParam > 取值本身」的顺序决定。修完截图确认:`nhentai` 的 Language 分类列出真实作品;`manhuagui` 与 `copy_manga` 的活体检查仍通过。

**`dynamic` 维度现在也读得到(2026-11-01,pin `d90b7ed`)**:官方源里有 12 处分类维度声明成 `type:"dynamic"` —— 它们**不列取值,只给一个 `loader`**,由运行时的 loader 返回(还可能是异步的,`manga_dex` 的「Tags」就是从标签表现算的)。此前我们只读静态的 `categories`,这类维度在界面上会是空的。Core 现在把每个维度都当承诺处理:声明了 `loader` 就调用它,再把返回项归一 —— 可以是字符串,也可以是 `{label, target:{param}}`,参数取自 `target`。实测 `manga_dex` 的「Tags」维度已列取值(如 `Oneshot`),加载 20 部、688 页,第 2 页 0 重复。**还没处理 `random` 那一种**:`nhentai` 的「Tags」是 `type:"random"` + `itemType:"search"`,语义是「随机挑若干标签去搜索」,不是分类加载,留作下一步。

**分类翻页与多维度的活体检查(2026-11-01)**:新增 `internal/library/live_category_test.go`(`NK_LIVE_JS` 门控,平时离线跳过):读声明 → 每个 part 各加载一次 → 有下一页时再读第 2 页,并要求第 2 页非空且不与第 1 页完全重复。实测两源:`manhuagui` 的「漫画柜」一个维度「类型」42 部、1479 页,第 2 页 42 部 0 重复;`copy_manga` 的两个维度(排行、主题)各自加载成功,主题有 2675 页,第 2 页同样 0 重复。顺带一个值得记的事实:`copy_manga` 的**详情**会稳定返回 210,而**分类列表**是好的 —— 同一站点对不同端点策略不同,巡检里要分开看。

**分类页接进界面了(2026-11-01)**:顶栏多了一个「分类」按钮,进去是源自己的分类页 —— 每个维度一行、取值是按钮,选中的那个是实心按钮,下面直接列漫画(封面 + 打开),带翻页。**每个取值是一个独立入口**,不是「各维度各选一个再组合」:加载器收的是「一个名字 + 它对应的参数」,`copy_manga` 的两个维度(排行、主题)正是各自取值直接进加载器。真实截图验证:`manhuagui` 的「漫画柜」39 个类型值全部列出、「全部」处于选中态、下方列表有封面与打开按钮。**分类选项也可以调了**:源声明的 `categoryComics.optionList` 现在和搜索筛选共用同一套渲染(`optionChips`),点一下就按新选项重新加载当前这一条;两组渲染合到一起,以后改筛选样式不会只改一半。真实截图:`manhuagui` 的四个选项(地区/年龄/状态/排序)全部列出,默认值处于选中态。

**分类页的声明现在读得到了(2026-11-01,pin `722d8d2`)**:34 个官方源里有 **32 个**声明了 `category`,而这是官方客户端的主浏览方式 —— 一个标题、若干维度(类型/地区/状态)以及加载器自己的选项。桌面端此前完全没有这条路,连声明都读不到:那些值是**数据不是函数**,而 `Call` 只能调函数。Core 新增 `Runtime.Category`:一次求值把对象压成 JSON —— 每个维度的取值、每个取值对应的参数、以及 `categoryComics.optionList`(与搜索选项同形状,客户端可以复用筛选代码);桌面端包了 `CategoryPage` 与 `CategoryComics`。真实源验证:`manhuagui` 的 `漫画柜` 一个维度「类型」39 个值 39 个参数、4 个选项,`categoryComics.load` 返回 42 部、4 页。**界面还没接**:浏览视图目前只有源自己的 `explore`,分类入口是下一步。

**标签现在可以点(2026-11-01)**:官方有 24 个源实现 `comic.onClickTag(namespace, tag)`,而桌面端从没调过 —— 详情页的标签是一行拼接文本,点了没反应。现在每个标签值是一个按钮,点了会问源「这个标签要做什么」:源返回 `action:"search"` 就带着它给的搜索词(`copy_manga` 给的是 `作者:<名字>`)直接搜;返回 `action:"category"` 时按分类参数打开 —— **这一种还没有实现**,因为我们这条浏览路径传不进源自己的分类参数,所以如实提示「当前版本还打不开分类参数」,而不是拿标签当关键词去搜一个源没要求的词。源不支持该命名空间时会抛自己的话(例如「未支持此类Tag检索」),那句话原样显示。Core 侧 `ClickTag` 有 4 类断言(搜索词、分类参数、源拒绝时说原因、源没有这个钩子时 `ErrNoTagAction`),界面侧把「标签变成什么」抽成 `tagOutcome` 并有 1 条用例;`goda` 的详情页截图确认标签已渲染成按钮。

**封面不跑像素脚本(2026-11-01)**:官方在 `onThumbnailLoad` 的注释里写明 `modifyImage` 与 `onLoadFailed` 对缩略图**会被忽略**。我们原来对封面也照做,现在按规格只对正文页执行 —— 封面本来也小,给它跑一遍解码/编码不划算。`TestModifyImageIsIgnoredForCovers` 用同一个源同时声明两个钩子,确认 `thumbnail=true` 的请求不带脚本、正文请求带。改完用 `manwaba_fixed` 实测封面:**加密的封面仍然解得开**(取回 40,111 字节、内容类型被清空,说明 `onResponse` 照样跑了)—— 被跳过的只有像素脚本与镜像回退(`onLoadFailed` 同样按规格对封面忽略)。

**`modifyImage` 的覆盖面,以及一处统计订正(2026-11-01)**:真正在 `onImageLoad` 里挂脚本的只有 **6 个文件** —— 官方只有 `jm`,vensrc 里是 `hanabimanga`、`jmcomic`、`recode-jm`、`shonen_jump_plus`、`sunday_webry`。此前写成「11 个源」是统计错误:那把 `hitomi`、`manhuagui`、`manhuaren`、`nhentai` 等只在**注释里提到** `modifyImage` 的文件也算进去了(它们写的是「这个字段在封面上会被忽略」这类说明)。脚本的两种典型用法:一种是 `jm` 把长条图按 `num` 等分并倒序重排,另一种是 `shonen_jump_plus` 把被切成 4×4 乱序块的页面还原。**这两种现在都已由桌面端执行**,`shonen_jump_plus` 实测从 232,750 字节 JPEG 变成 879,725 字节 PNG。

**桌面端已接上图片回调,六个源当场修好(2026-11-01)**:`ImageRequest` 现在把 Core 给的句柄一起带回来,`doImage` 取回后按三点走 —— 有 `onResponse` 就先把响应体交给源改写(解密),取回成功但**是一张 HTML 页面**而源又声明了 `onLoadFailed` 就当成失败去问下一个镜像,真失败同样问下一个镜像;最多跳 4 次,每一跳都重新过 http(s) 校验与大小上限。`4khd_cosplay` 这类源正是「200 + 错误页」,官方客户端靠解码失败触发回退,我们这边解码在 WebView 里,所以由取回方判断。实测:原先归在「客户端缺口」的 6 个源全部转为通过 —— `cxc`(659,568 字节密文 → 494,647 字节真实 PNG)、`4khd_cosplay`(1,243 字节 HTML → 回退后 248,428 字节 webp)、`ccc`、`mojoin`、`mojoin_fixed_v107`、`manwaba_fixed`、`manwaba_fixed_v113`。

**Core 已提供回调能力(2026-11-01,pin `69ed1f9`)**:Core 的 `comic/js` 新增 `ImageConfig`/`ApplyImageResponse`/`ImageFallback`,把 `onResponse`/`modifyImage`/`onLoadFailed` 留在源的运行时里、用句柄命名,宿主可以「把响应体交给 onResponse」「问下一个镜像」;不认句柄的宿主行为完全不变。**本仓库已把 pin 移到 `69ed1f9` 并重新取回 vendored Core,但客户端还没调用这三个方法**,所以下面这条症状暂时不变 —— 差别在于它现在只差一次接线,不再卡在 Core。

**图片配置里的三个回调,客户端一个都没接(2026-11-01,已确认,未修)**:`ImageLoadingConfig` 除了 `url`/`method`/`headers`/`data`,还可能带 `onResponse`(把响应体解密或改写后再用)、`modifyImage`(用像素缓冲脚本改图)、`onLoadFailed`(这次地址不行就换下一个镜像)。**这三个都在 `ImageRequest` 里被丢掉了**,于是同一类源分别坏在三个地方:`cxc`/`mojoin`/`manwaba` 把密文当图片(见下条),而 `4khd_cosplay` 的首选镜像 `img.uuss.uk` 返回的是 1243 字节的 HTML 错误页 —— 源本来就写了「这一家失败就换 4khd/wordpress」,客户端没调 `onLoadFailed`,就直接把这张 HTML 当成了图片。两处都能用同一个机制修:让 Core 留住配置里的函数并允许「按句柄调用」,宿主再把响应体交给它。:`cxc` 这类源给的不是图片地址,而是「地址 + `#cxc-v1=密钥:IV`」,再由 `comic.onImageLoad` 返回一个 `ImageLoadingConfig`,里面用 `onResponse` 回调把响应体做 AES-CBC 解密、转成真正的图片。**客户端的 `ImageRequest` 只取了 `url`/`method`/`headers`/`data`,把 `onResponse` 与 `modifyImage` 直接丢掉了**,于是 `/img` 把密文当图片返回:`cxc` 实测 659,568 字节、`Content-Type: image/png`,但文件头不是任何已知图片格式,阅读器只能显示破图。`comic/js/init.js` 里 `ImageLoadingConfig` 是有这两个字段的,所以这不是源写错,是宿主没实现。修它要让宿主把响应体交回 JS 回调、再把返回的字节拿回来 —— 需要 Core 提供「按配置 id 应用 onResponse」的入口,属于 Core 接口变更(会动 pin 与两个客户端),因此先记录、不擅自扩大。

**源写了哪种搜索入口就调哪种(2026-11-01)**:官方的搜索有两个入口 —— `search.load(keyword, options, page)` 收页码,`search.loadNext(keyword, options, next)` 收上一页返回的游标。客户端只认前者,于是 `ehentai` 这种只声明 `loadNext` 的源在搜索时直接报「source does not define this function」,而它在官方客户端里是能用的。现在先看源声明了哪个;声明了 `loadNext` 就按游标翻页,游标按源和页码记住,新搜索(第 1 页)清掉上一轮。之前 `search` 的 `load` 与 `loadNext` 混在两个文件里的判断也统一了 —— 视频那条路早就有,漫画这条一直缺。

**游标翻第二页的活体检查(2026-11-01)**:只声明 `loadNext` 的源,第一页能出结果不代表第二页也能 —— 游标丢了或形状不对时,第一页照样正常。新增 `internal/core/live_cursor_test.go`(`NK_LIVE_JS` / `NK_LIVE_QUERY` 门控,平时离线跳过)读两页并要求第二页非空且不与第一页完全重复。用 `ehentai.js` 实测:第 1 页 25 条、第 2 页 25 条、重复 0 条。iOS 端改成同样的调用后,这条路径在 Core 侧先有了证据。

**多选筛选组必须传 JSON 数组(2026-11-01)**:`multi-select` 组的值是「选中键的 JSON 数组字符串」,源的 `JSON.parse(options[i])` 就是按这个读的。我们给的是单个键,`ehentai` 因此在 `JSON.parse("0")` 上报 `Unexpected token ,`。Core 的 `SearchOptionGroup` 现在有 `DefaultValues`(来源声明的数组默认值),桌面端把多选组编码成 JSON 数组再传,单选组仍然只传一个键 —— 两种形状不混用。

**`-<none>` 这种空值选项被读成了字面量(2026-11-01)**:选项格式是「`值-文字`」,左边可以是空的,`-<none>` 的意思是「值为空、文字是 `<none>`」。原来只要 `-` 出现在第 0 位就把整串当值,于是「没选任何筛选」被传成字符串 `-<none>` —— 源用 `if (stars)` 判断有没有选,`-<none>` 是真值,于是往请求里塞进一个不存在的筛选。现在按「第一个 `-` 分隔、左边可以为空」解析。

**没有章节列表的源打不开阅读(2026-11-01)**:`ehentai`、`hitomi`、`nhentai`、`wnacg` 这类画廊源在 `comic.loadInfo` 里根本不返回 `chapters`,整个画廊就是一个阅读单元,它的 `loadEp` 收空章节 id。客户端三处都按「有章节」写死:详情页的「开始阅读」把章节列表置空后 `readerURLFor` 直接返回空串(按钮是死的)、阅读页的 `/img` 拿页面引用(画廊源给的是 `"0"`、`"1"` 这样的页码)当地址校验于是回 502、`/switch` 与 `switchChapter` 要求章节 id 非空于是回 404。三处都改成「空章节 id 是一个合法的阅读单元」:阅读地址里 `chapter=` 为空,`/img` 先问源的 `onImageLoad` 拿到真实地址**再**校验 http(s)(安全检查没有放宽,只是校验的是真正要抓的那个地址),`/switch` 把空 id 交给窗口判断。**这不是为某一个源打补丁**:官方 34 个源里有 4 个是这种形状,修之前它们全被判成「失败」。

**源接口版本从 1.6.0 提到 1.6.3(2026-11-01)**:官方客户端当前发布版是 v1.6.3,只有 `cxc` 与 `mangabz` 两个源把 `minAppVersion` 写在 1.6.3,其余源都 ≤ 1.6.0。抬高之前先在这两个源上把搜索、详情、章节、取图整条线跑通,所以这个数字是有实测撑着的,不是照着官方版本填的;真要遇到本客户端没实现的接口,源会在调用处报自己的错,而不是被一个版本门槛挡住。

### 两批源的真实巡检(2026-11-01,两轮)

**官方源复查(2026-11-01,含限流说明)**:34 个官方源一次性跑完是 **16 个通过**,但**这个数字不能当结论** —— 同一批源单独重跑时,整轮里失败的 `ykmh`(403)与 `mycomic`(Cloudflare 挑战)都立刻通过,说明连续全量跑本身会触发站点限流/反爬。剩下的逐个看过:官方 `manwaba` 连 `onImageLoad` 都没声明,而 `vensrc` 的 `manwaba_fixed`(带 `onResponse` 解密)在同一客户端下通过 —— 是源没跟上站点把图片打乱;`copy_manga` 是**搜索能过、详情稳定 210**(换了关键词搜索拿到 30 条,详情每次都 210),同一套请求头与签名在搜索上被接受,所以属站点对详情端点的策略变化,不是客户端拦住了自己。判断一个源该不该算失败,要按上面的分类法看,而不是数通过个数。

官方源(`venera-app/venera-configs`,34 个)第一轮 5 个通过,修完上面几处后 **10 个通过**:`copy_manga`、`copy_manga_multi_accounts`、`ehentai`、`goda`、`hcomic`、`hitomi`、`komiic`、`manhuagui`、`nhentai`、`wnacg`。第三方源(`laincat/vensrc`,`index.json` 24 个 + `extended.json` 47 个,去重后 71 个)从 31 个提升到 **34 个**。

剩下的失败逐条看过,没有一条是本客户端的解析错:

| 失败 | 数量(示例) | 判定 |
| --- | --- | --- |
| 站点改版/接口变更 | `manga_dex`、`hot_manga`、`jm`、`jmcomic`、`recode-jm`、`damotu` | 源或站点侧 |
| 需要登录/账号 | `baihehui`、`cxc`、`hanabimanga`、`kanman`、`kiutaku`、`kuaikan`、`xiutaku`、`myhl` | 源或站点侧 |
| 反爬/Cloudflare | `ykmh`、`311`、`haokan`、`zymk` | 站点侧,单轮结果不稳定 |
| 搜索无结果 | `baihehui`、`dongman`、`one_kkk`、`mojoin` | 关键词不匹配,不是缺陷 |
| 源脚本自身 bug | `copy_manga_multi_accounts`(`res is not defined`)、`hot_manga`(只替换 `.jpg.hNNNx.jpg`,站点已返回 `.webp`) | 源侧 |

其中三条值得单独记:**`manga_dex`** 的 `at-home` 接口对任意章节都返回空 `data` 数组(`hash` 也是空),同一部漫画换 5 个章节结果一样,是站点侧;**`hot_manga`** 的设置确实到达了源(另写用例证明 `SaveSetting` → `loadSetting` 生效),坏在源自己的替换正则不匹配 `.webp`;**`cxc`** 的图片在 `/private/` 路径下,未登录返回的 191,184 字节不是任何图片格式,`Content-Type` 却写 `image/jpeg` —— 需要账号。

## 源本身的变动(记录,不是回归)

2026-10-31 复跑真实源时,拷贝漫画的详情请求开始返回 **HTTP 210**,而该源的脚本自己只接受 200
(`copy_manga.js` 里 `throw \`Invalid status code: ${res.status}\``),于是报 `js: Invalid status code: 210`。
这不是客户端缺陷:同一天复跑,MangaDex(6.4s)、Komiic(1.7s)、Comick(56s)三条端到端全过,错误文本也是源脚本自己抛的。210 属于站点侧的非标准响应,
可能与人机验证或限流有关;在站点恢复正常前,拷贝漫画这条 E2E 会失败。


### 第三方源的一次巡检(2026-11-01)

**先说方法上的一个错误**:第一轮我用 `NK_LIVE_REPEAT=1` 跑了一遍就下了结论,第二轮用 `NK_LIVE_REPEAT=2` 复跑,结果直接打脸 —— `copy_manga` 与 `copy_manga_multi_accounts` 第一次 PASS、第二次 FAIL(站点 210);`ykmh` 第一次 FAIL(Cloudflare)、第二次 PASS。**单次巡检不能当结论**,这正是 `live-sources.sh` 里 `FLAKY` 这个分类存在的理由。

用 `venera-miuix`(Miuix 风格的 Venera 重写版,作者自用)自带的 **34 个漫画源**,关键词「火影」:

**两轮都通过的 6 个**:`goda`、`hcomic`、`hitomi`、`komiic`、`manhuagui`。
**一轮过一轮不过(FLAKY)**:`copy_manga`、`copy_manga_multi_accounts`、`ykmh`。

其余失败的,**绝大多数不是客户端缺陷**,而是源脚本与站点现状不匹配,分三类:
- **需要登录或凭据**:`baozi`、`comick`、`komga`、`mh18`、`picacg`。
- **站点改了返回结构**(返回 HTML 而不是 JSON,或加上了 Cloudflare 挑战):`ccc`、`comic_walker`、`shonen_jump_plus`。
- **源脚本与站点不匹配**:详情页没有章节(`nhentai`、`mxs`、`mycomic`、`manhuaren`、`lanraragi`、`gelbooru`)、章节页懒加载所以 `src` 属性为空(`jcomic`)、封面地址返回的不是图片(`manwaba`,该站自己返回加密字节,带不带 Referer 都一样)。

**巡检里唯一由客户端引起的一个,已修**:见上面「协议相对 URL」那条。

这个仓库本身**不是主题插件** —— 它是完整的 Android 客户端,主题是编译进代码的(`VeneraTheme.kt`、`MiuixMotion.kt`、`VeneraLiquidGlassNavBar.kt`),随包带的是上面那 34 个标准格式的 `.js` 源,我们的 Core 能直接吃。


## 未验证

- **真实 Cloudflare 挑战**:没有遇到可复现的挑战页。18comic.vip 对命令行返回 403,但 WebView2 能直接打开且**不设置 `cf_clearance`**,因此按现在的完成规则无法判定通过;这类站点是否能在验证后放行,需要再找样本。
- **macOS 与 Linux**:交叉编译与 `go vet` 都过(`GOOS=darwin GOARCH=arm64`、`GOOS=linux GOARCH=amd64`,2026-10-31 复查),但**没有运行**。能编译不等于能开窗:窗口那一层要各自平台的系统库,真实运行仍需在对应机器上做。
- **视频源**:Core 支持,桌面端还没接上;`browser.Sniffer` 目前明确返回不支持。
- **制作台**:未开始,设计见 [ARCHITECTURE.md](ARCHITECTURE.md)。
- **源本身失效**:vensrc 的 `wnacg` 1.4.1 安装失败,因为它依赖的两个发布页现在只留「邮件获取地址」,不再列出漫画域名,`init` 因此抛错。官方客户端同样把 `init` 的错误当成安装失败,所以这不是客户端的偏差。

## 验证工具

两个开发用程序,不进发行包:

```sh
# 真实源跑完安装→搜索→详情→章节→取图,并核对阅读器网页的两个接口
NK_LIVE_JS=/path/to/manga_dex.js NK_LIVE_QUERY="one piece" \
  go test -run TestLiveSource -v ./internal/library

# 打开阅读器窗口,读出网页状态(图片是否解码)并保存窗口截图
go run ./cmd/readercheck -script /path/to/manga_dex.js -query "one piece" -out reader.png
go run ./cmd/readercheck -script ... -query ... -layout   # 双页 + 右起排版
go run ./cmd/readercheck -script ... -query ... -scan     # 逐章页数,找有图的章节

# 打开主窗口做一次真实搜索并保存截图;加 -open 会先打开第一个结果
# -favourite 先把第一个结果收藏,用来核对结果网格的「已在书架」标记
go run ./cmd/uicheck -script /path/to/manga_dex.js -query "one piece" -open -out window.png

# 人机验证:窗口本身,以及 Core 的整条 403→验证→重试 链路(本地伪造站点)
go run ./cmd/verifycheck -local
go run ./cmd/verifycheck -e2e
```

`NKDEBUG=1` 会打印封面下载与解码的日志。

## 下一步

1. 找可复现的真实 Cloudflare 挑战(需要站点当前对命令行回 403 且会下发 `cf_clearance`),用它核对完成规则;目前 18comic.vip 这类「WebView 能开但不发 `cf_clearance`」的站点,验证后仍无法判定通过,可能要放宽完成规则,这属于设计决定。
2. 在 macOS 与 Linux 上编译并运行。
3. 接上视频源与嗅探 —— 三个选项各自的工作量与代价已量在 [VIDEO-PLAYBACK.md](VIDEO-PLAYBACK.md):WebView + `<video>`(纯 Go,复用阅读器那套代理)、原生播放器(需 cgo 与自写解码,MyGo 只带 GIF 解码器)、只嗅探直链交给系统播放器(请求头带不过去)。
4. 制作台模块。
