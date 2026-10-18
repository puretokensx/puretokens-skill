# 文本向量请求与回执

```text
<执行器> embedding --host <当前宿主> --request <绝对 UTF-8 JSON 文件>
```

只接受 `--host` 和 `--request`。示例中输出路径须替换为用户明确的绝对新文件路径；Windows 使用如 `C:\work\embeddings.jsonl` 的绝对路径，在 JSON 中转义反斜杠。

```json
{
  "model": "qwen3.7-text-embedding",
  "input": ["雨天的城市街道", "晴天的山间湖泊"],
  "output_file": "/workspace/results/embeddings.jsonl"
}
```

请求仅含这三个字段，不接受 `dimensions`、`user`、自选地址、文件引用或输出目录 flags。审核 ID 为 `qwen3.7-text-embedding`、`qwen3.7-text-embedding-flash`，不猜别名。`input` 为 1–64 条非空字符串，每条最多 32768 个 Unicode 字符；整个请求最多 1048576 字节，响应最多 2097152 字节。父目录必须已存在且可写，文件必须是尚不存在的绝对 `.jsonl`；不覆盖、跟随符号链接改写或删除原文件。请求和路径校验先于凭据读取或网络请求。

固定 API 为 `POST https://api.puretokensx.com/v1/embeddings`，body 仅有 `model`、`input` 和固定 `encoding_format: "float"`；`output_file` 是本地产物路径，不发送到 API。单次同步 90 秒，不重试、不轮询。

每行文件为 `{index,dimensions,embedding}`：index 是零基原输入编号，按编号排序；dimensions 是该向量长度，embedding 为校验后的有限数值数组。模型名不构成维度证据；条数、编号和维度一致性由执行器校验。文件是用户产物，不能写入受管 Skill、隐藏缓存或数据库。

成功为 `ok=true`、`command=embedding`、`submission_outcome=accepted`、
`next_step=done`、`output_verified=true`。`artifact` 含 `path`、
`format=jsonl`、`item_count`、`dimensions`、`sha256`、`bytes`；
摘要校验值不是独立的实机交付证据。聊天只报告条数、实际维度、用户文件路径，
不读取或粘贴整个向量。没有验证完成的文件不能报成功。

失败沿用 evaluation 回执字段：`ok`、`command`、`submission_outcome`、`next_step`，以及适用的 `failure_phase`、`local_error_code`、`http_status`、`api_error_code`、`error_message`、`next_action`；`output_verified=false`，错误经 typed support 脱敏包装，不发明字段、错误码或额外查询。`not_submitted` 只说明本次未提交；`rejected` 是已确认拒绝；`unknown` 不证明未处理或未扣费。响应接受后文件写入／校验失败仍为 `accepted`、`failure_phase=content`、`local_error_code=embedding_output_unavailable`，没有 result／artifact，保留费用不确定性，不自动重试。进程中断且无回执同样不能断言未发请求。

没有 `task_id`、wait、resume、content、查找同步结果或服务端幂等保证。用户明确重做前，须消除“没有文件／超时等于未扣费”的误解；不借更新、换模型或换宿主自动再提交。
