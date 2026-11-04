# Nekophage 状态

[English](STATUS.en.md) | 简体中文

## Windows 0.2.9 新Core包与安装版媒体（2026-11-04）

fe101db/Core ccb6268已打包为0.2.9。七项工程、ZIP CRC/逐文件摘要/版本、NSIS和实际隔离安装通过。使用旧测试资料副本，仅更新夹具端点及源摘要，原生搜索重新认证后从继续观看实际看到English与既有3.030709秒画面，视频裁剪逐像素一致；清单鉴权200且无Range，媒体和英文VTT鉴权206。UIA没有改变字幕选择，随后前台关闭字幕操作被USER_INPUT_ACTIVE拒绝且未发送，故不计新的切换/关闭/跨进程关闭恢复。结束仅清理本任务进程，不算正常关窗；卸载及隔离/原始资料摘要保留通过。

交付[0.2.9包与回执](../../deliveries/2026-11-04-nekophage-029/README.md)，ZIP21,567,624字节，安装器18,720,179字节。日志 `E:/Temp/nk-desktop-029/`。包未签名、未公开发布，Mac证据继续归4d947098非Markdown源码范围；本轮夹具无音轨，不证明扬声器。

## Core ccb6268 桌面主线整合（2026-11-04）

桌面main已快进到4d947098，功能pin为ccb626825bc116dad3b3e9a082867fd22c09caf4。Windows Lead完成七项工程检查、真实WebView阅读2/2页及scrollY1200恢复、HLS鉴权/Range/2.011秒续播、18718音频解码字节、原生MP4 Range206，以及候选安装版主窗启动、原生关闭、进程退出、卸载和资料保留。安装媒体未重跑，包未签名，公开目的地仍待指定。

Orca Worker mac-validation在干净c545212加同一5文件补丁上验证205个非Markdown文件指纹与Windows候选一致，不宣称完整Git树一致。macOS27.0.1/arm64、Go1.27.3以GOMAXPROCS=2/GOFLAGS=-p=2完成7/7工程检查、2张320×1200页图解码、scrollY1200恢复、HLS.js320×180、鉴权/Range、2.063912541秒落盘及2.064秒重开。原日志仅在Mac `/tmp/nk-desktop-ccb6268-evidence/`，本地回执 `E:/Temp/nk-core-source-release/mac-consumer-receipt.json` 不是原日志副本。Mac未验安装版、正式包、跨进程、音频或原生MP4；开发构建commit字段为unknown。

## Core ccb6268 候选消费（未整合主线，2026-11-04）

本分支候选1fbd2647使用Core ccb626825bc116dad3b3e9a082867fd22c09caf4，正式客户端主线和0.2.8交付包仍为3be57b5。Windows七项工程检查通过；现有readercheck在隔离资料中实际解码2/2页，重开恢复scrollY=1200。uicheck通过带Referer/Cookie鉴权的HLS/Range请求、320×180解码和页面进度落盘/重开2.011秒；音频解码18718字节，不代表扬声器。videocheck原生MP4播放、进度报告及1024字节Range206通过。这些是开发WebView路径，不是新安装包或Mac验收。Core Apple CI尚未有终态，日志 `E:/Temp/nk-core-source-release/`。

## Windows 0.2.8 本地交付（2026-11-04）

已构建包含bfc3484图片转换失败回退修复的0.2.8/1de7f996，Core仍3be57b5。Lead在Windows完成scripts/check.sh七项检查、ZIP CRC/逐文件摘要/版本、NSIS构建及实际隔离安装。安装版主窗口和控件可见，原生关闭按钮经UIA调用后窗口消失，进程已退出；没有捕获退出码。随后卸载移除安装目录，隔离资料的全部已有文件摘要保持不变。

ZIP为21,567,833字节，安装器18,722,840字节；摘要、截图和回执在[交付目录](../../deliveries/2026-11-04-nekophage-028/README.md)，日志 `E:/Temp/nk-desktop-028/`。本轮未重跑包装阅读/媒体/字幕/跨进程续播，不提升旧包或Mac证据。包未签名、未公开发布，公开目的地仍待指定。

## 图片响应/像素处理失败的镜像契约(2026-11-04)

