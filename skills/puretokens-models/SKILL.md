---
name: puretokens-models
description: 查询当前 Pure Tokens 连接的图片／视频模型、参数、媒体操作，或明确查询音频／Jev／embedding／rerank 模型在认证目录的可见性、精确型号对应哪个线上分组时使用。
---

# Pure Tokens Models

宿主绑定与本地失败停止：首次调用前按当前应用或用户明确指定绑定一个 host，全部命令沿用，不从安装目录推断。`active_connection_*`、`workbuddy_*` 或 `host_credential_adapter_unavailable` 本地失败后停止 API 流程，保留用户原任务和已有 task ID；不得自动调用 puretokens-connection、init、doctor、models 或其他 Skill 探测，不得枚举、更换 `--host` 或借用凭据。仅用户明确重查或有可验证的新依据时，在原 host 检查一次；换宿主须用户明确指定。“继续生成”本身不是重查、换宿主或重提授权；本次未发送不代表此前未提交。

通过安装的单次 Go 执行器查询模型目录。从当前 SKILL.md 的绝对目录解析同级 `../.puretokens-executor/puretokens-api`；Windows 使用 `puretokens-api.exe`，不要依赖工作目录或 PATH。调用 `<绝对执行器路径> models --host <当前宿主 ID>`。仅执行器私密读取文档列出的当前宿主明确连接记录；凭据格式夹具通过不代表项目／会话覆盖或真实宿主端到端验收。Skill 不读取配置、不传 Key／Base URL，不自行 HTTP，也不使用其他传输。

不得申请、调用或使用 Computer Use，也不得打开、点击或控制浏览器、Pure Tokens Switch、Pure Tokens Desktop 或其他图形界面来发现模型、读取配置或替代执行器；不得调用其他 Skill 作为回退。

只处理执行器返回的结构化结果；不得把响应原文、请求头或配置内容直接展示给用户。仅当执行器返回实际本地或 API 失败时，按安全回执处理，明确目录请求是否执行且不猜测配置原因。

详细字段按需读 `references/execution-contract.json`；只有对应异常时才读 `references/behavior-scenarios.json`。

用户需要选型比较或提出“最好／最快／最便宜”时，按需读
[选型指南](references/selection-guide.md)：先按必需能力筛选，明确哪些比较项没有数据。
普通生成不增加这一步；仅问价格且没有可查询的兼容条件时，不调用无报价字段的目录。

## 音频模型查询

用户明确查询当前可见音频模型时，用 `{"kind":"audio"}` 筛选文件，
可加精确 `model` 或 `operation`（speech／transcribe／generate／music），
执行 `<绝对执行器路径> models --host <当前宿主 ID> --request <筛选文件>`。
该模式只请求一次 `GET https://api.puretokensx.com/v1/models`，取返回 ID
与已安装音频审核模型的交集，去重后展示；不接受 parameters 筛选。
`matching_scope=reviewed_audio_models_listed_by_api` 仅证明本次目录可见。
操作与音色参数来自本地审核契约，不能冒充目录返回的 schema，不能据此
承诺路由上线、权限、价格或生成质量；空结果不证明模型下线。
普通配音、录音转写、声音和音乐生成交给 puretokens-audio，不先查目录。
仅问用法、安装版本支持范围或音色时读取 puretokens-audio 本地说明，
不发请求。未限定范围的 models 保持图片／视频目录语义，只有用户明确
要求多个类别时分别查询，禁止从一个目录推断其他类别可用。

## Jev 评估模型查询

