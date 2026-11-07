# Sanitized host diagnostics

各系统的 `cleanup_status: pending` 表示清理未完成，不等于文件同步失败；按独立 init 结果说明连接状态，不绕过删除守卫或自动重装。锁文件存在不等于被占用；缺少管理清单不能归因于清理失败，也不能称为无害。

连接验证失败必须按脱敏状态解释，适用于所有宿主。`active_connection_record_missing` 只说明执行环境中未找到适配器声明的记录；`active_connection_unavailable` 表示读取或解释未完成；`active_connection_selection_unconfirmed` 表示无法确认有效选择；`active_connection_format_unsupported`、`active_connection_endpoint_missing`、`active_connection_endpoint_unsupported` 表示适配器支持范围或字段识别问题；历史码 `active_connection_not_puretokens` 仅表示未通过固定端点识别，不证明用户从未配置。`active_connection_credential_missing` 表示适配器未取得可用凭据，不等于服务端判定凭据无效。`active_connection_ambiguous` 表示无法安全选择，不要求删除其他连接。保留现有可用连接，反馈宿主版本和诊断码；不展示配置，不要求重装、切换聊天模型、重配或更换凭据。`api_request_executed: false` 时明确本次未发送 API 请求、未认证凭据。`host_credential_adapter_unavailable` 表示执行环境不受支持，不是用户没配置。身份或网络检查失败也不能归因于未配置。

宿主绑定与本地失败停止：首次调用前按当前应用或用户明确指定绑定一个 host，全部命令沿用，不从安装目录推断。`active_connection_*`、`workbuddy_*` 或 `host_credential_adapter_unavailable` 本地失败后停止 API 流程，保留用户原任务和已有 task ID；不得自动调用 puretokens-connection、init、doctor、models 或其他 Skill 探测，不得枚举、更换 `--host` 或借用凭据。仅用户明确重查或有可验证的新依据时，在原 host 检查一次；换宿主须用户明确指定。“继续生成”本身不是重查、换宿主或重提授权；本次未发送不代表此前未提交。
