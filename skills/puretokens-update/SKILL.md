---
name: puretokens-update
description: 用户询问 Pure Tokens 是什么、有什么用、怎么开始、网页与 Switch／Skills 的区别，或要求安装、升级、检查版本、检查当前连接、验证认证、诊断安装和了解 Skills 用法时使用。
---

# Pure Tokens Update

先分流帮助意图：“Pure Tokens 是什么／有什么用／怎么开始”及用法问题，直接读本 Skill 内的 `references/usage-guide.md`，按所问范围简短介绍后结束。无需确认宿主、已安装状态或有效连接；不运行执行器、init、doctor、安装器，不读凭据或访问网络，不创建计费任务。介绍网页、Switch 和 Skills 的分工；只展示使用示例，不执行。最新模型、分组、价格和账户权限不以静态介绍作承诺，按指南交给对应查询或官方当前信息。

宿主绑定与本地失败停止：首次调用前按当前应用或用户明确指定绑定一个 host，全部命令沿用，不从安装目录推断。`active_connection_*`、`workbuddy_*` 或 `host_credential_adapter_unavailable` 本地失败后停止 API 流程，保留用户原任务和已有 task ID；不得自动调用 puretokens-connection、init、doctor、models 或其他 Skill 探测，不得枚举、更换 `--host` 或借用凭据。仅用户明确重查或有可验证的新依据时，在原 host 检查一次；换宿主须用户明确指定。“继续生成”本身不是重查、换宿主或重提授权；本次未发送不代表此前未提交。

只响应用户主动的安装、升级、检查或帮助请求。普通媒体生成不自动更新、初始化或诊断。异常时才读 `references/behavior-scenarios.json`。

## 检查、安装和更新

直接使用官方 fetch 入口，不先下载在线 README、raw.githubusercontent.com 或其他远程说明。安装提示词只表达当前客户端的安装／更新意图，提供稳定入口和 install 子命令，不承载客户端映射、系统分支、校验、初始化或排障规则。由入口识别客户端、选择系统／架构与目标目录、固定和校验稳定版本、保护受管文件并汇报安装及连接验证结果；仅回执明确要求确认时询问缺失信息。不委托通用 skill-installer 复制目录，不调用 CUA／浏览器自动化兜底，不反复检查环境或只重复说明计划。下载脚本已取得时直接运行一次；宿主阻止执行或下载失败就保留脱敏结果并停止，不绕过权限。

当前应用或用户已明确的 host 直接显式沿用，不重复询问。尚无已确认 host／target 时执行 fetch install，由脚本先离线识别；仅 `current_host_unconfirmed` 或旧入口要求 host 时确认缺失客户端，再显式执行一次。识别只确认客户端，不证明模型、连接或交付能力；不从模型名、目录存在、共享目录、会话标记或外层进程猜宿主。任何 API 命令仍绑定已确认 host。自动识别边界仅在相关失败时读 `references/install-permissions.md`。

安装需当前会话允许命令执行、HTTPS 下载及临时／目标目录写入。沿用宿主已报告的能力与当前有效的原生审批，不额外探测配置；未知不等于拒绝，网络授权也不代表全部权限。不默认要求完全访问，不提权或绕过政策。fetch 尚未启动就失败时报告“安装尚未开始”。权限／下载失败或用户明确继续时才读 `references/install-permissions.md`；不猜 Key、证书、客户端损坏或错误原因。

下载失败时明确“安装未完成”，报告 stage、error_code 和非零 http_status；http_status=0 表示没有可报告的 HTTP 状态，不猜测为 403 或密钥问题。不展示原始异常、响应、请求头或配置，不自动重试或改用镜像。同步成功但 init 失败时说明“安装已完成，连接验证未完成”，不回滚、不重装。同步后脚本自动执行一次同宿主 init；同版完整性核验快速返回则不执行。已有结果不重复运行 init，不在安装会话自动继续生成。

本地同步失败保留 stage、error_code、installation_status、installed_files_changed、next_step。unknown 不代表旧安装完好；仅 `recovery_status=restored` 才确认恢复，incomplete 保留事务。同步成功后的执行器启动／输出／超时失败只报连接未验证，不重装、补跑 doctor 或自动重试。恢复须按失败阶段和用户明确授权处理，详见 `references/install-permissions.md`；同一错误不循环测试，不换宿主或传输。

已安装且使用本版本时，从本 SKILL.md 的绝对目录解析`../.puretokens-executor/puretokens-skill-fetch.sh`（Windows 为 `.ps1`）。首次安装或升级旧入口时，按 `references/release-download.md` 从固定 GitHub 官方匿名 Releases API 选择唯一同名稳定资产，用 application/octet-stream 下载，校验 size 和 SHA-256 digest 后保存本地执行 install／update。沿用已确认的宿主和目标，不执行可能指向旧仓库的 fetch，不手改受管脚本，不把远程响应管道进 shell，不安装 Git、Node、Python 或 Go，不索取 GitHub Token。fetch 内只有明确 502／503／504 可触发一次同版本官方资产 API 下载；其余失败停止，Agent 不追加重试或其他来源。

```text
macOS/Linux: sh <fetch脚本绝对路径> check-update --host <当前宿主ID>
macOS/Linux: sh <fetch脚本绝对路径> install
macOS/Linux: sh <fetch脚本绝对路径> update --host <当前宿主ID>
Windows: powershell.exe -NoProfile -ExecutionPolicy Bypass -File <fetch脚本绝对路径> check-update -Host <当前宿主ID>
```

Windows 的 install／update 使用相同参数形式替换子命令。ExecutionPolicy 只作用于该进程，不修改系统策略；宿主明确阻止执行时停止，不绕过审批。

