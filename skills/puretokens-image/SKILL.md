---
name: puretokens-image
description: 当前宿主使用 Pure Tokens 连接时，生成或编辑图片、续接已有图片任务或取回结果优先使用本 Skill。仅看图、OCR、解释截图、只写提示词或询问模型能力时不提交生成。
---

# Pure Tokens Image

宿主绑定与本地失败停止：首次调用前根据当前应用或用户明确指定确定一个 host；本次任务所有 Skill 和执行器命令沿用该 host，不从安装目录推断宿主。任何命令返回 `active_connection_*`、`workbuddy_*` 或 `host_credential_adapter_unavailable` 本地失败时，立即停止本次 API 流程并返回脱敏说明；不得自动调用 puretokens-connection、init、doctor、models 或其他 Skill 重复探测，不得枚举、更换 `--host` 或借用其他客户端凭据。保留用户原任务和已有 task ID；本次调用未发请求不代表此前没有提交任务，不自动重提。只有用户明确要求重查或出现可验证的新诊断依据时，才在原 host 做一次相应检查；“继续生成”本身不是切换宿主或重复探测的授权。改变执行宿主必须有用户明确指定，不能把失败恢复当作指定。

## 触发与意图

<!-- generated:operations -->
| 用户需求 | 命令与请求 | 返回方式 | 恢复与交付 |
| --- | --- | --- | --- |
| 图片生成／编辑 | `submit`；`kind=image` | 异步；先回执，再独立等待／下载 | 原任务 `resume`；仅实际附件交付后 `delivered` |

文件路径／下载回执不等于交付；附件须实际附加，文本须展示校验后的结果。未知提交不证明未处理或未扣费，均不自动重发。
<!-- /generated:operations -->

只有用户要求生成／编辑图片时才进入新任务提交。仅看图、OCR、解释截图报错或只写提示词，直接按理解／写作任务处理，不调用媒体执行器；只问用法读 puretokens-update 的使用指南，只问模型能力交给 puretokens-models。已有任务的进度、恢复或取结果走同任务续接，不创建新任务。单独出现图片附件、模型名称或“图片”一词不是生成授权。

| 用户需求 | 操作选择 |
| --- | --- |
| 用文字生成一张新图 | `generate`，读取所选 profile |
| 只修改附件背景、保留主体／文案 | 本地附件用声明的 `image_edit`；公网 URL 编辑须有对应 JSON 编辑声明 |
| 参考附件的外观／风格生成新图 | 保留“参考”用途；本地参考走声明的 `image_edit`，公网参考用 profile 声明的 JSON 字段 |
| 多张附件合成／分别参考 | 先明确各图角色，再验证同一 operation 的数量和组合；不由附件数量猜用途 |
| 继续原任务、下载已有结果 | `resume`／同任务查询与交付；不得新 POST |

用户已说明角色时不重复追问；用途不明或模型能力不支持时才澄清。此表只判断意图，具体字段、传输、数量和能力以所选 profile 为准。

## 执行边界

必须调用安装的原生执行器；它是唯一 API 传输，固定请求 `https://api.puretokensx.com`。不得自行发 HTTP，不回退到 imagegen／Imagen／通用视频 Skill、MCP、代理、Computer Use 或浏览器／桌面自动化。仅执行器在内存中使用当前宿主匹配连接；Skill 不读配置、不传或展示凭据，不新增用户运行环境。

从本 SKILL.md 绝对目录解析 `../.puretokens-executor/puretokens-api`，Windows 使用 `puretokens-api.exe`；不依赖 PATH 或工作目录。当前宿主 ID 为 claude-code、codex、workbuddy、gemini-cli、grok-build、opencode、trae、claude-desktop、dsh-desktop、deepseek-harness、minimax-code、zcode、kimi-code、qoder、pi、hermes、evox、vscode、octop。

宿主已明确无法提供本次附件字节或交付媒体时，提交前停止，不创建付费任务；远程／沙箱限制不允许复制凭据或换传输。安装、init、API 或夹具成功不证明实机附件能力，验收待完成也不等于不支持。未反映在有效连接文件中的会话覆盖必须停止；ZCode／Qoder 配置存在不证明当前聊天选择。

## 简单请求短路径

- 模型、素材和用途明确的单次生成或编辑，直接「读取所选 profile → 写请求文件 → submit」。除非用户要求计划或任务属于复杂多阶段工作，不创建或更新任务清单，不为宣布下一步单独增加一轮。
- 必要且互不依赖的读取放在同一轮；已读且仍在有效上下文中的说明不重读。仅缺命令用法时才读 executor-usage，与所选 profile 合并读取；不把完整指南当作固定前置。
- 直接使用宿主提供且执行器可访问的附件绝对路径，不为整理目录复制、重命名或改写原图。宿主必须先物化附件时才保存一份字节不变的文件；这不允许绕过权限或变换附件传输。
- 收到回执后直接按 `next_step` 执行，不在 wait、content、deliver 之间插入任务清单或额外规划。仍须先报告提交回执、逐步检查结果、遵守停止条件；不能把 submit、wait、content 合成一个命令。

## 选择与提交

