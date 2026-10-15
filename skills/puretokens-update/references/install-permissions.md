# 安装权限与失败后继续 / Installation permissions and continuation

仅在安装／更新出现权限或下载失败、用户明确要求继续时读取。本说明不是正常生成的前置检查，不增加网络探测、运行环境或凭据读取。

## 安装前需要什么

当前客户端会话必须能执行本地命令、HTTPS 下载、写入下载临时目录和目标 Skill 目录，并允许按既有契约运行原生执行器。优先使用宿主已报告的能力和审批结果；未报告即未知，不扫描或展示配置来猜权限。未知不等于拒绝，没有明确阻止时照正常流程执行所需命令并遵守原生审批，不为未知状态增加探测或反复询问。不把“聊天正常”“可读文件”或一次网络授权当成全部权限已就绪。

权限由用户通过当前客户端实际提供的正规入口授予；不要猜界面按钮、伪造不可用的审批方式、切换账号或自动开启完全访问。先申请所需范围。客户端政策拒绝或无审批能力时说明具体限制并停止。不同宿主使用各自的授权方式，不把 Codex 设置推广到其他客户端。

2026-10-04 的 Windows Codex 个案中，用户报告调整其所称的“沙盒权限和完全访问权限”后恢复。没有分别验证两个选项的作用或最小所需权限，不能由此认定所有用户必须完全访问，也不能保证相同错误一定由相同原因导致。

## 先确定失败阶段

| 已有证据 | 向用户说明 | 下一步 |
| --- | --- | --- |
| 旧提示词停在读取 README，fetch 尚未请求 | 安装尚未开始；只确认文档读取失败，不能推断发布附件不可达或权限被拒 | 用户明确继续、宿主已确认且无宿主拒绝时，使用自带步骤的完整官方提示词直接下载稳定 fetch 一次，不重试文档 |
| 下载 fetch 或启动命令被阻止 | 安装尚未开始；没有安装脚本执行证据 | 根据实际错误及宿主已报告的命令／网络／目录权限处理；无法确认时只问一个必要问题 |
| fetch 的 read_release_manifest、download_selector、download_platform_archive 失败 | 本次安装／更新未完成；仅当回执确认 installed_files_changed=false 才说旧安装未修改 | 保留 stage、error_code、非零 http_status、next_step；按类别处理 |
| sync 已明确成功，init 失败 | 安装已完成，连接验证未完成 | 用户要求继续连接验证后只执行一次 init，不重新下载或同步 |
| 同步中断、命令超时或输出不确定 | 安装状态未确认 | 用户另行授权诊断本地版本、受管清单和未完成事务；不自动重跑，不把 doctor 当完整性校验 |
| 同版完整性检查成功 | 已是当前稳定版且文件完整 | 不下载、不写入、不补 init；需要加载时新开对话 |

完整官方提示词自带宿主映射、稳定 fetch 地址和命令；首次所需下载就是稳定发布入口，不先请求 README、raw.githubusercontent.com 或其他远程说明。默认目录由脚本解析，Octop 仍须明确当前工作区目标，MiniMax Desktop 要求确认运行目录时不猜路径。只给仓库地址的一句话入口仍依赖文档可达性。

文档读取失败和发布附件下载失败是不同阶段；不能把一次文档连接中断定性为 Key、权限、证书或整个 GitHub 的问题。旧流程已经停止时不自动恢复，只在用户明确继续后执行上述官方入口；不是镜像、main 源码或其他传输兜底。真实宿主拒绝、入口或发布附件失败仍按原规则停止。

bootstrap 下载发生在 fetch 启动之前，其异常无法由尚未下载的脚本捕获。安装入口必须直接给权限提醒；不能声称只更新安装器即可覆盖这个阶段。此指南及 README 不会自动改写外部网页硬编码或用户已复制的提示词。

## 下载错误分类

以下是 Windows fetch 回执的分类；其他系统保留各自实际错误，不能猜成 Windows 错误。读取异常类型／数值，不展示原始异常文本、用户名、路径、配置、响应或凭据。

