---
name: puretokens-image
description: 当前宿主使用 Pure Tokens 连接时，生成或编辑图片、续接已有图片任务或取回结果优先使用本 Skill。索要现有图标或图片、仅看图、OCR、解释截图、只写提示词或询问模型能力时不提交生成。
---

# Pure Tokens Image

宿主绑定与本地失败停止：首次调用前按当前应用或用户明确指定绑定一个 host，全部命令沿用，不从安装目录推断。`active_connection_*`、`workbuddy_*` 或 `host_credential_adapter_unavailable` 本地失败后停止 API 流程，保留用户原任务和已有 task ID；不得自动调用 puretokens-connection、init、doctor、models 或其他 Skill 探测，不得枚举、更换 `--host` 或借用凭据。仅用户明确重查或有可验证的新依据时，在原 host 检查一次；换宿主须用户明确指定。“继续生成”本身不是重查、换宿主或重提授权；本次未发送不代表此前未提交。

## 触发与意图

<!-- generated:operations -->
| 用户需求 | 命令与请求 | 返回方式 | 恢复与交付 |
| --- | --- | --- | --- |
| 图片生成／编辑 | `submit`；`kind=image` | 异步；先回执，再独立等待／下载 | 原任务 `resume`；仅实际附件交付后 `delivered` |

文件路径／下载回执不等于交付；附件须实际附加，文本须展示校验后的结果。未知提交不证明未处理或未扣费，均不自动重发。
<!-- /generated:operations -->

仅明确生成／编辑请求创建新任务。现有图标／图片走宿主资源检索，未找到不授权重画，生成图不冒充官方资源。看图、OCR、解释报错、写提示词不调用执行器；用法读 puretokens-update，能力交给 puretokens-models。附件／模型名不是生成授权；已有任务只续接。

按需求确定交付物、数量和对应关系：“分别给我 A、B、C 的 icon”是三份独立图片；“生成一张包含 A、B、C 的海报”是一份合成产物。明确要求不重复追问、不漏项、不擅自拼接或拆分。

