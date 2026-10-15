---
name: puretokens-update
description: 用户询问 Pure Tokens 是什么、有什么用、怎么开始、网页与 Switch／Skills 的区别，或要求安装、升级、检查版本、检查当前连接、验证认证、诊断安装和了解 Skills 用法时使用。
---

# Pure Tokens Update

先分流帮助意图：“Pure Tokens 是什么／有什么用／怎么开始”及用法问题，直接读本 Skill 内的 `references/usage-guide.md`，按所问范围简短介绍后结束。无需确认宿主、已安装状态或有效连接；不运行执行器、init、doctor、安装器，不读凭据或访问网络，不创建计费任务。介绍网页、Switch 和 Skills 的分工；只展示使用示例，不执行。最新模型、分组、价格和账户权限不以静态介绍作承诺，按指南交给对应查询或官方当前信息。

宿主绑定与本地失败停止：首次调用前按当前应用或用户明确指定绑定一个 host，全部命令沿用，不从安装目录推断。`active_connection_*`、`workbuddy_*` 或 `host_credential_adapter_unavailable` 本地失败后停止 API 流程，保留用户原任务和已有 task ID；不得自动调用 puretokens-connection、init、doctor、models 或其他 Skill 探测，不得枚举、更换 `--host` 或借用凭据。仅用户明确重查或有可验证的新依据时，在原 host 检查一次；换宿主须用户明确指定。“继续生成”本身不是重查、换宿主或重提授权；本次未发送不代表此前未提交。

只响应用户主动的安装、升级、检查或帮助请求。普通媒体生成不自动更新、初始化或诊断。仅问用法时直接读 `references/usage-guide.md` 回答，不运行 init／doctor，不查凭据或访问网络。异常时才读 `references/behavior-scenarios.json`。

## 检查、安装和更新

直接使用官方 fetch 入口，不先下载在线 README、raw.githubusercontent.com 或其他远程说明。安装提示词只表达当前客户端的安装／更新意图，提供稳定入口和 install 子命令，不承载客户端映射、系统分支、校验、初始化或排障规则。由入口识别客户端、选择系统／架构与目标目录、固定和校验稳定版本、保护受管文件并汇报安装及连接验证结果；仅回执明确要求确认时询问缺失信息。不委托通用 skill-installer 复制目录，不调用 CUA／浏览器自动化兜底，不反复检查环境或只重复说明计划。下载脚本已取得时直接运行一次；宿主阻止执行或下载失败就保留脱敏结果并停止，不绕过权限。

安装入口尚无已确认的 host／target 时，执行 fetch install，由脚本先离线识别客户端；已有确认值则显式沿用。它最多检查八层直接启动进程的名称和父 PID，仅明确的原生 Codex／OpenCode 启动器可自动识别，不越过 Node、Electron 或模糊的 Claude 进程借用外层身份。`current_host_unconfirmed` 或旧入口要求 host 时，只询问缺失的客户端，再带参数执行一次；不是下载失败后的自动重试。不根据模型名、目录存在或继承的会话标记猜宿主。`detect-host` 只读、无网络，不是普通生成预检；识别不证明当前聊天模型、凭据或 API／交付能力。任何 API 命令仍须绑定已确认的 host。

安装需当前会话允许命令执行、HTTPS 下载及临时／目标目录写入；优先使用宿主已报告的权限与审批能力，不额外探测或读取宿主配置。未知权限不等于拒绝，按正常流程及原生审批执行；审批须覆盖当前动作，遵守宿主实际有效范围。网络授权不证明完整执行权限，不默认要求“完全访问”或自动扩大权限。fetch 下载或启动失败应报告“安装尚未开始”。旧提示词若停在读取说明，仅在用户明确继续且宿主已确认时，使用完整官方提示词直接请求稳定 fetch 一次，不重试文档；历史文档失败不能证明发布附件不可达，也不允许绕过宿主拒绝或重试失败的发布下载。权限错误、Windows 安全上下文错误及用户明确要求继续安装时，按需读 `references/install-permissions.md`，不要一律归因于 Key、证书、客户端损坏或权限问题。

下载失败时明确“安装未完成”，报告 stage、error_code 和非零 http_status；http_status=0 表示没有可报告的 HTTP 状态，不猜测为 403 或密钥问题。不展示原始异常、响应、请求头或配置，不自动重试或改用镜像。同步成功但 init 失败时说明“安装已完成，连接验证未完成”，不回滚、不重装。同步后脚本自动执行一次同宿主 init；同版完整性核验快速返回则不执行。已有结果不重复运行 init，不在安装会话自动继续生成。

