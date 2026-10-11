---
name: puretokens-evaluate
description: 当前宿主使用 Pure Tokens 连接，用户要求用 Jev／JEV／TypeSafe 对文本或结构化文本做分类、选项选择、按标准评分或是否判断时使用。仅解释概念、设计评估标准、看图、生成媒体或普通聊天不触发付费评估。
---

# Pure Tokens Evaluate

宿主绑定与本地失败停止：首次调用前按当前应用或用户明确指定绑定一个 host，全部命令沿用，不从安装目录推断。`active_connection_*`、`workbuddy_*` 或 `host_credential_adapter_unavailable` 本地失败后停止 API 流程，保留用户原任务和已有 task ID；不得自动调用 puretokens-connection、init、doctor、models 或其他 Skill 探测，不得枚举、更换 `--host` 或借用凭据。仅用户明确重查或有可验证的新依据时，在原 host 检查一次；换宿主须用户明确指定。“继续生成”本身不是重查、换宿主或重提授权；本次未发送不代表此前未提交。

## 适用范围

<!-- generated:operations -->
| 用户需求 | 命令与请求 | 返回方式 | 恢复与交付 |
| --- | --- | --- | --- |
| Jev 文本评估 | `evaluate`；`model / state / questions` | 同步；一次请求最多90秒 | 无任务查询或结果找回；不自动重发 |

文件路径／下载回执不等于交付；附件须实际附加，文本须展示校验后的结果。未知提交不证明未处理或未扣费，均不自动重发。
<!-- /generated:operations -->

把用户明确提供的文本、记录或对话作为 `state`，把独立判断放在 `questions` 中。评估结果是结构化数值／选项，不是聊天回复、事实证明或执行外部操作的授权。只问 Jev 用法／能力时读本地说明，不运行 init、doctor、models 或评估。

| 需求 | 问题类型 | 必需信息 |
| --- | --- | --- |
| 工单分类、选择候选项 | `choice` | 1–255 个明确选项；名称已足以表达含义时，描述可为 null |
| 按相关性、质量等标准评分 | `score` | 2–10 档从低到高排列的标准 |
| 是否紧急、是否符合条件 | `noul` | 明确的是／否问题；可描述 true、false |

保留指定文案、选项（包括 other）的含义及评分档位顺序，不擅自添加评分标准、阈值或事实。只澄清缺失的必要信息或冲突。首版仅接受文本／结构化文本；图片、视频、音频不能直接评估，不下载附件、做 OCR 或把媒体概括为文字后冒充 Jev 看过原件。用户另行明确提供或同意使用文本时，才评估该文本。

## 最短执行路径

1. 确定当前 host。默认模型 `jev-latest`，另支持精确 `jev-1.13.0`、`jev-preview`；“Jev”作为产品名称使用默认模型，不把 `jev` 当 API ID。未知 ID 不猜测映射。用户明确查可用模型才交给 puretokens-models 的 evaluation 查询。
2. 同一 `state` 的独立问题合并到一个请求；问题不能引用本次尚未产生的另一个答案。不把独立记录混成一个状态再假装逐条评分。单次本地限制为 1 MiB 请求、最多 64 个问题、JSON 深度 32；这不是服务端 token 上限或价格保证。超限时先说明必要拆分和预计请求次数，范围未明确才澄清，不静默拆成更多收费请求；不截断 state 或改变问题含义以绕过限制。
3. 按需读 [命令与请求示例](references/executor-usage.md)，用宿主文件工具创建权限受限的 UTF-8 JSON 临时文件，保留文本原样。仅有 `model`、`state`、`questions`；不传 stream、async、附件或自选地址。
4. 从本 SKILL.md 绝对目录解析 `../.puretokens-executor/puretokens-api`，Windows 为 `puretokens-api.exe`。执行 `<执行器> evaluate --host <当前宿主> --request <绝对请求文件>`，随后清理本次临时文件。提示词、state 和凭据不进命令行。
5. 执行器同步等待最多 90 秒，仅 POST 一次到 `https://api.puretokensx.com/typesafe/v1/systemone`。`next_step=done` 才展示成功结果；失败按回执停止。没有 task_id、wait、resume 或 content，不追加查询或下载。

必须使用安装的原生执行器；它是唯一 API 传输。仅执行器在内存中使用当前宿主匹配连接，不读／展示宿主配置、不索取另一套 Key，不新增 Node、Python、SDK、MCP、代理、浏览器或其他 Skill 作为回退。沿用当前已绑定的 host；宿主与模式说明见 [客户端交付](references/host-delivery.md)。Jev 直接展示文本或结构化结果，不要求媒体附件工具。

## 结果与停止条件

- choice 展示选项及其概率；score 展示分数与原评分档位（从 0 起，可为小数）；noul 展示“是”的概率，不私自设 0.5 阈值转成确定结论。只报告返回且通过校验的数据。
- confidence 是模型给出的置信度，不是经过验证的正确率。用户另有明确决策阈值时可标注是否达到，不能由此执行退款、删除、发送或其他外部操作。
- 只改变已有分数的展示权重／排序时，根据用户明确公式本地计算，标明为派生结果，不再调用 Jev；缺失分数不编造。
- 批量请求按用户指定的有限范围逐项处理，串行、每项一次；先说明记录数和预计请求次数，同一 state 能合并的问题不拆请求。用户已给定范围不重复索要同一授权；范围或拆分不明确才澄清。记录已完成项，遇失败或取消立即停止，用户明确继续时只处理未尝试项，未知项不自动重做。没有后台循环或无限重试。
- `submission_outcome=unknown` 表示响应丢失、超时或无法可靠校验；不能说未处理／未扣费，不自动重提、换模型或更新。没有同步结果找回／幂等保证。用户明确要求重做时先说明可能再次计费，消除“超时等于免费”的误解后再按其决定执行。
- 失败仅展示阶段、脱敏说明和下一步；不打印原始请求／响应或把完整结果当支持摘要。详细字段读 `references/execution-contract.json`，异常场景读 `references/behavior-scenarios.json`；均非普通评估前置。
- 需要解释进度或区分未知与失败时才读 [回执说明](references/receipt-guide.md)，不增加诊断请求。