用户明确查询当前 Jev／JEV／TypeSafe 评估模型时，用 UTF-8 筛选文件
`{"kind":"evaluation"}`（可加精确 `model`），执行
`<绝对执行器路径> models --host <当前宿主 ID> --request <筛选文件>`。
该模式只发一次 `GET https://api.puretokensx.com/v1/models`，只列出返回 ID
与已审查 `jev-latest`、`jev-1.13.0`、`jev-preview` 的交集，不以名称前缀
猜能力，不接受 operation／parameters 筛选，不回退静态清单或媒体目录。
返回的 evaluation 类型来自已安装契约，输入 schema 不由此目录声明；
`matching_scope=reviewed_evaluation_models_listed_by_api` 只代表目录可见，
不证明原生入口、评估权限、价格或质量。没有返回只说明本次未列出。
仅询问 Jev 的概念／用法时读 puretokens-evaluate 本地指南，不发请求。
普通评估不先查目录；明确评估任务交给 puretokens-evaluate。
用户同时要媒体和评估目录时分别执行其明确请求的两个查询，不把一个响应
冒充另一个；未限定范围的现有 models 命令仍只查询媒体目录。

## 文本向量与重排模型查询

仅用户明确查询当前可见模型时，创建 UTF-8 筛选文件
`{"kind":"embedding"}` 或 `{"kind":"rerank"}`，可加精确 `model`，
运行 `<绝对执行器路径> models --host <当前宿主 ID> --request <绝对筛选文件>`。
每次只发一个 `GET https://api.puretokensx.com/v1/models`：
embedding 取返回 ID 与 `qwen3.7-text-embedding`、
`qwen3.7-text-embedding-flash` 的交集；rerank 只与
`qwen3.7-text-rerank` 相交。去重后展示，不按名称前缀猜能力，
不接受 operation／parameters，不回退媒体目录或用静态名单冒充实时结果。

`matching_scope=reviewed_embedding_models_listed_by_api` 或
`reviewed_rerank_models_listed_by_api` 仅代表本次认证目录可见；
不证明固定 POST、当前权限、价格、真实宿主、实际计费或输出质量。
没有列出只说明本次未返回，不能断言模型下线。用户同时明确查询两类时
分别执行各自一次查询，不能把一个响应冒充另一类。

明确生成向量交给 puretokens-embedding，明确重排交给 puretokens-rerank；
普通任务不先查 models、init 或余额。仅询问概念／用法／安装版本支持范围
时读对应 Skill 的本地指南，不联网、不提交。向量不是聊天模型，
重排分值不是概率，不以目录可见为由修改聊天模型列表或发付费探测。

## 图片／视频查询规则

- 每次图片／视频认证目录查询只调用一次 `GET https://api.puretokensx.com/v1/media/models`；不得调用 Images/Videos 提交、任务状态、内容、余额或其他路径，也不得重试。
- 当前认证目录是唯一事实来源。只报告实际返回的精确模型 ID、明确返回的 `capabilities`、`input_schema.properties` 参数资料、`input_schema.constraints` 条件限制和 `input_schema.operations`；不得以 README、安装包静态清单、模型名称或过往任务补全、猜测或回退。
- 用户可查询：当前可用图片/视频模型；某个精确模型；支持某个能力或操作的模型；以及某项时长、画幅、分辨率、尺寸、数量或参考媒体要求有哪些兼容模型。
- 需求匹配只基于当前条目明确声明的 capability、`input_schema.properties` 字段和值、适用的 `input_schema.constraints`、以及 `input_schema.operations`。例如图生视频只匹配发布 `image_to_video` 的模型；参考图、参考视频、参考音频和视频编辑分别只匹配 `reference_image_video`、`reference_video`、`reference_audio`、`video_edit`。没有明确声明就不列为兼容。
- 不对模型质量、速度、价格、用量、排队时间、内容效果或未返回的可用性做推荐或排序。多个模型都满足已声明条件时，完整列出并说明它们在目录声明的差异；用户的需求无法映射到明确 capability、参数或 operation 时，请其选择明确约束或显示完整目录。

## 图片／视频用户可见输出