保留下载回执的 next_step；network_access_denied／tls_security_context_unavailable 提示用户先核实客户端当前会话权限，不宣称原因已唯一确定。用户处理阻塞并明确要求继续后：同步前只在原宿主再执行一次官方 fetch；已同步且仅 init 失败，明确要求继续连接验证才执行一次 init；同步状态未知先另行诊断本地状态，不自动重跑。未变更环境的同一错误不反复测试，不更换传输、宿主或权限模式。

已安装且使用本版本时，从本 SKILL.md 的绝对目录解析`../.puretokens-executor/puretokens-skill-fetch.sh`（Windows 为 `.ps1`）。用户通过当前官方安装提示词安装或升级旧版本时，由宿主现成 HTTPS 下载能力把 `https://github.com/puretokensx/puretokens-skill/releases/latest/download/puretokens-skill-fetch.sh`（Windows 改为 `.ps1`）下载为唯一临时本地文件，沿用已确认的宿主和目标目录执行 install／update。旧 fetch 可能指向旧仓库；不执行它、不手改受管脚本，不在下载失败后自动改路。不把远程响应管道进 shell，不安装 Git、Node、Python 或 Go。

```text
macOS/Linux: sh <fetch脚本绝对路径> check-update --host <当前宿主ID>
macOS/Linux: sh <fetch脚本绝对路径> install
macOS/Linux: sh <fetch脚本绝对路径> update --host <当前宿主ID>
Windows: powershell.exe -NoProfile -ExecutionPolicy Bypass -File <fetch脚本绝对路径> check-update -Host <当前宿主ID>
```

Windows 的 install／update 使用相同参数形式替换子命令。ExecutionPolicy 只作用于该进程，不修改系统策略；宿主明确阻止执行时停止，不绕过审批。

宿主 ID 为 claude-code、codex、workbuddy、gemini-cli、grok-build、opencode、trae、claude-desktop、dsh-desktop、deepseek-harness、minimax-code、zcode、kimi-code、qoder、pi、hermes、evox、vscode、octop；自动识别不了时才询问。显式选择使用 --host／-Host；自定义路径同时传 --target／-Target 绝对目录并保留 host，Octop 必须指定工作区目标。fetch 固定最新稳定发布清单中的版本、源码提交和 SHA-256，先下载并校验该发布的目录选择器，不复用旧版同目录选择器。普通安装只下载当前系统／架构平台包，不回退 main、源码归档或镜像。维护者明确指定本地官方检出时仍可执行其 sync。

check-update 只比较版本，不证明文件完整，也不修改安装或执行 init。同版 install／update 只有在发布执行器校验和与九份受管清单全部通过、无未完成事务时，才返回 `already current and verified`：不下载 ZIP、不写入、不 init。缺失、修改或未完成事务必须停止并保留现场，不自动修复。

用户明确要求检查缺失、文件冲突或中断安装时，按需读 `references/installation-recovery.md`。`install-inspect --directory <安装根绝对路径>` 只读九个受管组件，返回 missing／modified／additional／transaction／unreadable 分类及安全相对路径，不读凭据或发请求。用户明确要求修复后才使用官方 fetch 的 `repair`，仍完整校验当前稳定包；仅恢复可证明的受管缺失和完整受管事务，不覆盖本地定制或删除未知文件。普通 update 不自动转 repair。

只由 sync 写入八个 Skill 和一个当前平台原生执行器及下载／同步脚本。它保护非受管目录；更新锁阻止并发，中断的受管事务在下次显式 sync 时恢复。Gemini 如已有较高优先级的共享 `.agents/skills`，更新其有效目录并报告重复的低优先级副本，不擅自删除。只处理八个当前 Skill 和原生执行器；其他目录和用户配置保持不动。

受管识别核对 `.puretokens-managed.json` 的完整文件清单和校验值；没有记录的目录只有与当前官方源文件完全匹配才可接管，不能仅凭同名或目录中的自报版本判定归属。存在新增文件、本地修改、缺失文件或符号链接时停止覆盖；向用户说明需保留并处理该冲突，不自动删除、重建清单或声称更新完成。中断恢复遇到改动同样保留原目录和备份。

只在收到 `Pure Tokens Skills <版本> synchronized with the native API executor at <目录>` 后报告本次同步完成；`already current and verified` 只表示原安装完整、未修改文件，不补跑 init。超时／中断不能声称完成或自动重试。实际同步成功后复用脚本的当前宿主加载步骤与简短示例，不重复列出八份相同版本信息。

## 初始化与诊断

