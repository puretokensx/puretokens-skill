<p align="center">
  <img src="./assets/brand/puretokens-skill-hero.png" alt="Pure Tokens 官方 Skills" width="100%" />
</p>

# Pure Tokens Skills

用于检查 Pure Tokens 连接、余额与模型目录，生成图片和视频，以及用 Jev 评估文本的官方 Skills。

0.18.8 完善安装权限指引、Windows 下载错误提示及安装受阻后的继续流程。不默认要求完全访问，也不自动修改客户端权限。

0.18.7 增加 MiniMax Code Desktop 适配，限 macOS／Windows 本地默认连接。有桌面偏好配置的独立安装须使用已确认的目标目录或客户端运行目录环境。连接和会话要求见 `references/switch-skill-compatibility.md`。

## 让 Agent 安装

下方保留一句话兼容入口。需要明确安装步骤时，请复制完整的[中文安装提示词](#中文安装提示词)或[英文安装提示词](#英文安装提示词)。

安装前，当前客户端会话需要允许执行本地命令、HTTPS 下载，以及写入下载临时目录和目标 Skill 目录。通过客户端提供的正规审批入口授权；仅有网络授权不代表命令执行与文件访问权限齐备。不默认要求“完全访问”，也不自动修改权限。优先使用宿主已报告的权限信息，不在正常安装中增加探测命令。

如果读取本 README 或下载入口脚本失败，安装器尚未启动。先检查当前客户端会话的执行／网络权限并说明失败阶段，不据此判断 API Key 有问题或要求重装。下方完整提示词已包含首次下载前的权限指引，逐步安装时优先复制完整提示词。仅在排查失败时按需阅读[权限与继续安装说明](skills/puretokens-update/references/install-permissions.md)。

### 复制给可在本机执行命令的 Agent

```text
Install or update the official Pure Tokens Skills from https://github.com/puretokensx/puretokens-skill.
```

### 中文安装提示词

```text
请为我当前使用的客户端安装或更新 Pure Tokens 官方 Skills，仓库：https://github.com/puretokensx/puretokens-skill。

1. 下载官方说明前，先依据当前客户端已报告的信息确认能执行本地命令、HTTPS 下载并写入下载临时目录及目标 Skill 目录；未报告的权限记为未知，不等于拒绝；按正常流程执行，不增加探测命令或读取配置。需要授权时使用客户端实际提供的审批，过期或上轮授权不能代替当前审批；网络授权不代表全部执行权限，不默认要求完全访问或自动修改权限。阅读官方安装说明，只确定本次安装所需的当前客户端、官方宿主 ID 和执行环境。不要从模型名称或共享 Skill 目录猜客户端；无法确定时只问必要信息。不支持、审批被拒或没有执行能力时说明并停止。

2. 首次安装，将以下官方稳定版入口下载为本地文件：
   macOS/Linux：https://github.com/puretokensx/puretokens-skill/releases/latest/download/puretokens-skill-fetch.sh
   Windows：https://github.com/puretokensx/puretokens-skill/releases/latest/download/puretokens-skill-fetch.ps1
   已安装时，按官方说明定位该客户端官方 Skill 同级 .puretokens-executor 中的 fetch 脚本，执行 update。不要将远程内容直接管道进 shell。

3. 替换下面的绝对路径、操作和宿主 ID 后执行一次；首次安装用 install，已有安装用 update：
   macOS/Linux：sh "<fetch脚本绝对路径>" <操作> --host <当前宿主ID>
   Windows：powershell.exe -NoProfile -ExecutionPolicy Bypass -File "<fetch脚本绝对路径>" <操作> -Host <当前宿主ID>
   自定义目录按官方说明附加 --target / -Target 绝对路径；Octop 必须指定当前工作区的 .octop/skills。由官方脚本选择当前系统和架构的稳定版包，校验并同步完整 Skills 与原生执行器。不要复制单个 SKILL.md、使用通用安装器或回退 main、源码包、镜像；不要为此安装 Node、npm、Python、Go、Git 或服务。

4. 保留现有连接和用户文件。不要自行读取、展示或修改认证配置，也不要索取或转述凭据；由原生执行器按当前宿主规则完成只读验证。下载、校验、执行权限或文件冲突失败时，报告已完成阶段与脱敏原因并停止；不自动删除、修复、重装、换宿主或改用其他执行方式。读取说明或下载入口脚本失败时明确“安装尚未开始”，先引导检查当前会话权限，不仅凭 TLS 错误断定 Key、证书或客户端有问题。用户处理阻塞并明确要求继续后，尚未同步才在原宿主执行一次官方流程；已同步且仅连接验证失败时，只在用户要求继续验证后执行一次 init，不重装。超时、同步中断或输出不确定时不得猜测成功或直接重跑，先另行诊断。

5. 复用脚本的同步与自动 init 结果，不另跑 init、doctor、余额或模型查询。自动 init 最多两次只读请求，共用 20 秒预算；安装成功而 init 失败应报告“安装已完成，连接验证未完成”。同版完整性校验通过的快速返回表示“已是当前稳定版且文件完整”，无需再下载、写入或初始化。

6. 完成后简要汇报实际客户端、实际安装版本、安装状态和连接验证状态（已验证／未验证／未执行）。按实际安装版本的本地使用指南列出可用能力，给出两条可复制的使用示例，仅展示、不执行。实际同步后提醒我新开对话加载 Skills；不要声称当前会话已经加载，不自动开始生成或付费体验。
```

### 英文安装提示词

```text
Install or update the official Pure Tokens Skills for the client I am currently using. Repository: https://github.com/puretokensx/puretokens-skill.

1. Before downloading instructions, use information already reported by the current client to establish local command, HTTPS download, temporary-directory and target Skill-directory access. Unreported permissions remain unknown, not denied; proceed normally without extra probes or configuration reads. Use the client's actual approval controls when needed; expired or previous-turn grants do not replace current approval. Network approval does not establish all execution permissions; do not require Full Access by default or change permissions automatically. Read the official installation instructions and identify only the current client, its documented host ID and its execution environment. Do not infer the client from a model name or shared Skill directory; ask only for missing essential information. Explain and stop if the environment is unsupported, approval is denied or commands cannot execute.

2. For a first installation, save the official stable-release entry point as a local file:
   macOS/Linux: https://github.com/puretokensx/puretokens-skill/releases/latest/download/puretokens-skill-fetch.sh
   Windows: https://github.com/puretokensx/puretokens-skill/releases/latest/download/puretokens-skill-fetch.ps1
   For an existing installation, follow the official instructions to locate the fetch script in .puretokens-executor alongside that client's official Skills and use update. Never pipe remote content directly into a shell.

3. Substitute the absolute path, operation and host ID below, then run once. Use install for a first installation or update for an existing installation:
   macOS/Linux: sh "<absolute-fetch-path>" <operation> --host <current-host-id>
   Windows: powershell.exe -NoProfile -ExecutionPolicy Bypass -File "<absolute-fetch-path>" <operation> -Host <current-host-id>
   For a custom directory, add the documented --target / -Target absolute path; Octop requires the current workspace's .octop/skills. Let the official script select, verify and synchronize the stable package for this OS and architecture, including the complete Skills and native executor. Do not copy only SKILL.md, use a generic installer, or fall back to main, source archives or mirrors. Do not install Node, npm, Python, Go, Git or services for this task.

4. Preserve existing connections and user files. Do not independently read, display or modify authentication configuration, or request or relay credentials; let the native executor perform the documented read-only verification for this host. On download, checksum, execution-permission or file-conflict failure, report the completed stage and sanitized cause, then stop. Do not automatically delete, repair, reinstall, switch hosts or use another execution path. If reading instructions or downloading the entry script fails, say "Installation has not started" and guide the user to check current-session permissions first; TLS errors alone do not establish a key, certificate or client defect. After the user resolves the blocker and explicitly continues, run the official workflow once on the same host only if synchronization has not begun. After successful sync with failed connection verification, run init alone only when further verification is requested; do not reinstall. A timeout, interrupted sync or uncertain output requires separate diagnosis, not assumed success or an automatic rerun.

5. Reuse the script's synchronization and automatic init results. Do not add init, doctor, balance or model queries. Automatic init allows at most two read-only requests within one 20-second budget. Successful synchronization followed by failed init means “Installed; connection not verified.” A same-version verified fast return means “Already on the current stable version; files verified,” with no further download, writes or initialization.

6. Briefly report the actual client, installed version, installation status and connection verification status (verified / unverified / not run). List capabilities from that installed version's local usage guide and show two copyable examples without executing them. After actual synchronization, remind me to open a new conversation to load the Skills. Do not claim they are already loaded in this session or automatically start generation or a paid demo.
```

Agent 从官方仓库最新正式发布的附件下载 `puretokens-skill-fetch.sh`（Windows 为 `.ps1`）到本地文件，使用当前宿主 ID 执行 install。固定下载入口为 `https://github.com/puretokensx/puretokens-skill/releases/latest/download/puretokens-skill-fetch.sh`，Windows 将末尾 `.sh` 改为 `.ps1`。已安装时使用同级 `.puretokens-executor` 中的 fetch 执行 check-update 或 update。下载不能直接管道进 shell；没有正式发布资源就停止，不改用 main、源码归档或镜像。

安装执行边界：确认宿主后，将官方 fetch 下载为本地文件并执行一次，由脚本完成提交固定、下载、校验、同步和 init。不要在正常安装中额外审计整份脚本、逐项探测 PowerShell 能力、重复核对清单或再跑 doctor。必要的宿主审批照常遵守。遇到启动被拒绝、输出不可用、超时、下载或校验失败，报告已完成阶段与脱敏失败并停止；不得创建 probe、shim、Python 补丁、修改官方脚本、换启动路径或自动重试。只有用户另行要求开发排查时才进入调试流程。

用户处理权限问题并明确要求继续后，按最后确认的阶段恢复：尚未同步时，在原宿主使用同一官方 fetch 流程一次，由脚本重新解析并校验稳定发布，不复用半成品下载；同步已成功且仅 init 失败时，用户要求继续连接验证才执行一次 init，不重装。输出不确定或同步中断时，先按用户另行授权诊断本地安装状态，不自动重跑。安装恢复不适用于重新提交媒体任务。

## 包含的 Skill

| Skill | 用途 |
| --- | --- |
| `puretokens-connection` | 验证固定 Pure Tokens API 身份，不暴露配置。 |
| `puretokens-balance` | 查询账户钱包余额或当前 Key 的剩余额度，金额以 USD 显示。 |
| `puretokens-models` | 查询已认证的当前模型目录和模型声明能力。 |
| `puretokens-image` | 生图和已声明的图片编辑。 |
| `puretokens-video` | 生视频，以及已声明的图片/视频/音频参考和视频编辑。 |
| `puretokens-audio` | 文字配音、录音转文字、生成音效／声音场景、纯音乐和歌曲。 |
| `puretokens-evaluate` | 用 Jev 对文本分类、按有序标准评分或判断是否符合条件。 |
| `puretokens-update` | 初始化、展示使用须知并安全同步官方 Skills。 |

## 直连 API 的工作方式

媒体 Skill 始终请求 `https://api.puretokensx.com` 下的完整固定 URL：生图使用 `/v1/images/generations`，生视频使用 `/v1/videos`。不会将媒体请求发给任意用户配置的 Base URL、MCP、本地代理、sidecar 或第二个 endpoint。

每次请求由随 Skill 安装的单文件原生执行器完成。它只在调用期间运行，直连固定 API 后立即退出；不会启动端口、后台服务、代理、sidecar 或桌面自动化。执行器只读取受支持宿主在其已知配置路径中写入的当前有效连接，先验证该连接指向 Pure Tokens，再仅在内存中使用一把匹配 Key 请求固定 API。Pure Tokens Switch、CC Switch 或手工配置采用 `references/credential-adapters.md` 列出的格式时可使用这些适配器；没有写入这些记录的项目／会话覆盖尚未验证。Skill 和用户都不传递 Key 或 Base URL。不会扫描 Home、检查 provider 标签、索取、展示、保存、记录或报告 API Key、Base URL 或宿主配置。

生成图片和视频不需要 Node、npm、Python、Go、Pure Tokens Desktop、MCP 或上传中转。安装器会校验并放置当前平台的原生执行器；它负责固定请求、multipart 附件、同任务轮询和有边界的原生媒体字节交付。只有确认 POST 尚未开始时发生的校验或附件准备失败，才能说明未提交任务；POST 可能已开始后遇到网络或响应读取失败，结果仍是未知，Skill 会保留该状态且不自动重提。

Computer Use、浏览器自动化以及打开或点击 Pure Tokens Switch/Desktop 都不是备用执行路径。Skill 不会用它们寻找可见生成界面、获取凭据、提交媒体或交付结果，也不会调用其他生图或生视频 Skill 作为回退。

## 音频

例如：“把这段文字读成 MP3”、“把录音转成文字”、“生成雨声和脚步声”。
`puretokens-audio` 复用当前连接，同步调用固定音频接口，最多90秒，不轮询或
自动重试。支持1000字内配音、8MiB内 MP3/WAV/OGG 转写、500字内声音描述；
音色按模型审核，生成结果以 MP3/WAV 原文件交付。音色克隆、翻译及
实时对话暂未接入。超时不代表未扣费；附件交付失败可校验并重交原文件。
音频的生产调用、计费和真实客户端播放分别记录在维护侧验收记录中。

音乐同样使用此 Skill，支持 `stepaudio-3-music-preview`：描述纯音乐风格，
或提供歌词生成歌曲。音乐异步提交一次并立即返回任务回执，再查询和下载原
任务。描述最多1000字、歌词4000字，交付32 MiB内 MP3/WAV。明确的本地任务
记录支持续接，不保存描述和歌词。网关从音频保存时起保留24小时；已下载的
原文件可本地校验后重新交付。启用音乐请求前需部署配套 Web 音乐 API Key
补丁，见 `references/audio-execution-contract.md`。

[音频场景指南](skills/puretokens-audio/references/prompt-guide.md) 区分朗读正文、
声音描述、音乐风格与原样歌词。明确的组合需求可按
[有限多步骤流程](skills/puretokens-video/references/workflows.md) 执行：先生成图片，
再把原文件作为支持该传输方式的视频模型首帧；或生成视频和独立旁白／音乐素材。
每步分别保留进度并交付，后一步失败不重做前面的任务；没有合成、剪辑或混音能力。
单次生成不加载组合指南、不增加前置检查。

## Jev 文本评估

例如：“用 Jev 把这条工单分为账务、技术或其他”，或“用 Jev 判断是否紧急，并按这三个档位评估影响”。新增的 `puretokens-evaluate` 通过原生执行器同步调用固定 `/typesafe/v1/systemone`，复用当前客户端的 Pure Tokens 连接，不需要 TypeSafe SDK 或第二套 Key。同一文本的独立问题合并为一次请求。

默认模型 `jev-latest`，另支持已审查的精确 ID `jev-1.13.0`、`jev-preview`。Choice、Score、Noul 返回结构化选项／分数和概率；confidence 不等于正确率，也不授权外部操作。首版支持文本／结构化文本，不直接输入图片、音频或视频。超时／响应无法校验就停止，不自动重发、轮询或推断扣费。明确查询 JEV 模型时使用 `/v1/models` 与已审查 ID 的交集，目录可见不证明原生入口可调用。见 [评估契约](references/jev-evaluation-contract.md)；生产及真实客户端评估尚未验收。

## 媒体路由

仅看图、OCR、分析视频、解释截图报错或只写提示词不创建媒体任务；模型能力问题交给模型查询，已有任务的进度／取结果走同任务续接。图片和视频入口提供意图到操作的简表，场景提示词指南只在需要时读取；原样提示词与实际输出参数分开处理，不由附件数量推断首尾帧。

当前宿主使用 Pure Tokens 连接时，即使用户没有明确说“Pure Tokens”，普通图片请求也必须先选择 `puretokens-image`，再考虑通用 `imagegen`、Imagen 或其他图片 Skill；普通视频请求也必须先选择 `puretokens-video`，再考虑通用视频 Skill。一旦选择了 Pure Tokens 专项 Skill，就只能执行固定 API 路径或安全停止，绝不回退到通用媒体 Skill。

优先级由已安装 Skill 的元数据和宿主当前连接上下文共同提供。对固定请求，只有原生执行器可以私密地在内存中解析一把当前匹配连接的凭据，但绝不展示或报告连接配置；若某个宿主忽略已安装 Skill 的选择元数据，需要在该宿主修正自己的选择策略，Skill 无法在运行时强制第三方宿主或第三方 Skill 改变优先级。

## 支持的宿主

| 宿主 | Skill 目录 | 直连执行 |
| --- | --- | --- |
| Claude Code | `~/.claude/skills` | 凭据格式通过夹具测试；客户端端到端待验收 |
| Codex | `~/.agents/skills` | 凭据格式通过夹具测试；客户端端到端待验收 |
| WorkBuddy | `~/.workbuddy/skills` | 凭据格式通过夹具测试；客户端端到端待验收 |
| Gemini CLI | `~/.gemini/skills` | 凭据格式通过夹具测试；客户端端到端待验收 |
| Grok Build | `~/.grok/skills` | 凭据格式通过夹具测试；客户端端到端待验收 |
| OpenCode | `~/.config/opencode/skills` | 凭据格式通过夹具测试；客户端端到端待验收 |
| Trae / TraeWork（`trae`） | `~/.trae/skills`（仅历史安装） | 无凭据适配器；当前 TraeWork 的 Skill 发现未验证 |
| Claude Desktop | `~/.claude/skills`（与 Claude Code 共享） | 本地 Code 会话；Desktop 凭据夹具通过，端到端待验收 |
| DSH Desktop | macOS：`~/Library/Application Support/dsh-desktop/harness/skills`；Windows：`%APPDATA%\dsh-desktop\harness\skills` | 凭据格式通过夹具测试；客户端端到端待验收 |
| 官方 DeepSeek Harness | `~/.dsh/skills` 或绝对路径 `DSH_HOME/skills` | 0.18.7 默认本地 Desktop 连接 |
| MiniMax Code Desktop | `~/.minimax/skills` 或明确的运行数据目录 | 本地默认会话；已有桌面偏好时须确认安装目标 |
| ZCode | `~/.zcode/skills` | 本地连接适配；真实 API 和附件交付待验收 |
| Kimi Code | `~/.kimi-code/skills` | 凭据夹具覆盖；真实 API 与附件交付待验收 |
| Qoder | `~/.qoder/skills` | IDE／CLI 本地执行；真实 API 与附件交付待验收 |
| Pi | `~/.pi/agent/skills` | 内联认证优先级通过夹具测试；真实 API 与附件交付待验收 |
| Hermes | `~/.hermes/skills`；Windows：`%LOCALAPPDATA%/hermes/skills` | 支持选中的内联自定义连接；不支持凭据池和动态认证 |
| EvoX | `~/.evox/agent/skills` | 默认实例的新版配置／认证夹具覆盖；不读取旧存储 |
| VS Code | `~/.copilot/skills` | 默认本地 profile；0.18.7 支持同 Key 的单连接多模型 |
| Octop | 明确指定当前工作区的 `.octop/skills` | 只读本地 SQLite 连接；不假设全局 Skill 安装目录 |

四个新增适配器的夹具覆盖不等于真实客户端媒体验收。Hermes、EvoX 支持文档声明的绝对数据目录覆盖；Octop 使用 `--host octop --target <绝对工作区>/.octop/skills`（PowerShell：`-HostId octop -Target <绝对工作区>/.octop/skills`），不枚举代理、不写全局 Skill 包数据库。Cursor 未注册为 Skill 宿主。

宿主列表以 `references/host-support.json` 为唯一契约。表中为默认目录；Claude／WorkBuddy 支持明确配置目录覆盖，Pi 支持不含父级跳转的绝对 `PI_CODING_AGENT_DIR`，DSH 支持本地 Harness 明确设置的 `DSH_HOME`。Gemini 如已有较高优先级的 `.agents/skills` 安装，会更新该目录并报告重复副本。不会依据 provider 名判断。

注册、安装、API 验证和完整媒体交付是不同状态。实机证据需记录客户端、系统、Shell 版本、架构与执行模式；只有完整用例通过，才能确认该具体环境能打开图片、播放视频并完成附件交付。当前逐项结果以 `references/host-acceptance.json` 为准；部分通过不等于完整媒体验收。本地结果不代表 WSL、远程或沙箱模式也已通过，验收方法见 `references/host-acceptance-guide.md`。

Claude Desktop 请选择本地 Code 会话并使用宿主 ID `claude-desktop`；它读取 Desktop 当前第三方连接，不借用 Claude Code 的凭据。云端、SSH、WSL 或 Cowork 隔离环境不等于本机环境，无法访问本机连接或执行器时会停止。DSH 使用 `dsh-desktop`；项目或自定义 Skill 目录可能覆盖用户目录，应核对实际加载位置。安装示例及边界见 [桌面宿主说明](skills/puretokens-update/references/desktop-hosts.md)。

[Switch × 专用任务矩阵](references/switch-skill-compatibility.md)分别记录 Skill
执行、连接读取、模型权限、专用接口和真实交付。0.18.7 包含音频／Jev指令入口；
向量／重排没有 Skill 入口。配置成功、列出模型或 init 成功不等于专用调用通过。

## 图片、视频与异步任务

普通生成使用默认模型或精确 ID 时只读选中 profile；需要选择模型或解析别名时才读小型索引。不加载全部模型，也不在每次提交前查实时目录。`puretokens-image` 默认使用 `gpt-image-2.5-flare`；`puretokens-video` 默认使用 `minimax_h3`。用户明确指定模型时优先使用指定值；已有任务保留原模型。只有用户明确查询当前模型、请求选中 profile 没有的参数/媒体操作，或需要诊断模型/参数/capability 拒绝时，才读取实时目录。

正常等待窗口结束表示任务仍在生成，不算失败。当前已授权的前台交付最多再续一个窗口，随后暂停并询问用户；任务记录保存该预算。错误、对账及无法容纳的等待要求停止自动续等。视频、多图和跨会话任务默认使用明确的工作区任务记录；附件交付失败时重交已验证文件，不重新提交或下载。

生图和生视频都是异步任务。每个新请求最多一次 POST；得到顶层任务 ID 后，只轮询和获取同一个任务。若 POST 可能已开始但未取得任务 ID，提交结果即为未知：不会重复提交，也不会声称未扣费。多图按 `0..n-1` 顺序交付；视频仅在任务终态成功后交付。

当前请求中的本地图片、视频或音频附件，只会随模型声明的准确 multipart Images/Videos API 操作发送。Skill 不单独上传、不转存、不把附件改写成提示词，也不会把参考媒体请求静默降级为文生。公网 HTTPS URL 只会放入模型资料允许的字段；不支持文件／语音 ID。

普通请求由选中的安装 profile 约束。认证目录用于明确的当前能力查询，不是每次生成的必需前置条件；访问权限以媒体 API 的实际响应为准。权限拒绝时再引导检查连接授权，不自动重提。

<!-- media-model-catalog:start -->
## 媒体模型清单

已与基础模型目录同步：2026-10-04T06:19:22.134Z。

这份清单用于安装后的模型选择，不是每次请求的认证检查。普通生成只读选中 profile；仅明确查询、profile 缺口或拒绝诊断时读取实时目录。经审查的本地兼容补充定义单独注明来源。

README 只从基础目录中带有明确图片/视频能力的模型生成，不通过模型名称推断。已安装模型索引用于选择模型，只有被选中模型的 profile 承载已知参数；实时目录只在明确查询、安装 profile 缺口或提交被拒后的诊断时按需读取。发布前从受控基础目录刷新，并运行 `npm run release:validate`；当快照超过七天时发布校验会失败。

### 图片模型

| 模型 ID | 提供方 | 也可以这样说 | 适合 | 示例 |
| --- | --- | --- | --- | --- |
| `gpt-image-2` | OpenAI | `image2` | 图片生成 | `用 gpt-image-2 生成一张图片。` |
| `gpt-image-2.5` | OpenAI | 仅精确 ID | 图片生成 | `用 gpt-image-2.5 生成一张图片。` |
| `gpt-image-2.5-flare` | OpenAI | 仅精确 ID | 图片生成 | `用 gpt-image-2.5-flare 生成一张图片。` |
| `gpt-image-2.5-sunburst` | OpenAI | 仅精确 ID | 图片生成 | `用 gpt-image-2.5-sunburst 生成一张图片。` |
| `gpt-image-2(Sub)` | OpenAI | 仅精确 ID | 图片生成 | `用 gpt-image-2(Sub) 生成一张图片。` |
| `grok-imagine-image` | xAI | `grok image` | 图片生成 | `用 grok-imagine-image 生成一张图片。` |
| `grok-imagine-image-2.0` | xAI | `grok image 2.0` | 图片生成 | `用 grok-imagine-image-2.0 生成一张图片。` |
| `grok-imagine-image-quality` | xAI | 仅精确 ID | 图片生成 | `用 grok-imagine-image-quality 生成一张图片。` |
| `nano-banana-2` | Google | `nano banana 2` | 图片生成 | `用 nano-banana-2 生成一张图片。` |
| `nano-banana-2-lite` | Google | 仅精确 ID | 图片生成 | `用 nano-banana-2-lite 生成一张图片。` |
| `nano-banana-pro` | Google | `nano banana pro` | 图片生成 | `用 nano-banana-pro 生成一张图片。` |
| `qwen-image-3.0` | Qwen | 仅精确 ID | 图片生成 | `用 qwen-image-3.0 生成一张图片。` |
| `qwen-image-3.0-pro` | Qwen | 仅精确 ID | 图片生成 | `用 qwen-image-3.0-pro 生成一张图片。` |
| `wan2.7-image` | Qwen | 仅精确 ID | 图片生成 | `用 wan2.7-image 生成一张图片。` |
| `wan2.7-image-pro` | Qwen | 仅精确 ID | 图片生成 | `用 wan2.7-image-pro 生成一张图片。` |

### 视频模型

| 模型 ID | 提供方 | 也可以这样说 | 适合 | 示例 |
| --- | --- | --- | --- | --- |
| `grok-imagine-video` | xAI | `grok video` | 视频生成 | `用 grok-imagine-video 生成一条视频。` |
| `grok-imagine-video-1.5` | xAI | 仅精确 ID | 视频生成 | `用 grok-imagine-video-1.5 生成一条短视频。` |
| `minimax_h3` | MiniMax | 仅精确 ID | 视频生成 | `用 minimax_h3 生成一条短视频。` |
| `omni` | Google | 仅精确 ID | 视频生成 | `用 omni 生成一条短视频。` |
| `seedance-2.0` | ByteDance | 仅精确 ID | 视频生成 | `用 seedance-2.0 生成一条视频。` |
| `seedance-2.0-fast` | ByteDance | 仅精确 ID | 视频生成 | `用 seedance-2.0-fast 生成一条视频。` |
| `seedance-2.0-mini` | ByteDance | 仅精确 ID | 视频生成 | `用 seedance-2.0-mini 生成一条视频。` |
| `seedance-2.5` | ByteDance | 仅精确 ID | 视频生成 | `用 seedance-2.5 生成一条视频。` |
| `seedance2.0` | ByteDance | 仅精确 ID | 视频生成 | `用 seedance2.0 生成一条短视频。` |
| `veo_fast` | Google | 仅精确 ID | 视频生成 | `用 veo_fast 生成一条短视频。` |
| `veo_lite` | Google | 仅精确 ID | 视频生成 | `用 veo_lite 生成一条短视频。` |
| `veo_quan` | Google | 仅精确 ID | 视频生成 | `用 veo_quan 生成一条短视频。` |
| `wan3.0-video` | Qwen | `wan3 video`, `wan 3 video` | 视频生成 | `用 wan3.0-video 生成一条短视频。` |
| `wan3.0-video-prime` | Qwen | `wan3 video prime`, `wan 3 video prime` | 视频生成 | `用 wan3.0-video-prime 生成一条短视频。` |

<!-- media-model-catalog:end -->

## 失败提示与回执

提交响应无法读取、约定位置缺少任务编号、编号类型／格式不兼容使用三个不同的本地分类，不能一概说服务器未返回编号或未生成。诊断以出错客户端原回执中的执行器版本为准，不用另一台电脑的版本代替，也不为获取诊断而重跑付费任务。

执行器机器回执保留已知模型、任务 ID、原 operation、状态、安全参数和进度。用户只看到必要的状态、实际附件或可操作失败。失败会给出安全的失败阶段、API 明确返回时的公开错误码、API 明确返回时的 HTTP 状态、经清理的提示和下一步操作。Skill 不会暴露原始响应、请求头/体、内部 URL、凭据或用户媒体。

错误提示按受控分类生成，只保留服务端确实返回的已知公开分类码；未知码和任意原始错误文字不进入回执。GIF 会检查完整帧数据，WebM 会检查容器边界、视频轨道和块结构；损坏输出不复用。结构校验不代替实际附件打开或视频播放验收。对账中的记录在用户明确 `resume` 时只查询同一任务一次，确认新状态后再继续。

## 更新

`puretokens-update` 的原生 fetch 读取最新正式发布清单，固定其中的版本、源码提交、目录选择器和平台包校验和。只下载当前系统／架构的平台包，不回退整库源码；包内仅含八个通用 Skill、一个平台执行器及当前系统脚本。检查更新不写入或执行 init。同版安装／更新先核验执行器与九份受管清单，完整则直接返回，不下载包、不重写、不 init；缺失、改动或未完成事务则停止，保留文件。显式本地源码 sync 仅供维护开发使用。

首次安装或实际更新后仍自动 init，最多两次只读请求共用 20 秒总预算，不自动重试。文件同步成功与连接验证分别报告，验证超时不回滚安装、不重装，也不证明凭据无效。只有带版本的同步成功回执才表示本次安装完成；同版核验回执表示原安装完整，未修改文件。

源码同步脚本是 macOS/Linux 的 `runtime/puretokens-skill-install.sh` 和 Windows 的 `runtime/puretokens-skill-install.ps1`。它们只负责安装更新及校验复制平台执行器；用户不需要 Node、npm、Python、Go 或包管理器。

每个受管目录保存 `.puretokens-managed.json` 文件清单和校验值。更新和中断恢复遇到新增、修改、缺失文件或符号链接时停止覆盖并保留现有内容。没有受管记录的目录只有与当前官方源完全匹配才可接管；同名、版本号或自报哈希不构成归属证明。该记录用于发现意外改动，不是抵抗本机篡改的签名。

首次安装或实际版本更新后，安装器自动执行 `init`；同版完整性核验通过时不执行。先做不计费的固定 `/v1` 身份检查，再用一次 `/v1/media/models` 请求验证当前凭据认证，两次共用 20 秒总预算，不展示凭据或宿主配置，然后输出当前使用须知和示例。验证未完成时，会给出经过脱敏的原因，例如没有当前匹配连接、缺少凭据、API 拒绝及 HTTP 状态、网络失败或 API 身份未确认；绝不打印配置 URL、provider 或 Key。需要再次验证认证时，可要求“初始化 Pure Tokens Skills”或“验证 Pure Tokens 认证”，执行一次 `init`，不修改配置。一般“检查当前 Pure Tokens 连接”只执行 `connection` 检查公开身份，不验证认证；安装诊断使用 `doctor`，用法问题只读本地指南，不串行重复这些检查。

## 维护者校验

匿名完成回执默认关闭。显式设置 `PTP_OPERATIONS_RECEIPTS=1` 后，可报告安装、
已验证连接和执行事件次数；不含账户、凭据、提示词或配置。回传失败不会改变操作结果。
每次发送最多一秒，提交成功与作品交付分别理解，详见
[回执契约](references/operations-receipts-contract.md)。

共享宿主说明与停止规则由 `references/desktop-hosts.md`、`references/skill-fragments/host-binding.md` 和宿主注册表维护。修改后运行 `npm run docs:sync-guidance`，同步八个独立可安装 Skill 及入口摘要；工程门禁会检查漂移。这不是用户运行依赖。

```bash
npm run check
npm run release:validate
```

PNG/JPEG 下载和复用会验证像素数据，解码上限为 33,554,432 像素；WebP 检查完整容器及非空图像块，不等同于完整解码。模型查询与提交共用组合约束，但线上目录查询不会改写本地 profile；本地已有枚举／范围不自动放宽，需要维护者适配并发布后显式更新。

## 执行回执与验收

0.17 链路为：提交 → 立即返回任务 ID → 有界等待／查询 → 按索引下载 → 宿主交付附件。普通生成不增加余额、init 或目录预检。用有限的请求 JSON 文件输入，避免交互 stdin 一直等待；已下载不等于已交付。

余额与官方 CC Switch 使用同一个 API Key 查询接口：`GET https://console.puretokensx.com/api/product/console/api-keys/usage`，随后仅再读取一次公开 `/api/product/console/status` 获取 USD 换算比例，无需浏览器登录。不限额 Key 返回账户钱包余额，限额 Key 返回该 Key 的剩余额度；默认用一行金额说明查询范围，不含订阅套餐额度。不会把旧计费接口的占位值当余额。每次最多两次 GET、总计 30 秒，失败给出可操作的原因，不编造金额。

执行器与 CC Switch 导入时一样，仅为余额请求在内存中补齐缺失的 `sk-` 前缀，不修改用户配置或生图／视频认证。余额返回 401／403 而其他调用正常时，会说明是余额查询被拒绝，并引导保留可用连接。

`references/host-acceptance.json` 分开记录凭据格式测试与真实客户端验收。本地自动化测试通过，不能替代每个客户端在 Windows／macOS 上的附件交付验收。

完整验收步骤见 `references/host-acceptance-guide.md`。`npm run acceptance:validate` 核对宿主与系统版本、执行器哈希、证据文件及结果；缺少实际证据的项目保持待验收，不会因工程检查通过自动变为成功。

用户明确要求检查参数时，用 `preflight` 校验而不提交媒体；`doctor` 检查本地安装并执行只读连接检查。仅问用法则直接读安装的指南，不访问网络。这些都不是普通生成的前置步骤。可选任务记录保存在用户／工作区明确位置，用于同任务续接与交付索引跟踪，不含凭据、prompt、参考 URL 或媒体字节。

精确媒体报价、服务端幂等保证和未知提交的任务查找尚未实现；本地校验与任务记录不能代替这些服务端能力。

ZCode 使用 `--host zcode`；绝对路径 `ZCODE_DATA_BASE_DIR` 指定 `<base>/.zcode/skills`。需要唯一启用且匹配的连接，但这不证明当前会话模型选择。仅同步远程工作区的 Skill 不会同时提供执行器或连接。

构建验证：`npm run executor:build` 使用 `runtime/executor/build-config.json` 指定的 Go 工具链，生成源码／构建输入摘要和六份二进制证明。`npm run validate` 核对摘要、嵌入身份和产物；`npm run executor:verify` 在临时目录重建六个平台并逐字节比较，`npm run release:validate` 包含此门禁。用户安装无需 Go 或 Node。真实宿主验收另行记录，Linux 产物存在不等于所有客户端支持 Linux。

文件复用要求显式任务记录中的 SHA-256、字节数和媒体类型均匹配。旧记录可继续查询原任务；缺少摘要或文件被改写时保留文件，换输出目录取回同一任务。ZCode 路由需要明确宿主上下文或用户指定，不能从连接存在推断当前会话选择。
