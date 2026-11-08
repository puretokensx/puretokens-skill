# 进度、交付与失败说明

仅在需要解释回执时读取。依据原执行器 JSON 与本次宿主实际附件结果，不从终端退出码、
文件路径、URL 或供应商原始响应自行推断成功。不为补充说明再发请求。

## 先判断是否必须停止

优先处理 `reconciliation_required=true`、`submission_outcome=unknown`、
`ok=false` 或 `next_step=await_user`，不能被 `status=completed`、已有路径或编号覆盖。下述明确的新请求本地拼装修正是唯一窄例外。
有编号仅表示能关联原任务，不证明已完成或已扣费。回执缺失／无法解析时也可能已经提交，
停止自动操作，不重发；保留用户原任务及已有记录。

仅图片／视频新请求同时返回 `next_step=correct_request`、`failure_phase=validation`、
`submission_outcome=not_submitted`、`local_error_code=media_request_invalid`、受控 `validation` 且无 task_id／对账时，
可依据所选 profile 修正一次 Agent 自己的拼装错误，继续原已授权请求。不得改变用户模型、内容、附件、数量、格式或输出范围；
用户明确值不受支持、修正有歧义或再次失败时，简短说明并询问，不把 JSON 编辑交给用户。
仅有 validation 阶段不够；unknown、已有记录、权限／传输／附件失败不适用，不自动换模型、补预检或重提。
`validation` 只含受控字段、原因及有限公开选项，不含输入值，不进入任务记录或支持摘要。

## 根据回执展示

| 已确认事实 | 给用户的简短说明 | 后续 |
| --- | --- | --- |
| 异步 submit 成功，明确 accepted | 已接受，任务编号为原 task_id | 必须先展示回执，再按 next_step 继续 |
| next_step=wait，正常窗口结束 | 仍在生成，已保留原任务 | 仅按该操作剩余前台窗口继续，不重置计数 |
| next_step=content | 已生成，接下来取回文件 | 只下载原任务；此时不能说已交付 |
| next_step=deliver，文件已校验 | 文件已下载 | 实际附加原生文件；失败只恢复交付 |
| 本次宿主已确认实际附件交付 | 已交付本次附件 | 异步记录才执行 delivered；同步音频没有该命令 |
| 异步 next_step=done | 原记录显示已完成交付 | 不重复下载、不声称本次又附加了文件 |
| 用户明确要求再次发送已交付的文件 | 按原记录、原索引执行 redeliver | 优先校验原本地文件；需要时只取同任务结果，实际交付后再 delivered |
| 同步转写／Jev next_step=done | 展示校验后的文本／评估结果 | 无任务轮询、下载或 delivered |

普通进度一句话即可；不逐次播报轮询、不展示内部命令或整份 JSON。
多图报告实际已交付数量及剩余项；各操作等待窗口不同，沿用其原规则。
结果优先：成功回复先交付实际附件或校验后的文本，再给必要的一句说明。
正常路径由 Agent 执行，用户无需管理 submit／wait／content、任务文件或 JSON。

## 失败时说清三件事

说明“在哪一步停止、已保留什么、可以怎样继续”。仅使用回执中的受控字段和脱敏说明。
失败只给一个与当前证据相符的下一步：依据原执行器的 next_action 或本次已确认的宿主交付失败，
选择当前阻塞点；不并列建议重装、换客户端、换模型和跑诊断，也不自动执行这些动作。
原因未知时直接说明未确认，只请求解决当前阻塞所必需的信息；已知任务和原文件由 Agent 保留，
不让用户抄任务号、编辑 JSON 或理解内部阶段后才能继续。

| 阶段／证据 | 说明与处理 |
| --- | --- |
| validation + not_submitted | 本次未提交；报告具体不支持的参数／附件条件，不推断此前没有任务 |
| submission + unknown | 提交结果未确认，可能已处理；不自动再提交或承诺未扣费 |
| submission + rejected | 本次请求被拒绝；按 next_action 处理，不擅自换模型或删参数 |
| status 失败／对账 | 原任务保留，当前状态未确认；停止自动等待，不报生成失败或退款 |
| 异步 content 失败 | 原任务不重做；按回执在用户明确继续后取原结果 |
| 同步音频 accepted + content 失败 | 响应已接受但未保存为可用文件；没有结果查询，不能照搬异步 content 恢复 |
| 宿主附件未交付 | 执行器未证明宿主交付；异步用原记录 resume，同步音频用原 artifact 的 audio-verify |

`task_connection_changed` 表示映射任务的连接已改变；本次未发请求。保留原任务与已下载文件，不修改 connection_id、不借 Key、不重提；已验证本地文件可继续交付，远程恢复需支持方核对原连接。

跨会话失败回执的 `failure_source=task_record` 表示原任务记录的受控分类，
不是本次再次调用的结果。按 `failure_category` 与已有公开 `api_error_code` 处理；
旧记录没有分类时原因未知，不能据此排除内容、权限或额度限制，也不能主动推荐换模型。
`redelivery_index` 表示本次明确请求再次交付的索引；历史 delivered 数量不证明本次已交付。

宿主交付失败是当前宿主的观测，不伪造执行器的 failure_phase 或 support 字段。
本地连接未确认不等于未配置，不要求用户重装、粘贴 Key 或切换客户端。
需要私密支持时仅提供原回执已有的 support 块；不补采配置、提示词、媒体、路径或原始日志。

### 提交没有 HTTP 响应

`support.transport_error_code` 是本次 HTTP 调用的本地类型诊断，不是服务端错误码。
`api_request_timeout` 为超时，`api_request_cancelled` 为取消，`api_dns_failed` 为解析失败或超时，
`api_tls_failed` 为 TLS／证书错误，`api_network_permission_denied` 为网络操作被系统拒绝，
`api_connection_failed` 为连接失败／中断，`api_transport_failed` 为未能细分的传输错误。
仅说明已观测类别，不猜代理、网络故障责任方或服务端负载；不关闭证书校验、绕过沙盒，
也不一概要求完全访问权限。按回执 next_action 给出针对性处理，保留原任务记录。

`api_request_attempted=true` 只说明已调用 HTTP 客户端，不证明到达服务器。
提交 `unknown` 时，即使是 DNS／TLS 错误，也不能承诺未受理／未扣费，不自动重发或换模型。
没有 `http_status` 不是“HTTP 0”，也不能说接口返回了空任务编号。旧回执缺少类型诊断时，
原因仍未知；不能靠对话总耗时倒推超时。`observed_at` 是回执时间，不是提交开始时间。

## 仅在用户询问耗时时

可选 `timings_ms` 是本次命令的本地毫秒计时。`total` 截止回执快照，
`preparation` 为命令准备／本地处理，`catalog` 为目录读取，`request` 为请求与响应处理，
`wait` 为实际前台等待，`content_response` 为内容请求至响应头／请求失败（含连接及网络／服务端等待），
`download` 为响应头后的正文读取／落盘（含等待字节），`verification` 为本地文件校验。
各阶段互斥；未发生的阶段省略，已测但不足 1 毫秒可为 0。它不含此前命令、Agent 思考、
宿主交接或用户播放时间，也不能将网络响应时间全归为服务端生成。
不为计时重跑任务或补发请求；平时不向用户展开这份指标。
旧版本未拆分的 `download` 不能反推响应等待或纯传输耗时；新字段也不是服务端独占计时，
更不证明文件持续传输。`media_content_response_timeout` 表示响应等待超限，
`media_download_stalled` 表示读取无字节进展超限；两者都保留原任务且停止自动取回。
HTTP 200 只证明收到响应状态，不证明完整图片／音视频已经取回或交付。