对照[上游图片加载流程](https://github.com/venera-app/venera/blob/a0eba914f4c2a84ac1bc925adec2baabe920b9be/lib/network/images.dart)的同一重试范围,onResponse异常、非字节返回值和像素脚本异常都应交给onLoadFailed。旧桌面适配器在这些失败路径提前返回,备用地址没有被请求。真实Core脚本+loopback HTTP负向用例复现三项失败,无镜像时保留原错误的对照通过。

现在把单次响应转换独立成applyImageTransforms,错误统一回到既有有界镜像循环,保留配置完整替换、无镜像原错误和未知原生图片格式直通策略。四种情况及已有镜像/像素用例均过Windows race回归,完整scripts/check.sh七项通过,日志 `E:/Temp/nk-image-error-contract/full-check.log`。日志 `E:/Temp/nk-image-error-contract/before.log`、`after.log`;参考源码的精确提交/blob记录在 `E:/Temp/nk-image-error-contract/reference.json`。

Mac上的darwin/arm64定向回归也已完成:Worker在`c545212`加`0db2ead`两份受测文件的小型隔离工作区上,核验补丁和结果文件SHA后,运行`go test -race ./internal/core -run '^Test(ImageTransformFailuresFollowMirrors|ImageFallback|ModifyImage)' -count=1 -v`(7个顶层、11个叶子用例)、`go build ./...`与`scripts/check-core-pin.sh`,结果均通过,Core实际为3be57b5。原始日志在Mac `/tmp/nk-desktop-mirror-bfc-evidence/`,Windows回执 `E:/Temp/nk-image-error-contract/mac-receipt.json` 是Worker回传记录。

这不是完整`bfc3484`树、GUI、媒体播放、包装或iOS验收;Windows与Mac都只证明这两个受测文件及其受限Core/HTTP/race路径。临时传输gist `b697850a8e126c1f659d070755ba21c9` 在Mac核验后已由Lead删除,认证重读404,日志 `E:/Temp/nk-image-error-contract/gist-cleanup.log`。该轮测试时交付包仍对应旧`0db2ead`；后续0.2.8/1de7f996已包含本修复，包装验收见本页顶部。

这里只核对公开源码契约并实际运行本地Core/HTTP,没有运行官方App。本项是客户端底层适配修复,Core pin仍3be57b5;此前0db2ead交付包不包含它,尚未新增Mac或包装UI结果。

## 图片镜像回退配置修复(2026-11-04)

桌面图片请求在 `onLoadFailed` 返回备用配置后,旧逻辑只替换部分字段:省略method/data的备用地址会收到原POST与正文,而modifyImage始终沿用旧值。真实Core源脚本和loopback HTTP已复现:应为GET/空正文的请求仍是POST/original-request-body;应保持4×2的图被旧脚本旋转为2×4,新镜像的旋转则被忽略。

现按新的ImageLoadingConfig完整替换请求,method缺省GET,正文和像素脚本可清空或替换,新onResponse仍在像素处理之前执行。两项回归与原像素处理用例通过race测试,完整 `bash scripts/check.sh` 七项也全部通过,日志 `E:/Temp/nk-image-fallback/full-check.log`;旧实现负向日志 `E:/Temp/nk-image-fallback/before.log`,修复后 `after.log`。这是Lead在Windows执行的真实Core/HTTP/PNG解码验证,不代表Mac或GUI重新验收。Core功能pin仍为3be57b5;旧c545212包不包含本次修复。修复提交0db2ead4381ad67c5e3075daaf261b76909e12bf已生成Windows ZIP与安装器,位于 `E:/Code/deliveries/2026-11-04-image-mirror-fix/README.md`;ZIP摘要/CRC/版本及隔离安装/卸载/资料保留均通过,新包GUI/媒体/Mac尚未复验。

随后Lead在干净42b9465(功能修复0db2ead)/Core3be运行现有 `readercheck`:真实源脚本先发POST/旧正文得到503,再以GET/空正文且无旧请求头取备用地址。备用响应经onResponse反转和modifyImage旋转后,Windows WebView2实际解码1/1页,尺寸96×64→64×96。日志 `E:/Temp/nk-image-fallback/reader-688d774f/verify.log`,同目录receipt.json记录HTTP序列和二进制身份。这是开发阅读器路径,不计包内GUI、Mac或截图视觉通过。

随后Orca Worker在Laincat-Mini独立临时检出c545212上加上述两个文件(非完整0db2ead文件树),补丁5279字节/SHA26db4295…及两文件SHA与Lead逐字节一致。Go1.27.3 darwin/arm64上 `go test -race ./internal/core` 的镜像回退与5条像素处理用例全部通过,`go build ./...`、Core pin3be57b5检查也通过;原c545检出未改动。日志 `/tmp/nk-desktop-mirror-0db-evidence/`,Lead回执 `E:/Temp/nk-image-fallback/mac-desktop-mirror-receipt.json`。该轮未开窗口、未跑媒体矩阵与完整套件,故Mac GUI与打包媒体仍按旧提交证据单列。

Lead随后实际安装0db2ead包,从隔离复制的既有English/3.030709秒资料启动,原生搜索建立夹具会话后继续观看。截图与AX都见Subtitle rendering proof,视频区域与旧保存位置截图逐像素一致;清单鉴权200且无Range,媒体及VTT鉴权206。后台写入未改变字幕选择,前台点击随后被USER_INPUT_ACTIVE拒绝且未发送,所以不计新的切换/关闭/跨进程关闭恢复。仅清理时终止本轮进程,不计正常关窗;卸载及测试资料/原始夹具资料哈希保留通过。回执 `E:/Temp/nk-0db-installed-media/run-a7f2d847/receipt.json`,日志 `verify-media.log`。

## Mac包装窗口正常关闭与Core主线复验(2026-11-04)

用户确认Laincat-Mini已解锁后,Orca Worker对已核验c545212/Core3be的0.2.7 arm64包做隔离验收:包SHA bb39921a…、二进制SHA d955c9ab…及ad-hoc严格签名复核通过,以独立 `-data-dir /tmp/nk-c545-close-only/data` 启动PID78539/窗口11481;原生关闭按钮一次AX点击后窗口消失、原进程退出码0,未发终止信号。独立资料目录前后无文件。原始日志 `/tmp/nk-c545-close-only/`,Lead回执 `E:/Temp/nk-c545-mac-transfer/normal-close-accepted.json`。这不代替Gatekeeper/公证、其他机器运行或完整包装媒体流程。

Core单提交主线c0f0dfa的[CI41734985028](https://github.com/laincat/VeneraCat-Core/actions/runs/41734985028)也在该精确SHA完成三平台Go、Apple xcframework及macOS/iOS模拟器Swift冒烟四作业通过,回执 `E:/Temp/core-gbk-differential/main-ci-accepted.json`。本轮Core变动仅测试/文档,两端功能pin仍为3be57b5;不将不同提交的CI执行记录混同。

## Core测试与兼容说明已整合(2026-11-04)

经用户确认,Core main按单初始提交约定更新为c0f0dfa00c8a4daf7ce58fea5b615bc24eab88b7,文件树与通过完整CI41722875239的8922993完全一致,PR #3已关闭。只有测试与文档差异,生产转换实现未变;Nekularity/Nekophage仍使用功能pin3be57b5。保树/远端/两端pin回执 `E:/Temp/core-gbk-differential/main-published.json`。新主线SHA与已有CI受测SHA分别记录,不把旧CI改记为新SHA执行。

## 主线整合 Core 3be57b5(2026-11-04)

**当前主线 macOS arm64 包已构建并核验(2026-11-04)**:Orca Worker 在精确 c545212/3be57b5、工作树干净的检出上运行 `sh scripts/package-macos.sh`,生成 0.2.7/`com.laincat.nekophage`、最低 macOS12.0 的 `.app`,并以 ad-hoc 签名完成归档往返。ZIP 10,425,020 字节,SHA-256 `bb39921af5e62891944584b5ac56bb734cf24d10d1b9456eb9f195972e4b614c`;包内二进制与原始 `.app` 的 SHA-256 均为 `d955c9ab422e1628389a0a5b5db72cabae24a6b16662bd2e186cbf177548fd9a`,权限 0755,plist 一致。原始包与解包后 `codesign --verify --deep --strict` 退出码均为0,签名显示 `Signature=adhoc`、`TeamIdentifier=not set`;包内二进制 `-version` 的提交与 Core 均匹配。回执/日志 `/tmp/nk-c545-acceptance/packages/`。初次预检误用不存在的 `build/core` 路径已按 `go list` 修正,失败日志保留。

该项范围是构建、包内版本、归档与 ad-hoc 签名验证,不是 Gatekeeper、公证或新一次窗口启动;Mac 正常关窗仍受 `screen_locked` 限制。本次 macOS 包未重复打包 pin 的负向对照;此前反例保留各自提交与日志归属。

本次主线整合把 Core pin 更新为3be57b5,保留服务器关闭竞态修复1a83ef2、WebKit原生轨加入后的字幕修复079ef6b、打包pin检查047de37,并带回候选已验证的播放器探针等待页面加载和4a4caed的已保存关闭跨进程探针。整合树的Windows七项检查通过;新进程实际WebView2在1秒解码画面恢复关闭,保持value=-1、无活动轨/cue,页面重开也通过。播放器HTML与已验收97e24b6逐字节相同,证据 `E:/Temp/nk-desktop-core-integration-evidence/`。

当前主线功能提交c545212/3be57b5的Windows安装包已实际用/S /NOREGISTER /D隔离安装:安装后exe与ZIP摘要一致,版本/pin正确,原生主窗实际出现搜索/书架/源管理/AI/视频入口。点击窗口关闭后确认原进程消失;卸载退出0、安装目录删除,独立资料逐文件摘要不变。回执/截图/日志 `E:/Temp/nk-desktop-core-integration-evidence/install-b42482cb/`。本轮只补安装与启动退出,不把较早97e24b6的完整媒体流程移用为本包全流程通过。

当前pin的本地音轨解码补验(2026-11-04):Lead在精确c545212/3be57b5、独立资料的Windows开发WebView2复用uicheck -video-audio。HLS.js实际解码音频18537字节,原生MP4为17976字节,两条命令成功;无音轨MP4与清单虚报AAC的无音轨HLS均为0,命令按预期非零退出。日志/回执 `E:/Temp/nk-c545-audio/`。这是音频解码管线证据,不代表扬声器发声、安装版手工播放或Mac音轨通过。

消费端依据分别保留:Windows97e24b6/Core3be安装版已验英文活动cue→关闭→正常退出→新进程关闭、3.030709秒与画面恢复、鉴权/Range206,回执 `E:/Temp/nk-ca8-package/media97-fe96ad14/receipt.json`。Orca Worker在Mac ca8f370/Core3be通过搜索/详情、两页解码与失败重试、右起双页、页面进度和章节重开1200、视频时钟与Range206;原字幕呈现失败经同一WebKit补丁修复,英文和关闭/页面重开通过,日志 `/tmp/nk-ca8-acceptance/evidence/`。Mac错误stamp拒绝且旧.app不变、正确pin ad-hoc打包也通过。

Mac包装窗口已出现,正常关窗仍被screen_locked拒绝;未借SIGTERM或Windows结果计通过。Mac旧探针因强制活动字幕而失败的缺口现已补验:Orca Worker在精确c545212/Core3be干净检出上通过播放器包race回归(2.018秒),并将此前页面实际保存的off资料按摘要核对复制,未手写选择。独立进程13544、13564的before均恢复关闭、stable=2、width=320、time=1.000、无showing轨/cue,且各自页面重开也通过,两次退出0。日志 `/tmp/nk-c545-acceptance/evidence/`,Lead回执 `E:/Temp/nk-c545-mac-transfer/mac-acceptance.json`。两份窗口截图全白且不含Web层,不算视觉证据;跨进程事实来自两个PID的初始状态,不来自同进程LoadURL。临时源码gist已在确认接收后删除并核对404。阅读重开不等于跨App进程恢复,夹具无音轨。Nekularity main现已整合6eab317/3be57b5,受测组合3574346的CI/IPA及两条签名模拟器AI闭环已分范围核验;公开0.3.2仍为旧pin包。整合回执 `E:/Temp/nk-ios-final-integration/receipt.json`。下列旧版本记录保留原提交与pin。

## 主线字幕修复安装包验收(2026-11-03)

Lead 已构建并实际安装 0.2.7/`bf9cbd9`(Core `4f213ff`)。ZIP 21,566,978 bytes,安装器 18,720,774 bytes;CRC、逐文件摘要、实际版本与安装后 exe 一致性通过,产物和构建日志位于 `E:/Temp/nk-subtitle-package/`。使用此前测试实际保存的英文字幕/3.030709 秒资料的副本,仅调整本地夹具地址,经原生搜索建立夹具会话后继续观看:英文 Subtitle rendering proof 实际可见。通过字幕控件选择关闭后文字消失且 off=true 落盘;正常逐窗退出并启动新进程后,关闭选择仍保留,恢复位置仍为 3.030709,视频区域逐像素一致。

回执、校验和截图见 `E:/Temp/nk-subtitle-package/accept-8acc7abc/receipt.json`、`verify.log`、`restored-english.png`、`off.png`、`reopened-off.png`。两轮应用均正常关闭,本任务夹具停止、临时安装卸载,实际资料文件哈希不变。此结果属于 bf9cbd9/Core 4f213ff 的 Windows 安装版,不代替 Core 3be57b5 候选安装包或 macOS 验收;夹具无音轨,不证明扬声器或真实账号自动登录。

## 修复已保存字幕被 HLS 默认选择覆盖(2026-11-03)

安装版 f7b99d1 曾出现 English 已选但未请求字幕清单。Lead 用既有 subcheck 加预置英文资料在 Core 3be57b5 上复现:视频已解码 320 像素宽,时间为 1 秒,下拉框 English,却无 showing 字幕轨或活动文本。HLS.js 1.7.4 的 switchLevel 先计算默认轨,发出 SUBTITLE_TRACKS_UPDATED 后才应用它;页面在该通知里同步恢复设置,会被覆盖。现将该回调里的恢复动作放到 microtask,待 HLS 完成本次内部选择后应用用户选择。

同条件修复前失败、修复后真实 Windows WebView 显示 Subtitle rendering proof 并重开保留;默认中文切英文、预置英文切关闭并重开也通过。已有 desktop-probes 字幕流程增加 saved-english.json 回归资料,复用 subcheck,并在选择关闭前断言英文活动 cue。证据 `E:/Temp/nk-subtitle-restore-order/`。候选功能提交为 `ca8f370`/Core 3be57b5;相同修复已回移到真正 main 的 `bf9cbd9`/Core 4f213ff,主线七项检查与英文活动 cue→关闭→重开回归也通过,证据 `E:/Temp/nk-subtitle-restore-order/main-runtime/`。这些是开发窗口回归,不是旧 f7b99d1 安装包已修复,也不代表 macOS 已验证;安装版需基于新提交重建复测。

## Core 3be57b5 Windows 阅读补验(2026-11-03)

Lead 在候选 `7c0dac0`(功能代码 `f7b99d1`)、Core `3be57b5` 的全新业务/WebView 资料下运行既有 readercheck,并令本地图片服务首次返回错误。初次两页中解码一页、出现一条重试提示;调用阅读器重试按钮后失败数归零且两页均解码。页面自身写出 55% 进度,仅有本次章节存储键;双页右起时第一页面位于右侧;移动到 1200 后保存并重开恢复 1200。命令 `go run ./cmd/readercheck -script internal/reader/testdata/loopback_source.js -query probe -serve 18731 -fail 1 -retry -progress -layout -scroll` 通过,日志与精确参数见 `E:/Temp/nk-3be-candidates/reader-f06c59ad/reader.log`。这是 Windows 原生开发阅读器、页面内控制操作和同进程重开;不是安装版阅读器 UI 或 macOS 结果,原生截图不包含 Web 图层。

Mac 源码传输仍待解决:物理网段检查未发现 192.168.31.* 活动 en 接口,因此未继续 curl;Cindy 当前 SSH 主机列表为空,没有可直接复用的 SSH 连接。既有 Mac Worker 正在做 iOS 响应头的真实 HTTP 定向验证,未重复派发桌面或完整 UI。

## Core 3be57b5 安装版媒体追加验收(2026-11-03)

Lead 在 Windows 实际安装功能提交 `f7b99d1`/Core `3be57b5`,使用全新业务与 WebView 目录,只预置本地源定义,不注入 Cookie 或进度。原生搜索→详情→选集进入 HLS 播放,中文与英文字幕实际可见。首次播放到片尾后按既定规则清除进度;通过原生进度条定位后,页面保存 `3.030709` 秒。旧进程经正常逐窗关闭退出,新进程经搜索建立夹具会话后点「继续观看」,页面 start 值为 `3.030709`,英文选择保留,恢复视频区域与保存帧逐像素相同。所有清单均为无 Range 的 200,媒体均为鉴权通过、长度准确的 206 Range。

本次关闭字幕的 Home 按键被电脑工具以 `foreground_unavailable` 拒绝且未发送,未通过其他通道重试该输入,因此关闭字幕保留待验。后续单独复测尝试 UIA expand/set_value,选择读回仍为 English;随后 End 按键被 `DESKTOP_INPUT_BUSY` 拒绝,未发送。该轮保留为未验收,回执 `E:/Temp/nk-3be-candidates/subtitle-off-5261666d/receipt.json`;只按路径身份终止本轮临时进程并卸载,资料哈希不变,不把这次清理当正常关窗证据。夹具无音轨,不证明扬声器、PiP、真实站点、自动登录、源安装 UI 或 macOS。两次进程均正常关闭;本任务夹具经身份核对停止并确认端口关闭,临时安装卸载,业务资料文件哈希不变。回执、HTTP/截图核验及清理日志见 `E:/Temp/nk-3be-candidates/installed-media-55cca19b/receipt.json`、`verify-media.log`、`cleanup.log`。客户端 main pin 仍为 `4f213ff`。

**Core 发布与消费端候选更新(2026-11-03)**:Core main 已发布 `3be57b5`,功能源码与通过 CI `41631619233` 四个作业的 `6129f59` 相同,保树收拢回执见 `E:/Temp/core-stream-release/publish-receipt.json`。新的独立 pin 候选为 Nekularity `ea87eb5` 与 Nekophage `f7b99d1`,均指向 3be57b5;桌面七项检查与实际二进制版本核对通过,iOS 候选 `ea87eb5` 的 Mac CI `41638119772` 通过 Kit 单测、App 与 UI 测试编译;下载的未签名 IPA 的 SHA-256、ZIP CRC、App/Widget 0.3.2/14、最低 iOS 26.0 和 Core marker 已核对,UI 未执行。Swift 提前响应头候选 `65eba9f` 的 CI `41635149850` 在 Kit 单测失败;后续 `77c2c6` 的 Mac 定向三项断言也失败,首个结束原因正在诊断,尚未并入新 pin 候选。Core 可在响应头提前交付的宿主上增量返回文字,但旧客户端宿主仍延后,标准 AI 页面仍整包;新 pin 的窗口、Mac 消费端、UI 与真机结果不能借用旧版本。两端 main pin 均保持 4f213ff。

**桌面主线整合(2026-11-03)**:先前把 Nekophage 工作分支称作 main 的口径已更正;整合前真正 main 为 `28be4bd`/Core `214e778`,核对日志 `E:/Temp/core-magpie-first-chunk-evidence/desktop-main-ref.log`。现已将 `fix/subtitle-record-cap` 快进整合为真正 main 的 `67fc877`,pin 为 `4f213ff`,原分支保留;同时带回 WebView 探针隔离修复并移除误入版本控制的 Python 缓存。七项检查与 Windows 隔离阅读回归通过:两页解码、双页、55% 进度、重开恢复 1200,仅有本次章节存储键。回执 `E:/Temp/nk-desktop-integration/receipt.json`。`a079ce7`/Core `b55c1a4` 的 Windows 包仍属于独立候选,新 pin 的 macOS 消费端尚未验收;Nekularity main 仍 pin `4f213ff`。

更新:2026-11-03。以下按章节区分 Windows 真实站点、loopback 媒体夹具、原生窗口、单测与 macOS 的证据。

## 已发布 Core b55c1a4 的桌面候选(2026-11-03)

独立分支 `update/core-b55c1a4`、提交 `e21b04c` 使用干净的固定 Core 工作树 `E:/Temp/core-b55-consumer`。`bash scripts/check.sh` 七项通过,生成二进制 `E:/Temp/nk-b55-candidates/Nekophage-b55.exe` 的实际 `-version` 报告 `e21b04c` 与 Core `b55c1a488746efc1454e4f4781e4e83dc577d187`。日志 `E:/Temp/nk-b55-candidates/desktop-check.log`、`desktop-build-version.log`。本候选新 pin 的窗口与 macOS 验收待补,main 仍用 `4f213ff`;下面 `2c9c83b` 的包内媒体结果保留原提交归属。

## 新 Core 候选的安装包内媒体补验(2026-11-03)

Lead 在 Windows 实际安装 0.2.7/`db024bd`(Core `2c9c83b`)后,通过原生视频搜索→详情→选集进入包内 WebView。`-data-dir` 使用全新资料,只预置仓库本地视频夹具的源定义,未注入 Cookie 或进度;本轮不验证源安装 UI。播放器先显示中文字幕,同一暂停画面通过字幕选择框切到 English 后显示 `Subtitle rendering proof`,再关闭后字幕消失,`videoSubtitles.choice.off=true` 写入磁盘。原生播放控件让画面与时钟从 0:00 前进到 0:10。夹具日志确认所有 Referer/Cookie 校验为 true,清单 200 且无 Range/If-Range,媒体为精确长度的 206 字节范围,英文清单与 VTT 被请求。

通过页面进度条保存 2.977162 秒后,正常关闭全部四个窗口,确认旧 PID `38364` 已退出,再启动同一已安装 exe(PID `20864`)与同一实际资料。主窗单独关闭不会结束仍有子窗的进程,该初次检查未通过的日志保留。新进程通过原生搜索重新建立夹具会话,点击「继续观看」,页面 URL 带 `start=2.977162`;恢复视频区域与保存画面逐像素一致,与起始帧和片尾均不同,字幕保持关闭。没有向播放器补写进度,也不声称自动跨进程登录。

回执、窗口截图、HTTP 与比对日志在 `E:/Temp/nk-core-2c9-runtime/media-installed-0c334515/`: `receipt.json`、`chinese.png`、`english.png`、`off.png`、`resume-before.png`、`resumed.png`、`fixture.log`、`verify-media.log`。最后正常关闭本轮全部窗口,停止本轮夹具并确认端口关闭,静默卸载临时安装且实际资料逐文件哈希保持不变(`cleanup.log`、`uninstall.log`)。这次补上新 pin 安装包内媒体和跨进程续播证据,不覆盖无音轨夹具无法证明的音频/扬声器、PiP、真实站点或 macOS;两个客户端 main 的 pin 仍未升级。

## 可指定资料目录

启动参数 `-data-dir <目录>` 将应用业务资料指向指定目录,并在使用前创建该目录;不传参数时沿用原有用户资料目录。Windows 本机构建实测:`-version` 报告 Core `4f213ff`,`-data-dir` 指向全新目录后该目录内实际出现 `sources`,日志 `E:/Temp/nk-data-dir-win/verify.log`。它覆盖源、书架、设置和 Cookie 等应用数据,不承诺把系统框架的缓存与偏好路径全部重定向。

Orca Worker 已在 `c387e3f` 上重新打包 macOS App,`-version` 报告提交 `c387e3f`、Core `4f213ff`,ad-hoc 签名通过,二进制 SHA-256 `1f61424525835456dbf586e733119f6eed07285811ae9ed125d135689ae52271`;隔离审查日志 `/tmp/nk-neko-949/isolation-check.log`。按当时过宽的用户域写入边界,窗口未启动,不计通过。后续源码复核确认主窗启动不创建 WebView,MyGo 的 `registerDefaults` 注册内存默认值;这两项调用本身不证明会写入持久偏好。随后按修正后的边界实际启动了打包应用:`-data-dir` 指向全新临时目录(其中实际创建了 `sources`),窗口可见并截图,显示搜索、浏览、书架与源管理控件及空源提示,截图与日志在 Mac `/tmp/nk-neko-949/evidence/packaged-c387/`。原业务资料目录的文件元数据未观察到变化,所查 WebKit、HTTPStorages、Preferences、Caches、Saved State 路径均未新增;元数据对比不能单独证明未发生读取。

**正常关闭仍未验收**:桌面工具报告 `screen_locked` 并拒绝输入,清理时只对本轮 PID 发送 `SIGTERM`,包装进程记录的退出码 0 不能当作正常窗口退出。启动日志另有 `sandbox_extension_issue_file_to_process … Operation not permitted` 一行,窗口随后仍正常显示,原因未查。该项需解锁桌面后重测。

## 当前 Core pin 的 macOS 阅读验收(2026-11-03)

Orca Worker 在 macOS arm64 对 Nekophage `90e784f`、Core `4f213ff` 完成隔离资料验收:源码 bundle 两层 SHA-256、全新克隆 HEAD/tree 和 Core pin/stamp 一致;`sh scripts/check.sh` 7 项通过。`uicheck` 的本地源搜索与详情通过;`readercheck` 实际解码两页并由页面上报进度,滚动位置移动/保存/重开恢复均为 1200,双页布局两列通过。日志和截图在 Mac `/tmp/nk-neko-949/`。这些均是本地夹具,不是在线站点验证。

`sh scripts/package-macos.sh` 生成 0.2.7 `.app`,其 `-version` 实际报告 Core `4f213ff` 和提交 `90e784f`,二进制 SHA-256 `c70a6567ae85292b2535b2d7f08818e0e44b8e2f7276e84d9c09ce4ca11cde9d`;ad-hoc 签名验证通过,不等于 Developer ID 签名或公证。由于应用无隔离资料目录参数,Worker 未启动打包后的 `.app` 窗口以免读写用户资料,该项仍未验收。传输所用临时 secret gist 在下载和哈希核验后已删除。

## HLS 播放器的拒绝提示(2026-11-03,`efc1a97`)

上轮 Windows 续播在未登录夹具上收到 HTTP 403 时,页面只显示 `manifestLoadError`,用户无法判断是登录/权限还是格式问题。现在 HLS.js 致命错误的 `response.code` 为 401/403 时显示「媒体访问被拒绝 (HTTP 403),请检查视频源的登录状态或访问权限。」;其他 4xx/5xx 带状态码;没有 HTTP 响应的错误仍保留原有 details/「播放中断」文案。

用 `efc1a97` 的本地构建启动同一实际资料(无持久化会话 Cookie),HLS 清单请求确实返回 403,播放器窗口显示上述提示,截图 `E:/Temp/nk-media-http-error/403-player.png`,HTTP 日志同目录 `fixture.log`。常规 HLS 与字幕未受影响:`go run ./cmd/subcheck -dir internal/player/testdata/hls-subs` 通过,重开后 English 选中且活动 cue 为 `Subtitle rendering proof`,日志 `E:/Temp/nk-media-http-error/hls-ok.log`。`bash scripts/check.sh` 全项通过,日志 `E:/Temp/nk-media-http-error/check.log`(提交前)。这是本地窗口与夹具验证,未改 Core,也未证明真实站点返回 403 时的提示路径。

## 最新 Windows 包与包内 HLS(2026-11-03,`87e2ffc`)

最新本地测试包仍为版本 0.2.7,构建提交 `87e2ffc`,Core `4f213ff`,包含清单 Range 修复。便携 ZIP 为 **21,565,682 字节**,SHA-256 `c39857ee67c2186854488ae49d5237e706d1d077f5b367f6d49ea48b5c7df3f8`;setup 为 **18,722,739 字节**,SHA-256 `04416d2895222bac3d40084e279b1eda79e92cdf1a5f03abe20f2a9da41702c0`。`package-windows.py --nsis` 的 CRC、逐文件哈希、解压程序版本核对通过,日志 `E:/Temp/nk-package-87e2ffc-build.log`。

这次用 `/S /NOREGISTER /D=E:/Temp/nk-package-87e2ffc-accept/install` 明确指定临时安装目录,安装后的 exe 与 ZIP 中的文件哈希一致,资料使用隔离 APPDATA。通过安装包真实原生窗口完成视频→搜索 clip→详情→Episode 1→播放,实际看到视频与「第0段字幕 / 字幕渲染检查」,截图 `E:/Temp/nk-package-87e2ffc-accept/chinese-paused.png`。测试源派生自仓内 HLS 字幕夹具,额外声明 `Range: bytes=0-` 与匹配的 If-Range;服务器日志确认三份清单均为 200、没有 Range/If-Range,媒体 MP4 字节请求均为 206,Referer/Cookie 鉴权均为 true。页面进度自行落盘,实际位置 `0.95312` 秒、总时长 `10.066666` 秒;没有从测试工具注入进度。

字幕选择的定向按键被桌面工具以 `foreground_unavailable` 拒绝,该次没有发送输入,随后停止界面操作。该轮英文切换与关闭重开续播未验证,不与上轮开发工具 `subcheck` 的通过结果合并。清理时只终止本轮测试进程;临时卸载返回 0,安装目录移除,真实落盘的资料内容保留。完整回执 `E:/Temp/nk-package-87e2ffc-accept/receipt.json`,HTTP 日志 `fixture.log`,安装记录 `setup.log`,卸载记录 `cleanup.log` 均在同一目录。包未签名,未公开发布。

随后用同一 ZIP 的便携二进制(哈希与安装版一致)和原样保留的实际资料,补验了**进程重启后的继续观看**。视频列表读回同一集的 9% 进度,播放器 URL 带 `start=0.95312`。首次请求清单返回 403:夹具登录 Cookie 是会话 Cookie,按既有策略不跨进程保存;在原生搜索界面执行夹具登录后再次点「继续观看」,HLS 和中文字幕恢复。恢复画面的 900×350 视频裁剪与保存位置的旧截图逐像素一致,且与片尾截图不同;没有注入播放位置、Cookie 或进度上报。回执 `E:/Temp/nk-package-87e2ffc-accept/resume-receipt.json`,画面对照 `resume-frame-check.json`,截图 `resumed-authenticated.png`,HTTP 日志 `resume-fixture.log`。该结论以重新建立夹具会话为前提,不声称已自动跨进程登录。后台可访问性写入 English 未改变实际中文选择,英文/关闭操作仍未验收。测试进程已清理,资料未变。

## HLS 清单不继承源的 Range(2026-11-03)

源声明的默认 `Range`/`If-Range` 原先也会用于 HLS 清单。扩展现有嵌套 HLS 回归后,真实 HTTP 服务在修复前收到 `Range: bytes=0-7`,返回 206 与截断的清单,清单未能完成 URL 重写,日志 `E:/Temp/nk-hls-declared-range-before.log`。现在代理对已识别的清单移除这两个头,保留 Referer 等源请求头;媒体分片仍沿用源声明并接受播放器提供的范围,不会改写原始声明。

`go test -race -count=1 ./internal/player` 通过,原有用例同时核对根清单重定向、嵌套清单重写、密钥/初始化文件读取、分片 `206`/`Content-Range`/实际字节以及请求头,日志 `E:/Temp/nk-hls-declared-range-after.log`。Windows WebView 的 `go run ./cmd/subcheck -dir internal/player/testdata/hls-subs` 也通过:重开后 English 仍选中,活动 cue 为 `Subtitle rendering proof`,日志 `E:/Temp/nk-hls-declared-range-subtitle.log`。前者是 HTTP 回归,后者是本地 HLS 字幕窗口验证;尚未验证真实站点。先前 `e578d29` Windows 包不含此修复。

## Core pin 升到 `4f213ff`,并用真实源脚本验 `URL`(2026-11-03)

此前 pin 是 `8e14aad`,源作用域还没有 `URL`/`URLSearchParams`。Core 补上这两个类后,桌面端仍然链的是旧 pin,所以「Core 有修复」并不等于「客户端用上了」。现在 `scripts/core-version` 与 `build/core` 都是 **`4f213ff`**,并在客户端仓内新增 `TestSourceURLClassIsUsableThroughAComicSource`:装一条真源脚本,让它用 `new URL` 取 `host`/`href`、用 `URLSearchParams` 改查询,再走 `Core.Search` 拿结果。这个探针先暴露了 Core 未按标准把特殊 scheme 主机名转小写(`Images.Example.Test` 原样返回),Core `4f213ff` 修掉后通过。

`sh scripts/check.sh` 全项通过(gofmt、pin、vet、race 测试、docs、partitions、capabilities),日志 `E:/Temp/nk-pin-4f213ff-check.log`;URL 源探针日志 `E:/Temp/nk-pin-4f213ff-urlprobe2.log`。随后在桌面提交 `c36875d`、Core `4f213ff` 上设置 `NK_OFFICIAL_SRC=E:/Temp/nkprobe/official-srcs`,运行 `bash scripts/desktop-probes.sh`,结果 **19 PASS / 0 FAIL / 0 SKIP**,日志 `E:/Temp/nk-desktop-pin-4f213ff-probes.log`。其中视频、阅读与字幕使用本地夹具;官方源目录用于回调与标签测试,不代表真实站点或账号已验证。

## 升 pin 后的 Windows 包(2026-11-03,`e578d29`)

重打 `e578d29`、Core **`4f213ff`** 的 Windows 包。便携 ZIP 为 **21,565,167 字节**,SHA-256 `41bd37c3e4bcec3a72e78ae316db8523361ccb4e6b25b8570ac2aa9661dced83`;NSIS setup 为 **18,719,324 字节**,SHA-256 `23dd1b453b19e6a8734937b35e86af585c73756de0e5d94ab86aa88224a3e69e`。构建脚本核对了 ZIP CRC、解压后的逐文件哈希及 `-version`,日志 `E:/Temp/nk-package-4f213ff-build.log`。

安装器 `/S /NOREGISTER` 返回 0;安装后二进制报告版本 0.2.7、Core `4f213ff` 和提交 `e578d29`,自带卸载器,状态为 `skipRegistration=1`。安装版启动后系统报告标题 Nekophage、窗口 1100×760,随后窗口正常退出;卸载返回 0、安装目录移除、隔离 APPDATA 下的测试资料保留,日志 `E:/Temp/nk-package-e578d29-install2.log`。这次先误以为重定向 `LOCALAPPDATA` 会改变 NSIS 目标目录,实际安装器通过 shell 文件夹 API 仍选真实用户的 `Programs/Nekophage`;该次安装后已完成卸载。安装版的截图被其他窗口遮挡,不作界面证据。

便携 ZIP 解压到独立目录后可启动和关闭,置顶后可见搜索/浏览/书架/分类/源管理/AI/视频栏及空库提示,截图 `E:/Temp/nk-portable-e578d29-window.png`,日志 `E:/Temp/nk-portable-e578d29-capture.log`。这是本地未签名测试包,不声称已发布或做过新 pin 的全套窗口回归。

## 媒体断流保留传输错误(2026-11-03)

代理原先忽略 `io.Copy` 的错误:上游以 chunked 方式给出部分视频后断开,本地代理仍正常结束响应,播放器收到的是残缺正文加干净 EOF。真实 `httptest` HTTP 连接复现了该问题:修复前无长度声明的子例得到 `body="first" err=<nil>`;带 `Content-Length` 的对照则已能识别短正文。现在已发出响应头后若复制失败,处理器用 `http.ErrAbortHandler` 中止下游响应,两种情况下客户端都收到部分字节及 `io.ErrUnexpectedEOF`,不把断流当作播放完成。测试同时确认源 Referer 仍到达上游。

`go test -race -count=1 ./internal/player` 与 `go vet ./internal/player` 通过。负向对照与修复日志分别为 `E:/Temp/nk-media-truncated-before.log`、`E:/Temp/nk-media-truncated-after.log`。这是 HTTP 传输错误回归;不声称播放器已在真实站点完成自动重试。

## 打包前核对实际 Core(2026-11-04)

切换分支后残留的 go.work/corestamp.go 曾让 pin 为 `3be57b5` 的 `97e24b6` 打出 `4f213ff` 安装包,脚本因只核对自身提交而错误报告通过。错误产物已被正确pin版本替换,原日志 `E:/Temp/nk-ca8-package/build-fixed.log` 保留。现在 Windows 与 macOS 打包入口复用 check-core-pin.sh,在创建/替换产物前核对实际模块、干净检出与生成戳记。错误模块和旧戳记两种 Windows 实验均在输出变更前拒绝,已有产物保留,日志 `E:/Temp/nk-package-pin-evidence/negative.log`。Orca Worker 已在 Mac 的 ca8f370/3be57b5 工作区加入同一检查并实际验证:旧戳记以 generated stamp differs/exit1 拒绝,原.app的4个文件内容均不变;finally精确恢复戳记后正确pin打包通过,实际3be57b5、0.2.7及ad-hoc校验成功。证据 `/tmp/nk-ca8-acceptance/pin-gate-evidence/`,Lead回执 `E:/Temp/nk-package-pin-evidence/mac-receipt.json`。此轮未操作窗口。Windows 开发 WebView 的引擎是 WebView2,不是 WebKit。

## 媒体服务器关闭后拒绝新会话(2026-11-04)

关闭后调用 Server.Open 仍返回会话和已无法访问的页面地址;新建与 Close 并发时也可在清空会话表后登记一个未取消的媒体代理。两个生命周期用例在修复前复现失败。现在关闭标记与会话登记共用服务器锁:关闭后的新建取消刚分配的代理并返回 http.ErrServerClosed;已登记的会话仍由 Close 取消。

Windows 的 go test -race -count=1 ./internal/player 已通过,修复前后日志 `E:/Temp/nk-player-close-evidence/before.log`、`after.log`。这是 Go 接口生命周期回归,不等于新安装包或 Mac 窗口验收。

## 媒体会话关闭后的迟到写入(2026-11-03)

同步屏障复现了一个底层生命周期缺陷:HTTP 请求已通过 token 查到会话,但还在接收请求体;此时关闭会话或整个服务,旧处理器仍返回 204 并调用进度/字幕保存回调。旧状态可能覆盖后来保存的状态。`TestClosedSessionDiscardsPendingReports` 用 `io.Pipe` 明确控制「读到正文→关闭→正文 EOF」的顺序,四种组合在修复前均失败(`status=204 callbacks=1`),日志 `E:/Temp/nk-session-late-reports-before.log`。

现在在接收完正文、接受上报前重新检查媒体代理已有的取消 context,关闭后的迟到上报返回 410,不调用保存回调;服务关闭与单会话关闭共用此信号,不另建生命周期状态。race 回归包含上述四种组合及存活会话正常进度/字幕路径,通过日志 `E:/Temp/nk-session-late-reports-after.log`。这是 HTTP 处理器的确定性时序测试,不声称做了实际界面或 TCP 断连验收。取消前已被回调接纳的操作仍允许结束,本修复不回滚已接受的保存。0.2.5 本地包 `e1c2da4` 不包含此后续修复,0.2.6 包(`3359910`)包含。

## macOS App 打包与启动(2026-11-03,提交 `580f9fd`)

新增 `scripts/package-macos.sh` 后,在 macOS 27.0.1 arm64 / Go 1.27.3 上真实构建:`go build -trimpath -ldflags "-s -w"` 产出 `build/macos/Nekophage.app`,Info.plist 由脚本生成并通过 `plutil -lint`,版本取自程序自报的 `0.2.7`,Core pin `214e7782`。App 总体积 `25,487,983` 字节,可执行文件 `25,449,712` 字节、SHA-256 `0d8ae671d4b55473790b29068085f829f0a44af6b6ee5f29e45d89e638384d58`。

签名是 **ad-hoc**(`codesign --force --sign -`):`codesign --verify --verbose=2` 报 `valid on disk` 与 `satisfies its Designated Requirement`,这是 Apple Silicon 上运行 Go 二进制的必要条件,**但不是 Developer ID,也没有公证**;换一台机器仍会被 Gatekeeper 拒绝,需要仓库没有的凭据。

启动核对:`open build/macos/Nekophage.app` 后进程存在,CoreGraphics 窗口列表确认 owner `Nekophage`、标题 `Nekophage`、尺寸 **1100 × 760**、位置 `(410, 78)`;窗口截图 `/tmp/nk-mac-window.png` `81,740` 字节(1146×806),可见工具栏与空库提示;全屏截图因终端无屏幕录制权限为全黑,只作权限证据。`osascript quit` 后进程与 PID 均消失。日志在 `/tmp/nk-mac-pkg-evidence/`。

这一轮还暴露并修掉了一个环境缺陷:`fetch-core.sh` 传外部 Core 路径时,`go.mod` 里指向 `./build/core` 的 replace 仍生效,构建直接失败。现在工作区改用 `go work edit -replace`,外部模式跳过 `go mod tidy` 并说明原因;两种模式都复验过(外部 fetch/build/test 通过,默认模式仍取到 pin 并通过 `check.sh`),提交 `16025c4`。限制:这是开发构建打包,不是发布产物;播放、字幕与真实站点未在这个 `.app` 内验证。

## 升 pin 后的 Windows 包(2026-11-03)

`daf55ac`(Core `8e14aad`)上重打并装了一遍:

| 产物 | 字节 | SHA-256 |
|---|---|---|
| `…-windows-amd64-daf55ac.zip` | 21,561,961 | `d7dc173b…4d79bf` |
| `…-windows-amd64-daf55ac-setup.exe` | 18,716,090 | `b70d7799…fec015` |

解压到任意目录(不是安装目录)直接运行,`Nekophage.exe -version` 同样自报 `0.2.7` / `8e14aad…` / `daf55ac`,说明便携包不依赖安装位置。`Start-Process setup.exe -ArgumentList '/S' -Wait` 退出 0;装出来的可执行自报 `Nekophage 0.2.7`、`Core 8e14aade75f563a95f50d4ec80be27bae0d02c63`、commit `daf55ac`,与包名一致;`Uninstall.exe /S` 退出 0,三秒后安装目录消失,`APPDATA/Nekophage` 保留。前一份包(`5c5ce67`)记录的是 Core `214e7782`,两份都是各自提交上的真实产物,不互相替代。日志 `E:/Temp/nk-package-pin-build.log`。

## Core pin 推到 `8e14aad`(2026-11-03)

Nekophage 的 pin 从 `214e7782` 升到 `8e14aad`,也就是包含 `b2610a5`(异步 `rejection` 保留错误名)的那个 Core 提交。按既定规则,生产改动在下一次客户端构建时随版本升 pin,这次正是那个时点。

`214e7782..8e14aad` 之间 Core 只有一个生产文件变了(`comic/js/engine.go`),`apple/` 下的 Swift 表面没有任何改动,所以链接的行为差异仅是错误文案,不会改变 API。

本机验收:`scripts/set-core-pin.sh` + `scripts/fetch-core.sh` 把 `build/core` 切到该提交,`go build ./...` 通过,`go test ./...` **16 个包全过**,`scripts/check.sh` 通过,窗口套件 **14 PASS / 0 FAIL / 0 SKIP**,`-version` 自报 `Core 8e14aade75f563a95f50d4ec80be27bae0d02c63`。日志 `E:/Temp/nk-pin-build.log`、`E:/Temp/nk-pin-test.log`、`E:/Temp/nk-pin-probes.log`。

## 当前 HEAD 的 Windows 包重打并验收(2026-11-03)

在 `5c5ce67` 上重打 Windows 包,不只看构建通过,还装了一遍:

| 产物 | 字节 | SHA-256 |
|---|---|---|
| `Nekophage-0.2.7-windows-amd64-5c5ce67.zip` | 21,561,030 | `5ba010a7…d8d3af3` |
| `Nekophage-0.2.7-windows-amd64-5c5ce67-setup.exe` | 18,714,941 | `847cda97…d8576f8` |

安装:`Start-Process setup.exe -ArgumentList '/S' -Wait` 退出 0,`LOCALAPPDATA/Programs/Nekophage/Nekophage.exe` 存在且 `-version` 自报 `0.2.7`、Core `214e7782…`、commit `5c5ce67`(与包名一致);`Uninstall.exe` 存在。卸载:`/S` 退出 0,三秒后安装目录**已消失**(`dir_left=False`),`APPDATA/Nekophage` 仍在——`0.2.7` 那个 `SetOutPath "$TEMP"` 的修复在重新打的包上依然成立。

一个操作坑:在 bash 里直接 `./setup.exe /S` 会静默退出而不安装(参数被 shell 拧了),必须走 PowerShell 的 `Start-Process -ArgumentList '/S'`。日志 `E:/Temp/nk-package-head-build.log`。

## macOS 阅读主线在当前 pin 上通过(2026-11-03,`56c9f0c` / Core `8e14aad`)

仓库没有 remote,代码经私有 gist 传:`base64` 3,255,720 字节、SHA-256 `c6bd87de…1181fa`,解码后 bundle 2,441,789 字节,`git bundle verify` 报 `contains this ref` 与 `records a complete history`,HEAD `56c9f0c`,`scripts/core-version` 与 `fetch-core.sh` 拉到的 Core 都是 `8e14aade75f563a95f50d4ec80be27bae0d02c63`。`go build ./...` 退出 0;`-version` 自报 Core 就是这个 pin(客户端 commit 显示 `unknown`,因为只有 Core 打了 stamp,客户端 commit 由打包时注入)。

三条阅读检查都在仓内夹具上跑,退出码均为 0:

- **进度上报**:`pctKeys` 含本章节的 `nkreader.at:loopback_pages/one/1:pct`,`pct=55`;
- **双页布局**:两列各 `440.5px`,第 1 页 `x=443` 在右、第 2 页 `x=0` 在左,`dirLabel` 为 `右→左`;
- **阅读位置**:滚到 1200 后重开,`scrollY=3207`。

第三项要打个问号:macOS 回到 **3207**,而 Windows 上同样操作用本篇上面的严格判据是**精确回到 1200**。我随后把判据从「≥80%」改成「落在保存值的 ±20% 内」(下限 80px),Windows 连跑两次仍是 `scrollY=1200 saved=1200`;macOS 那次是在旧判据下跑的,所以 3207 究竟是不是缺陷还没有定论——已请 Mac 侧用新判据复跑。截图 20,756 字节只含原生层,不含 WebView 像素,渲染证据是 `decoded=2 / images=2`。

## Mac 复跑定性:3207 是旧判据下的假象(2026-11-03,`12ad82e`)

同一份夹具在 Mac 上用新判据连跑两次,**两次都通过**:

```
run1  settled at [1200,5,"1200"]  → 重开 scrollY=1200 saved=1200
run2  settled at [1240,5,"1240"]  → 重开 scrollY=1240 saved=1240
```

run2 的 `moved` 是 1240 而不是 1200,正是「±2px 内再偏 40px」在起作用:上一跑存下的 1200 让新进程已被恢复到 1200,探针因此偏移后再滚。两处修复(探针偏移 + 阅读器「布局稳定前不写、最后一张图到尺寸后再应用」)正好咬合。`-layout` 与 `-progress` 同批也都退出 0。

五轮收敛的轨迹值得留档:

| 提交 | `-scroll` | `settled at` / 重开后 | 判据能看到的 |
|---|---|---|---|
| `baa35a4` | 2/2 通过 | `[1200,5]` / `3207 saved=3207` | 自证:拿存档值当基准,看不见漂移 |
| `44410bf` | 0/2 失败 | `[1200,3,null]` / `0 saved=<nil>` | 暴露「根本没写」 |
| `2dbbc29` | 0/2 失败 | 同上 | 只有页面侧时序修复,探针仍在竞态 |
| `12ad82e` | 2/2 通过 | `[1200,5,"1200"]` / `1200` | 漂移被拦住,`saved == moved` |
| `1db4c91` | 2/2 通过 | `[1200\|1240,5,"1200"\|"1240"]` / 对应值 | 加了 boot 等待后稳定复现 |

顺带纠正回执里的一处推断:Mac 侧认为 `-progress` 的 `y=3207` 来自 Core 持久存储,**不对**——那正是探针自己滚到页面 55% 的位置(`0.55 × 5752 ≈ 3164`,加上取整)。真正的问题在同一个日志里:`pct` 读的是 `Object.keys(localStorage)` 的**第一个** `:pct` 键,而存储是跨夹具共享的,所以它报过别的章节的 21。现在改成按本章节的键读,并新增两条断言——键在但值为 `null` 时报「这一章没有存百分比」,报告里没有百分比也失败。本机实测 `pct="55"`、`scrollY=3127`。

## 探针不再赌「一秒后键就写好了」(2026-11-03)

Mac 侧跑 `44410bf` 时两次都失败,失败行是 `settled at [1200,3,null]` —— 位置键**压根没写**,重开自然没东西可恢复。它们指出的另一件事也成立:我那条「存储值必须等于基准」的硬校验其实已经命中(`saved` 为 nil),但紧接着的恢复判定**无条件覆盖**了它,日志里只剩「没恢复」。

两处都改了:

- 滚动后不再固定睡 1 秒,而是**轮询最多 5 秒**等 `localStorage.getItem(位置键)` 出现;拿不到就明确报 `the page stored no place for … after …`,并在此时直接结束,不再让后续检查覆盖这条更重要的错误;
- 顺带把「键写上了、值不对」与「键根本没写」区分成两句不同的话——前者是漂移,后者是保存没发生,维修方向完全不同。

再往前一步:Mac 侧那几次 `localStorage.length` 只有 3 条,说明**页面 boot 还没走完**探针就动手了。现在开滚之前先等阅读器自己写出一条位置键(它在 900ms 必写),最多等 10 秒;等不到就直接报 `the reader never reported its own progress; storage never showed a position key`——因为这种状态下滚一下量的是探针,不是阅读器。

实测:本机 `-scroll` 连跑 5 次全过(`1200 saved=1200` / `1240 saved=1240`),窗口套件关掉重试 **12 PASS / 0 FAIL**,`check.sh` 通过。日志 `E:/Temp/nk-poll-{1,2,3}.log`、`E:/Temp/nk-boot{2}-*.log`、`E:/Temp/nk-probes-boot.log`。

## 阅读位置的漂移:恢复跑在图片定尺寸之前(2026-11-03)

Mac 侧把 3207 这件事查清了:不是“恢复坏了”,是**存档值本身从 1200 漂到了 3207**——恢复发生在图片拿到尺寸之前,布局随后变高,浏览器带着滚动位置走,防抖保存再把漂移后的值写下来,于是重开读到 3207,拿 3207 跟自己比当然“一致”。它们的估算也对得上:`1200 × 6752/2522 ≈ 3212`。

修的是**时序**,不是采样:

- 位置改为在**最后一张图拿到尺寸之后**再应用一次,并且在布局稳定前**不写入**任何位置;
- 图片监听对象从 `images`(URL 字符串数组)改为 `pages.querySelectorAll('img')`——第一版直接对字符串调 `addEventListener`,把后续初始化全抛掉了,表现为“既不恢复也不保存”(本机三次全红才发现);
- 缓存命中的图片可能早于监听器就绪,`img.complete` 时立即计入,否则计数永远到不了零。

实测:本机 `-scroll` 连跑 3 次全过(`1200 saved=1200` / `1240 saved=1240`),窗口套件关掉重试连跑 2 轮 **12 PASS / 0 FAIL**,`check.sh` 与 `internal/reader` 测试通过。这是页面行为改动,真正定性仍要看 Mac:新探针把基准换成「探针实际滚到的位置」,漂移会直接报错。

## 阅读位置检查的偶发失败:其实是“没动”而不是“没记住”(2026-11-03)

改成严格判据(落在保存值 ±20%)之后,`-scroll` 开始约一半概率失败:重开后 `scrollY=0`、存储里根本没有位置键,而滚动那一刻 `scrollY` 明明是 1200。

读到根因在探针自己身上,不在 App:**探针先清存储、再滚到 1200,而页面可能已经被旧键恢复到了 1200** —— 滚到当前位置不会触发 scroll 事件,阅读器就不会写入任何东西,重开自然没有可恢复的位置。这解释了为什么失败时 `localStorage` 少两个键(位置与百分比都没写)。

修法不是在采样上打补丁,而是让这一步**永远不是空操作**:目标与当前滚动位置相差 2px 以内就先偏 40px,并把**实际**的 `scrollY` 作为基准返回;判据同时要求存储值与基准一致(`saved`),否则报「存储说 X,但探针滚到了 Y」。另外在移动前等页面真正就绪(`readyState === 'complete'` 且 `#pages figure` 有内容),避免在即将被替换的空文档上滚动。

实测:连跑 4 次全部通过(`scrollY=1240 saved=1240` / `scrollY=1200 saved=1200`,取决于是否触发了那 40px 偏移),窗口套件关掉重试连跑 3 轮均 **12 PASS / 0 FAIL**。日志 `E:/Temp/nk-nudge-{1..4}.log`、`E:/Temp/nk-probes-nudge-{1,2,3}.log`。

这同时解释了 macOS 上那次 `scrollY=3207`:旧判据(≥80%)对比的是「探针想要滚到的位置」,而重开后的位置可能来自存储里的旧值。新探针会把「存储值 ≠ 基准」直接判失败,所以 Mac 侧复跑一次就能定性。

## 窗口探针的偶发失败查到了根上(2026-11-03)

连续跑套件时偶发「某一条 FAIL、日志里却全是 PASS」或「a dialog window was never opened」。分了三种情况处理,不是简单加重试:

- **对话框探针**:它只睡 2 秒就去取窗口,机器上已经开着别的窗口时对话框会晚一点才出现。改成最多轮询 10 秒(每 100ms),单独连跑 **8/8** 通过。根因修掉,不是掩盖。
- **`readercheck`**:页面状态是在 goroutine 里等一段时间后读的;如果窗口在检查前就结束,它照样打印 `done` 并以 0 退出——一次套件运行就是这样“静默通过”的。现在加了 `checked` 标记,窗口提前结束会以 `the window ended before the page could be checked` 失败退出。
- **套件本身**:一个接一个起十几个窗口进程,偶尔仍会丢窗口。`run()` 现在允许**一次重试**,但会打印 `RETRY <条目>` 并把首次日志留成 `.first`——失败可见,套件不变成噪声;`NK_NO_RETRY=1` 可关掉重试,用来测量真实的偶发率。

实测:修好后连续 **3 轮 12 PASS / 0 FAIL / 0 RETRY**;关掉重试时仍偶尔出现一次环境级失败(登录/对话框各见过一次),说明重试是兜底而不是根因。日志 `E:/Temp/nk-probes-final-{1,2,3}.log`、`E:/Temp/nk-dialog-{1..8}.log`。

## 分类页也离线了,只剩官方脚本才需外部目录(2026-11-03)

夹具补齐 `category`(一项「类型」、两个值「连载/完结」)与 `categoryComics.load`,`category-probe.sh` 默认用它,套件里的 `NK_PROBE_SRC` 条件块彻底删掉。

```
category: parts=1 values=2 page=1 comics=1 err=""
```

两半都要算:画出了选项却没加载到漫画,或者返回了漫画却没把声明画出来,都不是完整分类页。这条检查同时读 parts/values 与 comics,所以两种情况都会被抓。

实测:无外部变量 **17 PASS / 1 SKIP**,只给 `NK_OFFICIAL_SRC` 就是 **19 PASS / 0 FAIL / 0 SKIP**。剩下唯一需要仓库外资源的,是**官方脚本回调与标签**——它本来就应该拿官方脚本验,不能用自造的代替。日志 `E:/Temp/nk-probes-cat.log`、`E:/Temp/nk-probes-cat-full.log`。

## 评论与源收藏也走仓内夹具了(2026-11-03)

`loopback_source.js` 原来只回答搜索、详情与章节页;现在补了 `comic.loadComments` 与 `favorites`(`addOrDelFavorite` / `loadFolders` / `loadComics`),于是这两条探针不再需要借 Nekularity 的 `callback_probe.js`:

- `comment-probe`:`comments: count=1 author="Fixture Reader" body="a comment from the fixture"`
- `favorites-probe`:`favorites: count=1 first="Loopback Comic"`

两个脚本都改成默认用仓内夹具,套件里也从「有 `NK_PROBE_SRC`/`NK_FAV_SRC` 才跑」移到默认执行。

同一轮还把**分区**也接了进来:夹具声明两个 `explore` 段(`multiPageComicList`),因为只有一个 section 时界面不画 chip 行——`browse: sections=2 chips=2 comics=1`。剩下真正需要外部脚本的只有两项:**分类页**(要先声明 `category` 与 `categoryComics`,夹具暂未提供)与**官方脚本的回调与标签**。

实测:无外部变量 **16 PASS / 2 SKIP**(两项跳过都写明了原因),给齐外部脚本 **19 PASS / 0 FAIL / 0 SKIP**,`check.sh` 通过。日志 `E:/Temp/nk-probes-favcom5.log`、`E:/Temp/nk-probes-favcom-full.log`。

## 书架也能离线验了(2026-11-03)

`scripts/shelf-probe.sh` 原来必须传一个源脚本,套件里也只在 `NK_PROBE_SRC` 存在时才跑。它自带 loopback 伺服,而仓内夹具的章节正好指向 loopback,所以现在默认就用仓内夹具,并移出条件分支。

判据仍是文件而不是画面:读一章后看 `shelf.json` 里记的是哪部漫画、哪一章、第几页——

```
shelf-probe: shelf row: Loopback Comic / Chapter 1 / page 1
```

实测:无外部变量 **13 PASS / 3 SKIP**,给齐外部脚本 **19 PASS / 0 FAIL / 0 SKIP**(19 条逐条核对过,没有重复条目)。日志 `E:/Temp/nk-shelf-offline.log`、`E:/Temp/nk-probes-shelf.log`、`E:/Temp/nk-probes-shelf-full.log`。

## 原生界面的搜索也进了默认套件(2026-11-03)

上一轮我把 `uicheck` 判为“加了也只是永远绿”:它只截图、不断言,而**找到结果**和**什么都没找到**的界面截图大小几乎一样。这轮补上了断言,才把它纳入:

- App 增加 `SearchForCheck()`:返回结果数、第一条标题、失败文案——和已有的 `FavoritesForCheck`、`CommentsForCheck` 同一思路;
- `cmd/uicheck` 增加 `-expect`:第一条标题必须包含给定文本,否则非零退出。

两个方向都实测过:`-expect 'Loopback Comic'` → `search: count=1 first="Loopback Comic" err=""`、退出 0;`-expect 'Nonexistent Title'` → `first result "Loopback Comic" does not contain "Nonexistent Title"`、退出 1。

同一条检查还打开了第一部漫画,判据是详情页**自己报告**的标题与章节数:`detail: title="Loopback Comic" chapters=1 err=""`;`-expect-detail 'Wrong Comic'` 会以 `detail title "Loopback Comic" does not contain "Wrong Comic"` 非零退出。空白详情页与「这部漫画没有章节」截图几乎一样,所以只看图不算数。

窗口套件因此保持无外部变量 **12 PASS / 3 SKIP**(断言加在既有条目里),给齐变量 **16 PASS / 0 FAIL / 0 SKIP**。日志 `E:/Temp/nk-uicheck-ok.log`、`E:/Temp/nk-uicheck-neg.log`、`E:/Temp/nk-detail-ok.log`、`E:/Temp/nk-detail-neg.log`、`E:/Temp/nk-probes-detail.log`。

## 字幕选择进入默认窗口套件(2026-11-03)

`cmd/subcheck` 也进了 `scripts/desktop-probes.sh`。判据不是「存下了名字」,而是重开后**渲染的正是那条字幕**:`chosen: selected="English" active="Subtitle rendering proof" cues=[en:1]` → `reopened: selected="English" active="Subtitle rendering proof" cues=[en:1]`。这条路径以前真的错过:选择只活在页面里,关掉播放器再打开会退回清单默认值。

窗口套件因此变成无外部变量 **11 PASS / 3 SKIP**,给齐变量 **16 PASS / 0 FAIL / 0 SKIP**(两处跳过的仍是需要外部脚本的那三项)。日志 `E:/Temp/nk-sub-head.log`、`E:/Temp/nk-probes-with-subs.log`。

## 视频播放进入默认窗口套件(2026-11-03)

`cmd/videocheck` 早就有,但只在手动跑。现在它进了 `scripts/desktop-probes.sh`:仓内夹具 `internal/player/testdata/hls/stream.mp4` 经媒体代理播放,判据是三条——位置**自己**前进(`currentTime` 两次读取递增)、页面没有错误文本、播放器在暂停时上报了进度。窗口截图只拍到原生层,所以证据是页面自己报的数字。

实测:`readyState=4 videoWidth=320 duration=10.066667 time=2.0674 -> 4.0688 hls.js=1.7.4 err=""`,退出 0。

同一条检查还验证**字节范围**:从页面真实请求过的 `/media/…` 地址再发一次 `Range: bytes=0-1023`,要求 `206`、有 `Content-Range`、且只回 1024 字节。实测 `status=206 content-range="bytes 0-1023/178570" bytes=1024`——拖进度条要用到它,而整包返回的播放器在拖动前看起来一切正常。窗口套件因此变成无外部变量 **10 PASS / 3 SKIP**,给齐变量 **16 PASS / 0 FAIL / 0 SKIP**。日志 `E:/Temp/nk-video-head.log`、`E:/Temp/nk-probes-with-video.log`。

## 进度上报与阅读位置进入默认窗口套件(2026-11-03)

`-progress` 和 `-scroll` 原来只能手动跑。现在两者都用仓内夹具(readercheck 在 18731 供图),进了 `scripts/desktop-probes.sh`:

- **进度**:只读页面自己写入的上报键——探针自己发一次请求只能证明端点能答,不能证明页面在报告;
- **阅读位置**:滚动后重开,按页面实际可滚动距离的 80% 判断。

实测:不给外部变量 **9 PASS / 3 SKIP**,给齐变量 **16 PASS / 0 FAIL / 0 SKIP**。日志 `E:/Temp/nk-probes-7plus.log`、`E:/Temp/nk-probes-7plus-full.log`。

## 失败页的提示与重试有了端到端检查(2026-11-03)

阅读器在图片加载失败时会显示「第 N 页加载失败 / 重试」,但此前没有任何检查碰过这条路径。新增 `probesite.ServeOnePageFailing(port, failures)`:前 N 个请求返回 500,之后正常给图;`readercheck` 增加 `-fail N` 与 `-retry`。

`-retry` 的判据是两步:首次加载后必须**存在**失败提示(`.retry` 元素数量大于 0),点掉重试后必须**没有**提示且至少有一张图解码成功。两个方向都验过:

- `-fail 2 -retry`:失败提示 2 个 → 重试后 `failures=0 decoded=2`,日志 `E:/Temp/nk-retry-recover.log`;
- `-fail 0 -retry`:以 `a page that failed did not say so: .retry=0` 非零退出,日志 `E:/Temp/nk-retry-negative.log`——没有失败时这条检查不会假通过。

窗口套件新增这一项:无外部变量时 **7 PASS / 3 SKIP**,给齐变量时 **14 PASS / 0 FAIL / 0 SKIP**(`E:/Temp/nk-probes-retry-default.log`、`E:/Temp/nk-probes-retry-full.log`)。

## 滚动检查改按页面实际高度取目标(2026-11-03)

`readercheck -scroll` 原来写死「滚到 1200、重开后至少 800」。这有两个毛病:夹具页比 1200 矮时会因为页短而失败,而一个根本不能滚的页面在阈值为 0 时又会被放过。现在目标由页面算:`min(1200, max(0, 可滚动高度 - 40))`,算出的距离小于 100 就直接判「页面不足以验证滚动」;重开后要求至少回到该距离的 80%。

调这个的过程中我自己踩过一次坑,记在这里:第一版改写把「滚动后等 1 秒」的等待删掉了,而位置是滚动结束 400ms 之后才写入 localStorage,于是重开必然回到顶部——那不是 App 的缺陷,是检查器把要观察的行为先掐掉了。现在等待保留,日志会打出落定的位置。

实测:两次连跑都是 `scrollY=1200`;窗口套件仍 **13 PASS / 0 FAIL / 0 SKIP**。日志 `E:/Temp/nk-scroll10-1.log`、`E:/Temp/nk-probes-scroll-target.log`。

## 阅读解码检查不再依赖别的仓库(2026-11-03)

`scripts/reader-probe.sh` 原来说「给我一个源脚本」,而仓内没有;实际跑的时候借用的是 Nekularity 的 `UITests/Fixtures/callback_probe.js`。现在有仓内 `internal/reader/testdata/loopback_source.js` 之后,不传 `NK_READ_SRC` 就用它(查询词 `probe`、由 readercheck 在 18731 供图),`scripts/desktop-probes.sh` 里这条也从「有环境变量才跑」改成默认跑。

实测:不给任何环境变量时窗口套件 **6 PASS / 3 SKIP**(跳过的三项分别是源收藏、评论线程、官方脚本回调与标签,它们仍需要外部脚本);给齐变量时仍是 **13 PASS / 0 FAIL / 0 SKIP**。日志 `E:/Temp/nk-probes-default-reader.log`、`E:/Temp/nk-probes-full-default-reader.log`。

## 双页布局可以离线验了(2026-11-03)

`-layout` 以前需要真实站点:夹具章节只有一页,而检双页要两页。现在仓内新增 `internal/reader/testdata/loopback_source.js`(章节两页,都指向 `127.0.0.1:18731`,由 readercheck 自己供图),并新增 `scripts/layout-probe.sh` 接进窗口套件。

同时修了这个检查本身的两处问题:它原来断言页面 class 里出现 `spread rtl`,而方向是一个切换按钮,点一下是「另一侧」,断言写死某个词就等于依赖起始状态;另外判断「右→左」时用了 `contains("右")`,而「左→右」里也有「右」,会把左到右误判成右到左。现在按标签开头的箭头方向判断,并检查两列存在且第一页确实在右侧。

实测(Windows 真实窗口):两列 `433.5px 433.5px`,第一页 `x=436`、第二页 `x=0`(右→左),捕获 `build/probes/layout.png`;反例用一页章节跑,工具以 `few figures` 非零退出。窗口套件现在是 **13 PASS / 0 FAIL / 0 SKIP**,日志 `E:/Temp/nk-desk-probes-layout.log`。

## 夹具页改成可滚动尺寸(2026-11-03)

`probesite.ServeOnePage` 原来供一张 8×8 的 PNG:读者能解码,但一页比窗口矮,读位置永远是 0,`-progress` 与 `-scroll` 只能证明上报链路,证明不了「位置被记住」。现在供 **320×1200** 的页(每 100 行一条色带,截图里能看出滚到哪),单测断言文档高度与宽度。

同时改了 `-progress` 的判据:它原来自己发一个 `progress?...percent=55` 的 XHR,那只证明端点可用,不证明页面自己上报。现在只读页面写入的 `localStorage`,并要求出现**本章节**的上报键(单页章节的百分比按页序号算,本来就是 0,所以键才是证据)。本地复跑(`callback_probe.js`、`-serve 18731`):`decoded=1 first=320x1200`;`-progress` 退出码 0,读到 `y=1335 h=3388` 且含 `callback_probe/1/1`;`-scroll` 重开章节后 **scrollY=1200**。日志 `E:/Temp/nk-reader-tall.log`、`E:/Temp/nk-reader-scroll.log`、`E:/Temp/nk-reader-progress4.log`。

## macOS 阅读主线(2026-11-03,提交 `3c42463`)

仓库没有远端,代码以 git bundle 经私有 gist 传到 Mac(base64 `3,190,624` 字节,SHA-256 `093cbfc4…c2e3a173`;bundle `2,392,967` 字节)。`git bundle verify` 通过,clone 后 HEAD 为 `3c42463`,Core 由 `scripts/fetch-core.sh` 取到 pin `214e7782`。macOS 27.0.1 arm64 / Go 1.27.3 上 `go build ./...` 通过。

阅读主线用 `cmd/readercheck` 与离线夹具(`callback_probe.js`,页码指向 `127.0.0.1:18731`,由 readercheck 自己供图)跑通:退出码 0,章节 1 页中**实际解码 1 / 1**,`figures=1 images=1 decoded=1 first=8x8`;截图 `/tmp/nk-mac-reader.png` `20,756` 字节(900×920)。阅读进度上报返回 HTTP 200 与 `{"ok":true}`,键为 `nkreader.at:callback_probe/1/1:pct`。单页高度不足以滚动,进度值为 `0`,这里只证明上报链路成功。运行结束后 18731 端口无监听。

限制:截图是原生窗口捕获,不含 WebView 画面,解码结论来自工具的解码计数;夹具在 loopback,不涉及真实站点或账号;这是开发构建,不是 macOS 打包产物。日志在 `/tmp/nk-mac-reader-evidence/`。

## Windows 0.2.7 本地测试包(2026-11-03,提交 `f114dea`)

这一版第一次带上表头修复:无漫画源时不再把提示挤成一字一行。便携 ZIP `21,559,053` 字节,SHA-256 `7d348877318ec1af88b5c2991933e2441b902c296887b23dd596065ea31e3bc5`;setup `18,712,668` 字节,SHA-256 `c0ff8bfaa9fa89fbe4a4648bfc8443756bf01c514b229dc010a425b360b44da0`。`package-windows.py` 的构建、ZIP CRC、逐文件哈希与解压程序版本检查通过;程序自报 0.2.7、Core `214e7782`、提交 `f114dea`。

便携包启动核对:ZIP 解出的 `Nekophage.exe` 以隔离 `APPDATA`/`LOCALAPPDATA` 启动,窗口标题 `Nekophage`、1100×760,表头只有标题与控件,截图 `E:/Temp/nk-run-027/window.png`,关窗后进程退出。安装核对同样不是隔离安装(NSIS 走 shell 文件夹 API),装到真实 `%LOCALAPPDATA%\Programs\Nekophage` 后自报 0.2.7。这一轮发现卸载器会留下一个空的 `Nekophage` 目录:Windows 不能删除仍被某进程当作工作目录的目录。在 `Uninstall` 段开头加 `SetOutPath "$TEMP"` 后重建 setup,再装机→静默卸载,目录已不在,`%APPDATA%\Nekophage` 配置保留。播放与字幕流程仍未在包内验收。

## Windows 0.2.6 本地测试包(2026-11-03,提交 `3359910`)

这一版把媒体会话迟到写入的修复带进包里(0.2.5 的 `e1c2da4` 不含该修复)。版本号在 `main.go` 中升到 0.2.6;便携 ZIP `21,557,923` 字节,SHA-256 `5b67cd240f2faf9b6097a5bafef8b2542eb9ab43d05bd2919b6c47dabe8cc335`;setup `18,712,671` 字节,SHA-256 `0d2955eaf9e3881fdad4210122e761d76c8f9157ed284b4d68440c182b20ad30`。`package-windows.py` 的构建、ZIP CRC、逐文件哈希与解压程序版本检查通过;`scripts/check.sh` 在 `7bd9965` 上通过。

安装核对**不是隔离安装**:NSIS 的 `$LOCALAPPDATA` 来自 shell 文件夹 API,重定向环境变量无效,`/S /NOREGISTER` 实际装进了真实 `%LOCALAPPDATA%\Programs\Nekophage`,会先覆盖此前的测试安装。安装后程序报 0.2.6、Core `214e7782`、提交 `3359910`、`go1.27.3 windows/amd64`;随后用 `Uninstall.exe /S` 静默卸载,目录已删除,`%APPDATA%\Nekophage` 配置保留。

0.2.6 的便携包另做了一次启动核对:用 ZIP 解包出的 `Nekophage.exe`(不经安装)配隔离 `APPDATA`/`LOCALAPPDATA` 启动,窗口标题 `Nekophage`、1100×760,截图 `E:/Temp/nk-run-026/window.png`,关闭窗口后进程退出。截图同时暴露一个真实布局缺陷:无漫画源时表头那句提示被固定宽度控件挤成一字一行,与标题重叠;该缺陷在当前开发构建中已修(`e54944d`,提示删掉,空状态本来就有说明),修复后的窗口截图 `E:/Temp/nk-run-dev-out/window.png`。**0.2.6 包不含该修复**,播放与字幕流程仍未在包内验收。0.2.6 **未做**安装包窗口的播放、字幕或续播验收,那部分证据仍属于 0.2.5;无 Git remote,未做任何远端发布。

## Windows 0.2.5 本地测试包(2026-11-03,提交 `e1c2da4`)

已把字幕偏好独立存储及迁移修复打包:便携 ZIP `21,556,043` 字节,SHA-256 `a43520be5753c7d983a151f681b4c5fbd287a47cf581950ea08740534bda2e6e`;setup `18,711,551` 字节,SHA-256 `e7db5370ec9a3d948a08680fde552cc4b7fd08a4809df5edbc783384c887980e`。`package-windows.py` 的 ZIP CRC、逐文件哈希、解压程序版本检查通过;隔离 `/S /NOREGISTER` 安装后的程序报 0.2.5、Core `214e7782`、提交 `e1c2da4`。真实安装包窗口完成视频搜索→详情→第 1 集→中文字幕显示,截图 `E:/Temp/desktop-package-025-dropdown.png`;Range、字幕清单/VTT 请求的本地鉴权日志均为 auth=true。

用户要求底层优先后停止进一步界面操作。包内语言切换和偏好恢复未完成验收,不沿用开发工具的通过结论。任务窗口已关闭、进程退出、夹具停止、安装目录卸载且隔离 APPDATA 标记保留,日志 `E:/Temp/nk-package-025-cleanup.log`;回执 `E:/Temp/nk-package-025-acceptance/receipt.json`。包未签名,仅本地交付,没有远端发布;SmartScreen、无 WebView2、macOS 和 iOS 不在本轮范围。

## Windows 0.2.4 安装包(2026-11-02,提交 `29bab49`)

便携 ZIP `21,547,137` 字节,SHA-256 `da5098c58b519c91544e6d5decee79a690cbf0a501d518f8347df0c4f04f336c`;按用户 setup `18,705,377` 字节,SHA-256 `f089f62aa5429ffee6a8adc345fe044f3f048d11f8dce5d04661c9dafb55e105`。回执 `E:/Temp/desktop-package-0.2.4-receipt.json`。

打包过程抓到一个真实缺陷:0.2.3 包在真实窗口中播放同一夹具时**没有任何字幕轨,也不请求 `subs.m3u8`**。原因是默认走了浏览器原生 HLS,而 WebView2 的原生路径忽略 `EXT-X-MEDIA` 字幕组;用 `cmd/uicheck` 去掉 `-video-hlsjs` 可复现(`decoder=native`、`activeSubtitleCues=0`)。`851884d` 改为 HLS 会话优先内置 HLS.js,原生只作回退,并保留 `-video-native` 复现回退路径。

实测:隔离静默安装后 `-version` 报 `0.2.4`/Core `214e7782`/提交 `29bab49`;在**打包后的 GUI** 里点「视频→搜索→详情→第 1 集→播放」,字幕选择框显示「中文」,画面出现「第0段字幕 / 字幕渲染检查」,夹具服务器日志记录到 `subs.m3u8` 与 `sub0.vtt` 请求,截图 `E:/Temp/desktop-package-0.2.4-subtitle.png`;`Uninstall.exe /S` 删除安装目录。该包内字幕语言切换/关闭的后续验收见下一节。**未验证**:macOS/iOS 字幕、签名与无 WebView2 的机器。

## 字幕语言与关闭操作(2026-11-02,安装包 `29bab49`)

Lead 通过 Cindy 桌面工具操作未修改的 0.2.4 安装包,在暂停的同一 `0:01` 画面完成「中文→English→关闭→中文」:英文时画面有 `Subtitle rendering proof`,关闭后该文字消失,恢复中文后显示 `字幕渲染检查`。这次选中了真实下拉选项,没有用 DOM 注入代替控件操作。三张对照截图为 `E:/Temp/desktop-subtitle-024-english-paused.png`、`E:/Temp/desktop-subtitle-024-off-paused.png`、`E:/Temp/desktop-subtitle-024-chinese-restored.png`;完整回执 `E:/Temp/desktop-subtitle-language-024-receipt.json`。

最终夹具使用与 `cmd/uicheck` 相同的 Go `http.FileServer`、登录 Cookie 与 Referer 校验,支持真实 Range;日志 `E:/Temp/nk-subtitle-language-024/range-fixture.stderr.log` 中中文/英文清单、VTT 与媒体范围请求均为 `auth=true`。最初不支持 Range 的 Python 服务不计入这些结论。所有任务窗口通过界面关闭、进程退出、卸载与夹具停止已确认(`E:/Temp/desktop-subtitle-language-024-cleanup.log`)。这证明当前 Windows 包内的语言切换/关闭。跨播放器重开的持久化已在后续单独补上,见「字幕选择跨播放器重开保持」;变体组刷新后的选择、其他字幕样式与 macOS/iOS 仍不在这些结论内。

## 活动字幕与关闭重开验证(2026-11-03)

`cmd/subcheck` 改为暂停在同一 1.000s、要求真实解码宽度大于零且 seek 完成,只读取 showing 轨的 `activeCues`。旧探针读取 `cues`,包含未来时间的已加载文本,因此旧记录不能单独证明当时有活动字幕。新会话从磁盘重读字幕偏好,不只依赖原书架对象的内存;选择落盘也必须匹配本次选择,而非任意旧记录。探针现支持 `-choose 关闭` 与 `-text`。

真实 Windows WebView2 / HLS.js 1.7.4 的三次独立运行均通过:English、中文、关闭在新会话重开后分别得到英文活动文本、中文活动文本、`value=-1` 且 showing 轨与活动文本均为空;媒体解码宽度均为 320,暂停位置均为 1.000s。日志分别为 `E:/Temp/nk-subtitle-active-english.log`、`E:/Temp/nk-subtitle-active-chinese.log`、`E:/Temp/nk-subtitle-active-off.log`。反例把英文 cue 移到 100–110s,在 1s 仍选中 English 却得到 `en:0` 与空活动文本,探针按预期非零退出(`E:/Temp/nk-subtitle-future-cue-negative.log`),说明未来字幕不能再造成通过。

这是开发工具操作真实 WebView 中的选择框并触发 change,不属于打包 GUI 的人工控件验收或屏幕像素证据。只覆盖本地夹具、磁盘偏好重读与新会话/页面,不声称验证了整个 App 退出重启、变体组切换、macOS 或 iOS。仍沿用已交付 0.2.4 安装包,本轮未打新包。

## 字幕选择跨播放器重开保持(2026-11-02)

此前字幕选择只活在本页:`subtitleChoice` 是页面变量,关闭播放器再打开同一集会回到清单默认轨。字幕选择独立存入书架文件(关闭也是一种选择),播放器页加载时读取、变更时写入。首次播放在进度落盘前选字幕也能保存,不会因此创建观看历史;播放结束清除进度后选择仍保留。旧版嵌在进度记录里的选择会在读取时迁移。这类记录有意不随观看史裁剪,因此另有上限:超过 2000 条时丢最早的,新增用例 `TestVideoSubtitlesAreCapped` 覆盖上限与重开后保留最新记录(`go test ./internal/shelf` 通过)。

更正此前的映射缺陷结论:最初选择 English 却选中中文是开发探针错把选项下标当成选项值,不足以证明产品中存在该缺陷。当前页面以轨道列表映射选项,并使用轨道自身的 id;真实站点的不同轨道排列仍待验证。

证据来自新的开发工具 `cmd/subcheck`:它在真实 WebView2 里开 HLS 会话(仓内夹具 `internal/player/testdata/hls-subs`,英文 `DEFAULT=NO`、中文 `DEFAULT=YES`),用真实书架存偏好,然后**新建会话重开同一集**。第一轮显示默认中文;选中 English 后书架记录为 `{Name:English, Language:en}`,showing 轨中已加载文本为 `Subtitle rendering proof`;重开后的新页面 `selected="English"`、`value=0`、`cues=[en:1]`,仍包含已加载的英文 cue(该旧计数并非活动 cue)(`E:/Temp/nk-subcheck-run3.log`)。窗口截图 `build/probes/subtitle-reopen/reopened.png` 只含原生层,不含 WebView 画面。

2026-11-03 在提交 `3c42463` 上重跑三条:英文(重开后 `selected="English"`、`value=0`、活动文本 `Subtitle rendering proof`)、中文(`value=1`、活动文本 `字幕渲染检查 第0段字幕`)、关闭(`value=-1`、无活动 cue 与文本),三条都解码宽度 320、暂停 1.000s,日志 `E:/Temp/nk-subcheck-3c42463-{en,zh,off}.log`。后续首次播放修复使用未预置历史的 `cmd/subcheck` 重跑通过(`E:/Temp/nk-subtitle-first-play-webview.log`);独立单测覆盖清除进度和旧数据迁移。限制:这是开发工具打开的真实窗口与真实 HLS.js 会话,不是 0.2.4 安装包内的 GUI 点击;夹具在仓内,不涉及在线站点;字幕字号/位置样式、画中画、变体组切换后的选择、macOS 与 iOS 仍未验证。`sh scripts/check.sh` 通过,日志 `E:/Temp/nk-subtitle-persist-full-check.log`。

## 字幕链路(2026-11-02)

此前默认走浏览器原生 HLS:同一夹具在真实 WebView 中**既不请求 `subs.m3u8` 也没有任何字幕轨**(`decoder=native`,`activeSubtitleCues=0`),而强制 HLS.js 时字幕正常。这是产品缺陷而非夹具问题——WebView2 的原生 HLS 忽略 `EXT-X-MEDIA` 字幕组。现在 HLS 会话一律优先内置 HLS.js,只在 HLS.js 不可用时回退原生;`-video-native` 仍可复现回退路径,该路径无字幕是有意接受、并已记录的限制。默认路径实测 `decoder=hls.js`、`activeSubtitleCues=2`、`2.006s` 续播(`E:/Temp/desktop-subtitle-prefer-hls.log`)。

字幕组经媒体代理读取,请求带源声明的 Referer/Cookie。早期 `41a89da` 强制选第一条;本轮改为尊重 `DEFAULT=YES`,没有默认轨时关闭,并增加「字幕」语言/关闭选择框,在列表刷新时保留本页显式选择。

夹具将英文放第一条(`DEFAULT=NO`)、中文第二条(`DEFAULT=YES`)。真实 Windows WebView 的默认轨为中文,`activeSubtitleCues=2`、文本 `字幕渲染检查`、模式 `disabled,showing`,暂停 `2.103s` 后恢复 `2.103s`;日志 `E:/Temp/desktop-subtitle-choice-default.log`。两条都改为 `DEFAULT=NO` 的反例得到 `cues=0`、模式 `disabled,disabled`,未请求字幕清单/VTT,探针以退出码 1 结束(`E:/Temp/desktop-subtitle-no-default.log`)。探针现在只接受 showing 轨的活动 cue,可用 `-video-subs-text` 校验语言,停用轨的缓存 cue 不算通过。

Lead 经真实窗口的「视频→搜索→详情→剧集→播放」操作看到中文字幕叠加,截图 `E:/Temp/desktop-subtitle-visible-window.png`,回执 `E:/Temp/desktop-subtitle-choice-receipt.json`。这补上了中文在该窗口中的像素证据。此前 `foreground_unavailable` 阻止了该轮操作;后续 0.2.4 安装包的语言切换/关闭已按上一节补验通过。**仍未验证**:字号/位置样式、macOS、iOS 与真实在线字幕。`bcbe602` 的旧 0.2.2 包尚无本轮选择框;视频 canvas 截帧仍不包含字幕叠层。

## 音轨解码管线(2026-11-02)

H.264+AAC 的 fMP4 HLS 夹具在真实 Windows WebView 中播放后,`webkitAudioDecodedByteCount` 实读为 **18,537 字节**,画面 `320×180`、鉴权/Range 正常,暂停 `1.975s` 重开恢复 `1.975s`;日志 `E:/Temp/desktop-audio-decoded-counter.log`。探针 `-video-audio` 现在要求实际解码字节数大于零;不提供计数的浏览器只能记为未观测。

原生 MP4 音轨同样通过:把同一带音轨素材的初始化片段与分片拼成一个 `clip.mp4`(`internal/player/testdata/mp4-audio`)后,原生解码路径实读 `audioDecodedBytes=17802`,`decoder=native`,暂停 `2.054s` 重开恢复 `2.054s`,日志 `E:/Temp/desktop-mp4-audio-native.log`;同一方式拼出的无音轨文件得到 `audioDecodedBytes=0` 并以退出码 1 结束(`E:/Temp/desktop-mp4-video-only-negative.log`)。

更强的负向对照故意让清单声明 AAC,实际媒体仍无音轨:得到 `audioDeclared=yes audioDecodedBytes=0`,退出码 1,日志 `E:/Temp/desktop-audio-false-manifest-negative.log`。此前 `8f1d812` 的 `audio=yes` 仅来自清单/编解码器信息,不能证明解码;该旧判据已替换。这次的实际字节计数仍**不等于扬声器实际发声**;原生 MP4 音轨已按上述夹具验证到解码字节计数。

## 视频读取取消(2026-11-02)

视频列表、详情与播放地址准备分别使用可取消的代次。新搜索取消旧搜索/分页并清掉旧详情准备;切源同时取消三类读取;关闭列表停止该列表的 HTTP,关闭详情停止详情与尚未开始的播放准备。列表和独立详情窗各有加载/错误状态,详情完成不能清掉仍在运行的搜索。已开始播放的媒体会话独立存在,不随关闭搜索窗口停止。

分页固定使用生成当前列表的源与关键词,而非用户后来编辑但未提交的输入。切换剧集或线路会取消旧播放准备;返回的播放地址必须在主线程再次确认仍属于当前代次,才能打开播放器。关闭后的列表/详情窗可重建;没有线路的详情不会因索引空数组而崩溃。

`go test -race` 以真实 Core、JS 视频源和 loopback 慢响应测试新搜索、分页被替代、切源、列表/详情关闭、历史续播取消及播放准备替代,要求服务端观察连接取消,随后实际读取成功。另验证迟到的同源同 ID 结果不能清掉新加载状态、列表与详情状态独立、分页不混入新输入词。日志 `E:/Temp/desktop-video-cancel-targeted.log`。这组是控制器/HTTP 验证;另外用真实 Windows WebView 复跑 HLS.js 解码、鉴权/Range、暂停落盘和 `2.051s → 2.051s` 续播,日志 `E:/Temp/desktop-video-cancel-hls-window.log`,不把网络用例当作原生窗口手势验证。

## 视频源安装与重启恢复(2026-11-02)

「源管理」新增漫画源/视频源/Kazumi 规则类型选择、URL 安装与系统文件对话框安装。库层统一负责类型明确的安装、同类型更新和持久化;漫画源默认入口不变,跨类型同名覆盖由共享 Core 拒绝。文件读取与 URL 下载共用 4 MiB 上限。设置页只为 JS 类型显示;视频选择器包含视频 JS、Kazumi 与已有内置视频记录,漫画选择器只列漫画。

**真实 Windows 原生 UI 通过**:独立 APPDATA 下启动正式程序,在「源管理」选择视频源,从系统「选择视频源」文件对话框安装 `hls_probe.js`,再输入 loopback URL 并点击安装 `mp4_probe.js`。窗口显示两个视频源,磁盘 `plugins.json` 保存 `video-js` 类型、来源和启用状态。通过窗口关闭后重新启动同一程序,视频选择器仍列出两者。回执 `E:/Temp/desktop-video-source-manager-ui-receipt.json`,截图 `E:/Temp/desktop-source-installed-both.png`。从这份 GUI 安装的 profile 再运行播放探针(不重新安装源),HLS.js 实际解码与 `2.042s → 2.042s` 续播通过,日志 `E:/Temp/desktop-restored-video-source-playback.log`。

`go test -race ./internal/library ./internal/ui` 覆盖文件安装、URL 更新、停用状态保持、类型冲突拒绝、失败输入不破坏现有记录、从磁盘恢复并启用后实际调用新版本 provider。Core 对停用 JS 更新只暂存新正文,不执行它刷新元数据;启用后才更新版本字段,客户端保留该语义。测试同时修正了误以为停用更新会立刻刷新版本号的假设,未修改共享 Core。Kazumi 的类型入口已实现,此回执不证明真实 Kazumi 站点播放。

## 桌面视频播放器(2026-11-02)

主窗口新增「视频」入口:视频源搜索 → 详情(线路/分集)→ 播放窗口。播放页由 `player.Server` 在 `127.0.0.1` 上提供,会话 URL 带随机 token,页面只能拿到代理资源 ID;页面同时内置 HLS.js 1.7.4(Apache-2.0,`internal/player/assets`),MP4 使用原生解码,HLS 在浏览器原生支持时优先原生,否则使用 HLS.js。观看位置写入 `shelf.json` 的 `videos`(与漫画历史分开,同一 key 不冲突),重开详情时显示已看时长,再次点击从上次位置续播;看到最后 2% 且距结尾 10 秒内视为看完,下次从头开始。

最新真实窗口证据(`cmd/uicheck -video …`,独立新 profile):HLS.js 解码 `320×180`、时钟自行推进,回读真实视频帧 `build/probes/hls-playback/frame.png`;页面暂停事件将 `2.0381s` 写入磁盘,重新打开书架并经「继续观看」实际恢复到 `2.0381s`。主清单、子清单、初始化片段和五个 fMP4 分片全部带源声明的 Referer 和搜索时取得的 Cookie;上游日志记录了 `bytes=0-1122` 等真实字节范围请求。日志 `E:/Temp/desktop-hls-browser-playback.log`。原生 MP4 同样通过鉴权、`640×360` 解码及 `2.0921s → 2.0921s` 真实续播,日志 `E:/Temp/desktop-mp4-auth-real-resume.log`。

此前 `desktop-player-ui-probe7.log` 只能证明 MP4 画面与时钟:探针残留的手工 `3s` 进度 POST 污染了保存判据,而且失败仍以退出码 0 结束。该注入已移除,现在只接受页面自身暂停事件、磁盘重读和重新打开后的实际播放位置。负向验证缺失登录 Cookie 时返回 403,探针以退出码 1 结束,日志 `E:/Temp/desktop-video-negative-check.log`。

本轮修复:MP4 不再依赖浏览器的 HLS 能力选择解码器;保存与打开剧集共用同一完成判断,10 秒短片的两秒进度可续播;位置保留小数秒,HLS 等待 metadata 后恢复位置;「继续观看」使用历史所属源而非当前源选择器。媒体夹具及复跑命令见 [testdata](../internal/player/testdata/README.md),正式程序不依赖 FFmpeg。

普通用户的视频源文件/URL 安装、重启恢复与视频读取取消见前述记录;音轨解码、字幕活动文本及中文窗口显示分别有独立证据。字幕选择框的语言切换/关闭也已在 0.2.4 安装包实测。**未验证**:画中画、macOS 实机运行、在线真实站点与扬声器输出。0.2.2(`bcbe602`)是较早的交付物,最新 0.2.4(`29bab49`)已包含并验证字幕选择框。

## 桌面视频媒体传输层(2026-11-02)

`internal/core` 新增 VideoSearch/VideoDetails/VideoPlay 对共享 Core provider 的直接委托,不在客户端重写解析。媒体读取复用现有传输层的 Cookie 与重定向策略,返回可关闭的响应流,不把完整影片读进 64 MiB 源响应缓冲。视频 JS 的 Network 使用现有 `comic:<key>` 分区,媒体也使用同一分区,真实 Cookie 用例已验证登录状态不会丢失。

`internal/player.MediaProxy` 是每次播放的媒体代理:播放器只能访问代理生成的资源 ID,HLS 清单解析与改写仍调用 Core。loopback 端到端用例覆盖主清单重定向后的相对地址、子清单、AES 密钥、初始化片段、Referer、Range/206/Content-Range;首块数据在响应结束前可读,关闭代理会话会取消上游 HTTP。声明为 `Encrypted` 的媒体(例如未支持的 DRM)、待嗅探与 DASH 明确返回不支持;HLS AES-128 密钥资源按清单代理,不等同于已实现 DRM。

这些传输层用例的历史日志为 `E:/Temp/desktop-media-proxy-tests.log`;当前的界面、HLS 解码与续播证据单列在上节,不把网络单测当作浏览器解码验证。既有漫画请求仍保留整包大小限制;媒体流不复用整包超时,生命周期由调用方上下文与播放会话关闭控制。

## Windows 安装包与便携包 0.2.2(2026-11-02,提交 `bcbe602`)

含字幕轨选择与音轨/字幕判据。便携 ZIP `21,544,794` 字节,SHA-256 `c69db3d7762366784f891dd1cc23e99c97f89378621fcae3d90a7a754f6cf554`;按用户 setup `18,701,040` 字节,SHA-256 `461e4fb02d046f22ffe72014d68a04dfd6899b1f4718e517bd1ca8247e145f74`。

实测:静默安装到隔离目录(`skipRegistration=1`)后 `-version` 报 `0.2.2`/Core `214e7782`/提交 `bcbe6025`;安装后的程序开出标题为 `Nekophage` 的真实窗口并正常响应,关闭后 `Uninstall.exe /S` 删除目录。打包脚本自身通过 ZIP CRC、解压后逐文件哈希与解压程序 `-version`。**未验证**:打包后的 GUI 里逐项点到字幕播放(字幕证据来自同一提交的 `cmd/uicheck` 窗口),以及签名、无 WebView2 的机器。

## Windows 安装包与便携包 0.2.1(2026-11-02,提交 `e27d5f7`)

`python scripts/package-windows.py --nsis <makensis>` 现在生成便携 ZIP(`21,544,447` 字节,SHA-256 `a5e9f549927a2e937ba196e0fb862140d4c2fa39a3c9278ed05747d7b8f5c3b0`)和按用户安装的 setup(`18,697,739` 字节,SHA-256 `58e980bacf3d6337abc8935ddfab950c392d4cf0ed6b2512b2bdc3d8d1e90025`)。打包脚本自身验证 ZIP CRC、解压后逐文件 SHA-256 与解压程序的 `-version`;安装行为单独实测。

实测(回执 `E:/Temp/desktop-installer-0.2.1-receipt.json`):静默安装到隔离目录后 12 个文件齐全、逐文件哈希与 `SHA256SUMS.txt` 一致、`-version` 报 `0.2.1`/Core `214e7782`/提交 `e27d5f79`;正常安装写入当前用户卸载项与开始菜单,安装后的程序开出真实窗口(标题 `Nekophage`,工具栏含搜索/浏览/书架/分类/源收藏/当前源/源管理/AI/视频,截图 `E:/Temp/desktop-installed-0.2.1-window.png`);`Uninstall.exe /S` 后目录、卸载项与快捷方式均删除。`/NOREGISTER` 安装写入 `skipRegistration=1` 且不写注册表与快捷方式;卸载后仍不写;预置在 `%APPDATA%\Nekophage` 的哨兵文件在卸载后仍存在,说明卸载不清用户资料。

未验证:签名与 SmartScreen 声誉、没有 WebView2 的机器、其他 Windows 版本。此包已包含 `0d8ce15`、`30d460f`、`8f1d812` 与音轨字节计数判据,取代下面的 `0.2.0` 旧包。

## Windows 便携包 0.2.0(2026-11-02,提交 `7dc3051`)

视频窗口合入后重新打包:`build/releases/Nekophage-0.2.0-windows-amd64-7dc3051.zip`,**21,520,864 字节**,SHA-256 `18250017734e129ab00ff9573392909aea847b72cc6e6f7c719bebd9ac332322`。ZIP CRC、解压后逐文件 SHA-256、解压后 `-version`(报告提交 `7dc3051`)全部通过。随后在独立 `APPDATA`/`LOCALAPPDATA` 下启动解压出的正式程序:窗口标题 `Nekophage`,1100×760,截图 `build/probes/package-0.2.0/packaged-window.png`;点窗口关闭按钮退出,后置检查确认进程结束。**没有动用户真实配置目录。**

打包时发现并修掉一个真实的界面缺陷:新加的「视频」按钮把工具栏挤出了 1100px 的默认宽度,右端被裁掉。收窄源选择、模式切换与搜索框后,开发构建与打包构建的窗口截图都确认最后一个控件完整可见。日志 `E:/Temp/windows-package-build-0.2.0b.log`、`E:/Temp/nk-package-window-0.2.0b.log`。

这仍是未签名便携包,不是安装程序(本机没有 Inno Setup / NSIS,也没有代码签名证书);没有远端可推。

## Windows 便携包验收(2026-11-02,提交 `dbbaf0a`)

`python scripts/package-windows.py` 生成 `build/releases/Nekophage-0.1.0-windows-amd64-dbbaf0a.zip`(20,755,698 字节)。ZIP CRC、逐文件 SHA-256、解压后 `-version` 全部通过,程序报告 Core `214e7782190d` 与提交 `dbbaf0a`。另直接启动解压后的正式程序,只在子进程中改用独立 APPDATA/LOCALAPPDATA:宿主窗口回执确认标题 Nekophage 及搜索/浏览/书架/漫画源/AI 控件存在,通过窗口关闭按钮退出且后置检查确认窗口消失。

这是未签名便携包,不是安装程序,没有推送远端。窗口截图 `build/probes/package-dbbaf0a/packaged-window.png`;打包日志 `E:/Temp/windows-package-build.log`,界面回执 `E:/Temp/windows-package-ui-receipt.json`。Windows 本机证据不代表 macOS 或另一台 Windows 已验证。

## 当前 Windows 窗口复跑(2026-11-02,代码 `00a1055`)

用独立测试 profile 与 loopback 图片夹具运行 `cmd/uicheck`:原生详情实际显示 `Probe Comic`、封面、标签与 Chapter 1,并打开阅读器。磁盘 `shelf.json` 读回 `callback_probe / Chapter 1 / page 1`。独立 `cmd/readercheck` 再次验证 `figures=1 images=1 decoded=1 first=8x8`。这是当前取消/缓存改动后的真实 Windows 窗口与解码证据,不使用截图体积推断阅读器内容;夹具不证明真实站点当前可用。

日志 `E:/Temp/desktop-delivery-ui.log` 与 `E:/Temp/desktop-delivery-reader.log`,原生详情截图在 `build/probes/delivery-00a1055/detail.png`。新增 `scripts/package-windows.py` 生成未签名便携 ZIP,并验证解压后的文件校验与 `-version`;窗口验证与打包检查分别记录,不把命令行启动等同于界面通过。

## 页面读取取消(2026-11-02)

代次取消已扩展至浏览、分类/排行、详情、标签与详情中的源收藏读取。切换模式、切源、关闭详情或打开另一部作品时,过期读取不会继续等待网络,也不能把迟到的内容写回新页面。详情的评论能力检查移到后台并共享详情上下文,避免在 UI 锁内等待 Core。切换源时同时清掉旧分类声明;从多源结果点入详情仍保留原网格,返回不丢结果。

已完成的列表在切页后保留,被中断的列表回到页面时重新加载,避免留在永久加载状态。源收藏写操作不随离开详情取消。真实 Core + JS + loopback HTTP 分别验证详情、浏览和分类的服务端连接取消、旧调用退出与下一次读取成功;状态回归覆盖切源清理与已完成/未完成列表的区别。

## 图片缓存与下载预算(2026-11-02)

下载缓存上限 32 MiB、400 条;位图缓存上限 128 MiB、160 条,先达到哪项就淘汰最旧条目。计费按下载切片的容量,并为解码位图计算 RGBA 与完整缩小图链。MyGo `DecodeBitmap` 的闭包会保留压缩数据以便重新解码,所以那份数据也计入位图预算;之前只把下载 map 清空并不代表它已释放。超单项预算的图片使用占位并保留失败标记,避免反复下载/解码;解码前检查尺寸,避免先分配超预算的像素。预算不含 GPU 纹理与全部进程开销,不宣称进程内存峰值已受限。

封面与收藏页图片共用 4 个下载槽位;排队时取消不发起下载,结束后释放槽位。loopback HTTP 用例验证峰值并发与后续下载能继续,竞态测试覆盖队列取消。缓存测试覆盖字节淘汰、替换计费、压缩数据保留、奇数尺寸缩小链、坏图释放、超大头尺寸与缓存键重访;条目淘汰测试同时修正了原先 `a-0` 与真实键 `a/0` 不一致的无效断言。

## 搜索取消(2026-11-02)

搜索代次现在拥有可取消的 `context`:新搜索会停止旧搜索的 HTTP 工作,切换源或离开搜索模式同样停止旧请求。粘贴链接、单源分页、多源搜索都使用该代次的上下文;多源任务在等待并发槽位时也能退出。每页保留独立超时,一页结束不取消另一页。过期票据若延迟启动,收到的也是已取消上下文,不会误加入新搜索。

`TestSearchReplacementStopsHTTPAndLeavesNewRequestUsable` 用真实 Core + JS 夹具 + loopback HTTP 验证服务端观察到旧连接取消、旧 Core 调用返回错误、同源下一次搜索返回新结果;没有用 sleep 推测取消完成。代次用例另覆盖延迟启动、分页隔离、离开搜索模式。本段记录搜索修复;浏览/详情和封面限制见上方后续验证记录,不据此声称任意写操作都会取消。

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

**同一个 AI 窗口在 Core `d81445f` 上重测(2026-11-01)**:`uicheck -ai -ai-test -ai-model group/laincat -ai-prompt …` 连到本机 Magpie(`127.0.0.1:3425`,53 个模型或路由组),界面出现真实的流式回答「我在验证结果是否符合你的要求，以及关键行为是否按预期工作。」。用的是真模型、真流式,不是假服务。

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

**删除收藏会静默失败:少了第 4 个参数(2026-11-01)**:六个官方源的 `addOrDelFavorite` 有第四个参数 `favoriteId` —— 源**自己的**收藏条目句柄。`wnacg` 的删除直接用它拼 URL:

```js
Network.get(`${this.baseUrl}/users-fav_del-id-${favoriteId}.html?ajax=true...`)
```

我们只传三个参数,于是删除请求实际打到 `del-id-undefined`,而它仍返回 200、源也返回成功 —— **看起来删掉了,其实没有**。这类"成功但不生效"比报错更难发现。

现在 Core 解析 `Comic.favoriteId` 与 `details.favoriteId`(两处都可能带),`SetFavorite` 始终传第四个参数:有句柄就传,没有就是 `null`(不传的源本来就忽略它)。假源单测钉住了两种形态:`["c1",null,false,"fav-77"]` 与 `["c1","f2",true,null]`。

**边界**:未登录时源不返回 `favoriteId`,所以真实的 `wnacg` 删除无法在本机验证 —— 只有单测证据。

**收藏到源:从"只能看"到"能加能删"(2026-11-01)**:上一轮只把源的收藏**列出来**,详情页没有入口。现在详情页在「收藏」(本地书架)旁边多一个「收藏到源」:显示源是否支持、多收藏夹时给文件夹选择、已在该收藏夹时按钮变成「已在源收藏」再按一次即移除。源抛的错误按原文显示。

几个刻意的选择:

- **两种收藏分开**:本地书架与源收藏是两份数据,各有各的按钮,不合并成一个"收藏"键。
- **单收藏夹的源不传文件夹**:实测 `wnacg` 的 `add`/`remove` 在未登录时都返回成功(源自己接受),`nhentai` 则回 `Authentication required. Please login again or provide API Key in source settings.` —— 客户端原样转述,不判断哪种"应该"。
- **只声明收藏夹切换的源**(`ehentai`)不出现在详情页的添加入口,因为它没有列表可读,加进去也无处可看。

单测覆盖参数形态:单收藏夹源必须收到 `null` 而不是空串;改名/移除的收尾都验证过。截图确认按钮与文件夹选择渲染正常。

**又一(N)轮读取巡检(2026-11-01,pin `cf9eb9b`)**

34 源重跑:**20 通过 / 11 失败 / 3 需自建服务**。与上一轮相比有两处变化,都不是客户端回归:

- **`jm` 稳定通过** —— 上一轮修的 BOM 容忍起效了(以前它连 init 都过不去),现在搜索、详情、章节、取图整条走通。
- **`manga_dex` 这次失败**:实测 `api.mangadex.org` 现在返回 **302 → "Redirecting to bot challenge page"**,源拿到 HTML 就去 `JSON.parse`。这是站点侧的机器人防护,不是我们的传输层坏了;我们的挑战验证器只在 Cloudflare 式的 403 上触发,**302 跳转到挑战页这类没有覆盖**,结论是"站点改了",不是"客户端可用"。
- **`ikmmh` 这次搜索 0 条、首页也空**,上一轮通过 —— 典型的单轮不可结论,已在文档里保留这个例子。

`manga_dex` 这条值得单独留证 —— 实测响应就是:

```
HTTP/1.1 302 Found
location: https://mangadex.org/edge/challenge?src=…
content-type: text/plain
server: MangaDex

Redirecting to bot challenge page... (if this isn't a browser, don't send a browser UA)
```

也就是说它既不是 403、也没有 `cf-mitigated`,而是**自己的 edge 挑战**:302 → 文本提示页。我们现有的挑战检测刻意保守(只在 403/429/503 + `Server: cloudflare` + 页面标记时才算),**覆盖不到这一种**;而"把 302 当成挑战"需要一个新的挑战类型与完成规则(现在只有 cloudflare / captcha 两种 DoneRule,没有"离开某路径"这种判据),不是一行判断能补的。所以这里如实记为**站点侧新机制、客户端未覆盖**,不做半成品。

仍失败的其余源与之前归类一致:`picacg` 需登录、`mh18`/`mycomic` Cloudflare、`baozi` 403、`comic_walker`/`shonen_jump_plus` 返回 HTML、`happy` 搜索 EOF、`jcomic` 站点把图地址移进需 JS 解密的 `data-locked`、`mh1234` 连接超时。

**「2 条回复」是个数字,点了没反应(2026-11-01)**

评论里 `replyCount` 一直只是文字。真正的回复要**用同一个 `comic.loadComments` 再调一次**,把 `replyTo` 设成那条评论的 id —— 两个客户端都只读了第一层。

- 桌面端:「N 条回复」变成按钮,点了读那层回复,顶部显示「回复 X 的评论」和「返回全部评论」;回到线程本身时 `replyTo` 传 `null`(不是空串,源按 null 判断)。
- 单测钉住这条:`replyTo` 为 null 读线程,传 id 读回复,内容能分辨。
- 截图 `E:/Temp/nkprobe/ui-comments-thread.png` 显示「2 条回复」已成按钮。

**未做**:iOS 的同一处刚接上,但只有语法/键核对;`replyCount` 里没有账号维度的问题,但**真实源的回复层仍需登录才能读到**,这点不当作已验证。

**评论能发不能赞(2026-11-01)**

4 个源声明了 `comic.likeComment`(`ccc` / `lanraragi` / `picacg` / `zaimanhua`),契约是 `likeComment(comicId, subId, commentId, isLike)` —— **同一个函数、传 `false` 就是取消赞**。Core 层早就实现了,界面上一直没有按钮。

现在评论区每条评论带「赞 / 取消赞」,只在源声明了 `likeComment` 时出现;点完重新读当前页,因为源可能只在下一读里给出新的赞数。

**活体验证**:`ccc` 与 `zaimanhua` 都声明了赞(`canLike: true`);它们的评论列表要登录才能读,所以**投票的端到端仍需真实账号**,这一点不当作已验证。

**粘贴源自己的链接,客户端当关键词搜(2026-11-01)**

8 个源声明了 `comic.link`(域名 + `linkToId`),用来把「https://nhentai.net/g/691529/」这类地址解析成作品 id。两个客户端都没接:搜索框把整串地址当关键词,结果当然是 0 条。

- Core 客户端层:`LinkDomains` 读源声明的域名(**用 `Runtime.Value` 读属性,不是 `Call`** —— `comic.link` 是对象,当函数调用只会得到 "source does not define this function");`LinkToID` 让源自己解析;`OpenLink` 按域名匹配后逐个问,第一个答得出来的源胜出。
- 桌面端:搜索框里像地址的输入先走 `OpenLink`,认出就**直接打开那部作品**;没人认得时明说「没有源认得这个链接」,再按关键词搜也不迟。
- 判定「像地址」不只看有没有点:`12.5`、`v1.2` 这类带点的关键词必须留在关键词一侧,否则每次搜索都白跑一轮源调用。

**活体验证**:`nhentai` 链接 → id `691529` → 详情页正常打开(标题、封面、标签都来自源),截图 `E:/Temp/nkprobe/ui-link-open2.png`。顺带看到一个不存在的 id `177013` 会如实报 404,而不是假装打开了。

**未做**:iOS 侧还没接 `linkToId`;`enableTagsTranslate` 仍未接。

**点标签遇到分类就断在半路(2026-11-01,pin `0c721f3`)**

6 个源声明了 `comic.onClickTag`,桌面端一直在用,但**「分类」这一支从来没走通** —— 点了只弹一句「当前版本还打不开分类参数」。iOS 则连标签都不可点。

这轮把桌面端走通:标签被源归到分类时,**切到分类页并选中源点名的那个参数值**;参数在当前声明里找不到时明说,而不是悄悄落到第一个值。测试用一个 fixture 钉住这两条(选中的是 value 0、未知参数会出提示),并顺手记下两处真实契约细节:`params` 在声明里叫 `categoryParams`;声明对象是 `category`,`categoryComics` 只是加载器。

Core 侧加了 `ClickTag(callID, key, namespace, tag)`,给手机端同一份语义(含「源没意见」时返回空 action 而不是报错,以及源抛错时把原话记进日志)。

**未做**:iOS 的标签点击还没接(`LinkToId`、`enableTagsTranslate` 也仍未接)。

**20 个源有评论,客户端一处入口都没有(2026-11-01)**

官方源里 **20 个声明了 `comic.loadComments`,13 个声明了 `comic.sendComment`**,而两个客户端从来没有调用过它们 —— 这些源的评论区在客户端里等于不存在。

- **路径一开始就查错了**:评论函数挂在源的 `comic` 对象下(`comic.loadComments`),不是顶层 `loadComments`;按顶层查会把「有评论」的源全报成「没有」。
- Core 客户端层新增:`CommentsSupported` / `CanSendComment` / `CanLikeComment` / `Comments` / `SendComment` / `LikeComment`;`Details` 补上 `subId`(部分源的评论线程用它定位作品),空 id 传 `null` 而不是空串。
- 桌面端详情页多一个「评论」按钮(源没声明就不显示),评论区独立窗口:列表、翻页、回复某人、发送、刷新。`canSend` 在开窗时读一次 —— 放进渲染路径等于每次重绘都跑一次源调用。

**活体验证**:`komiic` 搜索→详情→评论,读到 3 条真实评论(用户名、时间、正文),截图 `E:/Temp/nkprobe/ui-comments-komiic.png`;`nhentai` 声明支持但未登录时返回 0 条,属账号维度,不是客户端问题。

**未做**:iOS 侧还没有评论 UI(这轮只做桌面端),也没有 Mac 编译验证。

**浏览巡检变成一条命令(2026-11-01)**:`NK_LIVE_DIR=/path/to/scripts scripts/live-explore.sh` 跑 `cmd/explorecheck`,逐源列出浏览页并加载第一页;失败按源逐条报,需要自建服务的源单列,`NK_LIVE_REPEAT=2` 可复跑以区分「站点偶发」与「稳定坏」。

同时清掉了桌面端 `internal/core` 里已成死代码的 `exploreType` / `parseParts` / `parseOrderedParts` —— 留着只会让下一个人从这里再抄出第二份实现,而"两端各一份"正是上一轮 iOS 浏览页坏掉的原因。

最近一次全量(34 源):**23 通过 / 8 失败 / 3 需自建服务**。

**浏览页的归一化移进 Core,两端不再各写一份(2026-11-01,pin `c855fd3`)**:上一节 iOS 那条缺口(自己拼 `explore[i].load`)的根因是**同一段逻辑在两端各有一份**。现在四种浏览形状与 `load`/`loadNext` 回退都在 Core 的 `comic/js` 里,桌面端只把每个 comic 映射成界面要的形状。

修完后又跑了一遍全量浏览巡检:**23 通过 / 8 失败 / 3 需自建服务**,`ehentai` 三页可用,`mh1234` 这次通过(上一轮是站点超时,印证了单轮结果不能当结论)。

**「浏览」页从来没被活体验证过,而且对一类源完全不可用(2026-11-01,pin `110b0d8`)**

34 个官方源**全都**声明了 `explore`,但活体测试只走搜索→详情→章节,浏览页一眼都没看过。写了 `cmd/explorecheck` 逐源列出浏览页并加载第一页,第一次跑:

**24 通过 / 10 失败**,其中 `ehentai` 三个浏览页全部报 `source does not define this function`。

原因是**契约形状**:官方 API 允许浏览页用 `load(page)`,也允许用 `loadNext(next) → {comics, next}`(游标式)。`ehentai` 只声明了后者,而我们永远只调 `load` —— 浏览页存在,客户端问错了函数。有 5 个源用 `loadNext`:ehentai、hitomi、jcomic、manhuagui、manhuaren。

- Core 客户端层新增游标支持:`Explore` 发现 `load` 不存在时改调 `loadNext`,`ExploreResult.Next` 带回下一个游标;桌面端「下一页」按游标翻,没有页码时不显示页码。
- 顺带修掉一个放大伤害的行为:**没有 UI handler 时,没有返回值的 UI 调用(showMessage / launchUrl / cancelLoading)不再报错**。`ehentai` 的「watched」页在未登录时先 `showMessage` 再返回空页;以前整个调用失败,现在按源的意图返回空页。需要答案的对话框(showDialog / showInputDialog)仍然报错 —— 那个答案决定源接下来做什么,不能编。

修完 `ehentai` 三个浏览页全部可用(25 + 77 + 0 部,登录态相关的那页按源意图返回空)。`kavita`/`komga`/`lanraragi` 在探针里单列为 SKIP(需要用户自建服务),不再混进失败。

**同一处缺口在 iOS 也存在(2026-11-01,pin `7019844`)**:mobile 层的 `CallComic` 原本失败也不记录,已按与桌面端相同的形状补上(源 key + 调用路径 + 原因);负向验证:把那处判断改成 `if false`,测试立刻失败。桌面端只跟 pin。

**面板在真正需要它的时候是空的(2026-11-01)**:把「源日志」拿真实失败源(`jcomic`)去跑,截图是 **0 行** —— 因为那个源根本不打日志,错在 Core 侧(`image URL is not http(s)`)。也就是说,面板只在源自己 `console.log` 时有用,而用户最需要它的时刻恰恰相反。

- `Core.Call` 失败时记录一行:源 key + 调用的路径 + 原因。「搜索失败」不再只剩一句被下一屏覆盖的弹窗。
- 图片地址不是 http(s) 时记录**它拿到的那个字符串**(带引号,空串也看得出来是空串)。之前读者只看到「不是 http(s)」,却不知道源到底给了什么。

两条测试都钉住:源抛错后缓冲里最后一行带 key 与 `search`、且保留原因;空图片地址被拒时记录该源与该消息。

**源自己的话一直看不见(2026-11-01)**:巡检时我能看懂 `Failed to convert base64 ...`,是因为我在终端里跑;普通用户从启动器打开,**这行根本不存在**。他们只看到源随后抛出的「Cannot read properties of null」,真实原因被藏起来了 —— 而这正是「是客户端坏了还是站点变了」这个问题的答案。

- Core 把源日志(源 `console.log` 与 Core 代表源产生的错误)**按源 key 归属**,内存里保留最近 400 行。`Services.LogSource` 是可选回调,只设 `Log` 的宿主行为不变。
- 桌面端新增「源日志」面板:时间、源 key、级别、正文;error 红、warn 黄;可「复制全部」贴进 bug 报告,可「清空」。
- 内存有上限:在循环里打日志的源不会撑爆进程(测试打 500 行,确认只留 400 且**保留最新的**)。

视觉证据:`E:\Temp\nkprobe\ui-source-logs.png`(5 行,含一条被正确归属到该源的 base64 转换失败);`E:\Temp\nkprobe\ui-source-logs-baozi.png`(真实站点 403:`Core: search.load: js: Invalid status code: 403`,归属到 `baozi`)。

**真实源第 N 轮巡检:一个 BOM 让整条源失效(2026-11-01,pin `a93ac09`)**

34 个官方源重跑:**20 通过**、11 失败、3 个需要自建服务(单列 SKIP)。逐个看日志归类,发现一个真问题:

`jm` 的 init 从域名列表接口取一段 base64。这个响应现在**开头多了一个 UTF-8 BOM**,`base64` 在**第 0 字节**就失败 → `Convert` 按官方行为记日志并返回 null → 源拿到的数据是 null,`JSON.parse(null)` 之后在 `null.length` 上崩掉,错误信息于是变成「Cannot read properties of null (reading 'length')」,而它自己的**兜底域名列表根本没机会执行**。

修法是只去掉不可能属于 base64 字母表的东西:BOM 与 ASCII 空白。合法输入不受影响,垃圾输入照样报错。修完后 `jm`:安装 ✓、浏览 ✓、详情 62 话 ✓、第 1 话 118 张图 ✓、首图实取 10,098 字节 webp ✓;只剩分页端点站点返回 500。

其余失败按维度归类,没有一条指向客户端:

| 源 | 归类 |
| --- | --- |
| `picacg` | 需要登录 |
| `mh18` `mycomic` | Cloudflare 挑战 |
| `baozi` | 403 |
| `comic_walker` `shonen_jump_plus` | 返回 HTML 而非 JSON |
| `happy` | 搜索接口 EOF |
| `jcomic` | **站点改动**:页面 `src=""`,真实地址进了需 JS 解密的 `data-locked`(已抓页面确认) |
| `mh1234` | 本机连接超时,且源假定 `#total-page` 一定存在 |
| `jm` | 分页端点 500(原 init 崩溃已修) |

**导出可能写出自己都读不开的文件(2026-11-01,pin `9013ca5`)**:读端对 `comic_source/` 下每个文件都有 16 MiB 上限;写端只校验脚本,**没校验 `.data`**。所以一个存了很多数据的源,导出会成功,但生成的文件用 Core 自己的读取器打不开 —— 这种失败最糟:用户以为备份好了,等要恢复时才发现是坏的。

- 新增 `MaxWriteDataBytes`,与读端 `MaxSourceBytes` 同值;超限直接报错并**点名是哪个源**。
- 桌面端把这条错误翻成一句话:「这个漫画源存的数据太大,写进备份后会读不回来,所以没有导出」。
- 另有一条测试**手工构造**一个超过上限的成员,确认读端确实会拒绝 —— 用来证明这条上限是真实边界,不是我猜的。

**源的设置也没带走 —— 现在带走并按声明分回去(2026-11-01)**:上一条只解决了 `.data`(源自己存的),每个源的**设置**(用户在设置面板里选的)仍然不导出。

矛盾的根源是形状不一致:归档**每个源只有一个对象**,而这个客户端把它拆成 `data` 与 `settings` 两个桶。合并导出很容易,难点在导回来时怎么知道哪个键属于哪个桶。

答案用源自己的话:**源在 `settings` 里声明过的键就是设置,其余是数据**。所以导入改成 `Library.ImportSourceValues`:先读 `core.Settings(key)` 拿到声明键,再按键分流;源加载失败时全部按数据处理(与旧导入行为一致,不会更糟)。导出则把两个桶合并成一个对象,同名时数据优先 —— 数据是源自己写的。

验证:一个声明了 `settings = {domain: ...}` 又存了 `token` 的源,导出后导入到空客户端,`token` 在数据桶、`domain` 在设置桶,**并且有一条反向断言确认 `domain` 没有被当成数据**。这条同时修复了"官方备份里的设置被我们塞进数据桶"这个老问题。

**导出只带走了代码,源的登录态没走(2026-11-01,pin `e46c8f9`)**:上一版导出只写 `comic_source/<key>.js`。但归档格式里还有 `comic_source/<key>.data` —— 源自己存的东西(令牌、选定的域名)。读取器一直解析它,导入端也一直会写回去,**只有写端从来没产出过它**。结果就是"恢复成功了,但源醒来是登出状态"。

- Core `WriteSource` 新增 `Data`;有内容时写成 `<key>.data`,与读取器按 key 配对的规则一致。
- 桌面端从 store 取每个源的整个数据对象带进导出。
- iOS 侧 Core 无法枚举宿主的 store,所以 `ExportBackup` 的源数据由客户端传 `{"<key>":{"<name>":<value>}}`;Kit 从 `SourceStore.allData(key:)` 逐源收集。

验证是**两段闭环**:桌面端"有源的客户端导出 → 空客户端导入 → 源在";以及"源存了 `token` → 导出 → 另一客户端导入 → `LoadData` 读回原值"。

**仍然没带走的**:每个源的**设置**(与原客户端的数据分属两个桶)。合并进 `.data` 会让导入端把设置当成数据写回去,这不是修复而是换个地方错;留作后续单独处理。

**备份只能进不能出,现在能导出了(2026-11-01,pin `ffdfbe7`)**:Core 一直能**读** `.venera` 备份,但没有写的能力 —— 用户能迁进这个客户端,迁不出去。

- **Core** 新增 `backup.Write`:按读取器已经定义的格式写 `comic_source/<key>.js` 与 `appdata.json`;键顺序固定,同一份数据两次导出**逐字节相同**(这样"有没有变"这个问题才答得上来)。
- **刻意不写**:`cookie.db`(分区凭据,而导出文件是会被发到自己邮箱的东西)、`history.db` / `local_favorite.db`(官方 App 的 SQLite schema,本客户端的书架是另一种格式,写半个别人的数据库只会被读成损坏)。
- **桌面端**「导出 .venera 备份」按钮在导入旁边,写源与搜索历史,文案直接写明「不含 Cookie」。

验证是**往返**:写出文件后用 Core 自己的读取器读回来,确认源、脚本、设置与搜索历史都在,并确认压缩包里**没有** `cookie.db`;另有一条测试确认两次导出逐字节一致。

**边界**:这个文件能被 Venera 兼容客户端读出源与设置,但**不是官方 App 数据的完整克隆**(书架与图像收藏不迁);官方 App 读我们导出的文件这件事**没有验证过**,因为这台机器上没有它。

**源给的图片(验证码/二维码)之前被丢掉了(2026-11-01)**:`UI.showInputDialog(title, image, validate)` 的第二个参数是**图片** —— 源用它展示验证码或二维码。之前桌面端直接忽略,输入框就少了一样必需的东西。

现在图片通过**源自己的封面规则**取(`onThumbnailLoad`),所以源的请求头与图片改写照样生效;字节由对话框自己持有和解码 —— 不放进主窗口的图片缓存,因为那个缓存由主窗口线程读、这里写,共用就是竞态。

中间踩了三个坑,都记下来:

1. 第一版把字节写进主窗口缓存、绘制时从那里读 → 竞态,而且窗口不会重绘;
2. 改成处理器自持字节后,图片仍不显示 → 查日志确认**绘制确实重跑了、字节也确实到了**(`picture=true`),问题在布局:`Fill()` 列里只给 `Height(96)` 可能算出 0 宽,图片于是消失;
3. 改成显式 `Width(200).Height(96)` 后正常,截图里能看到图片。

**图是 httpbin 的测试图**,不是某个源真的验证码,但取图、鉴权、解码、显示这条链路与源驱动时完全相同。

**对话框窗口的排版也看过了(2026-11-01)**:前一节只证明"回调能跑完",没证明**那个窗口长什么样** —— `jsui.go` 里的窗口是从零写出来的,没有任何截图证据。

抓图工具加了 `-dialog`:它用同一个处理器开一个真实对话框(两个按钮),关掉自动确认,截图后关闭窗口。第一次拍出来按钮挤在左下角、内容区没用满;改成按钮靠尾端(和桌面端对话框的习惯一致)后重新拍,确认排版正常。这张图是**合成对话框**,不是某个源弹出来的,但窗口、按钮、回调链路都是同一份代码。

**对话框回调 panic 不再把脚本挂死(2026-11-01)**:对话框里的按钮执行的是**源自己的代码**。之前 `Invoke` 若 panic,欠结果的那个 goroutine 就没了,而调用方还阻塞在 `ShowDialog` 上 —— 源从此卡住,界面上只看到一个再也不动的窗口。现在 `invokeSafely` 把 panic 变成错误返回给源,和源自己 `throw` 走同一条路;四条单测覆盖 panic、源自己的错误、没有回调、以及校验函数 panic。

顺手补上 `ShowLoading` 的可见性:之前只是记账,主窗口没有任何反馈。现在只要源还持有加载号,主窗口就显示一行「源正在处理…」+ 转圈,取消后消失(重复取消不会把计数压成负数,也有测试)。同时给处理器加了 `app == nil` 的防护 —— 处理器可以早于窗口存在、也可能晚于窗口被调用,不该因此让源的调用崩掉。

**在真实进程里又验了一遍,并修掉一处潜在死锁(2026-11-01)**:上面那张表用的是测试处理器,不等于 `jsui.go` 本身跑得通。给抓图工具加了 `-callback <键>`:它在**真实 App 进程**里对这同一个 Core 调 `settings.<键>.callback`,窗口、线程和处理器都是生产路径。

```
$ go run ./cmd/uicheck -script wnacg.js -callback refreshDomains
nekophage: callback refreshDomains -> "null" err=<nil>
$ go run ./cmd/uicheck -script happy.js  -callback wipeCache
nekophage: callback wipeCache -> "null" err=<nil>
```

写这一段时发现自己代码里的一处**潜在死锁**:自动确认路径原本在 JS 线程上直接 `actions[0].Invoke()`,而那个线程正是阻塞在 `ShowDialog` 里的调用方 —— 回调一进来就又调回引擎。改成在另一个 goroutine 里执行并等它,和手动按按钮的路径一致。

**桌面端现在真的能给源一个 UI 了(2026-11-01)**:`Core` 一直有 `UIHandler`(消息、对话框、输入框、选择框、加载),但**桌面端从未提供实现**,于是任何调用 `UI.showDialog` / `UI.showMessage` 的源都会拿到 `No JavaScript UI handler provided`。这不是装饰性问题:有些源**把活干在对话框回调里**。

`internal/ui/jsui.go` 现在提供它:`ShowDialog` 按源给的按钮建窗口,按下后执行**源自己的回调**,关窗则让 promise 以 null 结束(不会把脚本挂着);输入框/选择框接上源自己的校验,不通过就把源的话显示在输入框下面;消息落到主窗口提示,链接交给系统浏览器。抓图工具显式开 `SetAutoConfirm(true)` —— 它没法按自己正在拍的那个窗口里的按钮。正常运行时不会打开这个开关。

**真实源验证**(带一个"按第一个按钮"的测试处理器):

| 源 | 回调 | 之前 | 现在 |
| --- | --- | --- | --- |
| `wnacg` | `refreshDomains` | `No JavaScript UI handler provided` | 完成,返回 null |
| `happy` | `wipeCache` | 同上 | 完成,返回 null |
| `ccc` | 账号登录 | 同上 | **跨过 UI 阻塞**,进到真实请求,改成站点回 HTML |

`ccc` 那一行说明改动确实解开了客户端侧阻塞:剩下的失败已经不是"客户端没给 UI"。

同时 `callback` 类型的设置项(9 个源声明)从"客户端暂不支持"变成真正的「执行」按钮:Core 在源上下文里调 `settings.<key>.callback`,结果或源的报错按原文显示。

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

桌面端现在在源设置面板里按声明画:密码方式给账号/密码/登录/登出;Cookie 方式按 `fields` 逐个给输入框 + 「校验并保存」,由源自己的 `validate` 判断并写入 Cookie(空值一律被源拒绝,不会静默通过);**只提供网页内登录的源**(`nhentai`)当时只能显示「桌面端暂不支持」,而不是给一个按不动的表单。`ehentai` 正是这种源 —— 它根本没有密码登录。**这一条后来补上了**,见下面的「网页内登录」一节。

源抛出的错误按原文显示:`picacg` 是 `Failed to login`,`ccc` 是 `No JavaScript UI handler provided`(它的登录要宿主提供 JS UI 会话,这是当前边界,不是客户端 bug)。活体测试 `TestLiveAccountStyles` 覆盖三种方式。

**网页内登录:5 个官方源用源自己的页面登录,客户端原本只能显示「暂不支持」(2026-11-01,pin `466b26f`)**:`nhentai`、`ehentai`、`ccc`、`mycomic`、`manga_dex` 都声明 `account.loginWithWebview {url, checkStatus, onLoginSuccess}`。三个部分都不能跨语言边界传:`checkStatus`/`onLoginSuccess` 是函数,所以 Core 用 handle 命名它们,客户端把 handle 传回来。桌面端在源设置里给出「在网页里登录」,窗口是挑战页用的同一个 WebView,源的 `checkStatus(url, title)` 决定何时算登录完成,完成后再跑 `onLoginSuccess`。

两条容易写错的地方已单独固定:

1. **交接触发顺序**:`ehentai` 的 `onLoginSuccess` 通过 `Network.getCookies("https://forums.e-hentai.org")` **从源分区读回** Cookie 再写进 `.exhentai.org`,所以必须先存 Cookie 再调回调。正反两向都测了(存了就有、不存就空)。
2. **Cookie 归属**:登录页会加载第三方,把整个浏览器存储倒进源分区等于让源背上它没要求的登录态。现在只留登录站点自己的 Cookie(浏览器规则:前导点=该域及子域、无点=精确匹配),并且**开始页和结束页两个域都算**(`manga_dex` 从 `forums.mangadex.org` 登录后落在 `mangadex.org`),同时命中的只存一次。

`mycomic` 的登录页当场被判为"已完成",这不是缺陷:它的"登录"本质是过 Cloudflare 挑战,落在真实页面即为成功。

另外两端都补了:登出与登录反馈不再只出现在密码方式下(`ehentai` 这类纯 Cookie 源原本算出了「这组 Cookie 不能用」却从不显示,也点不到登出)。

桌面视觉证据(2026-11-01):`ui-webview-login.png` 的 `nhentai` 设置区同时画出「在网页里登录」与「登出」;`ui-webview-ehentai.png` 的 `ehentai` 设置区同时画出网页登录、四个 Cookie 输入框、「校验并保存」与「登出」;`ui-callback-wnacg.png` 的 `wnacg` 设置区按源自己的 `buttonText` 画出 `Refresh`。三张图都在 `E:/Temp/nkprobe/`,由 `cmd/uicheck -settings` 在真实源上捕获。**新增两张对比图**:`ui-cb-quiet2-settings.png` 是修复前 —— 只按了 `Quiet` 一个回调,`Run` 行却同样显示「已完成」(`callbackMsg` 是面板级单一文本);`ui-cb-quiet3-settings.png` 是修复后,只有被按的那一行显示结果。另一张 `ui-wnacg-cb-settings.png` 是真实源上按下 `Refresh` 的结果:`刷新域名列表` 行显示「已完成」,其余设置行不受影响;`-callback` 现在走按钮用的同一个入口,所以截图里看到的就是用户会看到的。

**网页登录现在有整条链路的实测,不只是截图(2026-11-01,`455d0ec`、`487b15a`)**:`cmd/loginprobe` 起一个本地假登录页(`/login` 标题 `Sign in`,自动跳到 `/done` 标题 `Logged in` 并下发 Cookie),让窗口、WebView、轮询、源自己的 `checkStatus`、Cookie 导入源分区、`onLoginSuccess` 全部在无人值守下跑一遍。五种模式各自验证一件事:`ok` 保留 1 个 Cookie 且回调里 `Network.getCookies` 读得到;`timeout` 在源永不同意时以错误结束且**不留下半个登录**;`failcallback` 回调抛错时**立即报出源自己的原因**;`latecookie` 在页面先同意、Cookie 后到时继续轮询;`close` **关掉窗口立即结束等待**。

最后一项正是这一线程最初的缺陷,现在有端到端证据:给 `LoginWindow` 加了 `OnWindow` 钩子(仅工具使用),探针在页面加载 3 秒后真正关掉窗口——等待在 **3.4 秒**内以「窗口在源同意前被关闭」结束、什么都没存。**负向验证**:去掉 `OnClosed` 接线后,同一探针报 `the wait outlived the window by 1m0s — it kept polling a closed page`。

第三项是这一轮抓到的缺陷:回调抛错时客户端原先继续轮询到 90 秒超时,用户最后看到的是 `context deadline exceeded`,源写的原因被吞掉。现在只有「页面已同意但 Cookie 还没落地」才继续等,其余错误马上上浮,探针断言错误里必须出现 `onLoginSuccess`,退回超时行为就会失败。

顺带记一笔容易写错的地方:探针第一版自己 `cookies.Open` 了一个 jar 交给登录窗口,于是 Cookie 落进了没有任何源会去读的地方,看起来像产品缺陷。真实路径是 `Library.Jar() → core.Jar()`,同一个实例(`settings.go`)。

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


## 真实源巡检(2026-11-01,35 个官方脚本)

`NK_LIVE_DIR=… NK_LIVE_QUERY=love sh scripts/live-sources.sh` 的结果:**18 个通过、3 个需要自建服务、13 个失败**。失败逐个归因如下,没有一个是客户端回归:

| 源 | 现象 | 归因 |
|---|---|---|
| `manga_dex` | `Unexpected token <` | 站点返回 HTML(302 边缘挑战),核心不支持该挑战类型 |
| `mycomic`、`mh18`、`ykmh` | `CloudflareException` | 站点挑战,需要 clearance;该挑战类型同样未支持 |
| `baozi`、`ikmmh` | 403 | 站点拒绝 |
| `comic_walker`、`shonen_jump_plus` | `Unexpected token <` | 站点返回 HTML |
| `happy` | `Post …: EOF` | 站点提前断开 |
| `picacg` | `Not logged in` | 需要真实账号 |
| `zaimanhua` | `漫画不存在或已被删除` | 样本失效(源自己说的) |
| `jcomic` | 图片 URL 为空 | 源侧返回空串,客户端正确拒绝(`image URL is not http(s)`) |
| `manwaba` | 封面「不是图片」 | **实测**:取回那张 13328 字节的封面,`Content-Type: image/jpeg` 但首字节是 `aa254a…` 而不是 `ffd8`,即站点返回的是加密/混淆字节。客户端拒绝把它当图片是对的 |
| `mh1234` | `Cannot read properties of null` | 源脚本解析站点结构时期望的元素不存在,属源侧脆弱 |

## 十二项窗口探针在表头与安装修复之后复跑(2026-11-03,提交 `580f9fd`)

表头提示删除、卸载器改 `SetOutPath`、安装回滚与字幕上限之后,再用四个环境变量跑一遍:exit=0,**PASS=12,FAIL=0,SKIP=0**,日志 `E:/Temp/nk-desk-probes-6f120fe.log`。覆盖面与上一轮相同(登录/挑战/对话框/关闭后取页/收藏/阅读解码/评论/分区/书架/分类页/官方脚本回调与标签),说明这些改动没有破坏窗口行为。仍是开发构建的真实窗口与本地夹具,不是安装包内的验收。随后夹具页改为 320×1200、`-progress` 判据改为只读页面状态之后,又在 `fdfa83f` 上复跑一次:仍是 12 PASS / 0 FAIL / 0 SKIP(`E:/Temp/nk-desk-probes-fdfa83f.log`)。

## 十二项窗口探针在字幕上限提交上复跑(2026-11-03,提交 `cc85ad9`)

字幕记录上限合入后,用四个环境变量跑了一遍真实窗口探针(`NK_OFFICIAL_SRC` + `NK_PROBE_SRC` + `NK_FAV_SRC` + `NK_READ_SRC`):**exit=0,PASS=12,FAIL=0,SKIP=0**,日志 `E:/Temp/nk-desk-probes-full-cc85ad9.log`。覆盖登录窗口(成功/超时/回调失败/迟到 Cookie/关闭)、挑战页、关闭源对话框释放调用方、关闭窗口后仍接更新与取页、源收藏列表、阅读器解码 loopback 页面、评论线程、浏览分区跟随声明、阅读落书架、分类页绘制与加载,以及 34 份官方脚本的回调与标签契约。同一提交的 `scripts/check.sh` 也通过(`E:/Temp/nk-desk-check-shelf.log`)。

这是开发构建的真实 WebView2 窗口,不是 0.2.6 安装包内的可执行文件;夹具与官方脚本均为本地文件,不涉及在线站点或账号。

## 十二项全部通过,无一跳过(2026-11-01)

四个环境变量全给(`NK_OFFICIAL_SRC` + `NK_PROBE_SRC` + `NK_FAV_SRC` + `NK_READ_SRC`)的一次完整运行:

```
exit=0   PASS=12   FAIL=0   SKIP=0
```

这比「各项分别通过」更有意义:**新增的检查之间没有互相干扰**——分类声明没有改变分区标签的计数,runner 式的 loopback 供图没有影响评论线程,真实源阅读器与离线阅读器各跑各的。逐项冒烟证明不了这一点。

## 窗口检查现在有十二项(2026-11-01,Core `d81445f`)

`sh scripts/desktop-probes.sh` 一次跑完所有需要真窗口的检查,八项全过(三项按环境变量决定是否参与):

```
PASS  login window: ok, timeout, failing callback, late cookie, close
PASS  challenge window: cleared and dismissed
PASS  closing a source dialog releases its caller
PASS  a closed window still takes updates and page reads
PASS  the favourites the source keeps are listed (NK_FAV_SRC=测试源)
PASS  the reader decodes a real chapter          (NK_READ_SRC=manhuagui)
PASS  the reader decodes a served page           (只给 NK_PROBE_SRC 时的离线版本)
PASS  a comment thread the source answers        (NK_PROBE_SRC=测试源)
PASS  the browse chips follow the sections       (NK_PROBE_SRC=测试源)
PASS  reading lands on the shelf                 (NK_PROBE_SRC=测试源,离线)
PASS  the category page draws and loads          (NK_PROBE_SRC=测试源)
PASS  every callback the official sources declare  (NK_OFFICIAL_SRC)
PASS  what every source's own tags mean            (NK_OFFICIAL_SRC)
```

其中收藏与分区两项**都不看截图**:收藏问窗口 `count/first/err`,分区问 `sections/chips/comics`,并校验 `chips == (sections>1 ? sections : 0)`。分区那条是给「单分区不画标签行」这个改动加的护栏,负向样本是「两个分区但列表为空」——它会以 `the current section holds no comics` 失败。

阅读器那项是唯一需要**真解码器**的:源可以返回看着合理的字节数而那并不是图片,所以它要求 `decoded>0`,并用 `manhuagui` 的真实章节跑(210 页,首批解码 4 页,首图 1520×1300)。负向验证用了 `copy_manga`:它现在在详情阶段返回 210,探针据此判失败,并把**站点的答复**原样打出来,而不是只说「命令失败」。

评论那一项先被写成「截图有内容就算过」,拿一个**不声明评论**的源一跑也 PASS 了——空线程和真线程的截图只差几百字节。改成向窗口要 `count / author / body / err` 之后,同一个源立刻报出 `count=0 err="源返回错误:core: wnacg keeps no comments"`。这类假阳性只有负向验证能发现。

## 阅读器解码现在也能离线跑(2026-11-01)

真实源那次要网络、要站点配合。现在 `cmd/readercheck` 多了 `-serve PORT`:在本机 loopback 上回一张运行时画出来的 PNG,夹具的章节指向 `http://127.0.0.1:18731/page.png`,于是**不联网也能证明「图片真的解码成图」**:

```
go run ./cmd/readercheck -script UITests/Fixtures/callback_probe.js -query probe -serve 18731
→ page title=Probe Comic figures=1 images=1 decoded=1 first=8x8
→ the reader rendered 1 of 1 pages
```

探针里用 `NK_READ_SERVE=18731 sh scripts/reader-probe.sh` 即可。这和 iOS 界面测试用的是同一个端口与同一个夹具,两端因此可以用同一份输入检查同一件事。

## 阅读器在新 pin 上重测(2026-11-01,Core `d81445f`)

`go run ./cmd/readercheck` 对真实源各跑一次,归因分开记:

| 源 | 结果 | 归因 |
|---|---|---|
| `manhuagui` | 详情 210 页,阅读器 `figures=210 images=210 decoded=4 first=1520x1300`,图片经阅读器代理取回 | 通过 |
| `komiic` | 同为 `figures=210 images=210 decoded=4 first=1520x1300`(HTML 抓取源与 API 源各一,结果一致) | 通过 |
| `copy_manga` | 详情阶段 `Invalid status code: 210` | **站点侧**(该站此前也用 210 回应,不是客户端或源脚本的回归) |

`decoded=4` 不等于只渲染 4 页:阅读器按需解码,这是「屏幕上此刻解码出来的页数」。窗口截图里没有 web 层,所以判断依据是日志里这行数字,不是图。

## 未验证

- **真实 Cloudflare 挑战**:没有遇到可复现的挑战页。2026-11-01 用 `cmd/challengeprobe -mode inspect`(新增)直接看窗口里到底是什么,两个站点的结论不再靠推测:
  - `18comic.vip`:窗口里是**正常站点**(标题「免費A漫 - 禁漫天堂」,Cookie 有 `yuo1`/`AVS`/`ipcountry`/`theme` 等),**根本没有挑战页**,也就没有 `cf_clearance`。所以「无法判定通过」不是客户端看不到 cookie,而是这类站点**没有挑战可过**:完成规则要求 `cf_clearance`,而页面不需要它就已可用,于是客户端只能等到超时。
  - `mangadex.org`:**2026-11-01 取到了完整链路**。`GET https://api.mangadex.org/manga?title=love` 返回 `302`,`location: /edge/challenge?src=<原路径的 base64>`,正文只有一句 `Redirecting to bot challenge page... (if this isn't a browser, don't send a browser UA)`;该挑战页是 `200` + `challenge-assets/challenge.js`,脚本读 `#c`/`#d`/`#h` 做 **proof-of-work**(`SOLVER_LIMIT_MS = 400000`,难度默认 17),解出后 `location.replace` 回原地址并带上 `powblock=<nonce>&pbchal=<token>`。也就是说:**它的通行凭据在 URL 参数里,不是一个 Cookie**。现有 `DoneRule` 只能表达「某些标记不存在 + 某些 Cookie 必须存在」,因此描述不了它——要支持就得加一种「URL 带上指定参数」的完成条件。**这属于产品决定,没有擅自放宽规则。** 顺带一个观察:因为挑战是页面内 JS 自解的,真人在 WebView 里打开会自动通过,缺的只是判定规则。
  - 两者 `-mode live` 都以 `context deadline exceeded` 结束、0 个 cookie,与 inspect 的观察一致。**本地可复现的那条闭环**(页面自己写出 `cf_clearance` 并被窗口判为通过、cookie 进入分区、带回页面真实 UA)已由 `scripts/challenge-probe.sh` 的 `clear` 模式跑通;仍缺的是**真实站点**上的成功样本。
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
go run ./cmd/uicheck -script /path/to/manga_dex.js -query "one piece" -open -out build/uicheck.png

# 人机验证:窗口本身,以及 Core 的整条 403→验证→重试 链路(本地伪造站点)
go run ./cmd/verifycheck -local
go run ./cmd/verifycheck -e2e

# 无人值守跑完网页登录的三种结局(本地假登录页),以及 403 挑战窗口
go run ./cmd/loginprobe -mode ok|timeout|failcallback|latecookie
sh scripts/login-probe.sh
sh scripts/challenge-probe.sh

# 源的设置回调:不带界面跑一个源的某个回调,或跑完它声明的全部回调
go run ./cmd/cbprobe /path/to/source.js - --all
NK_OFFICIAL_SRC=/path/to/official-srcs sh scripts/callback-probe.sh

# 按下详情页上的一个标签,看源自己的 onClickTag 把它解释成什么(-open 打开第一个结果)
go run ./cmd/uicheck -script /path/to/source.js -query 关键词 -open -tag '类型|热血' -out tag.png

# 校验一个源脚本的阅读契约:安装→搜索→详情→**逐个标签组**的标签动作
# 一个源对不同命名空间可以分别给「搜索」「分类」或不处理,只看第一个组会误判

go run ./cmd/comicprobe /path/to/source.js 关键词

# 源对话框被用户关掉后调用方是否释放;窗口关掉后 Update 与页面读取是否仍安全
go run ./cmd/dialogprobe
go run ./cmd/updateprobe

# 跨官方源问一遍每个源自己的标签含义(需要网络)
NK_OFFICIAL_SRC=/path/to/official-srcs sh scripts/tag-probe.sh

# 评论线程开在真窗口里并截图(不碰网络;给它一个声明了 loadComments 的源脚本)
sh scripts/comment-probe.sh /path/to/source.js

# 阅读器在真实章节上真的解码出图片(需要网络;站点拒绝时它会说出是站点的问题)
NK_READ_SRC=/path/to/source.js sh scripts/reader-probe.sh

# 上面这些需要窗口的检查一次跑完
# 不设的项会打印 SKIP 而不是悄悄略过:
#   NK_OFFICIAL_SRC 跳过官方源回调与标签,NK_PROBE_SRC 跳过评论线程与分区标签,
#   NK_FAV_SRC 跳过源收藏,NK_READ_SRC 跳过阅读器解码
NK_OFFICIAL_SRC=/path/to/official-srcs NK_PROBE_SRC=/path/to/source.js NK_FAV_SRC=/path/to/source.js NK_READ_SRC=/path/to/source.js sh scripts/desktop-probes.sh
```

评论线程那项是这一轮补的:客户端对真实源的评论早就跑过,但那条路要网络和登录。给它一个自带回答的源脚本(例如 iOS 仓库里的 `UITests/Fixtures/callback_probe.js`),这项在任何机器上都能跑,检查的是评论界面本身有没有把作者、正文和按钮画出来。


分类页那项证明的是**两半契约**同时成立:`category` 给出可选项,`categoryComics` 给出加载器。三种输入实测过——完整声明 `parts=1 values=2 comics=1` 通过;只有 `parts` 没有加载器时 `comics=0` 并以 `source does not define this function` 失败;完全没声明时以「offers no category parts」失败。画出选择器却加载不出内容,正是它要区分的那种「看起来像站点空了」的故障。

`NKDEBUG=1` 会打印封面下载与解码的日志。

## 源收藏也有了可复跑的检查(2026-11-01)

评论那次假阳性给了教训,所以这项一开始就**问窗口内容**:`uicheck -favorites` 现在多打一行

```
favorites: count=1 first="Probe Comic" err=""
```

`scripts/favorites-probe.sh` 以 `count>0` 且标题非空为通过条件,失败时把源给的错误原样打出来。三种输入:

| 源 | 结果 | 归因 |
|---|---|---|
| 测试源(声明 `favorites`) | `count=1 first="Probe Comic"` | 通过 |
| `manhuagui` | `count=0`,错误「请先登录漫画柜账号」 | **需要登录** |
| `komiic` | `count=0`,错误 `Cannot read properties of null (reading '0')` | **源脚本侧** |

收藏声明在**顶层 `favorites` 块**而不是 `comic` 里,写错位置只会得到空页面——这正是必须问内容、不能看截图的原因。

## 门禁默认带竞态检测(2026-11-01)

`scripts/check.sh` 里的 `go test` 一直是普通模式,而这个客户端恰恰在后台 goroutine 里加载评论、浏览和图片,界面同时在读同一份状态——正是竞态检测该管的地方。现在默认 `-race`:

```
PASS  go test (race detector)      # 13 个包,exit 0
NK_NO_RACE=1 → PASS  go test (no race detector)
```

两种模式的标签会打印在结果里,**关掉时看得见**,不会让「跑过」和「带检测跑过」混成一件事。没有 C 工具链的机器用 `NK_NO_RACE=1`。

## 单分区源不再画一行单标签(2026-11-01)

同一个状态,两端原来给出不同画面:iOS 在「只有一个分区且没有分类」时不画分区行(`BrowseView.swift:50`),桌面却总是画一行——那行里只有一个标签,而且永远是选中的,读起来只是噪声。

现在桌面也按「`len(pages) > 1` 才画」处理,连那条分隔线一起收进条件里。实拍核对过两种情况:

| 源 | 结果 |
|---|---|
| 只声明 1 个分区 | 没有标签行,直接是 `Solo Comic` 与「第 1 / 1 页」 |
| 声明 2 个分区 | 两个标签照常出现,`Probe Shelf` 选中态、`Probe Second` 可切 |

`go vet`、`go test ./internal/ui` 通过。这类差异不会自己冒出来,只有把两端放在同一个输入下看才会发现。

## 种子生成变得跨平台确定(2026-11-01)

`cmd/seedgen` 直接把文件字节塞进记录的 `body`,于是**同一份夹具**在 Windows(CRLF)和 macOS(LF)上生成出不同 md5 的种子。后果不是理论问题:验证机拿到的是另一个哈希,「种子与树一致」这句话就没法用哈希回答,只能靠人去比内容——而这次就是这么绕进去的。

现在生成前把 `

` 与孤立 `
` 统一成 `
`:

```
同一命令连跑两次 → 字节完全相同
四个源里只有 callback_probe 的 sha256 变化(其余不含 CRLF)
iOS 侧 check-seed.sh 仍通过(它本来也会归一换行)
```

## 交叉编译:三个目标都能编过(2026-11-01)

「在 macOS 与 Linux 上编译」这条一直列在下一步,却只有 macOS 的真机证据(Mac worker)。这一轮先做了交叉编译,把「编不过」这种可能排除掉:

```
GOOS=linux  GOARCH=amd64  go build ./...   exit 0
GOOS=linux  GOARCH=arm64  go build ./...   exit 0
GOOS=darwin GOARCH=arm64  go build ./...   exit 0
```

**这证明的是可移植性,不是能运行**:窗口层依赖 GTK/WebKit 的运行时库,交叉编译出来的二进制没有在 Linux 上启动过。所以「Linux 上可用」仍然没有证据,「Linux 上编不过」现在已经可以排除。

这台机器上也确实没有能跑它的环境(2026-11-01 核对):`wsl.exe` 存在但**没有安装任何发行版**,`docker` 不存在。要补这条证据,需要先装一个 Linux 环境——那属于系统变更,没有擅自做。

## 下一步

1. 找可复现的真实 Cloudflare 挑战(需要站点当前对命令行回 403 且会下发 `cf_clearance`),用它核对完成规则;目前 18comic.vip 这类「WebView 能开但不发 `cf_clearance`」的站点,验证后仍无法判定通过,可能要放宽完成规则,这属于设计决定。
2. 在 Linux 上运行(macOS 已由 Mac worker 运行;交叉编译已通过,见上一节)。
3. 接上视频源与嗅探 —— 三个选项各自的工作量与代价已量在 [VIDEO-PLAYBACK.md](VIDEO-PLAYBACK.md):WebView + `<video>`(纯 Go,复用阅读器那套代理)、原生播放器(需 cgo 与自写解码,MyGo 只带 GIF 解码器)、只嗅探直链交给系统播放器(请求头带不过去)。
4. 制作台模块。