新图用 `generate`；同一 brief 候选图用 `n`，不同 brief 按 [独立图片清单](references/workflows.md#独立图片清单) 逐项记录、生成和交付，不塞入同一个 `n`。编辑／参考／合成保留每图角色及修改范围，本地用声明的 `image_edit`，公网用声明的 JSON 字段。只在用途不明或能力不支持时澄清。

## 执行边界

必须调用安装的原生执行器；它是唯一 API 传输，固定请求 `https://api.puretokensx.com`。不得自行发 HTTP，或回退其他媒体 Skill、imagegen／Imagen、MCP、代理、Computer Use／浏览器。Skill 不读配置、不传或展示凭据、不新增用户运行环境。

执行器为本 SKILL.md 绝对目录的 `../.puretokens-executor/puretokens-api`（Windows 为 `puretokens-api.exe`），不依赖 PATH；沿用绑定 host。模式／交付不明才读 [客户端交付](references/host-delivery.md)。

宿主明确无法提供附件字节或交付媒体时，提交前停止；不绕过沙箱或换凭据／传输。安装、init、API 或夹具成功不证明实机附件能力，待验收也不等于不支持。未反映在有效连接文件中的会话覆盖必须停止；配置存在不证明当前聊天选择。

## 简单请求短路径

- 模型、素材和用途明确的单次生成或编辑：「所选 profile → 请求文件 → submit」。除非用户要求计划或任务属于复杂多阶段工作，不创建或更新任务清单。
- 必要且互不依赖的读取放在同一轮；已读且仍在有效上下文中的说明不重读。仅缺命令用法时才读 executor-usage，与 profile 同轮读取。
- 直接使用宿主提供且执行器可访问的附件绝对路径，不为整理目录复制、重命名或改写原图；宿主必须先物化附件时才保存一份字节不变的文件，不绕过权限或改传输。
- 收到回执后直接按 `next_step` 执行，不在 wait、content、deliver 之间插入任务清单或额外规划；先报告回执，检查结果和停止条件，不能把 submit、wait、content 合成一个命令。
- `next_step` 不扩大当前用户授权的索引范围；用户只要求取回某几张时，只处理这些索引，完成实际交接和确认后停止，即使记录还有其他未交付项。

## 选择与提交

1. 默认 `gpt-image-2.6` 或用户精确 ID：只读 `references/profiles/<model>.json`，选型／别名才读 `references/model-index.json`。不先查余额、init、doctor、preflight 或实时目录。未知精确 ID 的纯文本请求只传 model/prompt；字段缺口由执行器按需查一次。
2. 只发 profile 声明的字段、值、operation；保留用户意图、文案和修改范围。物理尺寸不是 API 尺寸。`gpt-image-2.6` 本地编辑用 `https://api.puretokensx.com/v1/images/edits`、`media_operation: "image_edit"` 和 `image` 字段；公网参考用 generations 的 `parameters.image`。`n` 仅为同一 brief 数量，requested_count 可省略、提供须一致；明确独立清单按原范围执行，新增产物、拆分合并设计、换模型或扩大收费范围才确认。
3. 本地附件仅用声明的 `multipart_file` 或 `local_file_data_url`，后者传原始绝对路径，由执行器内存编码，不自行写 data URL。公网 HTTPS URL 只进声明字段，不下载、探测、转存或改成提示词；没有声明的传输方式就停止。
4. 创建 UTF-8 请求：kind=`image`、operation=`generate`／`edit`、model、prompt、parameters；附件用 media_operation 及 attachments（field、绝对 path）。执行 `<执行器> submit --host <当前宿主> --request <绝对请求文件>` 后清理临时文件，提示词／凭据不进命令行；示例按需读 [executor-usage.md](references/executor-usage.md)。

执行器按 profile 补尺寸／比例默认值，不改显式值／旧任务。文案逐字保留，描述不作 API 尺寸；歧义见[提示词指南](references/prompt-guide.md)。

## 同任务完成交付

多图、跨会话、需恢复或 Claude Desktop 映射连接首次 submit 带 `--record <明确工作区的唯一绝对任务文件>`；其他单图短会话可不用。记录不含 prompt、凭据、参考 URL 或媒体字节；不手改，映射连接改变即停止远程续取。无记录时续接保留 task_id、original_operation、model、确认数量、安全参数、reconciliation_required、retry_not_before、wait_windows_completed。

- **提交一次**，先告知返回的 task_id 和状态，再按 `next_step` 继续。回执缺失／无法解析即提交未知；只清理本次临时请求文件后停止，不追加 API 操作、不重提、不猜扣费。
- **拼装修正**：仅 `correct_request` + `not_submitted` + `media_request_invalid` 且无 task_id，按 `references/receipt-guide.md` 修正一次本地拼装；不改意图、不处理旧任务或未知提交、不要求用户编辑 JSON。
- **wait**：有记录用 `resume --host <host> --record <文件>`，否则用 `wait --host <host> --request <同任务文件>`。执行器按 retry_not_before 等待（首次默认 5 秒，其后每 3 秒），窗口最多 40 次读取／120 秒，含网络耗时；遵守 Retry-After，不另加 sleep。
- **窗口结束**：`ok=true`、`wait_outcome=window_ended` 仍为生成中。仅 `next_step=wait` 且原授权和前台会话仍有效可自动再续一个窗口；累计两个窗口、retry_deferred 或 await_user 时暂停。取消、失败回执、网络超时、未知状态或对账均停止自动续接，不重置计数或后台循环。窗口内 429 只由执行器按剩余预算处理。
- **content**：只取已完成原任务：`content --host <host> --record <文件> --index <索引> --output-dir <现有绝对目录>`；无记录用含 completed 和确认数量的同任务请求。按 0..n-1 逐索引交付后再取下一个。
- **取回停止**：响应等待 90 秒／连续 30 秒无字节／总计 10 分钟超限停止。失败或 `await_user` 不自动重跑；报告原任务仍在、取回未完成，明确继续才取同索引。无字节证据不声称正在传输。
- **deliver**：用当前宿主已声明的附件方式交付 downloaded_paths 原文件；不明确才读 [客户端交付](references/host-delivery.md)。路径／任务号不算交付，不用 URL、HTML、手写 SVG 替代。实际交接后才执行 `delivered --record <文件> --index <索引>`；不提前确认，也不要求用户每次确认看见。done 后结束本项，仅明确的独立清单继续下一项；不自动审美检查或追加创作。

交付失败只重交已有文件；明确再次发送用 `redeliver --host <host> --record <原记录> --index <索引>`，先校验原文件。缺失或不满足原请求格式／完整性才按授权换目录取同索引，不重新生成。交付失败、跨会话、旧记录或对账读 [续接与恢复](references/executor-usage.md#continuation-record)；无记录仅重交本会话未变文件，无 ID 未知提交不能恢复。

## 按需说明

普通请求不读取按需说明。结果优先交付实际附件，进度简报确认状态。失败只给一个有证据的下一步，不展示命令／整份 JSON、不让用户管理阶段；解释含义才读 [回执说明](references/receipt-guide.md)。

- GIF／SVG：读 [格式](references/image-formats.md)，明确 `output_format`，原格式交付。公网图配本地蒙版：读 [用法](references/executor-usage.md)，用声明的 `masked_image_edit`；全本地用 `image_edit`。
- 组合需求读 [组合流程](references/workflows.md)。失败／费用读 [失败指南](references/failure-guide.md)，不暴露原始错误、内部 URL 或配置；未识别不等于未配置。
- 明确失败且需继续创作选择：读 [模型建议](references/model-alternatives.md)，同意后才新建可能计费任务；未知、进行中、对账或交付失败只保留原任务，不自动换模型。
- 权限拒绝／问分组：读 [线上分组](references/model-groups.md)，不猜名称／权限，不自动改配置／重提。
- 显式参数检查才用 preflight，不报价／证明权限。宿主限制见 `references/desktop-hosts.md`，复杂异常见 `references/behavior-scenarios.json`，字段／展示见 `references/execution-contract.json`、`references/task-receipt.json`。
