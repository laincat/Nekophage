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
gofmt -l .          # 应无输出
go vet ./...
go test ./...       # 含端到端测试,不访问外网:用 httptest
./scripts/check-docs.sh
```

真实源与界面的核对(需要网络与桌面会话,可选):

```sh
NK_LIVE_JS=/path/to/manga_dex.js NK_LIVE_QUERY="one piece" go test -run TestLiveSource -v ./internal/library
go run ./cmd/readercheck -script ... -query ... -out reader.png        # 阅读器
go run ./cmd/readercheck -script ... -query ... -layout               # 双页/右起
go run ./cmd/uicheck -script ... -query ... -open -out window.png
go run ./cmd/verifycheck -e2e
NK_LIVE_JS=... NK_LIVE_QUERY=... go test -run TestLiveSource -v ./internal/library   # 整条主线   # 人机验证:403→验证窗口→重试,本地伪造站点
```

交叉编译检查:

```sh
CGO_ENABLED=0 GOOS=linux   go build ./...
CGO_ENABLED=0 GOOS=windows go build ./...
CGO_ENABLED=0 GOOS=darwin  go build ./...
```

需要把 Go 的临时目录和缓存放到别处时,设置 `GOTMPDIR` 和 `GOCACHE`。
