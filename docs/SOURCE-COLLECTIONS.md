# 统一源合集订阅

[English](SOURCE-COLLECTIONS.en.md) | 简体中文

桌面端复用 Core ccb6268 的合集解析和平台/能力匹配，客户端负责无 Cookie 下载、取消和资料目录内的快照。一个来源对应一个跨端合集；订阅跟踪目录，不自动下载、安装或执行其中的插件、主题、模块和服务。

## 程序入口

以下命令使用实际 Nekophage 可执行文件，输出 JSON 后退出，不打开主窗口。`follow` 参数为合集 URL；`refresh` 和 `unfollow` 参数为合集声明的 ID。

```sh
Nekophage -data-dir ./profile -collection follow https://example.org/source.json
Nekophage -data-dir ./profile -collection list
Nekophage -data-dir ./profile -collection refresh publisher.collection
Nekophage -data-dir ./profile -collection unfollow publisher.collection
```

`list` 从本地快照读取，可离线使用；重新启动进程后也保留订阅。所有条目都保留，`compatible` 和 `missing` 由 Core 给出，未支持平台、未知能力和设计条目不会从目录中消失。macOS 的平台值映射为 `macos`；桌面当前声明 `comic-source.js`、`video-source.js` 与 `service.js`。兼容不等于已获授权或已经安装。

## 存储与请求行为

数据保存在当前资料目录的 `collections.json`，与 `plugins.json` 分开；取消订阅不删除已安装源。每份下载上限4 MiB，所有持久快照合计上限32 MiB。使用现有原子写入，写盘失败时内存与旧文件保持一致。保存不对描述做HTML转义，避免有效文档在重启后因为转义膨胀超过下载上限而无法恢复。

刷新沿用原URL和合集ID。服务器返回不同ID、无效JSON、超限内容或HTTP错误时保留旧快照；其他URL不能静默占用已有ID。损坏或未来版本的存储文件返回错误，不自动清空。下载使用既有无Cookie传输，不读取源登录状态。

一个Manager内的新刷新会取消同URL的旧请求；取消订阅和关闭Manager也会取消所属请求。迟到响应不能覆盖新快照或重新建立已取消的订阅。命令路径绑定中断信号到请求context；系统Ctrl+C手势本轮未单独验收。

当前存储与其他桌面资料一样按单进程所有权使用。同一资料目录的命令应串行执行，不承诺多个独立进程并发写入的合并；不同资料目录各自隔离。

## 验证与后续

测试使用真实Core解析器、客户端Transport和loopback HTTP，覆盖快照重开、平台/能力匹配、保持已安装记录、资料隔离、失败保留、写盘失败重试、连接取消与替代、取消订阅防恢复、损坏快照拒绝及HTML描述往返。实际构建的桌面程序另用统一源工程的7条目合集做跨进程命令验证。


## 条目安装

`-collection install <合集ID> <条目ID>` 安装一个条目，先按平台与能力判定兼容，不兼容直接拒绝并列出缺失能力。

- `comic-source.js`：下载文本，按条目声明的字节数与 SHA-256 校验后交给源库安装；脚本注册的 key 与条目声明不一致时立即卸载回滚。
- `service.js`：下载 zip 服务包，校验摘要后解压到 `<data-dir>/services/<ID>/`，安装前先加载一次，语法错误或清单非法都会在落盘前拒绝。同名服务整体替换，不会留下旧版本文件。安装记录写在 `<data-dir>/services.json`。
- `video-source.js` 与其他类型仍明确拒绝。

服务包由 `-service` 命令使用：

```sh
Nekophage -data-dir ./profile -service list
Nekophage -data-dir ./profile -service info demo-service
Nekophage -data-dir ./profile -service call demo-service ping '{"a":1}'
Nekophage -data-dir ./profile -service remove demo-service
```

`call` 的第一个参数既可以是已安装服务 ID，也可以是磁盘上的包目录。命令行的宿主授予 `network` 与 `storage`，存储按服务分别落在 `<data-dir>/services/<ID>.json`；`secrets` 只有在用户为该服务存过账号值之后才授予：

```sh
Nekophage -data-dir ./profile -service account-set demo-service user alice
Nekophage -data-dir ./profile -service account-list demo-service   # 只列键
Nekophage -data-dir ./profile -service account-remove demo-service user
```

账号值保存在 `services/<ID>/accounts.json`，权限 0600，**未加密**：桌面还没有平台密钥库，与其假装有，不如写清楚。`account-list` 与 `-service list` 只暴露键名与数量，不打印值。服务契约见 Core 的 `docs/SERVICE-PLUGINS.md`。

本轮不新增目录GUI，也没有图形界面的服务入口或账号流程。现有源安装路径保持独立；后续接入目录GUI时复用本Manager，并在后台调用网络和解析操作。执行结果与具体日志见[STATUS](STATUS.md)。
