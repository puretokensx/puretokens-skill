# Gemini TTS / Lyria asynchronous audio

仅精确模型 `gemini-3.8-flash-tts`、`lyria-3.5`、`lyria-3-pro-preview`
使用本指南。默认配音／音乐模型不变。先读选中 profile，不用目录或 init 预检。

## 请求与边界

`kind=audio_task`、`operation=generate`、`requested_count=1`。`prompt` 是
Gemini 的原样朗读文本，执行器只在发请求时映射为 `input`；Lyria 则原样作为
音乐描述 `prompt`。不得改写正文、歌词或自动拆分成多次计费请求。
临时请求 JSON 使用受限权限、绝对路径，正文不进入命令行或任务记录。

- Gemini：可选 `voice`（省略为 Kore）、`instructions`、`temperature` 0–2，
  `speakers` 若提供，恰好两项 `{name,voice}` 且姓名不同。用户指定双人配音
  时保留明确的角色与音色，缺失才澄清。仅 WAV，不支持 PCM 交付。
- Lyria：可选原样 `lyrics`、`temperature` 0–2；仅 MP3。不要添加 StepFun
  的 `instrumental`、`caption`，也不要添加 duration、voice、group 或参考文件。
- 执行器本地上限：正文／歌词分别 8000 字符，配音指示 4000 字符，音色和
  角色名分别 120 字符，请求 JSON 总量 64 KiB，单个输出 32 MiB。这些是
  Skill 安全边界，不是供应商时长、价格或可用性承诺。超限停止，不静默截断。

Gemini 示例：

```json
{
  "kind": "audio_task",
  "operation": "generate",
  "model": "gemini-3.8-flash-tts",
  "prompt": "欢迎来到今天的节目。",
  "parameters": { "voice": "Kore", "response_format": "wav" },
  "requested_count": 1,
  "output_dir": "/absolute/user/output"
}
```

Lyria 示例：

```json
{
  "kind": "audio_task",
  "operation": "generate",
  "model": "lyria-3.5",
  "prompt": "一段轻柔的钢琴纯音乐",
  "parameters": { "response_format": "mp3" },
  "requested_count": 1,
  "output_dir": "/absolute/user/output"
}
```

## 提交、续接、交付

从当前 SKILL.md 定位原生执行器；Windows 使用 `.exe`。命令始终沿用已绑定
host，并提供现有可写输出目录、新的绝对任务记录路径。

```text
<executor> submit --host <current-host> --request <absolute-request.json> --record <new-absolute-task.json>
<executor> status --host <same-host> --record <same-task.json>
<executor> wait --host <same-host> --record <same-task.json>
<executor> content --host <same-host> --record <same-task.json> --index 0 --output-dir <absolute-output-directory>
<executor> delivered --host <same-host> --record <same-task.json> --index 0
```

不要一次性串行运行所有命令。提交一次后立即展示回执，再按 `next_step`
选择后续命令。等待最多两个前台窗口，保留计数、Retry-After 和原 task ID。
`SUCCESS` 才取回；`FAILURE`、平台或编号不匹配、未知状态、网络错误均停止
自动续接。不会通过 `/audio/speech` 或另一模型补发。

执行器固定调用 `POST https://api.puretokensx.com/v1/tasks/product-yutoux`、
`GET https://api.puretokensx.com/v1/tasks/{id}` 和
`GET https://api.puretokensx.com/v1/tasks/{id}/artifacts/audio/content`。
本地 Key 只来自当前宿主受支持连接；不借用其他客户端或网页会话。

下载成功只代表字节已保存。按 [客户端交付](host-delivery.md) 真正附加
音频后才执行 `delivered`。中断使用 `resume --record`；显式再次交付使用
`redeliver --record --index 0`，优先校验原文件，绝不重新生成。不要用同步
`audio-verify` 取代任务恢复。提交结果未知保留记录并停止，不推断未扣费。

源码与夹具通过不等于真实 Key 权限、计费或宿主播放通过；后者单独验收。