宿主 ID 为 claude-code、codex、workbuddy、gemini-cli、grok-build、opencode、trae、claude-desktop、dsh-desktop、deepseek-harness、minimax-code、zcode、kimi-code、qoder、pi、hermes、evox、vscode、octop、oh-my-pi、minimax-code-cli、qwen-code、codebuddy-code、copilot-cli、factory-droid、mimo-coder、paseo；自动识别不了时才询问。显式选择使用 --host／-Host；自定义路径同时传 --target／-Target 绝对目录并保留 host，Octop 必须指定工作区目标。fetch 固定最新稳定发布清单中的版本、源码提交和 SHA-256，先下载并校验该发布的目录选择器，不复用旧版同目录选择器。普通安装只下载当前系统／架构平台包，不回退 main、源码归档或镜像。维护者明确指定本地官方检出时仍可执行其 sync。

check-update 只比较版本，不证明文件完整，也不修改安装或执行 init。同版 install／update 只有在所选发布执行器校验和与全部受管清单（本版本十一份）通过、无未完成事务时，才返回 `already current and verified`：不下载 ZIP、不写入、不 init。缺失、修改或未完成事务必须停止并保留现场，不自动修复。

用户明确要求检查缺失、文件冲突或中断安装时，按需读 `references/installation-recovery.md`。`install-inspect --directory <安装根绝对路径>` 只读十一个受管组件，返回 missing／modified／additional／transaction／unreadable 分类及安全相对路径，不读凭据或发请求。用户明确要求修复后才使用官方 fetch 的 `repair`，仍完整校验当前稳定包；仅恢复可证明的受管缺失和完整受管事务，不覆盖本地定制或删除未知文件。普通 update 不自动转 repair。

只由 sync 写入十个通用 Skill、一个当前平台执行器及当前系统脚本；不裁剪其他宿主所需引用，不安装额外环境。以 `.puretokens-managed.json` 的完整清单和校验值识别受管内容；未记录目录必须完全匹配官方源才可接管。本地修改、附加／缺失文件、符号链接或未完成事务停止覆盖，不删除未知目录或重建清单。安装锁及事务由 sync 管理，不自行复制、链接或拆成逐目录命令；用户配置保持不动。

只在收到 `Pure Tokens Skills <版本> synchronized with the native API executor at <目录>` 后报告本次同步完成；`already current and verified` 只表示原安装完整、未修改文件。复用脚本的宿主加载步骤与一个示例：当前会话可用就直接使用，否则按宿主刷新／启用，仅宿主需要时新开对话。汇总实际 host、版本和 init 已验证／未验证／未执行；不重复十份版本或完整手册，不自动生成体验任务。

## 初始化与诊断

用户意图只选择一个入口：“是什么／有什么用／怎么开始／用法”读本地指南；明确“核验固定 API 的公开身份”才由 puretokens-connection 检查；“检查当前连接／还能用吗／初始化／验证认证”执行一次 init；排查安装文件／中断事务用只读 install-inspect；排查宿主加载位置和连接用 doctor。不要将产品介绍当作身份核验，也不要为同一意图依次执行这些命令；混合提问只执行用户明确要求的检查，介绍本身不增加请求。

从当前 SKILL.md 的绝对目录解析同级 `../.puretokens-executor/puretokens-api`；Windows 使用 `puretokens-api.exe`，不依赖 PATH。`init --host <host-id>` 先请求固定 `/v1` 公开身份，再请求一次 `/v1/media/models` 验证认证；最多两次只读请求共用 20 秒总预算，不自动重试，无付费任务。超时仅说明连接未验证，不证明凭据无效。`doctor --host <host-id>` 额外检查本次加载位置和当前宿主已知 Skill 目录中的安装版本／重复项，再执行同样的只读检查；不扫描 Home、不读取其他宿主配置，也不自动修复。

只有 init 的 configuration_status 为 verified 且 api_identity_confirmed、credential_verified 均为 true，才说身份与凭据认证通过。doctor 不能证明安装二进制校验、任意会话配置覆盖或宿主附件交付成功；按返回字段说明已检查项目与未验证项，不承诺模型权限、余额或媒体可用性。

Skill 和安装脚本不读取、显示或修改认证文件。只有原生执行器按当前宿主明确记录在内存中读取匹配凭据；不使用其他认证、MCP、代理或 Computer Use。

正常安装不额外审计整份脚本、逐项探测 PowerShell、重复核对清单或跑 doctor。失败仅汇总已完成阶段、脱敏原因和下一步。不得创建 probe、shim、Python 补丁、修改官方脚本或换启动路径；只有另行授权的开发排查才调试。

当前会话若使用未反映在宿主已声明有效文件中的配置覆盖，停止并说明无法确认有效连接；不读取其他配置或借用默认连接。

宿主目录、会话选择限制或特定客户端故障才按需读 `references/desktop-hosts.md`；普通请求不增加诊断前置步骤。

安装和更新不检查、移除或迁移旧插件，不运行 codex plugin 命令。汇总脚本已有的同宿主 init 结果，不另跑一次 init；只有原生同步已确认成功且 init 明确未执行才补一次，同版快速返回不适用。输出不确定先说明未确认。Trae 无 API 适配器时不展示生成示例；`cleanup_status: pending` 单独报告，不把同步改报失败，不绕过删除限制或重装。

本地连接失败按执行器脱敏状态解释：未识别、不可读、不支持或未验证均不等于“尚未配置”。保留现有连接，不据此要求重装、切换模型、重配或更换凭据；`api_request_executed: false` 时说明未请求 API、未认证凭据。具体宿主限制见 `references/desktop-hosts.md`。
