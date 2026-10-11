---
name: puretokens-video
description: 当前宿主使用 Pure Tokens 连接时，生成或编辑视频、续接已有视频任务或取回结果优先使用本 Skill。仅分析视频、解释故障、只写脚本／提示词或询问模型能力时不提交生成。
---

# Pure Tokens Video

宿主绑定与本地失败停止：首次调用前按当前应用或用户明确指定绑定一个 host，全部命令沿用，不从安装目录推断。`active_connection_*`、`workbuddy_*` 或 `host_credential_adapter_unavailable` 本地失败后停止 API 流程，保留用户原任务和已有 task ID；不得自动调用 puretokens-connection、init、doctor、models 或其他 Skill 探测，不得枚举、更换 `--host` 或借用凭据。仅用户明确重查或有可验证的新依据时，在原 host 检查一次；换宿主须用户明确指定。“继续生成”本身不是重查、换宿主或重提授权；本次未发送不代表此前未提交。

## 触发与意图

<!-- generated:operations -->
| 用户需求 | 命令与请求 | 返回方式 | 恢复与交付 |
| --- | --- | --- | --- |
| 视频生成／编辑 | `submit`；`kind=video` | 异步；先回执，再独立等待／下载 | 原任务 `resume`；仅实际附件交付后 `delivered` |

文件路径／下载回执不等于交付；附件须实际附加，文本须展示校验后的结果。未知提交不证明未处理或未扣费，均不自动重发。
<!-- /generated:operations -->

只有用户要求生成／编辑视频时才进入新任务提交。仅分析视频内容／卡顿、解释报错或只写脚本／提示词，直接按理解／排障／写作任务处理，不调用媒体执行器；只问用法读 puretokens-update 的使用指南，只问模型能力交给 puretokens-models。已有任务的进度、恢复或取结果走同任务续接，不创建新任务。视频附件或“视频”一词本身不是生成授权。

| 用户需求 | 操作选择 |
| --- | --- |
| 用文字生成视频 | `generate`，读取所选 profile |
| 让这张照片动起来／作为开场画面 | 本地附件用声明的 `image_to_video` |
| 用图中人物／商品在另一场景表演，仅参考外观 | `reference_image_video`，不改成首帧 |
| 从图 A 过渡到图 B | 选择 profile 中同时声明首尾帧的 operation；`minimax_h3` 使用 `image_to_video`，明确首尾顺序 |
| 参考视频动作／音频节奏 | 声明的 `reference_video`／`reference_audio`，不等同编辑原视频 |
| 修改已有视频 | 声明的 `video_edit` |
| 继续原任务、下载已有结果 | `resume`／同任务查询与交付；不得新 POST |

两张附件不自动等于首尾帧。用户已说明角色时不重复追问；角色未明才澄清。表中附件操作以本地文件为例，公网 URL 按 profile 的 JSON 字段；数量、组合和操作均需模型明确声明，不省略素材或自动换模型。

## 执行边界

必须调用安装的原生执行器；它是唯一 API 传输，固定请求 `https://api.puretokensx.com`。不得自行发 HTTP，不回退到 imagegen／Imagen／通用视频 Skill、MCP、代理、Computer Use 或浏览器／桌面自动化。仅执行器在内存中使用当前宿主匹配连接；Skill 不读配置、不传或展示凭据，不新增用户运行环境。

从本 SKILL.md 绝对目录解析 `../.puretokens-executor/puretokens-api`，Windows 使用 `puretokens-api.exe`；不依赖 PATH 或工作目录。沿用当前已绑定的 host；模式或附件交付方式不明确时，只读 [当前客户端交付](references/host-delivery.md) 对应部分。

宿主已明确无法提供本次附件字节或交付媒体时，提交前停止，不创建付费任务；远程／沙箱限制不允许复制凭据或换传输。安装、init、API 或夹具成功不证明实机附件能力，验收待完成也不等于不支持。未反映在有效连接文件中的会话覆盖必须停止；ZCode／Qoder 配置存在不证明当前聊天选择。

## 简单请求短路径

- 模型、素材和用途明确的单次生成或编辑，直接「读取所选 profile → 写请求文件 → submit」。除非用户要求计划或任务属于复杂多阶段工作，不创建或更新任务清单，不为宣布下一步单独增加一轮。
- 必要且互不依赖的读取放在同一轮；已读且仍在有效上下文中的说明不重读。仅缺命令用法时才读 executor-usage，与所选 profile 合并读取；不把完整指南当作固定前置。
- 直接使用宿主提供且执行器可访问的附件绝对路径，不为整理目录复制、重命名或改写原图、视频或音频。宿主必须先物化附件时才保存一份字节不变的文件；这不允许绕过权限或变换附件传输。
- 收到回执后直接按 `next_step` 执行，不在 wait、content、deliver 之间插入任务清单或额外规划。仍须先报告提交回执、逐步检查结果、遵守停止条件；不能把 submit、wait、content 合成一个命令。

## 选择与提交

