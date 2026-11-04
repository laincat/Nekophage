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
- 只验证了本机 Mac 开发构建和隔离测试源；没有验证安装包、Windows 或真实操作系统输入法的组合取消。
