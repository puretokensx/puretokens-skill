---
name: puretokens-connection
description: 用户询问 Pure Tokens API 的服务身份或公开标识时使用；检查当前连接能否使用、验证认证和安装诊断使用 puretokens-update。
---

# Pure Tokens Connection

宿主绑定与本地失败停止：首次调用前按当前应用或用户明确指定绑定一个 host，全部命令沿用，不从安装目录推断。`active_connection_*`、`workbuddy_*` 或 `host_credential_adapter_unavailable` 本地失败后停止 API 流程，保留用户原任务和已有 task ID；不得自动调用 puretokens-connection、init、doctor、models 或其他 Skill 探测，不得枚举、更换 `--host` 或借用凭据。仅用户明确重查或有可验证的新依据时，在原 host 检查一次；换宿主须用户明确指定。“继续生成”本身不是重查、换宿主或重提授权；本次未发送不代表此前未提交。

通过安装的单次 Go 执行器查询固定 API 公开身份。从当前 SKILL.md 的绝对目录解析同级 `../.puretokens-executor/puretokens-api`；Windows 使用 `puretokens-api.exe`，不要依赖工作目录或 PATH。调用 `<绝对执行器路径> connection --host <当前宿主 ID>`。仅执行器私密读取文档列出的当前宿主明确连接记录；凭据格式夹具通过不代表项目／会话覆盖或真实宿主端到端验收。Skill 不读取配置、不传 Key／Base URL，不自行 HTTP，也不使用其他传输。

不得申请、调用或使用 Computer Use，也不得打开、点击或控制浏览器、Pure Tokens Switch、Pure Tokens Desktop 或其他图形界面来读取配置、检查连接或替代执行器；不得调用其他 Skill 作为回退。

只处理执行器返回的结构化结果；不得把响应原文、请求头或配置内容直接展示给用户。仅当执行器返回实际本地或 API 失败时，按安全回执处理，明确固定 API 是否执行且不猜测配置原因。

本页足以执行普通身份检查；只有对应异常时才读 `references/behavior-scenarios.json`，详细约束见 `references/execution-contract.json`。

## 身份检查

- “这个服务是谁／是不是 Pure Tokens API”使用本页的公开身份检查；“检查当前连接／我的连接还能用吗／能否认证”使用 puretokens-update 的有界 init，检查身份与认证。一个意图只选一个入口，不先执行 connection 再重复 init。安装故障使用对应诊断，用法问题只读本地指南。
- 每次用户查询只调用一次 `GET https://api.puretokensx.com/v1`；不得调用模型目录、Images、Videos、余额、任务或其他路径，也不得重试。
- 只有响应 JSON 同时明确返回 `status: "ok"`、`name: "Pure Tokens API"`、`base_url: "/v1"` 时，才回答：`固定 API 返回了 Pure Tokens API 的公开身份标识；此检查未验证凭据、余额、模型权限或媒体可用性。`
- 响应不可读、字段缺失、值不匹配或请求失败时，只能回答：`固定 Pure Tokens API 未返回可确认的标识，当前无法完成检查。` 不得把该结果归因于用户 Base URL、API Key 或 provider 配置，也不得读取或要求用户粘贴它们。
- 该检查验证的是固定 API 端点的既有公开声明，不是密码学防伪证明；未确认不等于证明用户的其他连接属于某个服务。不得根据模型列表、模型名称、请求是否成功、Base URL 字符串或 provider 标签改写这一结论。
- 用户问当前真实 Base URL、API Key 或宿主 provider 时，明确说明：只有原生执行器可为固定请求在内存中使用文档列出的匹配连接凭据，但不展示、比较或报告这些配置；引导用户在 CC Switch、Pure Tokens Desktop 或宿主自身的连接设置中查看。

失败结论后可补充下一步：网络／服务异常可由用户稍后重查或联系 Pure Tokens 支持；标识不符可反馈身份检查结果。公开身份检查的 401／403 不验证凭据，不能据此要求更换 Key。执行器无法运行则说明检查尚未执行，并按回执处理本地可访问性。

当前会话若使用未反映在宿主已声明有效文件中的配置覆盖，停止并说明无法确认有效连接；不读取其他配置或借用默认连接。

宿主目录、会话选择限制或特定客户端故障才按需读 `references/desktop-hosts.md`；普通请求不增加诊断前置步骤。

本地连接失败按执行器脱敏状态解释：未识别、不可读、不支持或未验证均不等于“尚未配置”。保留现有连接，不据此要求重装、切换模型、重配或更换凭据；`api_request_executed: false` 时说明未请求 API、未认证凭据。具体宿主限制见 `references/desktop-hosts.md`。