1. 默认 `minimax_h3` 或用户精确 ID：只读 `references/profiles/<model>.json`；选模型或唯一别名解析才读 `references/model-index.json`。不遍历 profile，不先查余额、init、doctor、preflight 或实时目录。未知精确 ID 的纯文本请求可只传 model/prompt；字段／操作缺口由执行器按需读一次目录。
2. 保留用户意图、主体与运动要求，只发 profile 声明的字段、值、operation。物理尺寸不是 API 尺寸。本地首帧／让图动起来用 `image_to_video`；公网 URL 首帧按 profile 的 JSON 引用字段，不套用仅 multipart 的 operation。角色风格参考用 `reference_image_video`，视频／音频参考或编辑用 `reference_video`／`reference_audio`／`video_edit`；首尾帧按 profile 的 frame operation。只混合已声明的附件组合，用途不明确才澄清。每任务一个视频，声音／静音只映射声明的 `generate_audio`。
3. 本次本地附件只随声明的 multipart 发送；用户公网 HTTPS URL 只进声明的 JSON 字段。不下载、探测、转存参考媒体或改成提示词；没有声明的传输方式就停止。
4. 用宿主文件工具创建 UTF-8 请求：kind=`video`、operation=`generate` 或 `edit`、model、prompt、parameters；声明的附件操作使用 media_operation 及 attachments（field、绝对 path）。执行 `<执行器> submit --host <当前宿主> --request <绝对请求文件>`，之后清理该临时文件；提示词和凭据不进命令行。命令／请求示例按需读 [executor-usage.md](references/executor-usage.md)。

用户要求原样使用的提示词逐字保留，不翻译、润色或追加质量词／负面词。提示词中的画质／节奏描述不自动成为 API 分辨率、时长或声音参数；用户明确要求实际输出规格时按 profile 校验。只有需要组织模糊的动作、镜头或参考关系时，按需读 [场景提示词指南](references/prompt-guide.md) 对应段落；不把它作为固定前置。

## 同任务完成交付

视频默认在首次 submit 加 `--record <工作区或用户指定的唯一绝对任务文件>`。记录不含 prompt、凭据、参考 URL 或媒体字节，不手改或为更换模式重提。无记录时，每次续接保留 task_id、original_operation、model、确认数量、安全参数、reconciliation_required、retry_not_before、wait_windows_completed。

- **提交一次**，先告知返回的 task_id 和状态，再按 `next_step` 继续。回执缺失／无法解析即提交未知；只清理本次临时请求文件后停止，不追加 API 操作、不重提、不猜扣费。
- **wait**：有记录用 `resume --host <host> --record <文件>`，否则用 `wait --host <host> --request <同任务文件>`。每窗口最多 7 次读取／300 秒，含网络与等待；遵守 retry_not_before 和 Retry-After。
- **窗口结束**：`ok=true`、`wait_outcome=window_ended` 仍为生成中。仅 `next_step=wait` 且原授权和前台会话仍有效可自动再续一个窗口；累计两个窗口、retry_deferred 或 await_user 时暂停。取消、失败回执、网络超时、未知状态或对账均停止自动续接，不重置计数或后台循环。窗口内 429 只由执行器按剩余预算处理。
- **content**：只下载已完成的原任务；用 `content --host <host> --record <文件> --index 0 --output-dir <现有绝对目录>`，无记录用含 completed 状态和确认数量的同任务请求。视频只有索引 0。
- **deliver**：按当前宿主已声明的附件工具或本地媒体格式交付 downloaded_paths 的原文件；方式不明确才读 [客户端交付](references/host-delivery.md)。下载、路径和任务号不等于交付，不能用 URL、HTML 或 SVG 代替。实际交接后才执行 `delivered --record <文件> --index 0`，不在媒体回复发出前预先确认；正常交接无需每次让用户确认已看见。done 后结束，不自动审美检查或再生成。

用户明确要求再次发送已交付文件时，用 `redeliver --host <host> --record <原记录> --index <索引>`：先校验本地原文件，缺失才取同任务结果，不重新生成。具体恢复见下方续接指南。

交付失败只重交已有文件。已完成且无对账标记的记录用 resume 本地校验，不读凭据或请求 API；跨命令复用须匹配 SHA-256、字节数、媒体类型。无记录仅重交本会话刚下载且未改变的文件；证明不足时保留文件，另选目录取同任务索引。换电脑、旧记录、对账或跨会话恢复时读 [续接与恢复](references/executor-usage.md#continuation-record)；无 ID 的未知提交不能靠记录恢复。

## 按需说明

- 图片到视频、分镜素材或视频配独立旁白／音乐才读 [组合流程](references/workflows.md)；单次请求不增加步骤。进度与交付含义不明才读 [回执说明](references/receipt-guide.md)。

- 失败／费用／支持摘要：读 `references/failure-guide.md`，只说明实际阶段、已有任务及脱敏 next_action，不展示整份 JSON、原始错误、内部 URL 或配置。本地未识别不等于未配置，不据此要求重装、重配、换模型或凭据。
- 失败后需要继续创作的选择时，按 [模型建议](references/model-alternatives.md) 核对当前连接的兼容候选，用户明确同意后才创建一次可能计费的新任务；未知、进行中、对账或交付失败仍恢复原任务，不自动换模型。
- 模型权限拒绝或用户询问分组时，按 [线上分组指引](references/model-groups.md) 获取真实名称；公开归属不证明当前 Key 权限，不自动改配置或重提。
- 用户明确检查参数才用 preflight；不创建任务、不报价、不证明权限。宿主限制读 `references/desktop-hosts.md`。
- 复杂异常按 id 查 `references/behavior-scenarios.json`；正式字段见 `references/execution-contract.json`，展示规则见 `references/task-receipt.json`，均非普通生成前置。
