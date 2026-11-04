# 已装源源码查看

[English](source-code-viewer.en.md) | 简体中文

源管理里的每个已安装 JavaScript 源和 Kazumi 规则多了一个「查看源码」。它用 MyGo v0.7.3 自带的 `ui/editor` 打开只读副本：JavaScript 使用 JavaScript 语言配置，Kazumi 规则使用 JSON。不需要升级 MyGo，也没有新增依赖。

这不是编辑器：

- 文本只能选择、复制、查找和折叠，不能修改。`ReadOnly` 同时挡住键盘、粘贴和无障碍的写操作。
- 关闭窗口即释放这次查看的编辑器状态；迟到完成的后台读取会被丢弃。
- 不会重新执行、安装或保存源。重新打开会读取当前已安装记录，因此更新后的源会显示新正文。

实现位于 `internal/ui/sourcecode.go`，入口是 `internal/ui/manage.go` 的「查看源码」。`internal/ui/app.go` 持有当前查看器，避免源管理窗口每帧重建编辑器。后台读取通过 `Window.Update` 回到界面线程。

已知限制：

- 快照在窗口打开时固定，同一次打开期间不会跟随外部更新。
- 关闭源管理后，源码窗口仍可保留；两者各自关闭。
- 原Mac开发构建结果保留其旧Core归属；Windows当前组合的原生窗口验证见下文。系统剪贴板、真实操作系统输入法组合取消和安装版源码查看未验。

## Windows原生窗口验证（2026-11-04）

Lead在2b38049/Core313951e的隔离e2e构建中，通过MyGo自带自动化连接实际Windows窗口：漫画JavaScript、视频JavaScript和Kazumi JSON正文均与已安装记录逐字一致，显式setValue、文本输入和Backspace不能修改只读正文。切换源关闭旧查看器；关闭源管理后源码窗口保持，源管理重开可复用同一查看器。

实际从源管理URL安装入口将漫画源1.0.0更新为2.0.0，仅一次loopback请求；已打开窗口保持1.0.0，关闭重开显示2.0.0。通过原生WM_CLOSE关闭本任务窗口，两次独立进程均退出0；第二进程恢复2.0.0正文，其他源正文/类型/启用状态保留。日志及截图 `E:/Temp/nk-source-viewer-windows/`。这是e2e测试构建，发布构建不引入自动化插件。未验证系统剪贴板、真实IME或新Core组合的macOS窗口；Orca Worker旧Mac结果仍归此前ccb候选。
