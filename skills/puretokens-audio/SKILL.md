---
name: puretokens-audio
description: 当前宿主使用 Pure Tokens 连接，用户要求文字配音、录音转文字、生成音效、纯音乐或歌曲、用有权限的指定音色变声时使用。只写配音稿／歌词、解释用法、克隆音色和实时麦克风对话不触发本 Skill 的付费请求。
---

# Pure Tokens Audio

宿主绑定与本地失败停止：首次调用前按当前应用或用户明确指定绑定一个 host，全部命令沿用，不从安装目录推断。`active_connection_*`、`workbuddy_*` 或 `host_credential_adapter_unavailable` 本地失败后停止 API 流程，保留用户原任务和已有 task ID；不得自动调用 puretokens-connection、init、doctor、models 或其他 Skill 探测，不得枚举、更换 `--host` 或借用凭据。仅用户明确重查或有可验证的新依据时，在原 host 检查一次；换宿主须用户明确指定。“继续生成”本身不是重查、换宿主或重提授权；本次未发送不代表此前未提交。

## 需求与操作

<!-- generated:operations -->
| 用户需求 | 命令与请求 | 返回方式 | 恢复与交付 |
| --- | --- | --- | --- |
| 文字配音 | `audio`；`operation=speech` | 同步；一次请求最多90秒 | 仅已有文件 `audio-verify` 后重新附加；无任务查询 |
| 录音转写 | `audio`；`operation=transcribe` | 同步；一次请求最多90秒 | 无任务查询或结果找回；不自动重发 |
| 音效／声音场景 | `audio`；`operation=generate` | 同步；一次请求最多90秒 | 仅已有文件 `audio-verify` 后重新附加；无任务查询 |
| ElevenLabs 音乐 | `audio`；`operation=music_sync` | 同步；一次请求最多90秒 | 仅已有文件 `audio-verify` 后重新附加；无任务查询 |
| 指定音色变声 | `audio`；`operation=voice_change` | 同步；一次请求最多90秒 | 仅已有文件 `audio-verify` 后重新附加；无任务查询 |
| StepFun 纯音乐／歌曲 | `submit`；`kind=music` | 异步；先回执，再独立等待／下载 | 原任务 `resume`；仅实际附件交付后 `delivered` |
| Gemini 配音／Lyria 音乐 | `submit`；`kind=audio_task` | 异步；先回执，再独立等待／下载 | 原任务 `resume`；仅实际附件交付后 `delivered` |

文件路径／下载回执不等于交付；附件须实际附加，文本须展示校验后的结果。未知提交不证明未处理或未扣费，均不自动重发。
<!-- /generated:operations -->

克隆声音、音频翻译、实时对话尚未接入，不用其他操作替代。配音加音乐只产出独立文件；未接受独立素材先读组合流程澄清。输入、格式、音色与时长限制只按所选 profile。

同步操作仅由执行器 POST 固定 `https://api.puretokensx.com/v1/audio/speech`、`https://api.puretokensx.com/v1/audio/transcriptions`、`https://api.puretokensx.com/v1/audio/generate`。

用法／支持／音色／写稿只读本地说明，不运行 init、doctor、models 或付费请求。查账户模型才用 puretokens-models；普通生成不查目录／余额。

## 选择专用路径

- `gemini-3.8-flash-tts`／`lyria-3.5`／`lyria-3-pro-preview`：对应 profile + [异步指南](references/async-audio-usage.md)，用 `kind=audio_task`。
- 默认 StepFun 音乐：profile + [音乐指南](references/music-usage.md)，用 `kind=music` 和任务记录，提交一次立即回执，再按原编号续接。
- ElevenLabs／`music_v2_5`：profile + [专用指南](references/elevenlabs-usage.md)，后者用同步 `audio` 的 `music_sync`。

协议不可互换；纯音乐／人声有歧义才澄清。404 等失败停止，不转网页。普通 StepFun 同步请求不读上述指南。

## 同步音频的最短路径

