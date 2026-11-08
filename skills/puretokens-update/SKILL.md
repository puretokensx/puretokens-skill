---
name: puretokens-update
description: 用户询问 Pure Tokens 是什么、有什么用、怎么开始、网页与 Switch／Skills 的区别，或要求安装、升级、检查版本、检查当前连接、验证认证、诊断安装和了解 Skills 用法时使用。
---

# Pure Tokens Update

帮助问题（是什么／有什么用／怎么开始／用法）只读 `references/usage-guide.md` 对应部分，简短介绍后结束；不确认宿主、检查安装、运行 init／doctor、读取凭据或联网。示例不执行，静态介绍不承诺最新模型、分组、价格或账户权限。

宿主绑定与本地失败停止：首次调用前按当前应用或用户明确指定绑定一个 host，全部命令沿用，不从安装目录推断。`active_connection_*`、`workbuddy_*` 或 `host_credential_adapter_unavailable` 本地失败后停止 API 流程，保留用户原任务和已有 task ID；不得自动调用 puretokens-connection、init、doctor、models 或其他 Skill 探测，不得枚举、更换 `--host` 或借用凭据。仅用户明确重查或有可验证的新依据时，在原 host 检查一次；换宿主须用户明确指定。“继续生成”本身不是重查、换宿主或重提授权；本次未发送不代表此前未提交。

只响应用户主动的安装、升级、检查或帮助请求。普通媒体生成不自动更新、初始化或诊断。异常时才读 `references/behavior-scenarios.json`。

## 检查、安装和更新

直接使用官方 fetch，不先下载在线 README、raw.githubusercontent.com 或其他远程说明。提示词只保留安装意图、稳定入口和最小 bootstrap 校验；系统／架构、目标目录、版本固定、同步、init 与结果由工程负责。当前 host 已明确就沿用；不委托通用 skill-installer、CUA 或浏览器兜底，不重复环境探测、脚本审计或规划。脚本已取得就执行一次；失败停止，不绕过权限。

尚无已确认 host／target 时执行 fetch install，脚本离线识别；仅 `current_host_unconfirmed` 或旧入口要求 host 时询问缺失客户端一次。不从模型名、目录、会话标记或外层进程猜宿主；识别不证明连接或交付能力。相关失败才读 `references/install-permissions.md`。

安装需当前会话的命令、HTTPS、临时／目标目录写权限；沿用宿主已报告能力及有效审批，不额外探测。未知不等于拒绝，网络授权不代表全部权限，不默认要求完全访问。fetch 未启动就失败时报“安装尚未开始”；权限、下载或明确继续时才读 `references/install-permissions.md`，不猜 Key／证书问题。

下载失败报“安装未完成”，保留 stage、error_code、非零 http_status；0 不代表 403。不展示原始异常／响应／请求头／配置，不自动重试或换镜像。同步后自动一次同宿主 init；失败报“安装已完成，连接验证未完成”，不回滚、不重装。同版完整性快速返回不 init；已有结果不另跑一次 init，不自动生成体验任务。

本地失败按 `references/install-permissions.md` 保留安全状态：unknown 不代表旧安装完好，只有 `recovery_status=restored` 才确认恢复；incomplete 保留事务。不同步重跑、不补 doctor、不换宿主。同一个失败只说明已确认结果和一个由 `next_step` 支持的下一步；证据不足时说明原因未确定，不列一串重装／换模型建议。

已安装且使用本版本时，从本 SKILL.md 绝对目录解析 `../.puretokens-executor/puretokens-skill-fetch.sh`（Windows `.ps1`）。首次安装或升级旧入口才读 `references/release-download.md`：官方匿名 Releases API 下载唯一稳定 fetch，执行前校验 size 与 SHA-256 digest。支持本版参数时，把本次已验证元数据绝对路径通过 `--release-metadata`（Windows `-ReleaseMetadata`）交给脚本，避免重复查询；旧入口不传新参数。只复用本次 bootstrap，不复用历史缓存。保留原 host／target，不手改脚本、不把远程响应管道进 shell、不安装 Git／Node／Python／Go、不索取 GitHub Token。下载路线与校验由 fetch 管理，Agent 不追加重试、镜像或其他来源。

```text
macOS/Linux: sh <fetch脚本绝对路径> check-update --host <当前宿主ID>
macOS/Linux: sh <fetch脚本绝对路径> install
macOS/Linux: sh <fetch脚本绝对路径> update --host <当前宿主ID>
Windows: powershell.exe -NoProfile -ExecutionPolicy Bypass -File <fetch脚本绝对路径> check-update -Host <当前宿主ID>
```

