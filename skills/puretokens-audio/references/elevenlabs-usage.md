# ElevenLabs 同步音频

按精确模型读取对应 profile。本地实现与接口夹具不证明线上已开放、当前 Key
有权限、目标音色可用或客户端已实际播放。正常请求不先查目录；权限拒绝后停止，
不替换模型或音色。只读目录可见也不能算调用验收。

所有操作沿用 `audio --host <绑定宿主> --request <绝对 UTF-8 JSON 文件>`：
单次同步 POST，90秒总超时，不轮询、不重试。超时可能已生成或扣费。
输出是明确目录内的 MP3，须实际附加；重新附加前只用本地 `audio-verify`。
以下路径、正文和音色只作请求结构示意，不能直接运行作付费探测。

| 精确模型 | operation | 必要选择 |
| --- | --- | --- |
| `eleven_v3`、`eleven_v4`、`eleven_multilingual_v2`、`eleven_flash_v2_5` | `speech` | 原文 input，用户明确有权使用的 voice ID，MP3；最多4096字符 |
| `scribe_v2`、`scribe_v2_medical` | `transcribe` | 一份原始本地 MP3/WAV/OGG；8 MiB、5分钟以内；返回 text |
| `eleven_text_to_sound_v2` | `generate` | instruction 最多1000字符；duration_seconds 0.5–30；MP3 |
| `music_v2_5` | `music_sync` | instruction 最多1000字符；duration_seconds 3–600，最多毫秒精度；明确 instrumental 布尔值；MP3 |
| `eleven_multilingual_sts_v2` | `voice_change` | 原始本地 file 与有权使用的 voice ID；8 MiB、5分钟以内；MP3 |

配音示例：

```json
{"operation":"speech","model":"eleven_v3","input":"用户的原文","voice":"EXPLICIT_ENTITLED_VOICE_ID","response_format":"mp3","output_dir":"/absolute/user-output"}
```

voice 是有权限的供应商音色 ID，不能用 George、AnnaSu 等显示名、StepFun
预设或任意人物名称代替。示例 ID 或格式校验通过不表示获得音色权限。
不发送 speed、instruction、WAV、多角色、克隆或实时参数。

同步音乐示例：

```json
{"operation":"music_sync","model":"music_v2_5","instruction":"用户明确的音乐描述和原样歌词","duration_seconds":30,"instrumental":false,"response_format":"mp3","output_dir":"/absolute/user-output"}
```

歌词放在 instruction；不接受独立 lyrics 字段。只缺人声选择或时长时询问缺失项；
不为满足1000字符限制改写歌词。用户未选模型时仍沿用 StepFun 默认，
不因为有新型号就自动切换。长曲可能超过同步时限；结果未知时停止，不能重提。

变声示例：

```json
{"operation":"voice_change","model":"eleven_multilingual_sts_v2","file":"/absolute/user-recording.wav","voice":"EXPLICIT_ENTITLED_VOICE_ID","response_format":"mp3","output_dir":"/absolute/user-output"}
```

执行器读取稳定原始字节，在内存中构造受限 Base64 JSON；Agent 只提交本地路径，
不自行编码、上传或重托管。网关校验真实时长；不接受自报时长。
没有变声 instruction、input、speed 或 instrumental 参数。
