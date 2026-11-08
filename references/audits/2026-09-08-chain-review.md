# Skill 全链路审查 — 2026-09-08

> 历史审查材料：以下问题正文描述修复前状态，行号和探针只适用于当时源码。对应修复已随 3b60371 推送；后续安装修复见 5d7b22b、d1aefbb。不要将文末当时的未提交结论理解为当前状态，也不要在当前源码上运行旧探针来验收修复。


对象：当前 0.18.0 工作区候选，包含尚未提交的既有修复、Claude Desktop、DSH Desktop 和 ZCode 适配。此次为审查，未修改产品代码、发布产物或真实宿主配置，未使用真实凭据、网络媒体任务或付费 API。

审查覆盖 Skill 指令、宿主分派、凭据选择边界、模型校验、multipart、异步提交与续接、任务记录、下载复用、安装更新、发布和验收。不能用这次代码审查代替真实宿主验收，也不宣称已经穷尽所有缺陷。

## 修复状态（2026-09-08）

用户已要求实施本报告建议。以下问题描述保留为修复前证据，当前候选已完成对应实现：

1. 附件在校验阶段记录文件身份、长度、时间和内容摘要；发送前打开并验证句柄，发送中核对摘要、长度和总量，不创建附件缓存。即使大小及时间戳相同，内容变化也会拒绝。
2. 下载摘要、字节数和媒体类型保存在显式任务记录中；复用与交付确认都核对完整性。旧无摘要记录可续查原任务，但不能认领旧文件，保留文件后换目录下载。
3. Shell/PowerShell 下载入口始终使用本次固定提交的目录选择器；已补“带旧标记安装器 + ZCode”的隔离回归。
4. 参数验证先于任务记录创建；字符串安全投影按结构/枚举限制，非法自由文本不会进入新记录或回执。
5. 构建配置固定 Go 1.26.0；构建证明覆盖源码/依赖/构建输入摘要及六份产物，二进制内嵌源码身份。发布门禁实际重建六个平台并逐字节比较；Go 测试执行当前平台待发二进制的离线命令。
6. 新增 Linux 验收入口；五个 Linux 客户端目标 pending，未验证的其他 Linux 组合明确 unavailable。当前 100 个 host/OS 汇总检查 pending，与九类详细用例分开报告。

当前 Codex 的实际前置检查在网络请求前安全停止：`active_connection_not_puretokens`，`api_request_executed=false`。未读取/输出连接详情，未改客户端配置、未创建付费任务；因此真实图片、编辑、视频与附件验收仍未完成，不标记 passed。Windows 新用例未在本机执行。

当时的交接过程保留在 Git 历史中；本报告不作为实时进度日志。当前维护规则见 `CONTRIBUTING.md`，版本发布证据见 `references/acceptance/`，实机状态见 `references/host-acceptance.json`。

## 已确认问题

### 1. P1 — 发送中的附件没有保持校验时的完整性

位置：`runtime/executor/main.go` 的 `validateTaskRequest` 和 `taskRequestBody`，特别是 648–669 行。

先按路径 Stat 校验，之后发送时重新 Open；`io.Copy(part, io.LimitReader(file, limit+1))` 不检查实际复制长度，也没有再次验证文件身份、原始长度或累计总量。文件在两步之间被替换、增长或缩短，可能发送不同或不完整内容。文件增长超限时，截断后的 multipart 仍正常结束。

已复现：将一个通过校验的小附件扩为 `attachmentLimit(image)+100`，读取生成的 multipart；实际文件部分只有 `limit+1` 字节，读取无错误、正常 EOF。没有向外发请求。

影响：用户编辑请求可能携带残缺输入，API 是否拒绝、是否产生任务取决于服务端，不能假定绝不会产生费用。

建议：在发送前打开并验证待发送句柄，保持句柄对应关系；发送过程验证长度和累计上限；发生变化或超限必须关闭流并报错，已经开始传输时保留 unknown 的提交结果分类。避免通过先复制到隐藏缓存来解决。

### 2. P1 — 同名有效文件被当作已验证的同任务产物

位置：`runtime/executor/main.go:915`，`runtime/executor/task_records.go` 的 `recordedDownloadFormat` 和 `recordReceiptWriter.writeReceipt`。

复用依据是“内容接口路径的哈希文件名 + 媒体结构有效”，并没有保存并核对上次下载内容的摘要。文件名哈希并非文件内容哈希。原文件被编辑器覆盖成另一张仍有效的图时，也会被当作原任务结果。

已复现：在目标文件名下放一张独立合成 PNG，把 service origin 设为无效地址，`download` 仍直接成功返回这个文件，没有网络访问。记录恢复也仅检查名称和格式。

建议：在用户可见的显式任务记录中绑定 task/kind/index 与下载摘要、字节数、媒体类型；复用前核对内容。无可信下载记录时不能仅凭文件名认领既存文件，应安全报告冲突或要求显式输出目录。保留用户文件、不覆盖。

### 3. P1 — 旧目录选择器阻断新增宿主安装

位置：`runtime/puretokens-skill-fetch.sh:50`、`runtime/puretokens-skill-fetch.ps1:41`。

下载入口只看同目录安装器有没有 `puretokens-locate-v1` 标记，就决定复用它。HEAD 中旧安装器已有该标记，却不支持新增桌面宿主或 ZCode。新下载入口与旧安装器共存、尝试 `install --host zcode` 时，旧选择器失败，入口不会改用本次固定提交的选择器。

已复现：复用 `test/distribution.test.mjs` 的隔离 fixture/mockDownloads；把同目录 installer 换为 `git show HEAD:runtime/puretokens-skill-install.sh`，调用当前 fetch 的 `install --host zcode`。返回 `unsupported host`；模拟下载记录确认未请求固定提交的 installer。