用户意图只选择一个入口：“是什么／有什么用／怎么开始／用法”读本地指南；明确“核验固定 API 的公开身份”才由 puretokens-connection 检查；“检查当前连接／还能用吗／初始化／验证认证”执行一次 init；排查安装文件／中断事务用只读 install-inspect；排查宿主加载位置和连接用 doctor。不要将产品介绍当作身份核验，也不要为同一意图依次执行这些命令；混合提问只执行用户明确要求的检查，介绍本身不增加请求。

从当前 SKILL.md 的绝对目录解析同级 `../.puretokens-executor/puretokens-api`；Windows 使用 `puretokens-api.exe`，不依赖 PATH。`init --host <host-id>` 先请求固定 `/v1` 公开身份，再请求一次 `/v1/media/models` 验证认证；最多两次只读请求共用 20 秒总预算，不自动重试，无付费任务。超时仅说明连接未验证，不证明凭据无效。`doctor --host <host-id>` 额外检查本次加载位置和当前宿主已知 Skill 目录中的安装版本／重复项，再执行同样的只读检查；不扫描 Home、不读取其他宿主配置，也不自动修复。

只有 init 的 configuration_status 为 verified 且 api_identity_confirmed、credential_verified 均为 true，才说身份与凭据认证通过。doctor 不能证明安装二进制校验、任意会话配置覆盖或宿主附件交付成功；按返回字段说明已检查项目与未验证项，不承诺模型权限、余额或媒体可用性。

Skill 和安装脚本不读取、显示或修改认证文件。只有原生执行器按当前宿主明确记录在内存中读取匹配凭据；不使用其他认证、MCP、代理或 Computer Use。

失败时区分版本检查、下载／校验、文件同步和 init 阶段，说明哪些步骤已确认完成及下一步。网络失败可建议稍后重新检查；校验失败停止使用该包并反馈官方维护方；写入受限可建议检查目标目录权限，不提权或删除未知目录。sync 已成功但 init 失败时明确“文件已同步，连接验证未完成”，按 init 脱敏建议处理，不重复安装。未知中断先核对已安装状态，不自动重跑。

Windows 安装只运行一次官方 fetch／sync 流程并汇总同步、清理和 init 结果；不要自行拆成逐目录命令或重复执行已通过的校验。安装器捕获原生执行器输出并禁止创建额外控制台窗口；宿主自身的命令窗口仍由宿主控制。执行被策略拒绝时停止并说明，不改用备用启动方式或提权。

不要在正常安装中额外审计整份脚本、逐项探测 PowerShell 能力、重复核对清单或再跑 doctor。必要的宿主审批照常遵守；失败时报告已完成阶段与脱敏结果并停止。不得创建 probe、shim、Python 补丁、修改官方脚本、换启动路径或自动重试；只有另行授权的开发排查才进入调试流程。

当前会话若使用未反映在宿主已声明有效文件中的配置覆盖，停止并说明无法确认有效连接；不读取其他配置或借用默认连接。

宿主目录、会话选择限制或特定客户端故障才按需读 `references/desktop-hosts.md`；普通请求不增加诊断前置步骤。

安装时由官方入口识别或沿用当前应用／用户已确认的宿主 ID；不从模型名称、Skill 来源目录或共享 `.agents/skills` 推断为 Codex。Qoder 使用 `--host qoder`，默认目标由官方 locate 解析，不照抄 Codex 示例。普通安装只使用当前官方 fetch，不直接下载旧迁移载荷或复制 Skill 文件。

安装汇报保留实际宿主、同步版本、init 已验证／失败／未执行的结果，下一步只指向该宿主。仅复制文件不代表原生安装完成。只有原生同步已确认成功且 init 明确未执行时，补一次同宿主 init；同版快速返回不属于此情形。输出不确定先说明未确认，不重装或猜成功。

安装和更新不检查、移除或迁移 Codex 插件，不运行 codex plugin 命令，也不汇报旧插件状态。汇总脚本已有的同宿主 init 结果及加载指引，不另跑一次 init。默认仅输出短回执、当前宿主适用示例与帮助入口，不整段输出使用手册；ZCode 需刷新并启用，Trae 无 API 适配器时不展示生成示例。`cleanup_status: pending` 单独报告，不把已完成同步改报失败，不绕过删除限制或因此重装。

本地连接失败按执行器脱敏状态解释：未识别、不可读、不支持或未验证均不等于“尚未配置”。保留现有连接，不据此要求重装、切换模型、重配或更换凭据；`api_request_executed: false` 时说明未请求 API、未认证凭据。具体宿主限制见 `references/desktop-hosts.md`。
