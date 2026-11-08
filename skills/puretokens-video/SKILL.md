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

仅明确生成／编辑请求创建新任务。分析视频、解释报错、写脚本／提示词不调用执行器；用法读 puretokens-update，能力交给 puretokens-models。附件或“视频”本身不是生成授权；已有任务只续接。

文字生成用 `generate`；“让照片动起来”是首帧，仅参考人物外观用 `reference_image_video`，不能互换。A 过渡到 B 需声明首尾帧，明确顺序；动作／节奏参考不等于编辑原视频。两张附件不自动等于首尾帧，已说明角色不重复追问。数量、组合、本地／公网传输须 profile 声明，不省略素材或自动换模型。已有任务用 `resume` 续接，不新 POST。

## 执行边界

必须调用安装的原生执行器；它是唯一 API 传输，固定请求 `https://api.puretokensx.com`。不得自行发 HTTP，或回退通用视频／其他媒体 Skill、imagegen／Imagen、MCP、代理、Computer Use／浏览器。Skill 不读配置、不传或展示凭据、不新增用户运行环境。

执行器为本 SKILL.md 绝对目录的 `../.puretokens-executor/puretokens-api`（Windows 为 `puretokens-api.exe`），不依赖 PATH；沿用绑定 host。模式／交付不明才读 [客户端交付](references/host-delivery.md)。

宿主明确无法提供附件字节或交付媒体时，提交前停止；不绕过沙箱或换凭据／传输。安装、init、API 或夹具成功不证明实机附件能力，待验收也不等于不支持。未反映在有效连接文件中的会话覆盖必须停止；配置存在不证明当前聊天选择。

## 简单请求短路径

- 模型、素材和用途明确的单次生成或编辑：「所选 profile → 请求文件 → submit」。除非用户要求计划或任务属于复杂多阶段工作，不创建或更新任务清单。
- 必要且互不依赖的读取放在同一轮；已读且仍在有效上下文中的说明不重读。仅缺命令用法时才读 executor-usage，与 profile 同轮读取。
- 直接使用宿主提供且执行器可访问的附件绝对路径，不为整理目录复制、重命名或改写原图、音视频；宿主必须先物化附件时才保存一份字节不变的文件，不绕过权限或改传输。
- 收到回执后直接按 `next_step` 执行，不在 wait、content、deliver 之间插入任务清单或额外规划；先报告回执，检查结果和停止条件，不能把 submit、wait、content 合成一个命令。

## 选择与提交

1. 默认 `MiniMax-H3` 或用户精确 ID：只读 `references/profiles/<model>.json`，选型／别名才读 `references/model-index.json`。不先查余额、init、doctor、preflight 或实时目录。未知精确 ID 的纯文本请求只传 model/prompt；字段缺口由执行器按需查一次。H3 参考仅支持公网 HTTPS URL，不托管本地附件或自动换模型；旧 `minimax_h3` 不是别名，原任务保留原型号。
2. 保留主体、运动及素材角色，只发 profile 声明的字段、值、operation。物理尺寸不是 API 尺寸。首帧用 `image_to_video`／`first_frame_video`，参考用 `reference_image_video`／`reference_video`／`reference_audio`，编辑用 `video_edit`；组合按 profile。每任务一个视频；声音字段按模型声明，不混用 `audio` 与 `generate_audio`。
3. 本地附件仅用声明的 `multipart_file` 或 `local_file_data_url`，后者传原始绝对路径，由执行器内存编码并保留角色，不自行写 data URL。公网 HTTPS URL 只进声明字段，不下载、探测、转存或改成提示词；没有声明的传输方式就停止。
4. 创建 UTF-8 请求：kind=`video`、operation=`generate`／`edit`、model、prompt、parameters；附件用 media_operation 及 attachments（field、绝对 path）。执行 `<执行器> submit --host <当前宿主> --request <绝对请求文件>` 后清理临时文件，提示词／凭据不进命令行；示例按需读 [executor-usage.md](references/executor-usage.md)。

原样提示词不翻译、润色或增词。画质／节奏描述不自动变成 API 分辨率、时长或声音参数；明确输出规格按 profile 校验。模糊镜头／参考关系才读 [提示词指南](references/prompt-guide.md)。

## 同任务完成交付

