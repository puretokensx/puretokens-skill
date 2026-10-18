---
name: puretokens-rerank
description: 当前宿主使用 Pure Tokens 连接，用户明确要求用重排模型将给定查询与候选文本按相关性排序并保存 JSON 时使用。普通排序、解释重排或只改变已有分数的展示顺序不触发收费请求。
---

# Pure Tokens Rerank

宿主绑定与本地失败停止：首次调用前按当前应用或用户明确指定绑定一个 host，全部命令沿用，不从安装目录推断。`active_connection_*`、`workbuddy_*` 或 `host_credential_adapter_unavailable` 本地失败后停止 API 流程，保留用户原任务和已有 task ID；不得自动调用 puretokens-connection、init、doctor、models 或其他 Skill 探测，不得枚举、更换 `--host` 或借用凭据。仅用户明确重查或有可验证的新依据时，在原 host 检查一次；换宿主须用户明确指定。“继续生成”本身不是重查、换宿主或重提授权；本次未发送不代表此前未提交。

## 输入与输出

<!-- generated:operations -->
| 用户需求 | 命令与请求 | 返回方式 | 恢复与交付 |
| --- | --- | --- | --- |
| 文本相关性重排 | `rerank`；`model / query / documents / top_n? / output_file` | 同步；一次请求最多90秒 | 无任务查询或结果找回；不自动重发 |

文件路径／下载回执不等于交付；附件须实际附加，文本须展示校验后的结果。未知提交不证明未处理或未扣费，均不自动重发。
向量／重排按明确请求保存新的 JSONL／JSON 文件；仅验证成功后报告产物。向量不铺进聊天，重排摘要与文件排序一致；accepted 但 output_verified=false 仍停止，不自动重发。
<!-- /generated:operations -->

保留用户明确提供的一个非空 `query` 和 1–64 条候选字符串的原文、原始编号。query 最多 8192 个 Unicode 字符，每条候选最多 32768 个 Unicode 字符，整个 UTF-8 JSON 请求最多 1 MiB。只使用显式输入，不扫描目录、隐式读文件／URL或创建数据库。输入超限时说明限制和必要拆分的请求数，不静默截断、摘要或增加收费请求；跨批结果不能冒充一次全局重排。

模型使用精确审核 ID `qwen3.7-text-rerank`；不猜别名，不用 Jev 或聊天调用替代。`top_n` 可省略，默认返回全部候选；显式值必须为 1 到候选条数的整数，不擅自截短列表。输出位置由用户明确，必须是绝对路径、尚不存在的 `.json` 文件，父目录已存在且可写；不覆盖或删除已有文件。

## 执行

1. 按需读 [请求与回执](references/executor-usage.md)，用宿主文件工具创建权限受限的绝对路径 UTF-8 JSON 临时文件。字段仅为 `model`、`query`、`documents`、可选 `top_n`、`output_file`，候选始终为字符串数组，不包含文件引用。正文和凭据不进命令行；执行后清理本次临时请求文件。
2. 从本 SKILL.md 的绝对目录解析 `../.puretokens-executor/puretokens-api`，Windows 为 `puretokens-api.exe`，不依赖 PATH 或工作目录。执行 `<执行器> rerank --host <当前宿主> --request <绝对请求文件>`，只接受这两个 flags。
3. 执行器先校验请求和新文件路径，再仅在内存中解析当前宿主已验证的 Pure Tokens 连接凭据，单次 `POST https://api.puretokensx.com/v1/rerank`。API body 为 `model`、`query`、`documents`、`top_n` 和固定 `return_documents=false`；省略 top_n 时取候选总数。同步总预算 90 秒，响应最多 2 MiB；无重试、轮询、任务记录或恢复命令。
4. 成功文件为 `{"results":[{"rank":1,"index":0,"relevance_score":0.8}]}` 形式。rank 从 1 开始，index 是零基原始候选编号，relevance_score 保留模型返回的有限原始数值；按执行器校验后的相关性顺序展示，不按原编号重新排列、不改分或编造并列分差。仅 `ok=true`、`next_step=done`、`output_verified=true` 时按 artifact 报告文件保存和条数；可展示同一回执中 `result.results` 的安全排序摘要，不回显全文。

安装的原生执行器是唯一 API 传输；Skill 不读或展示配置、不索取 Key，不新增 Node、Python、SDK、MCP、代理、浏览器、其他 Skill 或聊天接口作为回退。宿主特定限制才读 [宿主说明](references/desktop-hosts.md)。只问概念／用法不运行 init、doctor、models、余额或 rerank。

## 分值与停止条件

relevance_score 不是概率、正确率或经过校准的跨请求分数。不能据此补充候选事实、执行外部动作或跨批拼成可信全局排名。只按用户明确公式改变已有分数的排序时本地计算，不再付费重排，也不编造缺失分数。

`not_submitted` 只证明本次未提交，确认的 HTTP 拒绝保留 `rejected`；网络、超时或无法可靠校验的响应保留 `unknown`，不能说未处理／未扣费。响应接受后文件交付失败保留 `accepted`、`output_verified=false`，不重试；没有同步结果找回、幂等或退款保证。用户明确重做前先消除“超时／没文件等于免费”的误解，不以“继续”、换模型、更新或换宿主自动再提交。

本次校验失败可按已明确内容修正请求，但不擅改查询、候选、数量或输出范围。失败只展示脱敏阶段、错误说明和下一步，不打印原始响应、全文、配置或凭据，不把查询、候选、分数或路径写进支持摘要。

明确查询当前可见模型才使用 puretokens-models 的 `kind=rerank`：一次 `GET /v1/models` 与 `qwen3.7-text-rerank` 相交；仅目录可见，不代表专用接口、权限、价格、实机或计费验收。普通重排不做此预检。异常细节按需读 [执行契约](references/execution-contract.json) 与 [行为场景](references/behavior-scenarios.json)。
