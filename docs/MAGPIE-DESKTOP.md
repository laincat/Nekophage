# 桌面端接 Magpie:评估

[English](MAGPIE-DESKTOP.en.md) | 简体中文

日期:2026-10-31。**本文只做评估,不含任何代码改动。** 「Magpie」指 Magpie 插件系统(供应商插件),与漫画源插件是两套东西;桌面端目前一个都没接。

## 结论

**不建议现在接。** 桌面端当前的产品定位是阅读器,而接 Magpie 的成本主要在「凭据落盘 + 对话界面 + 包的安装/授权流程」这三块,与阅读功能没有交集。若确实要在桌面端用 AI,建议走**路线 B(只支持自写模板包)**,把范围压在 OpenAI 兼容与 Anthropic Messages 两种方言上;路线 C(对齐 iOS 全部能力)成本约为 B 的两倍,而多出来的是社区包支持——社区包里目前能跑的只有两个。

## 现在有什么

| 能力 | 状态 |
|---|---|
| 包加载、相对模块解析、拒绝 Node 依赖 | Core 已有,桌面端编译进来即可用 |
| 授权(设备码/密钥)、刷新、用量、签到、选项 | Core 已有 |
| 三种入站方言(chat/responses/messages)、流式分块、取消、重试规则 | Core 已有 |
| 源的 cookie 分区分储、传输层 | 桌面端已有(`internal/cookies`、`internal/core`) |
| 包存储、Secret 存储、对话界面、安装入口 | **桌面端完全没有** |

Core 的入口很小:`magpieplugin.Load(bundle)` 得到 `Loader`,`Loader.NewInstance(opts)` 得到每个账号独立的 `Instance`,再调 `Authorize`/`Poll`/`Chat`/`Usage`。宿主只要实现两个接口——`Host.Do(ctx, req)`(发一次请求)和 `AuthStore.Get/Set`(按供应商存取凭据)。iOS 那份 `mobile/magpie.go` 之所以有 ~500 行,是因为要跨 gomobile 边界做 JSON 编解码;**桌面端直接调 Go API,这层不需要**。

## 要改哪些文件

| 文件 | 动作 | 内容 |
|---|---|---|
| `internal/magpie/providers.go` | 新增 | 供应商记录、包目录扫描、启用/停用 |
| `internal/magpie/package.go` | 新增 | zip/dir 读入 → `Bundle` → `Loader` 缓存(解包安全限制用 Core 的) |
| `internal/magpie/runner.go` | 新增 | `Instance` 生命周期、`Chat` 的 channel → UI 事件、取消 |
| `internal/magpie/signin.go` | 新增 | 设备码两阶段与密钥直登,状态机与 iOS 对齐 |
| `internal/magpie/usage.go` | 新增 | 用量归一(`null` 表示未报告,不是 0) |
| `internal/magpie/options.go` | 新增 | `provider|option` 键的开关,未改动的沿用包内默认 |
| `internal/magpie/auth_windows.go` | 新增 | DPAPI 加密后落盘 |
| `internal/magpie/auth_unix.go` | 新增 | 0600 文件;Linux 有 secret-service 时优先用它 |
| `internal/ui/magpie.go` | 新增 | 供应商列表、安装入口、用量与选项面板 |
| `internal/ui/magpie_chat.go` | 新增 | 对话窗口(MyGo 有 `TextInput` 与 `Scroll`) |
| `internal/ui/app.go` | 修改 | 打开 Magpie 窗口的入口、窗口生命周期 |
| `main.go` | 修改 | 装配 store/auth/host,与 `internal/core` 共用传输 |
| `docs/STATUS.md`、`README.md` | 修改 | 记录接与不接的结论 |

## 工作量

| 部分 | 估算 | 依据 |
|---|---|---|
| 逻辑层(providers/package/runner/signin/usage/options) | 700–900 行 Go | iOS 同名模块共约 930 行 Swift,去掉跨语言编解码 |
| 凭据落盘(含平台分支) | 150–250 行 | 两套平台实现 + 迁移与降级说明 |
| 界面(列表、安装、登录、对话、用量) | 900–1200 行 MyGo | iOS 侧五个视图共约 785 行 SwiftUI |
| 测试 | 400–600 行 | 包加载拒绝、授权状态机、流式与取消、凭据存取 |

合计约 **2200–3000 行**,不含文档。Core 侧不需要改动。

## 三条路线与取舍

| 路线 | 范围 | 代价 |
|---|---|---|
| A 不接 | 桌面端保持纯阅读器 | 用户在桌面端没有 AI;零维护成本 |
| B 只支持模板包 | OpenAI 兼容 + Anthropic Messages,用户装自写或第三方同类包 | 覆盖率有限;但方言由包自己声明,不写死在宿主里 |
| C 对齐 iOS | 含社区包与能力检测 | 成本近两倍,而社区包多数仍因 Node 依赖无法运行 |

## 风险

| 风险 | 说明 |
|---|---|
| 凭据落盘 | Windows 有 DPAPI;Linux 无 secret-service 时只能靠 0600 权限,必须在界面上说明 |
| 流式 UI | MyGo 是即时模式,需要把 `Chat` 的事件拉到 UI 线程;iOS 的「不可推、只能拉」经验可直接用 |
| 社区包预期 | 用户会期待「装上就能用」,而实际多数包在加载阶段就被拒绝;错误必须指名缺哪个模块 |
| 许可证 | Magpie 本体 MIT,但第三方包各自许可不同;不内置则继承风险最低 |
| 维护面 | 桌面端多一套与 iOS 平行的 UI 与状态机,任何 Magpie 行为变化都要改两处 |

## 建议

先在**路线 A 与 B 之间**选择,而不是直接讨论 C:

- 若桌面端短期没有 AI 需求 —— 选 A,把 `docs/OPEN-QUESTIONS.md` 里的结论写成「桌面端不接」,以后再说;
- 若希望桌面端能用自己的中转站 —— 选 B,先做 `providers.go` + `runner.go` + 一个对话窗口,用两个模板包跑通「安装 → 密钥登录 → 流式回复」,再补用量与选项。

无论选哪条,**参考包是否随客户端内置**都要单独定(见 Core 的 `docs/OPEN-QUESTIONS.md` 第 4 条):内置会让安装后立刻可用,也要承担随版本分发第三方代码的责任。