边界：这不是声称所有普通更新都失败。触发场景为旧标记选择器配合新增宿主；没有旧 sibling 的全新入口不受此条件影响。PowerShell 具有相同选择结构，本次只动态验证 Shell。

建议：目录选择器与本次固定提交绑定；或对本次宿主能力做明确兼容性校验，不支持时取得固定提交的选择器。不能只信长期不变的标记。增加旧版 → 新宿主路径测试。

### 4. P2 — 模型校验之前，非法自由文本已经写入任务记录

位置：`runtime/executor/task_records.go:279`，`runtime/executor/task_state.go:43`。

记录先通过 `recordFromRequest` 保存，然后才调用 `executeTask` 进行模型字段校验。安全投影对 size、resolution 等字符串主要做通用文本过滤，而不是确认它是声明的尺寸或枚举。模型校验失败时，原参数仍保留在记录中。

已复现：提交合成请求 `parameters.size = "private draft about project alpha"`；模型校验拒绝，不执行 POST，但留下 `submission_outcome: not_submitted` 的记录，仍含该自由文本。

影响：若宿主错误地把用户描述放入结构化参数，记录可能持久化本不应保存的请求文字。此处未观察或使用任何真实敏感数据，不应表述成已发生用户泄露。

建议：在首次持久化前完成声明参数验证和安全投影；仅保留已验证的枚举、尺寸、数值及布尔字段。保持 POST 前有记录保护，但不要把原始未验证参数写入该记录。

### 5. P2 — 发布溯源没有证明源码和可执行文件对应

位置：`scripts/package-platform-releases.mjs:40`、`scripts/build-executor.mjs`、`.github/workflows/release.yml`。

当前证明的是源码、构建脚本、可执行文件均与同一 Git 提交一致，且二进制与 manifest 校验和一致。它没有验证该二进制由这个提交的源码构建。若源码修改后忘记重建，并把源码与旧二进制一起提交，仍可以得到非空 sourceCommit。Go 工程测试执行当前源码，也不能代替测试实际交付的旧文件。

已有 `release provenance requires committed package and executor build inputs` 测试本身以 `package main` 的合成源码配合现成候选可执行文件，在提交后得到有效 sourceCommit；它验证了工作树一致性，并未验证构建对应关系。

建议：在受控工具链下生成构建证明，将源码摘要、依赖与构建输入摘要、各平台二进制摘要绑定；发布门禁验证对应关系，并让平台测试运行实际待发可执行文件。不要把只匹配版本号当作新鲜度证明。此项不表示当前六份候选已发现过期。

### 6. P2 — Linux 交付产物没有真实宿主验收入口

位置：`scripts/validate-host-acceptance.mjs:21`、`:35`、`references/host-acceptance.json`。

分发 Linux amd64/arm64 产物，但验收 schema/验证逻辑只接受 macOS/windows。即使完成 Linux 实机验收，也无法按当前结构记录。80 项 pending 是 10 宿主 × 2 系统 × 4 汇总项，不是所有媒体用例，更不包括 Linux。

建议：按宿主实际支持的 OS 建矩阵，允许 Linux；不适用组合明确 unavailable，不要求桌面客户端凭空支持 Linux。报告汇总项与九类完整用例的数量分别统计。

## 产品与实机验收风险

### 7. 当前不能把“可安装、有凭据夹具”解释为完整可用

`references/host-acceptance.json` 的 evidence 为空，80 个汇总项 pending。Skill 交付指令仍主要是“使用当前宿主的附件方式”，未通过真实宿主证明图片/视频手递交、刚生成图片的再次编辑和跨会话续接。

ZCode 还存在必须明确的语义边界：适配器选的是唯一启用的 Pure Tokens 连接，不能证明会话选中了它；Skill 的总路由条件却是“当前宿主使用 Pure Tokens”。这意味着两者之间仍需要宿主上下文或明确的用户选择，不能靠扫描设置补全，更不能据此承诺自动路由在所有宿主生效。其他宿主的项目/会话覆盖也只在现有文档中声明为未验收。

建议：先选择有授权的宿主做最小纵向验收：加载 Skill → 身份/认证 → 一张图 → 上传图编辑 → 刚生成图再次编辑 → 同任务恢复 → 原生附件 → 视频。逐宿主记录具体的输入字节和输出附件接口。暂时保持清晰的安装支持/凭据夹具/实机已验收三个层级。

## 小项与未成立的初步怀疑

- image/video 主指令中的宿主列表仍未列 zcode，只在文末另补说明，应统一，降低模型选择歧义。
- 当前 size/strength 等字段应按已安装 profile 判断；本轮检查确认 seedream 的 strength 是字符串枚举，不能误报成应保留数值而丢失的缺陷。
- 任务记录写入失败的 receipt 虽不带 failure_phase，但携带 local_error_code，现有 receipt schema 允许；不将其误报为 schema 违规。

## 验证与建议顺序

- 本轮隔离 Go 探针四项通过（表示当时缺陷成功复现，不表示修复通过）。`2026-09-08-chain-probes.go.txt` 仅保留为修复前审计证据，不复制到当前测试目录或作为现行验收指令。当前回归见 `runtime/executor/chain_regression_test.go`，统一运行 `npm run check`。
- Shell 旧选择器探针一项成功复现。既有发布溯源测试重新执行。
- 审查探针未留在常规测试目录，不用“断言缺陷存在”的测试增加工程通过数。
- 建议先修 1/2/3，随后 4/5/6，再在明确授权范围完成 7 的实机验收。各修复应补断言正确行为的回归测试，并重建六平台执行器、归档和候选包。
- 此次未修复上述问题，未提交、推送或发布；此前的 53 项工程通过不能证明这些新增边界条件已被覆盖。
