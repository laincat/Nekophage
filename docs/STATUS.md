# Nekophage 状态

[English](STATUS.en.md) | 简体中文

更新:2026-10-30。下列结论来自 **Windows** 本机实测,包括真实漫画源与真实网络。

## 已实现

| 部分 | 内容 |
|---|---|
| 装配 | Core 编译进同一进程;`internal/core` 提供运行时、注册表与漫画数据接口。2026-10-31 用**最新的 Core**(含 `magpieplugin`、`host/sse`、`mobile` 的 Magpie 接口)重新装配并全量跑过:`gofmt` 干净、`go vet ./...`、`go test ./...` 全过 —— 桌面端不接 AI,但 Core 新增的包不能把桌面端带坏。`build/core` 与 Core 工作树**逐文件一致**(`diff -rq` 无差异,不含 `.git`/`.github`/`apple`)。当前工作树 = `f798138` + 一个尚未推送的 `magpieplugin` 修复(`client.auth.get` 之前会忽略 provider id),所以「用的是哪一版 Core」由 `diff` 回答,不由记忆回答 |
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
| 人机验证 | 独立窗口加载挑战页,按 Core 的完成规则判断,通过后把 Cookie 与 User-Agent 交回并自动重试 |

## 已验证(真实源、真实网络)

| 源 | 结果 |
|---|---|
| 拷贝漫画 `copy_manga` v1.4.3 | 搜索「海贼王」21 部、2 页;详情 401 章、4 组标签;章节 209 张图;取图 339,596 字节 `image/webp`;封面 108,565 字节 `image/jpeg` |
| MangaDex `manga_dex` v1.2.0 | 搜索「one piece」20 部、2 页;详情 962 章、3 组标签;章节 53 张图;取图 122,856 字节 `image/jpeg`;封面 56,745 字节 `image/jpeg` |
| Comick `comick` v1.x | 搜索「one piece」：章节 51 张图、取图 37,912 字节、封面 44,668 字节(均为 WebP);书架记下 `page 3, 42%` |
| Komiic `komiic` v1.x | 分类浏览(分页型)第 1 页与第 2 页内容不同,封面与作者正常,翻页生效;端到端:章节 67 张图、取图 248,508 字节、封面 27,252 字节;书架记下 `page 3, 42%` |

Comick 顺带暴露了真实源测试自己的一个缺陷:测试直接拼 `...&chapter=<章节 id>`,而它的章节 id 本身带斜杠,拼出来的请求行变成非法 HTTP 版本,`httptest.NewRequest` 直接 panic。现在四个 reader 请求(`/chapter`、`/progress`、`/keep`、`/switch`)都改用 `url.Values` 编码,`/switch` 也改为解析返回的 JSON 再比对 `chapter` 参数,而不是在原始响应体里找子串。

Komiic 顺带暴露了两处服务端标注问题:页面图实际是 WebP,响应头却写 `image/jpeg`;封面响应头写 `application/octet-stream`。真实源测试因此改为按文件头判断图片类型(JPEG/PNG/GIF/WebP/HEIF/AVIF/BMP),不再信任 `Content-Type`——原来的写法会把一个能正常显示的 WebP 判成「不是图片」。WebP 的封面由 MyGo 的 `ui.DecodeBitmap` 解码(它已经引入 `x/image/webp`),阅读器里的页面图交给 WebView,两者都不受影响。

阅读器的两个接口按网页真实调用的方式各跑了一遍:拷贝漫画 `/chapter` 返回 209 张、`/img` 返回 339,596 字节;MangaDex 53 张、122,856 字节。

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

备份导入用三个 `.venera` 各自验证:`backup-test`(源 1、数据 2、搜索历史 2)、`backup-shelf`(历史 1、收藏夹 1 个含 1 部、搜索历史 1)、`backup-pages`(图片收藏 1 条,落盘 2 张:图片收藏与集数行各带一页)。最初用含真实 `copy_manga.js` 的 `.venera` 验证:预览显示源 1 个、带数据 2 个、搜索历史 2 条;导入后 `copy_manga` 以正常流程装成 1.4.3,4 项源数据写回,大整数 `12345678901234567890` 原样保留。

分类浏览在三个源上都通过:MangaDex 显示 Popular 分区与带封面的列表,拷贝漫画显示「推荐」「热门」两个分区,Komiic 的分页型 browse 第 1、2 页内容不同,翻页生效。阅读器窗口确实在 WebView 里运行:页面标题为漫画名,生成 53 个 figure,图片解码成功(第一张 757×1200),`currentSrc` 指向 `nkreader://`(Windows 上是 `http://nkreader.localhost`)的取图地址。重开同一章会回到上次的位置:滚到 1200 后重新加载,`scrollY` 仍是 1200。 Comick 的另一次检查记录了 51 个 figure、51 张图、其中 4 张解码成功,首图 750×1194,`currentSrc` 指向 `http://nkreader.localhost/img?source=comick&…`。

### 人机验证(2026-10-30)

- 窗口本身:本地伪装挑战页(标记消失 + 脚本写入 `cf_clearance`)在 3.5 秒内被判定通过,拿到 WebView 的 User-Agent,`cf_clearance` 写进分区罐。
- 整条链路:本地伪造 Cloudflare 站点对普通请求回 403(`cf-mitigated: challenge` + 挑战标记),Core 检测后开窗口,页面通过后把 Cookie 与 User-Agent 交回,重试带 `cf_clearance=clear` 拿到 `200:PASSED-OK`,且 Cookie 落在该源自己的 `comic:c` 分区。全程 3.4 秒。

## 本轮发现并修掉的问题

1. **版本号把真实源挡在外面**:应用原来报 0.1.0,而源要求 `minAppVersion` 1.3.4 或 1.6.0,源会直接拒绝安装。现在区分「应用版本」与「源接口版本」,后者报 1.6.0。
2. **Core 没有搜索选项**:`search.load(keyword, options, page)` 需要每个选项组一个值,缺了会变成 `undefined` 拼进 URL。Core 新增 `search.optionList` 读取(`js.Runtime.SearchOptions`、`mobile.SearchOptions`),桌面端按默认值传参。MangaDex 因此从协议错误变成正常返回 20 部。
3. **URL 里的空格**:MangaDex 的源把关键词直接拼进 `?title=one piece`,Go 会原样写出,服务端回 `PROTOCOL_ERROR`。现在发送前转义非法字节(Dart 的 `Uri` 也这么做)。
4. **封面永远是灰块**:首帧渲染时封面还没下载完,解码缓存把「没有图片」记成了最终结果。现在只缓存解码成功的结果和失败的下载。
5. **源列表陈旧**:`Library.Sources()` 原来读自己的缓存,从其他路径安装的源不显示;现在每次都向 Core 要。

## 未验证

- **真实 Cloudflare 挑战**:没有遇到可复现的挑战页。18comic.vip 对命令行返回 403,但 WebView2 能直接打开且**不设置 `cf_clearance`**,因此按现在的完成规则无法判定通过;这类站点是否能在验证后放行,需要再找样本。
- **macOS 与 Linux**:只做交叉编译,没有运行。
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
3. 接上视频源与嗅探。
4. 制作台模块。
