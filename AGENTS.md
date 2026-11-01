# Nekophage 开发约定

[English](AGENTS.en.md) | 简体中文

本文件写给参与开发的人和 AI 助手。修改代码前请先阅读。

## 原则

- 业务逻辑放在 VeneraCat Core。桌面端只负责界面、网络、WebView、播放器和存储;需要新的逻辑时先改 Core。
- Core 是本仓库的依赖,不是副本:`build/core` 由 `scripts/fetch-core.sh` 拉取,版本写在 `scripts/core-version`。不要在 `build/` 里改代码。
- 主界面是原生 UI(`ui` 包)。只有阅读器、人机验证和以后的播放/嗅探用 WebView,而且要嵌在原生界面里。
- 慢操作永远不在界面线程上做:goroutine 里完成,再通过 `Window.Update` 回到界面。
- 每个源的 Cookie、登录状态和源数据互相隔离:分区名是 `comic:<key>`,不使用系统共享 Cookie 存储。
- 源不可信:不把源的 HTML 加载进主界面,只进阅读器和验证窗口。
- JSON 保持原文:源数据的值是 Core 给的字节,不重新编码;章节等需要顺序的对象用文档顺序解析,不排序。
- 不要从 AGPL 项目(Animeko、AnimeFlow)或没有许可证的项目(AniCh)复制代码。参考 Magpie(MIT)、MyGo(MIT)时按行为重写。

## 仓库约定

- 根目录只放工具链或惯例要求必须在根目录的文件。
- 所有 `.md` 都提供中英双语:`X.md` 为中文,`X.en.md` 为英文,顶部互相链接。修改其中一份时同步修改另一份。`scripts/check-docs.sh` 会检查配对和根目录。
- 不是每个子目录都需要 README;只有根与 `docs/` 下的文档需要成对。

## 检查

在仓库根目录运行:

```sh
sh scripts/check.sh   # gofmt、真实 Core pin、go vet、go test(带 race)、docs、partitions、capabilities,逐项打印
```

单步等价于:

```sh
gofmt -l .          # 应无输出
sh scripts/check-core-pin.sh  # 实际模块 checkout、干净工作树与生成戳记必须匹配 pin
go vet ./...
go test ./...       # 含端到端测试,不访问外网:用 httptest
sh scripts/check-docs.sh
sh scripts/check-partitions.sh
```

`check-partitions.sh` 守的是分区名: `"comic:<key>"` 只允许出现在 `core.ComicPartition`。它来自一个真实的 bug —— 恢复备份 cookie 时写进了裸源 key 分区,而所有请求读的是 `comic:<key>`,cookie 落在没人读的地方,导入摘要却显示「恢复了 N 条」。**当时的测试断言的是同一个错误假设**,所以全绿。这类错误不是「没覆盖到」,而是**用例在帮我们确认误解**:唯一可靠的查法是对照代码之外的事实源(传输层真正用的 key),并让错写不成立(传源 key、内部加前缀),而不是靠「记得用助手」。

## 真实源与界面的核对(需要网络与桌面会话,可选)

```sh
NK_LIVE_JS=/path/to/manga_dex.js NK_LIVE_QUERY="one piece" go test -run TestLiveSource -v ./internal/library
go run ./cmd/readercheck -script ... -query ... -out reader.png        # 阅读器
go run ./cmd/readercheck -script ... -query ... -layout               # 双页/右起
go run ./cmd/uicheck -script ... -query ... -open -out build/uicheck.png   # 不写 -out 就不落盘
go run ./cmd/verifycheck -e2e
```

需要真窗口的检查一次跑完(三项按环境变量参与):

```sh
NK_OFFICIAL_SRC=/path/to/official-srcs NK_PROBE_SRC=/path/to/source-with-comments.js NK_READ_SRC=/path/to/source.js sh scripts/desktop-probes.sh
```

单独跑:

```sh
sh scripts/reader-probe.sh          # 需要 NK_READ_SRC:真章节真解码,decoded>0 才算过
sh scripts/comment-probe.sh /path/to/source.js   # 评论线程;向窗口要 count/author/body,不比截图
sh scripts/favorites-probe.sh /path/to/source.js # 源自己的收藏;同样问内容,不问截图
```

两处判据是踩出来的,别改回去:**评论**不能按截图大小判断(空线程与真线程只差几百字节,曾因此对不声明评论的源误报通过);**阅读器**必须要求 `decoded>0`(源可以返回看着合理的字节数而那并不是图片)。两者都做过负向验证。

## 交叉编译

```sh
CGO_ENABLED=0 GOOS=linux   go build ./...
CGO_ENABLED=0 GOOS=windows go build ./...
CGO_ENABLED=0 GOOS=darwin  go build ./...
```

交叉编译通过只说明**能编**:窗口层要 GTK/WebKit 运行时,编出来的二进制没有在 Linux 上启动过,不能写成「Linux 上可用」。

需要把 Go 的临时目录和缓存放到别处时,设置 `GOTMPDIR` 和 `GOCACHE`。

## 提交前跑检查,别接管道

`bash scripts/check-docs.sh | tail -1 && git commit ...` 看起来在把关,其实**退出码是 `tail` 的**:检查失败也会照常提交。这次就是这样把根目录多出的文件提交进去的。要么直接跑脚本,要么用 `set -o pipefail`,或者先跑后提交两步分开。
