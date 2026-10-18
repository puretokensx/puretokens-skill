# 文本重排请求与回执

```text
<执行器> rerank --host <当前宿主> --request <绝对 UTF-8 JSON 文件>
```

只有 `--host` 和 `--request`；正文、候选和凭据均不放命令行。示例中的输出须替换成用户明确的绝对新文件路径；Windows 绝对路径在 JSON 中转义反斜杠。

```json
{
  "model": "qwen3.7-text-rerank",
  "query": "如何为检索结果保留原始编号？",
  "documents": ["通过 index 关联原候选。", "通过颜色装饰界面。"],
  "top_n": 2,
  "output_file": "/workspace/results/rerank.json"
}
```

`model`、`query`、`documents`、`output_file` 必填，只有 `top_n` 可省略。model 仅为精确审核 ID `qwen3.7-text-rerank`。query 非空且最多 8192 个 Unicode 字符；documents 为 1–64 条字符串，每条最多 32768 个 Unicode 字符；整个请求最多 1048576 字节，响应最多 2097152 字节。`top_n` 是 1..候选条数的整数，省略时返回全部。输入保持原样，不自动摘要、读文件、抓取 URL、拆批或建库。

输出是用户明确的绝对新 `.json` 文件，父目录须存在且可写。不覆写、删除或通过符号链接改写已有文件；校验请求和路径先于读取凭据或网络请求。API 请求固定为 `POST https://api.puretokensx.com/v1/rerank`，仅 `model`、`query`、`documents`、`top_n`（缺省取候选总数）、固定 `return_documents: false`；不发送本地 output_file，不接受自选 endpoint、stream 或其他参数。

同步总预算 90 秒，单次 POST。文件为：

```json
{
  "results": [
    {"rank": 1, "index": 0, "relevance_score": 0.8},
    {"rank": 2, "index": 1, "relevance_score": -0.1}
  ]
}
```

示例仅说明格式，不代表实际返回值。rank 从 1 起；index 是 0 起的原始候选编号；relevance_score 保留原始有限数值，不裁剪到 0–1、不转为概率或准确率。文件按校验后的相关性排序；保留原始 index 关联，不能把排序位置当成原编号。摘要与文件须是同一排序，不能重新评分或添加候选全文。结果条数由 top_n 决定，编号范围／唯一性和有限数值须通过执行器校验。

成功为 `ok=true`、`command=rerank`、`submission_outcome=accepted`、
`next_step=done`、`output_verified=true`。`artifact` 含 `path`、
`format=json`、`item_count`、`sha256`、`bytes`，无 dimensions。
可显示同一回执中 `result.results` 的相同排序摘要及文件路径，
但不用原始 API 响应补充缺失数据。摘要校验值不是独立的实机交付证据。
不把相关分当事实证明、执行外部动作的授权或跨请求可比的全局分值。

失败沿用 evaluation 回执：`ok`、`command`、`submission_outcome`、`next_step`，以及适用的 `failure_phase`、`local_error_code`、`http_status`、`api_error_code`、`error_message`、`next_action`；`output_verified=false`，错误经 typed support 脱敏包装。`not_submitted` 仅表示本次未发请求，`rejected` 是已确认拒绝；网络中断、超时、无回执或无法校验的响应不能证明未处理／未扣费，保留 `unknown`。响应接受但文件交付失败为 `accepted`、`failure_phase=content`、`local_error_code=rerank_output_unavailable`，无 result／artifact，不再请求。没有 task_id、wait、resume、content、同步找回或幂等保证。

失败时只显示脱敏阶段、原因和下一步，不输出原始 API 错误／请求、查询、候选全文、配置或凭据。不把文件路径、分数与输入写进支持摘要。用户知情明确重做前不自动重试、换模型或换宿主。
