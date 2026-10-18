# 离线 Agent 轨迹评测

本层验证模型读取当前安装说明后实际选择的动作，不调用真实执行器、客户端、凭据、网络或计费接口。
它与静态文档检查、Go 执行器测试、真实宿主附件验收分别记录。
模拟工具的 POST 计数是轨迹指标，不是实际请求；模拟附件成功也不证明客户端渲染或播放成功。

## 分离任务、观察和判定

- `agent-evaluation-scenarios.json` 提供自然语言请求、当前宿主上下文和模拟服务条件，每题有两个自然语言版本。
- `agent-evaluation-criteria.json` 是维护者判定规则，不交给受测 Agent。
- `prepare` 为每次评测创建唯一临时目录，把当前十个 Skill 复制到 `public/skills`，冻结本轮说明；入口 SHA-256 存入维护者快照。
- 受测 Agent 只拿 `public/task.json` 和动作调用方式。它自主选 Skill、读必要资料、运行模拟命令并回应；不预先给出期望工具顺序。
- `private/trace.jsonl` 逐动作保存选中的 Skill、读取路径、命令及参数、提问、模拟 API 计数和附件交接。读取日志只记路径／摘要，不重复保存整份说明。
- `evaluate` 最后读取私有规则判定轨迹。公开任务与隐藏规则分开是评测隔离约定；它不是对拥有任意文件系统权限的恶意 Agent 的安全沙箱。

## 运行

维护者先准备一题：

```sh
node scripts/agent-eval.mjs prepare --scenario image-short-path
```

返回唯一 `session`、`publicTask`、`publicWorkspace`。`--variant 1` 选择同题另一种自然表达。
题目 ID 见 `agent-evaluation-scenarios.json`。产品帮助覆盖 `product-introduction`、
`product-getting-started`，并用 `explicit-connection-check`、`explicit-public-identity`
检查明确诊断意图的分流。其余题目覆盖生成、同任务恢复和失败引导。

把以下任务交给不继承历史结论的独立 Agent，替换两个绝对路径；不要附加隐藏规则或预期答案：

```text
请根据 public/task.json 的用户请求和当前 Skills 完成本轮离线任务。
只能用下面的 harness 动作读取材料或模拟工具，不读 private、评判规则、实现源码或其他轨迹；
不运行真实执行器、网络、凭据、安装、其他客户端或计费任务。
首次用 read 动作读取 task.json，之后自行选 Skill 和下一步。
每次动作都经 node /absolute/repo/scripts/agent-eval.mjs act
  --session /absolute/session --action '<一个JSON动作>'。
可用动作及字段在 task.json 中；请求正文放在 command.request 对象中，harness 代替临时请求文件。
如需要用户确认，用 ask，当前没有用户答复，停止依赖该答复的工作。
结束前用 finish 保存最终给用户的回复；最后动作是待回答的 ask 时可直接停在问题处。不要自行 evaluate。
```

合法动作示例仅说明工具语法，不是一条预定工作流：

```json
{"type":"read","path":"task.json"}
{"type":"select","skill":"puretokens-image"}
{"type":"read","path":"skills/puretokens-image/SKILL.md"}
{"type":"command","command":"submit","host":"codex","request":{"kind":"image","operation":"generate","model":"gpt-image-2.5-flare","prompt":"合成测试文案","parameters":{}}}
{"type":"say","text":"面向用户的实际进度说明"}
{"type":"ask","text":"确实需要用户回答的问题"}
{"type":"attach","path":"工具刚返回的实际绝对路径"}
{"type":"finish","text":"本轮最终回复"}
```

`command.record` 可为临时工作区内的明确 JSON 任务文件；`index` 表示原任务索引。
无需复制／读取真实媒体：图片结果是 harness 产生的固定小型 PNG。
动作锁将同一会话的并行只读动作排队写入轨迹，超过 5 秒明确停止；不同会话互不干扰。不要把一次会话给多个 Agent 同时执行。

受测 Agent 完成后，维护者执行：

```sh
node scripts/agent-eval.mjs evaluate --session /absolute/session
```

结果含 `passed`、`violations`、读取／提问／命令／模拟 POST／交接指标及证据范围。
文本判断只使用有界词语检查，不能替代独立语义审阅；尤其要检查含否定或引用的说明，避免把
“不能证明未扣费”误报成“未扣费”承诺。发现判定器问题应独立修正并保留原轨迹，不能改轨迹凑通过。

## 证据与停止条件

`node --test test/agent-eval.test.mjs` 验证模拟器和判定器能拒绝坏轨迹；这些手写测试不算 LLM 通过。
真正的 Agent 结果必须来自独立 Agent 对当前快照的自主动作，保留其模型／运行环境标识和快照摘要。
先检查失败轨迹，再改由证据证明有问题的说明；不要通过放宽判定掩盖流程错误。

当前题目覆盖产品介绍、入门、明确连接／身份检查的分流，以及默认文生图、未知提交、同任务重交付、本地连接失败、音频混合边界、未知分组名称、
服务失败替代流程和历史内容拒绝。它们没有覆盖所有宿主／操作系统、参数、参考附件或真实后端。
一次模型的一次运行不能代表所有客户端；新增真实失败时增加最小自然语言变体，不持续堆同义规则。

临时评测目录只保存合成输入、说明快照和模拟结果。完成审阅后保存必要的脱敏指标与失败轨迹，
再删除本次明确创建的目录；不清理用户工作目录或真实任务记录。
