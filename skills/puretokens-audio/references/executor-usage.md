# 同步音频命令与请求

StepFun 音乐使用异步任务，见 [音乐指南](music-usage.md)。ElevenLabs 的配音、音效、同步音乐和变声见 [专用指南](elevenlabs-usage.md)；本页示例和限制属于 StepFun。

统一执行 `<执行器> audio --host <当前宿主> --request <绝对 UTF-8 JSON 文件>`。
Windows PowerShell UTF-8 BOM 可用，UTF-16 不可用。请求文件及附件路径必须为绝对路径；
以下路径只是示意，执行前换成当前宿主的真实用户文件或输出目录。不得先运行示例试探付费接口。

## 文字配音

```json
{
  "operation": "speech",
  "model": "stepaudio-2.5-tts",
  "input": "欢迎使用 Pure Tokens。",
  "voice": "cixingnansheng",
  "speed": 1,
  "response_format": "mp3",
  "output_dir": "/absolute/user-output"
}
```

`input` 和 `voice` 必填；输入最多 1000 字符，速度可省略，提供时为 0.5–2。
配音 profile 按模型列出审核音色，不能把一个模型的音色自动套给另一个。
仅 `stepaudio-2.5-tts` 接受可选 `instruction`（本地最多 500 字符）；
其他支持的配音模型不发送该字段。首版只支持 MP3/WAV，默认值由 Skill 明确写入请求。

## 录音转写

```json
{
  "operation": "transcribe",
  "model": "stepaudio-2.5-asr",
  "file": "/absolute/user-recording.wav"
}
```

执行器读取一份稳定、非符号链接的用户文件快照，最多 8 MiB。
仅 multipart `model`、`response_format=json`、`file` 发往固定接口；
不上传到独立服务，不传原始文件名、路径、URL 或 Base64 JSON。
返回 `result.text`，不返回供应商任意元数据；内部支持摘要也不含转写文本。
ASR Max/Pro 依赖网关将供应商 SSE 聚合为最终 JSON；不能把半段流当完成。

## 声音场景

```json
{
  "operation": "generate",
  "model": "stepaudio-3-gen-preview",
  "instruction": "安静街道上的雨声与逐渐走近的脚步声。",
  "response_format": "wav",
  "output_dir": "/absolute/user-output"
}
```

本地最多 500 字符；执行器补固定 `task=text_to_audio`、`stream_format=audio`。
不接收 scripts、音色附件、音乐任务或流式选项，StepFun 音乐按 music-usage.md 的异步工作流执行；music_v2_5 按专用指南执行同步 music_sync。

## 结果与再次交付

三种操作都是一次同步 POST，总超时 90 秒，失败不重试。音频响应最多16 MiB，
先校验 Content-Type 和完整 MP3 帧／WAV 块，再以不覆盖现有文件的方式保存。
校验是结构完整性检查，不是听感或实机播放验收。OGG 转写输入检查页序、CRC、
音频标识与结尾；支持单流 Vorbis 或单／双声道 Opus（mapping family 0），不包含任意 OGG 视频。
WAV 支持8–192kHz、1–8声道的 PCM 8/16/24/32位及32位浮点；
其他封装／编码在提交前停止，不自动转码。

成功配音／声音回执为 `next_step=deliver`、`delivery_status=downloaded` 和
`artifact={path,sha256,bytes,media_type}`；转写为 `next_step=done` 和 `result.text`。
重交文件前运行 `audio-verify --host <原宿主> --request <绝对校验文件>`，
校验文件顶层只有 `artifact`，其值必须复制原回执而非根据当前文件重新编造。
该命令不联网、不读凭据、不创建任务，也不能恢复丢失的音频。

HTTP 4xx 完整响应标为 rejected；网络／读取／响应校验失败为 unknown。
音频已有效接收但落盘失败为 accepted + content 失败。任何一种都不能据此
承诺扣费／退款或自动重新生成。进程取消而无回执也可能已提交。