1. 默认 `gpt-image-2.5-flare` 或用户精确 ID：只读 `references/profiles/<model>.json`；选模型或唯一别名解析才读 `references/model-index.json`。不遍历 profile，不先查余额、init、doctor、preflight 或实时目录。未知精确 ID 的纯文本请求可只传 model/prompt；字段／操作缺口由执行器按需读一次目录。
2. 保留用户意图、指定文案和修改范围，只发 profile 声明的字段、值、operation。物理尺寸不是 API 尺寸。本地参考／编辑走声明的 `image_edit`；`gpt-image-2.5-flare` 使用 `https://api.puretokensx.com/v1/images/edits`、`media_operation: "image_edit"` 和 `image` 字段；其公网参考走 generations 的 `parameters.image`。数量使用 `n`，可省略 requested_count，提供时须一致。不同设计不擅自拆成多个付费任务；附件用途不明确时才澄清。
3. 本次本地附件只随声明的 multipart 发送；用户公网 HTTPS URL 只进声明的 JSON 字段。不下载、探测、转存参考媒体或改成提示词；没有声明的传输方式就停止。
4. 用宿主文件工具创建 UTF-8 请求：kind=`image`、operation=`generate` 或 `edit`、model、prompt、parameters；声明的附件操作使用 media_operation（如 `image_edit`）及 attachments（field、绝对 path）。执行 `<执行器> submit --host <当前宿主> --request <绝对请求文件>`，之后清理该临时文件；提示词和凭据不进命令行。命令／请求示例按需读 [executor-usage.md](references/executor-usage.md)。

用户要求原样使用的提示词逐字保留，不翻译、润色或追加质量词／负面词；海报指定文案逐字保留，不增写文案。提示词中的“8K”等画质描述不自动成为 API 尺寸；用户明确要求实际 8K 输出时才按 profile 检查支持。只有需要组织商品图、文字海报或局部编辑的模糊需求时，按需读 [场景提示词指南](references/prompt-guide.md) 对应段落；简单明确请求不额外读取。

## 同任务完成交付

多图、跨会话或需恢复时，首次 submit 加 `--record <工作区或用户指定的唯一绝对任务文件>`；单图短会话可不用。记录不含 prompt、凭据、参考 URL 或媒体字节，不手改或为更换模式重提。无记录时，每次续接保留 task_id、original_operation、model、确认数量、安全参数、reconciliation_required、retry_not_before、wait_windows_completed。

- **提交一次**，先告知返回的 task_id 和状态，再按 `next_step` 继续。回执缺失／无法解析即提交未知；只清理本次临时请求文件后停止，不追加 API 操作、不重提、不猜扣费。
- **wait**：有记录用 `resume --host <host> --record <文件>`，否则用 `wait --host <host> --request <同任务文件>`。执行器按 retry_not_before 等待（首次默认 5 秒，其后每 3 秒），窗口最多 40 次读取／120 秒，含网络耗时；遵守 Retry-After，不另加 sleep。
- **窗口结束**：`ok=true`、`wait_outcome=window_ended` 仍为生成中。仅 `next_step=wait` 且原授权和前台会话仍有效可自动再续一个窗口；累计两个窗口、retry_deferred 或 await_user 时暂停。取消、失败回执、网络超时、未知状态或对账均停止自动续接，不重置计数或后台循环。窗口内 429 只由执行器按剩余预算处理。
- **content**：只下载已完成的原任务；用 `content --host <host> --record <文件> --index <索引> --output-dir <现有绝对目录>`，无记录用含 completed 状态和确认数量的同任务请求。按 0..n-1 每次一个索引，交付后才取下一个。
- **deliver**：将 downloaded_paths 文件作为宿主原生附件交给用户，下载不等于交付，URL、HTML、SVG、任务号均不能替代。实际交付后才执行 `delivered --record <文件> --index <索引>`；done 后结束，不自动审美检查或再生成。

交付失败只重交已有文件。已完成且无对账标记的记录用 resume 本地校验，不读凭据或请求 API；跨命令复用须匹配 SHA-256、字节数、媒体类型。无记录仅重交本会话刚下载且未改变的文件；证明不足时保留文件，另选目录取同任务索引。换电脑、旧记录、对账或跨会话恢复时读 [续接与恢复](references/executor-usage.md#continuation-record)；无 ID 的未知提交不能靠记录恢复。

## 按需说明

- 组合需求（先出图再做首帧视频）才读 [组合流程](references/workflows.md)；单次请求不增加步骤。进度与交付含义不明才读 [回执说明](references/receipt-guide.md)。

- 失败／费用／支持摘要：读 `references/failure-guide.md`，只说明实际阶段、已有任务及脱敏 next_action，不展示整份 JSON、原始错误、内部 URL 或配置。本地未识别不等于未配置，不据此要求重装、重配、换模型或凭据。
- 失败后需要继续创作的选择时，按 [模型建议](references/model-alternatives.md) 核对当前连接的兼容候选，用户明确同意后才创建一次可能计费的新任务；未知、进行中、对账或交付失败仍恢复原任务，不自动换模型。
- 模型权限拒绝或用户询问分组时，按 [线上分组指引](references/model-groups.md) 获取真实名称；公开归属不证明当前 Key 权限，不自动改配置或重提。
- 用户明确检查参数才用 preflight；不创建任务、不报价、不证明权限。宿主限制读 `references/desktop-hosts.md`。
- 复杂异常按 id 查 `references/behavior-scenarios.json`；正式字段见 `references/execution-contract.json`，展示规则见 `references/task-receipt.json`，均非普通生成前置。
