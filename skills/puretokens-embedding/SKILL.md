---
name: puretokens-embedding
description: 当前宿主使用 Pure Tokens 连接，用户明确要求将给定文本生成 embedding／向量并保存为 JSONL 时使用。只解释向量、设计检索方案、普通聊天或建设数据库不触发请求。
---

# Pure Tokens Embedding

宿主绑定与本地失败停止：首次调用前按当前应用或用户明确指定绑定一个 host，全部命令沿用，不从安装目录推断。`active_connection_*`、`workbuddy_*` 或 `host_credential_adapter_unavailable` 本地失败后停止 API 流程，保留用户原任务和已有 task ID；不得自动调用 puretokens-connection、init、doctor、models 或其他 Skill 探测，不得枚举、更换 `--host` 或借用凭据。仅用户明确重查或有可验证的新依据时，在原 host 检查一次；换宿主须用户明确指定。“继续生成”本身不是重查、换宿主或重提授权；本次未发送不代表此前未提交。

## 输入与输出

<!-- generated:operations -->
| 用户需求 | 命令与请求 | 返回方式 | 恢复与交付 |
| --- | --- | --- | --- |
| 文本向量 | `embedding`；`model / input / output_file` | 同步；一次请求最多90秒 | 无任务查询或结果找回；不自动重发 |

文件路径／下载回执不等于交付；附件须实际附加，文本须展示校验后的结果。未知提交不证明未处理或未扣费，均不自动重发。
向量／重排按明确请求保存新的 JSONL／JSON 文件；仅验证成功后报告产物。向量不铺进聊天，重排摘要与文件排序一致；accepted 但 output_verified=false 仍停止，不自动重发。
<!-- /generated:operations -->

只处理用户明确提供的 1–64 条非空文本，保留内容和顺序。每条最多 32768 个 Unicode 字符，整个 UTF-8 JSON 请求最多 1 MiB；不把路径或 URL 当成隐式读取指令，不扫描目录、不自动抓取文件、不建立向量库。超限时说明限制和必要拆分的请求数，范围未明确才澄清，不静默截断、改写或扩大收费批次。

模型必须是精确审核 ID：`qwen3.7-text-embedding` 或 `qwen3.7-text-embedding-flash`。用户未指定模型时明确选择其中一个，不从名称猜别名、维度或可用性。缺少输出位置时确认用户要保存的工作区路径；输出必须是绝对路径、尚不存在的 `.jsonl` 文件，父目录已存在且可写，不覆盖或删除已有文件。

## 执行

1. 按需读 [请求与回执](references/executor-usage.md)，用宿主文件工具创建权限受限的绝对路径 UTF-8 JSON 临时文件。请求仅含 `model`、`input`、`output_file`；`input` 始终为字符串数组。正文和凭据不放进命令行，执行后清理本次请求文件。
2. 从本 SKILL.md 的绝对目录解析 `../.puretokens-executor/puretokens-api`，Windows 使用 `puretokens-api.exe`；不依赖 PATH 或工作目录。执行 `<执行器> embedding --host <当前宿主> --request <绝对请求文件>`，只有这两个 flags。
3. 执行器先校验请求与新文件路径，再在内存中解析当前宿主已验证的 Pure Tokens 连接凭据，只向 `POST https://api.puretokensx.com/v1/embeddings` 提交一次；API body 固定 `model`、`input`、`encoding_format=float`。同步总预算 90 秒，响应最多 2 MiB，无重试、轮询、任务记录或恢复命令。
4. 成功文件每行仅有 `{index,dimensions,embedding}`，按原输入的零基 `index` 排序。仅 `ok=true`、`next_step=done`、`output_verified=true` 时，按 artifact 报告条数、维度和用户文件路径；不把向量数组铺进聊天，不把保存文件宣称为已建库或完成检索。

安装的原生执行器是唯一 API 传输；仅它可读取当前宿主声明的有效连接并在内存中用凭据。Skill 不读或展示配置、不索取 Key、不增加 Node、Python、SDK、MCP、代理、浏览器或其他 Skill 作为回退。宿主特定限制才读 [宿主说明](references/desktop-hosts.md)。仅问用法／支持范围时读本地说明，不运行 init、doctor、models、余额或 embedding。

## 停止条件

`not_submitted` 只证明本次未提交；HTTP 拒绝为 `rejected`。网络中断、超时或无法可靠校验的响应保留 `unknown`，不能声称未处理／未扣费。响应已接受但文件交付失败保留 `accepted`、`output_verified=false`，也不自动重发；没有结果找回、幂等或退款保证。新请求须由用户知情决定，不能用“继续”自动重复未知或已接受的请求。

本次校验失败可按已明确内容修正请求，但不改变用户文本、模型或输出范围；未知提交不能当成本地校验重做。只展示脱敏阶段、错误说明与下一步，不打印原始响应、输入正文或凭据，不把向量、文件路径和输入放入支持摘要。

明确查询当前可见模型才使用 puretokens-models 的 `kind=embedding`；仅一次 `GET /v1/models` 与上述精确 ID 相交，不是普通执行的前置检查，目录可见不证明专用接口、权限、价格、实机或计费已验收。异常细节按需读 [执行契约](references/execution-contract.json) 和 [行为场景](references/behavior-scenarios.json)。