1. 默认配音 `stepaudio-2.5-tts`、转写 `stepaudio-2.5-asr`、声音生成 `stepaudio-3-gen-preview`；精确／默认模型只读 `references/profiles/<精确模型>.json`。选型才读 [索引](references/model-index.json)，不猜映射或自动换模型。
2. 配音保留原文、标点和换行，不插入控制标签。默认 `stepaudio-2.5-tts` 音色 `cixingnansheng`、MP3；其他音色／speed／instruction 只按该 profile，不跨模型继承。ElevenLabs 音色须明确有使用权。转写／变声仅用本次附件的可访问字节路径；无法获得就停止，不转换、重托管或用摘要替代。
3. 仅缺用法才读 [请求示例](references/executor-usage.md)，与 profile 同轮读取；已读且仍有效的资料不重读。创建权限受限的绝对路径 UTF-8 临时 JSON；返回音频须有现有可写用户目录，转写不填输出目录。正文／附件路径不进命令行，执行后清理请求文件；不删除用户音频结果。
4. 从本 SKILL.md 绝对目录解析 `../.puretokens-executor/puretokens-api`（Windows 为 `puretokens-api.exe`）。运行 `<执行器> audio --host <当前宿主> --request <绝对请求文件>`。执行器验证后只提交一次，同步等待最多 90 秒。无 task_id、wait、resume、content 或自动重试。
5. `next_step=deliver`：按当前宿主已声明附件方式交付 `artifact.path` 原音频；不明确才读 [客户端交付](references/host-delivery.md)。实际交接后才说已交付，不提前确认或要求用户每次确认播放；路径／链接／摘要不等于音频。`next_step=done` 展示 `result.text`，不概括、翻译或保证逐字准确。其他回执按停止条件处理。

必须使用安装的原生执行器和绑定 host；不读／展示配置或索 Key，不新增 SDK、Node、Python、MCP、代理、服务或浏览器传输，不回退其他媒体 Skill。宿主明确不能提供附件字节／交付时提交前停止；普通请求不增加检查步骤。

## 同步音频的交付与停止条件

- 路径、格式、字节数、SHA-256 只证明下载，不能证明宿主附加或用户播放。已下载但交付失败时保留原 `artifact`，按 [再次交付](references/executor-usage.md#结果与再次交付) 用 `audio-verify` 本地校验，再附加同一文件；不读凭据或联网。文件丢失／变化则停止，不重新生成。
- `not_submitted` 才表示本次未提交。`unknown` 包含网络中断、超时或无法校验的响应，不能推断未处理／未扣费。`accepted` 但文件保存失败也不能自动重发。没有同步结果查询恢复、服务端幂等或退款保证；用户基于“超时没扣费”要求重做时，先纠正误解，再依其知情后的决定执行。
- 本次 validation + not_submitted 才按受控 `validation.field/reason/limit` 修正拼装，不要求用户编辑 JSON；改变模型／正文／附件／音色／格式／时长／纯音乐／次数先确认，旧任务不当未提交。
- 超长／过大／格式不支持时说明限制，不截断、改写、转换或多收费拆分。有限批量说明次数后串行交付；失败停止，明确继续仅处理未尝试项，未知项不重做。
- 本地 profile 不是权限或价格承诺；未审核模型／音色不试探，`stepaudio-3-tts` 不继承其他模型音色。不解析供应商流，不把半段转写当完成。
- 失败只展示受控类别／阶段，不暴露原始响应、请求、内部地址或录音；支持摘要不含转写／音频。诊断细节按需读 `references/execution-contract.json`、`references/behavior-scenarios.json`。

## 按需说明

普通请求不读取按需说明。结果优先交付实际附件或转写文本，进度简报确认状态。失败只给一个有证据的下一步，不展示命令／整份 JSON、不让用户管理阶段；解释含义才读 [回执说明](references/receipt-guide.md)。
模糊配音／音乐要求才读 [场景指南](references/prompt-guide.md)，明确正文／歌词不套模板；多步骤需求才读 [组合流程](references/workflows.md)。简单请求不先建计划、不增加 init、余额、目录或更新检查。