Windows 的 install／update 使用相同参数形式替换子命令。ExecutionPolicy 只作用于该进程，不修改系统策略；宿主明确阻止执行时停止，不绕过审批。

宿主 ID 为 claude-code、codex、workbuddy、gemini-cli、grok-build、opencode、trae、claude-desktop、dsh-desktop、deepseek-harness、minimax-code、zcode、kimi-code、qoder、pi、hermes、evox、vscode、octop、oh-my-pi、minimax-code-cli、qwen-code、codebuddy-code、copilot-cli、factory-droid、mimo-coder、paseo；自动识别不了时才询问。显式选择使用 --host／-Host；自定义路径同时传 --target／-Target 绝对目录并保留 host，Octop 必须指定工作区目标。fetch 固定最新稳定发布清单中的版本、源码提交和 SHA-256，先下载并校验该发布的目录选择器，不复用旧版同目录选择器。普通安装只下载当前系统／架构平台包，不回退 main、源码归档或镜像。维护者明确指定本地官方检出时仍可执行其 sync。

check-update 只比较版本，不证明文件完整，也不修改安装或执行 init。同版 install／update 只有在所选发布执行器校验和与全部受管清单（本版本十一份）通过、无未完成事务时，才返回 `already current and verified`：不下载 ZIP、不写入、不 init。缺失、修改或未完成事务必须停止并保留现场，不自动修复。

明确检查缺失、冲突或中断安装时才读 `references/installation-recovery.md`，执行只读 install-inspect；明确修复才运行官方 repair。普通 update 不自动修复，不覆盖定制或删除未知文件。

只由 sync 管理安装锁、事务及 `.puretokens-managed.json` 完整清单：十个通用 Skill、一个平台执行器与当前系统脚本；不裁剪共享宿主引用、不自行复制或链接、不修改用户配置。未受管目录需完全匹配官方源才能接管；改动、附加／缺失文件、符号链接、未完成事务停止覆盖。

只在收到 `Pure Tokens Skills <版本> synchronized with the native API executor at <目录>` 后报告本次同步完成；`already current and verified` 只表示原安装完整、未修改文件。复用脚本的宿主加载步骤与一个示例：当前会话可用就直接使用，否则按宿主刷新／启用，仅宿主需要时新开对话。汇总实际 host、版本和 init 已验证／未验证／未执行；不重复十份版本或完整手册，不自动生成体验任务。

## 初始化与诊断

按意图只选一个入口：帮助读本地指南；固定 API 公开身份用 puretokens-connection；当前连接／认证用一次 init；文件／事务用 install-inspect；宿主加载位置和连接用 doctor。不依次串跑，不将介绍当成诊断。

从当前 SKILL.md 的绝对目录解析同级 `../.puretokens-executor/puretokens-api`；Windows 使用 `puretokens-api.exe`，不依赖 PATH。`init --host <host-id>` 先请求固定 `/v1` 公开身份，再请求一次 `/v1/media/models` 验证认证；最多两次只读请求共用 20 秒总预算，不自动重试，无付费任务。超时仅说明连接未验证，不证明凭据无效。`doctor --host <host-id>` 额外检查本次加载位置和当前宿主已知 Skill 目录中的安装版本／重复项，再执行同样的只读检查；不扫描 Home、不读取其他宿主配置，也不自动修复。

init 仅在 configuration_status=verified 且 api_identity_confirmed、credential_verified 均为 true 时认证通过。doctor 不证明二进制完整、会话覆盖或附件交付；不承诺模型权限、余额、媒体可用性。

只有原生执行器按当前宿主明确记录在内存中读取匹配凭据；Skill／脚本不读、显示或修改认证文件，不用其他认证、MCP、代理或 Computer Use。

正常安装不审计整份脚本、探测 PowerShell、重复清单或跑 doctor；不创建 probe／shim／Python 补丁或换启动路径。调试需另行授权。

当前会话若使用未反映在宿主已声明有效文件中的配置覆盖，停止并说明无法确认有效连接；不读取其他配置或借用默认连接。

宿主目录、会话选择限制或特定客户端故障才按需读 `references/desktop-hosts.md`；普通请求不增加诊断前置步骤。

安装不检查、移除或迁移旧插件，不运行 codex plugin。同步已确认成功但 init 明确未执行才补一次，同版快速返回不适用。`cleanup_status: pending` 单独报告，不把同步改报失败，不绕过删除限制或重装。

本地连接未识别／不可读／不支持／未验证均不等于“尚未配置”，保留现有连接，不要求重装、换模型或凭据。`api_request_executed: false` 只说明本次未请求 API、未认证凭据。
