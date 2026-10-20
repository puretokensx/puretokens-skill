# 音乐：提交一次，续接原任务

本指南属于 puretokens-audio，仅用于 StepFun 默认音乐。`music_v2_5` 见 [ElevenLabs 同步指南](elevenlabs-usage.md)，不得套用本页命令。StepFun 音乐是异步任务，执行器使用 `kind=music`；
配音、转写和音效仍用同步 `audio` 命令，两者回执不能混用。
唯一审核模型 `stepaudio-3-music-preview`，读对应 profile。普通生成不查目录。
只写／润色歌词或创作建议不发请求。明确要纯音乐时 `instrumental=true`；
明确要歌曲／有歌词人声时为 false。未明确是否有人声时先澄清这个必要选择。
不承诺精确时长、歌手声音、参考音频续写、翻唱或克隆，首版没有这些参数。

## 创建任务

UTF-8 绝对路径请求文件示例：

```json
{
  "kind": "music",
  "operation": "generate",
  "model": "stepaudio-3-music-preview",
  "prompt": "舒缓的钢琴与弦乐，温暖、安静，适合清晨阅读。",
  "parameters": {
    "instrumental": true,
    "response_format": "mp3"
  },
  "output_dir": "/absolute/user-output"
}
```

歌曲示例：

```json
{
  "kind": "music",
  "operation": "generate",
  "model": "stepaudio-3-music-preview",
  "prompt": "温柔的中文民谣，原声吉他伴奏。",
  "parameters": {
    "instrumental": false,
    "lyrics": "晨光轻轻落在窗边\n我们走过安静的街",
    "response_format": "wav"
  },
  "output_dir": "/absolute/user-output"
}
```

`prompt` 逐字映射网关 `caption`，最多1000个 Unicode 字符；可选 `lyrics`
最多4000个字符，保留用户原文／换行。纯音乐不能带非空歌词；有声歌曲可省略
歌词但不保证供应商生成歌词的内容。字段只允许 instrumental、lyrics、
response_format（MP3/WAV，未指定格式可明确写入MP3）；不能发送group、
duration、voice、参考URL、附件或自造控制参数。超限不截断、不自动拆分。

从当前 SKILL.md 解析安装执行器后执行：

`<执行器> submit --host <绑定宿主> --request <绝对请求文件> --record <新的绝对任务记录>`

必须先确定用户输出目录可写；每个任务用新的明确工作区记录，不覆盖旧记录。
执行后清理请求文件。执行器一次 POST 到固定
`https://api.puretokensx.com/v1/audio/music/submit`，立即返回安全任务回执，
先向用户报告接受状态和原 task_id，再进行后续独立命令。提交命令不等待完成
或下载；丢失响应无编号时保留 unknown 记录，停止且不重新 submit。

## 后续命令

- `status --host <同宿主> --record <原记录>`：只读取一次原任务状态。
- `wait --host <同宿主> --record <原记录>`：一个前台窗口最多7次读取、300秒；
  最多自动完成两个窗口，耗尽仍为 pending，然后等待用户明确继续。
- `resume --host <同宿主> --record <原记录>`：跨会话继续原任务；保留窗口计数。
  对账任务仅在用户明确续接时读一次状态；仍需对账则继续停止。
- `content --host <同宿主> --record <原记录> --index 0 --output-dir <绝对目录>`：
  仅终态成功且无需对账时下载，最多32MiB，检查完整MP3/WAV并记录哈希。
- 文件下载后用宿主原生附件交付实际文件，再执行
  `delivered --host <同宿主> --record <原记录> --index 0`；回执路径不等于交付。

这些命令前都加同一个安装执行器绝对路径。状态和内容仅访问固定
`GET https://api.puretokensx.com/v1/audio/music/tasks/{task_id}` 及 `/content`。
不直接请求供应商，不使用 `/pg`、网页会话、另一把Key或另一宿主进行恢复。
API任务由原账户和原 API Key 共同绑定；换Key可能不能读取旧任务，不因权限
失败生成新任务，也不让用户粘贴Key。用户明确切换宿主也必须满足原任务权限。

根据 `next_step` 执行：wait继续有限等待、content下载、deliver附加、done结束、
await_user停止。未知响应、读取错误、延后重试或
`reconciliation_required=true` 都停止自动继续，不推断失败／退款／未扣费。
上游未知提交可能仍得到网关公共编号，这表示可以定位对账任务，不代表已生成。

已下载未交付时使用 `resume` 验证记录中的原文件并重新附加，不使用同步音频的
`audio-verify` 回执替代音乐记录。文件被改写或证明缺失就保留文件，按回执选择
另一个目录下载同一任务。正常已完成本地恢复不读取凭据；实际附件成功后才确认。

网关从**保存音频制品时**起保留24小时，非从提交时间起。内容接口410表示本次
内容不可用／过期，不证明任务生成失败或退款。已有校验通过的本地文件仍可交付；
无可用本地文件则说明无法取回，不能自动重新收费生成。
任务记录仅存原编号、模型、格式／instrumental、等待与交付进度，不存歌词、
prompt、凭据、参考URL或音频。用户可保留明确输出文件，不作为隐式缓存管理。

有限批量逐项提交、续接并交付后才进入下一项，每项独立记录。pending属于已提交；
任何错误、unknown或对账即停止，明确继续批次仅处理未尝试项，不重提已提交项。
只问音乐用法读取本指南；明确查询当前可见音乐模型用
`models --host <同宿主> --request <筛选文件>`，筛选为
`{"kind":"audio","operation":"music"}`，目录可见不等于接口部署、价格或授权。

## 明确再次交付

用户明确要求再发已经交付的音乐时，使用 `redeliver --host <当前宿主> --record <原记录> --index 0`，
先校验本地原文件，缺失才取同一音乐任务结果；文件变化时保留原文件并指定另一个现有输出目录。
不调用 submit、同步 audio 或 audio-verify。历史 delivered 不代表本次交付，必须实际附加后再
对原记录执行 delivered；未完成的本次交付由 pending_redelivery_index 保存，resume 可续接。
