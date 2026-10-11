---
name: puretokens-balance
description: 查询当前 Pure Tokens 连接的账户钱包余额、当前 Key 剩余额度，或询问消费记录和额度范围差异时使用。
---

# Pure Tokens Balance

宿主绑定与本地失败停止：首次调用前按当前应用或用户明确指定绑定一个 host，全部命令沿用，不从安装目录推断。`active_connection_*`、`workbuddy_*` 或 `host_credential_adapter_unavailable` 本地失败后停止 API 流程，保留用户原任务和已有 task ID；不得自动调用 puretokens-connection、init、doctor、models 或其他 Skill 探测，不得枚举、更换 `--host` 或借用凭据。仅用户明确重查或有可验证的新依据时，在原 host 检查一次；换宿主须用户明确指定。“继续生成”本身不是重查、换宿主或重提授权；本次未发送不代表此前未提交。

仅询问消费记录入口或额度范围时，直接使用下方说明，不运行执行器、不读取凭据或请求网络。用户实际要求查询余额时，调用安装的单次原生执行器查询余额。从当前 SKILL.md 的绝对目录解析同级 `../.puretokens-executor/puretokens-api`；Windows 使用 `puretokens-api.exe`，不要依赖工作目录或 PATH。调用 `<绝对执行器路径> balance --host <当前宿主 ID>`。仅执行器私密读取文档列出的当前宿主明确连接记录；Skill 不读取配置、不传 Key／Base URL，不自行 HTTP，也不使用其他传输。

执行器复用现有 Pure Tokens API Key，与官方 CC Switch 查询使用同一接口：`GET https://console.puretokensx.com/api/product/console/api-keys/usage`。成功后仅再读取一次公开的 `GET https://console.puretokensx.com/api/product/console/status` 获取金额换算比例，该请求不携带 Key。最多两次 GET，共用 30 秒截止时间，不自动重试。余额是固定 console 域名的特例；其他 API Skill 的固定 `https://api.puretokensx.com` 地址不变。不需要 Desktop、Web 登录态或用户粘贴 Key；不加 init、模型目录、其他 endpoint 或浏览器查询。详细规则按需读 `references/execution-contract.json`，异常时读 `references/behavior-scenarios.json`。

余额入口要求 Key 带 `sk-` 前缀；执行器与官方 CC Switch 导入逻辑一致，在首次请求前仅在内存中补齐缺失的前缀，已有前缀不重复添加。不会改用户配置、截断 Key 后缀、影响生图／视频认证或在 401 后试另一把 Key。原始与补齐后的凭据均不得出现在回执中。

以执行器返回的 `result.remaining` 和 `unit` 为准，默认只回复一行（以下金额为示例）：

- `scope: account_wallet`：**当前账户钱包余额：10.00 USD。**
- `scope: key_allowance`：**当前 Key 剩余额度：10.00 USD（不代表账户钱包余额）。**

金额通常保留两位小数；非零但小于 0.01 的余额显示“低于 0.01 USD”，不要误报为零。负值如实说明欠额，不改为零。仅用户要求详情时补充 `used`／`total`；`total` 是接口返回的总额度，不叫充值总额或硬性上限。不输出 JSON、配置信息、长篇技术说明或无关诊断字段。

`unlimited_quota=true` 的官方接口分支返回账户钱包余额；false 返回该 Key 的额度。Key 不限额不等于钱包无限。这里的余额不含订阅套餐额度；不要把 Key 额度或钱包余额说成包含订阅的统一可用余额。只有用户问总可用、订阅或金额不一致时才解释此范围，并提供 [Pure Tokens 官网 · 钱包](https://console.puretokensx.com/wallet)，请用户查看钱包及套餐信息；不自动相加或把范围差异解释成需要充值。

剩余金额直接来自 `total_available`，按公开 `quota_per_unit` 转成 USD，与官方 CC Switch 脚本一致；不得再减已用额度，不使用旧兼容 billing 接口或其中的 `100000000` 占位值，不猜换算比例或人民币金额。查询不证明一次生成的实际价格、权限或一定可以付款；只有同币种、同范围的实际报价才能比较，最终由提交 API 判定。不要为每次生图／视频自动查余额。

失败时先读结构化回执，即使进程非零退出也不重复执行。余额 401／403 只说明余额接口拒绝了这次查询，不能推断整个连接未启用或 Key 无效。若对话中生图／视频或其他调用已成功，简要回复“余额查询被拒绝（401），但生图连接可用。请保留现有配置，并向 Pure Tokens 反馈余额查询失败”，使用实际 HTTP 状态；不要让用户换 Key、重装或修改可用连接。

执行器通过本地诊断码区分 `balance_usage_auth_rejected`（余额认证拒绝）、`balance_usage_unavailable`（余额读取失败）与 `balance_unit_metadata_unavailable`（公开换算信息失败）；这些是本地诊断码，不是服务端 `api_error_code`。默认用自然语言解释，用户需要排查时才提供脱敏码。429／网络／超时说明稍后可再查；公开换算信息失败不表示 Key 无效；格式或换算信息缺失则说“暂时无法确认余额，请稍后重试或到 Pure Tokens 官网控制台查看”。仅展示实际返回的脱敏原因，未返回余额绝不填零或估算，不索取凭据、不要求安装客户端。

失败答复应简洁包含实际失败范围和可执行的下一步，遵循余额专属 `next_action`，不只报错误码。额度／余额不足按已识别分类解释，并提供 [Pure Tokens 官网 · 钱包充值](https://console.puretokensx.com/wallet)，引导用户登录查看钱包并按需充值；额度不足还需提醒检查当前 Key 限额。此链接仅供用户自行访问，不作为 API 或浏览器代查入口。不把查询失败说成余额为零，不保证充值解决 Key 限额。

官网引导只适用于当前对话模型仍能正常回复，且已收到独立 API 请求的结构化结果或用户主动询问的情况。当前对话模型本身因认证或服务故障无法返回时，Skill 无法执行提示，应由宿主处理；不因此新增 Key 管理或渠道状态的官网跳转。官网链接仅供用户自行访问，不调用浏览器代查，不新增 API 请求。

用户主动询问消费明细、历史扣费或某次生成的实际费用，而现有结果无法确认时，提供 [Pure Tokens 官网 · 使用记录](https://console.puretokensx.com/usage-logs)，建议按发生时间及模型核对。不要用余额差、任务状态或暂未找到记录推断是否扣费，不承诺退款，也不把使用记录入口说成自动找回任务的能力。不在普通生成或所有失败回执中自动附加此入口。

当前会话若使用未反映在宿主已声明有效文件中的配置覆盖，停止并说明无法确认有效连接；不读取其他配置或借用默认连接。

宿主目录、会话选择限制或特定客户端故障才按需读 `references/desktop-hosts.md`；普通请求不增加诊断前置步骤。

本地连接失败按执行器脱敏状态解释：未识别、不可读、不支持或未验证均不等于“尚未配置”。保留现有连接，不据此要求重装、切换模型、重配或更换凭据；`api_request_executed: false` 时说明未请求 API、未认证凭据。具体宿主限制见 `references/desktop-hosts.md`。