| error_code | 可确认的事实 | 处理建议 |
| --- | --- | --- |
| network_access_denied | SocketError.AccessDenied，Windows 常见为 10013 | next_step=review_host_permissions；先检查命令执行和网络权限。不能只凭错误码指认某个沙箱、防火墙或策略 |
| tls_security_context_unavailable | Win32Exception 的 0x8009030E / SEC_E_NO_CREDENTIALS | next_step=review_host_permissions；先核实当前会话执行权限和 Windows HTTPS 安全上下文。不是 Pure Tokens Key 拒绝，也不足以证明证书损坏或客户端缺陷 |
| tls_failure | 其他 TLS／认证握手失败 | 保留 TLS 分类，不能一律引导开启完全访问；进一步诊断须有具体证据 |
| local_io_failure | 临时下载文件访问失败 | next_step=review_download_directory_permissions；检查当前会话的下载目录写权限，不自动改 ACL |
| dns_failure / timeout / connection_failure | 相应名称解析、超时或连接错误 | next_step=review_download_failure；针对实际错误处理，不反复执行相同请求 |
| http_error | 已取得 HTTP 拒绝状态 | 按实际状态处理；公开安装资源的 401／403 不等于 Pure Tokens API Key 无效 |
| transport_failure | 现有证据无法进一步分类 | 如实说明原因未确定，不根据异常文字猜测 |

下载回执的 http_status=0 表示没有可报告的 HTTP 状态。host 权限状态不能由下载脚本自行确认；它只提供检查建议，不宣称已检查 UI、授权期限或策略。Full Access 不是默认安装要求；不改变证书、代理、账号、系统权限或使用其他工具绕过失败。

## 用户明确继续后的行为

1. 复用已有失败阶段和授权结果作为诊断证据。只在用户已处理阻塞并明确要求继续后，恢复对应阶段；不后台等待或自动重试。授权必须仍覆盖当前会话／轮次／动作；本轮授权不能跨轮复用。授权过期时按宿主要求重新审批，不能因为历史获批跳过审批；同一有效或已拒绝的申请不循环提交。
2. 尚未同步：仍用原宿主的官方入口。若 fetch 未取得，先下载为本地文件；若已取得则执行同一命令一次。fetch 清理旧临时目录，重新解析、固定并校验稳定版本；不续用不完整文件，不手动跳过清单／校验，不保证继续的是之前未固定的版本。
3. 已同步仅 init 失败：仅在用户要求继续连接验证时执行一次同宿主 init（原有 20 秒、最多两次只读请求）；不下载、不同步、不执行付费体验。
4. 状态未知、文件冲突或校验失败：不得套用权限恢复流程。不自动删除文件、重建受管清单、重装或猜测成功；诊断／修复另行确定。
5. 第二次仍失败：保留新的脱敏结果并停止。没有新证据，不再循环收集截图、版本、相同网络请求；只有明确开发排查才收集一次必要的版本／执行模式／错误码信息。外部终端成功不是客户端内成功。

这只适用于安装及只读 init。图片、视频、音乐或其他计费调用仍遵守原任务回执和不自动重提规则；安装权限恢复不授权重发业务请求。

## English summary

Use host-reported command, HTTPS, temporary-directory and target-directory permissions; unknown means unknown, not denied. Proceed with normal required commands and native approval when there is no explicit block, without extra probes. Network approval is not proof of every execution permission. Use only the current client's available approval controls. Reuse old outcomes as evidence, not as an expired grant: a turn-scoped permission must cover the current turn or be approved again. Full Access is not a default requirement, and a Windows security-context error is not an API-key rejection or proof of a client defect.

The complete official prompts include host IDs and execution steps; the first required download is the stable fetch, without a README or raw.githubusercontent.com prerequisite. A documentation failure does not establish release-asset failure. If an older prompt already stopped at documentation, explicit user continuation and a confirmed host allow one direct stable-fetch attempt, respecting host approvals; do not retry documentation, a denied action or a failed release download.

Report the last confirmed phase. Before fetch starts, installation has not started; only a verified fetch receipt can establish that installed files were preserved. After the user resolves the blocker and explicitly continues, run the same official workflow once on the same host, with fresh manifest/checksum verification and no partial-download reuse. If sync succeeded and only init failed, run init alone only when connection verification is explicitly requested. Unknown sync output requires separate local diagnosis, not reinstall. No automatic retries, permission changes, alternate transport or paid submissions.

## 维护依据

- Microsoft Windows Sockets error codes：`https://learn.microsoft.com/en-us/windows/win32/winsock/windows-sockets-error-codes-2`
- Microsoft security error codes：`https://learn.microsoft.com/en-us/windows/win32/com/com-error-codes-4`
- OpenAI Windows sandbox：`https://openai.com/index/building-codex-windows-sandbox/`