视频首次 submit 默认带 `--record <明确工作区的唯一绝对任务文件>`；Claude Desktop 映射连接必须带记录。记录不含 prompt、凭据、参考 URL 或媒体字节；不手改，映射连接改变即停止远程续取。无记录时续接保留 task_id、original_operation、model、确认数量、安全参数、reconciliation_required、retry_not_before、wait_windows_completed。

- **提交一次**，先告知返回的 task_id 和状态，再按 `next_step` 继续。回执缺失／无法解析即提交未知；只清理本次临时请求文件后停止，不追加 API 操作、不重提、不猜扣费。
- **拼装修正**：仅 `correct_request` + `not_submitted` + `media_request_invalid` 且无 task_id，按 `references/receipt-guide.md` 修正一次本地拼装；不改意图、不处理旧任务或未知提交、不要求用户编辑 JSON。
- **wait**：有记录用 `resume --host <host> --record <文件>`，否则用 `wait --host <host> --request <同任务文件>`。每窗口最多 7 次读取／300 秒，含网络与等待；遵守 retry_not_before 和 Retry-After。
- **窗口结束**：`ok=true`、`wait_outcome=window_ended` 仍为生成中。仅 `next_step=wait` 且原授权和前台会话仍有效可自动再续一个窗口；累计两个窗口、retry_deferred 或 await_user 时暂停。取消、失败回执、网络超时、未知状态或对账均停止自动续接，不重置计数或后台循环。窗口内 429 只由执行器按剩余预算处理。
- **content**：只取已完成原任务：`content --host <host> --record <文件> --index <已授权索引> --output-dir <现有绝对目录>`；无记录用含 completed 和确认数量的同任务请求。普通视频索引 0，明确尾帧为 1，逐份交付，不扩大范围。
- **取回停止**：响应等待 90 秒／连续 30 秒无字节／总计 10 分钟超限停止。失败或 `await_user` 不自动重跑；保留原任务，明确继续才取同索引。无字节证据不声称正在传输。
- **deliver**：用当前宿主已声明的附件方式交付 downloaded_paths 原文件；不明确才读 [客户端交付](references/host-delivery.md)。路径／任务号不算交付，不用 URL、HTML、SVG 替代。实际交接后才执行 `delivered --record <文件> --index <索引>`；不提前确认，也不要求用户每次确认播放。授权产物交付后结束，不自动审美检查或再生成。

交付失败只重交已有文件；明确再次发送用 `redeliver --host <host> --record <原记录> --index <索引>`，先校验原文件，缺失才取同任务同索引，不重新生成。交付失败、跨会话、旧记录或对账读 [续接与恢复](references/executor-usage.md#continuation-record)；无记录仅重交本会话未变文件，无 ID 未知提交不能恢复。

## 按需说明

普通请求不读取按需说明。结果优先交付实际附件。失败只给一个有证据的下一步，不展示命令／整份 JSON、不让用户管理阶段；解释含义才读 [回执说明](references/receipt-guide.md)。

- Grok 编辑：读 [编辑限制](references/grok-edit.md)。非默认视频组合不明确：读 [新增参数](references/reviewed-video-usage.md)；`sourceReview` 只证明契约审核，`Super Vega` 内部空格保留。
- 独立尾帧：读 [尾帧用法](references/reviewed-video-usage.md)，仅精确 Super Vega Pro 声明的 `return_last_frame=true`；记录中 requested_count=1，索引 0 是视频、1 是尾帧。先视频再尾帧逐份交付，不漏交、不跟随产物 URL、不本地截帧／重新生成。
- Kling 高级输入：读 [专用指南](references/kling-native-usage.md)，六个精确 ID 的 `media_operation=native_video`；不创建主体／克隆音色。多阶段需求才读 [组合流程](references/workflows.md)。
- Wan 原生／文件参考：读 [专用指南](references/wan-native-usage.md)。
- 失败／费用／支持摘要：读 [失败指南](references/failure-guide.md)，不暴露原始错误、内部 URL 或配置；未识别不等于未配置。
- 明确失败且需继续创作选择：读 [模型建议](references/model-alternatives.md)，同意后才新建可能计费任务；未知、进行中、对账或交付失败只保留原任务，不自动换模型。
- 权限拒绝／询问分组：读 [线上分组](references/model-groups.md)，不猜名称或当前 Key 权限，不自动改配置／重提。
- 显式参数检查才用 preflight，不报价或证明权限；宿主限制读 `references/desktop-hosts.md`。复杂异常查 `references/behavior-scenarios.json`；字段／展示契约见 `references/execution-contract.json`、`references/task-receipt.json`。