- 按用户的筛选条件展示；未筛选时先按图片和视频 capability 分组。
- 每个模型仅展示实际返回的：精确 ID、capability、可选参数名称、`required` 标记、类型、默认值、`enum` 值、数值范围、非请求字段的条件限制（例如 `resolution_by_mode`），以及 operation 名称、请求方法、相对路径、content type、必需字段、附件数量和 transport。`constraints` 不是额外请求字段，绝不把它的名称或推断模式写入 API body。缺失字段写“目录未声明”，不得猜测。
- 若用户询问“哪个能做 X”，先给出满足 X 的兼容模型和匹配依据；不要直接提交生成请求。用户选定模型并提出生成需求后，再交由 `puretokens-image` 或 `puretokens-video` 按各自契约执行。
- 目录为空或精确模型缺失，只说明当前目录没有返回对应资料，不能断言 Key 分组错误。可展示实际返回的模型；用户预期有权限时，建议在自身设置中核对该模型权限或联系 Pure Tokens 支持。需要查询分组时按下节读取线上公开映射，不能把映射当作当前 Key 的权限证明，也不得猜分组名称。网络／服务失败建议稍后再查询；认证、权限、限流按回执 `next_action` 分别解释。不切换 endpoint、不以静态目录冒充实时结果、不自动重试或提交媒体。

用户明确筛选时，可用 `models --host <host-id> --request <UTF-8筛选文件>`，文件可含 `kind`、精确 `model`、`operation` 和 `parameters`，例如 `{"kind":"video","operation":"image_to_video","parameters":{"resolution":"720p"}}`。执行器只读取一次认证目录并按其声明筛选；缺少字段不视为兼容，不提交媒体。无筛选则省略 --request。查询目录不是普通生成的必需前置步骤。

## 线上声明与安装版本

当前目录契约未声明模型退役状态、下线时间或官方替代型号，不能从查询缺失、名称、快照日期推断下线或宣称某个型号是官方替代。可以基于当前认证目录明确声明的能力，列出满足用户要求的其他兼容候选；这不是健康检查、实际调用权限保证或生成授权。失败后的提议、确认及新任务处理由图片／视频 Skill 的 model-alternatives 指南约束，不能把普通目录查询变成自动换模型。官方退役／替代信息需 Web／网关先发布相应字段及日期／范围契约，再适配本地 profile 和展示；现有 input_schema.lifecycle 是任务创建／轮询协议，不是模型退役信息。不为此增加普通生成前的联网检查。

筛选复用提交的字段与组合校验，包含所选 operation 必需输入产生的互斥限制；查询不验证附件字节或保证实际提交权限。查询结果也不会改写已安装 profile。

用户明确要求核对新参数时，只查询选定精确 model。普通提交对缺失字段／操作可按需读一次目录，但已有字段的类型、枚举和范围不会自动放宽。线上声明与本地限制不一致时说明差异，用户要求更新后由 puretokens-update 安装稳定发布，再按新 profile 验证；没有覆盖新声明的稳定包就报告适配待发布。不要手改受管 profile、换模型、把查询成功当成提交成功，或在已有未知／进行中任务上重提。

当前会话若使用未反映在宿主已声明有效文件中的配置覆盖，停止并说明无法确认有效连接；不读取其他配置或借用默认连接。

宿主目录、会话选择限制或特定客户端故障才按需读 `references/desktop-hosts.md`；普通请求不增加诊断前置步骤。

本地连接失败按执行器脱敏状态解释：未识别、不可读、不支持或未验证均不等于“尚未配置”。保留现有连接，不据此要求重装、切换模型、重配或更换凭据；`api_request_executed: false` 时说明未请求 API、未认证凭据。具体宿主限制见 `references/desktop-hosts.md`。

## 线上分组查询

用户询问精确型号对应哪个分组，或已确认模型权限拒绝后需要分组指引时，读
[线上分组指引](references/model-groups.md)，使用独立的匿名 `model-groups` 命令。
公开归属不表示当前 Key 权限；旧服务缺字段就给通用指引，不写死分组名。
只问分组不调用认证目录或生成；仅问本 Skill 用法不联网。
