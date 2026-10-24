# Switch × Skills 专用模型兼容矩阵

核对时间：2026-10-24（Asia/Shanghai）。**0.19.8 已发布为最新稳定版**，发布提交为 `c25c11d44a12f8677c4fa695b5cb08be0f8811b7`；保留 0.19.6 的 TRAE 与新增模型适配，并同步本轮客户端说明、模型目录和独立图片交付。
发布身份以同名正式标签及经过校验的 `release-manifest.json` 为准，共 11 个发布附件。
0.19.5包含 Grok 限制与向量／重排 Skill，其发布身份与历史批准归档保留。
当前 `host-acceptance.json` 属于 0.19.8，维护者已于 2026-10-24 明确批准本版稳定发布；
该批准不继承旧版授权，不把 172 项待验收改为通过，也不替代工程与安装包校验。
正式标签、11 个匿名下载附件、三系统 CI、macOS arm64 隔离安装及同版快速返回均已验证，见[发布证据](acceptance/0.19.8-publication.json)。安装／更新流程不增加内部验收记录提示。

本轮 Switch 源码基线为 `878bf024` 加当前工作区，Web 为 `d80c6922` 加当前工作区；
源码和本地验证不代表这两个项目的新版本已经部署。音频／Jev指令自 0.18.7
起已分发；Switch 已有 Jev、音频、向量与重排引导，本轮不修改其配置。示例不等于
已执行、接口上线或真实宿主交付验收，正式安装入口仍取已发布稳定版本。

目标链路：Switch 选择分组并应用 **聊天连接** → 用户单独安装 Skills →
当前客户端执行对应 Skill → 原生执行器从该客户端有效连接取得凭据 →
固定专用 API 按精确模型 ID 鉴权 → 当前客户端真实展示或附件交付。
专用模型不需要进入聊天模型列表；用户仍需有一个能执行工具的聊天模型。
只选专用分组而没有兼容聊天模型时，Switch 不应伪造聊天模型来完成应用。

## 每格的五项证据

每格写作 `发布层级 [①,②,③,④,⑤]`：

1. 当前客户端实际加载正确 Skill 并运行原生执行器。
2. 从 **Switch 实际写入的当前客户端连接**取得正确 Key，而非任意默认连接。
3. Key 的所选分组及多分组完整模型名单包含精确目标模型。
4. 使用精确模型 ID 和对应专用 API，收到可解释的专用结果。
5. 图片实际打开、音视频实际播放／原生附件交付、转写和 Jev 文本实际展示，或向量／重排的明确用户文件保存并校验。

`发布` = **已发布支持**的实现，`源码` = **仅源码支持**，均不等于实机通过。
`夹` = **夹具通过**（仅合成配置／模拟 API）；`实` = **实机通过**；
`不` = **不支持**；`待` = 待实机验证，无通过声明。
③中的夹具只证明所选分组并集及刷新逻辑，**没有读取或验证任何用户真实 Key**。
①和⑤必须在同一客户端／系统／架构／会话模式上验收，不能拿安装器或文件校验替代。

当前 Switch 源码目录为 23 个客户端；Skills 注册 19 个安装宿主、
19 个凭据夹具适配器（含 0.19.6 的 TRAE）。DeepSeek Bot、Paseo、Rakazo、
OpenMausBot 不因此获得直接 Skill 支持。目录成员不代表线上已部署。

下表以 0.19.8 正式发布为分发边界；向量／重排的具体任务和输出见下文。
各行 ② 的精确路径及选择限制见
[凭证契约](credential-adapters.md)，系统支持范围见
[宿主验收记录](host-acceptance.json)。当前版本没有任何“实”格。
①均为“待”，虽然安装／执行器夹具已覆盖；⑤也均为“待”，模拟附件不是实际交付。

| 客户端（Switch ID） | 图片 | 视频 | 专用音频（含音乐） | Jev | 向量 | 重排 |
| --- | --- | --- | --- | --- | --- | --- |
| Claude Code (`claude-code`) | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] |
| ChatGPT / Codex (`codex`) | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] |
| WorkBuddy (`workbuddy`) | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] |
| Gemini CLI (`gemini-cli`) | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] |
| Grok Build (`grok-build`) | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] |
| OpenCode (`opencode`) | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] |
| Claude Desktop 本地 Code (`claude-desktop`) | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] |
| DSH Desktop (`dsh-desktop`) | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] |
| 官方 DeepSeek Harness (`deepseek-harness`) | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] |
| MiniMax Code Desktop (`minimax-code`) | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] |
| ZCode (`zcode`) | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] |
| Kimi Code (`kimi-code`) | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] |
| Qoder (`qoder`) | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] |
| Pi (`pi`) | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] |
| Hermes (`hermes`) | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] |
| EvoX (`evox`) | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] |
| **VS Code 多模型** (`vscode`) | **发布** [待,夹,夹,夹,待] | **发布** [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] |
| Octop 本地工作区 (`octop`) | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] | 发布 [待,夹,夹,夹,待] |
| **TRAE / TRAE SOLO** (`trae`) | 发布 [待,夹,待,夹,待] | 发布 [待,夹,待,夹,待] | 发布 [待,夹,待,夹,待] | 发布 [待,夹,待,夹,待] | 发布 [待,夹,待,夹,待] | 发布 [待,夹,待,夹,待] |
| DeepSeek Bot (`deepseek-bot`) | 不 [不,不,待,不,不] | 不 [不,不,待,不,不] | 不 [不,不,待,不,不] | 不 [不,不,待,不,不] | 不 [不,不,待,不,不] | 不 [不,不,待,不,不] |
| Paseo (`paseo`) | 不 [待,不,待,不,不] | 不 [待,不,待,不,不] | 不 [待,不,待,不,不] | 不 [待,不,待,不,不] | 不 [不,不,待,不,不] | 不 [不,不,待,不,不] |
| Rakazo (`rakazo`) | 不 [不,不,待,不,不] | 不 [不,不,待,不,不] | 不 [不,不,待,不,不] | 不 [不,不,待,不,不] | 不 [不,不,待,不,不] | 不 [不,不,待,不,不] |
| OpenMausBot (`openmausbot`) | 不 [不,不,待,不,不] | 不 [不,不,待,不,不] | 不 [不,不,待,不,不] | 不 [不,不,待,不,不] | 不 [不,不,待,不,不] | 不 [不,不,待,不,不] |

矩阵中②④是已有宿主适配器夹具和公共执行器夹具的**组合证据**，不是
每个宿主都执行过四类专用调用。VS Code与本次MiniMax Code测试贯穿实际命令分发、
Switch writer 对应文件、模拟权限拒绝和四类专用路径。上表向量／重排行的
②③表示共享连接／分组机制的夹具覆盖，④表示专用执行器的离线 API 夹具，
不是每个客户端都完成了真实调用。
Cursor 当前不在 Skill 注册表及 Switch 当前可用配置入口内，六类均不支持。

## 2026-10-24 客户端增量与边界

- Claude Desktop writer 已去掉 `inferenceDefaultModel`，默认模型置于
  `inferenceModels[0]`。执行器仍按 `_meta.json.appliedId` 选择唯一 gateway
  profile，不以模型顺序推断连接；新旧数组形状、重排、无效当前选择有仓库内夹具。
- Grok Build 的 `name` 只含模型 ID；执行器仍以 `models.default` 取连接，
  新旧显示名不参与认证。VS Code 配置形状及同组同 Key 守卫保持不变。
- TRAE 的 Switch 安装身份选择不扩展 Skills 范围；执行器遇到多个桌面
  状态库仍停止，不按 Switch 优先级取首项。
- DeepSeek Bot 是独立社区客户端，不是 DSH Desktop 或官方 Harness。
  Switch 通过 `DS_BOT_HOME`、shell `locations.json.dataDir` 或 `~/.ds-bot`
  定位 `profiles/ds-bot/cordis.patch.yml` 及同 home 的 `.credentials.yaml`；
  shell 根可由 `DS_BOT_SHELL_DATA` 指定。这只是 writer 契约，不授权 Skills
  读取这些文件。当前各 Bot 独立选模型，不能从保存的默认值猜当前 Bot 连接。
  接入前须独立定义 Skill 加载与绑定、当前会话有效连接及覆盖拒绝、凭据
  最小读取和真实结果交付，再实现安装器／适配器及验收；暂不注册宿主。

来源、合成配置摘要及本轮验证见
[2026-10-24 脱敏证据](acceptance/client-model-sync-20261024.json)。
本轮只同步 Skills；Switch 与 Web 的现有工作区改动均保留。

## 官方 DeepSeek Harness 新增范围

Skills 注册 19 个宿主，19 个有凭据夹具；0.19.6 包含 TRAE 适配器。官方Harness使用独立 `deepseek-harness`，仅默认本地Desktop组合；不是社区DSH。安装目录是DSH_HOME/skills或~/.dsh/skills，读取Desktop Cordis patch及其声明的refs。高优先级home patch、额外插件组合、会话覆盖、环境凭据覆盖或多匹配连接均不借用旧默认。0.18.7 起已分发该宿主实现，0.18.6 不含此适配。

共用writer夹具及实际dispatcher的图片200/401/403已覆盖；其他专用能力是共享执行器组合证据。①⑤仍待实机，不能宣传安装即可完成全部能力。来源及本次模型差异见[更新证据](acceptance/client-model-sync-20261003.json)。

## 分组权限、目录和已发布媒体

- Switch `config/catalog.rs` 的 `selected_group_model_ids` 保存完整有序并集；
  `compatible_client_model_ids` 仅决定聊天列表。专用模型名单不能用聊天子集覆盖。
- Switch `account/managed_keys.rs` 对相同分组的受管 Key 调用
  `update_managed_key_groups` 刷新模型名单，不旋转 Key。确认窗口过期时先刷新
  分组并拒绝不一致快照；配置事务检查 Key 覆盖所选分组。
- Web `product-client-resource-handlers.mjs` 从所选分组重算 `model_limits`，
  多分组写入 `auto_groups` 和限制，回读验证完整集合。单分组由组路由限制；
  不是“关闭名单限制就能用其他分组”。刷新必须重新应用／显式触发，不承诺后台自动刷新。
- 真实调用仍受 Key 状态、额度、渠道和网关专用路由约束。不能以目录可见、
  模型安装 profile、聊天可用或 init 成功证明专用 POST 有权限。普通生成
  不加目录预查；鉴权失败停止，不轮换 Key、不更换模型、不自动重提。
- 2026-10-24 匿名目录含 129 个模型，38 个明确媒体声明（15 图片、23 视频）。
  Kling V3/Turbo 和五款 Super Vega 新增专用声明，但已有独立审核适配；
  安装 profile 仍为 15 图片／24 视频，参数边界不变。`viduq4-preview`
  仍不在公开目录。当前默认图片为 `gpt-image-2.5-flare`、视频为 `MiniMax-H3`；
  下方 2026-10-18 记录中的旧默认仅为历史。
- 2026-10-18 12:14 UTC 上线后匿名目录有 114 个模型，其中 30 个图片／视频模型（15 图片、15 视频）；相比同日较早快照，新增 `wan3.0-video`、`wan3.0-video-prime`，移出 `grok-imagine-image-2.0`，同步 Grok 新画幅。默认图片 `gpt-image-2.5-flare`、默认视频 `minimax_h3` 保持。完整变化及独立负向限制见[当前审查](media-parameter-review.md)和[脱敏证据](acceptance/client-model-sync-20261018.json)。目录可见不证明某个 Key 有权调用；安装选项变化不改写已有任务 ID 或自动替换用户精确选择。新配置及多分组名单仍须由用户在 Switch 重新应用，不能把公开新增当作已有 Key 权限已刷新。
- 图片和视频仍按异步任务协议：提交一次、先回执、再独立等待／取原任务；
  所选profile声明multipart时，图片编辑才发送当前附件原字节至 `/v1/images/edits`；仅声明JSON URL的型号不接受本地附件，不自动上传托管。
  下载成功只进入待宿主交付，只有真实交付后才标记 delivered。

### 0.19.5 的 Grok 限制

`grok-imagine-video-1.5` 和 `grok-imagine-video-1.5-preview` 关闭新建
`video_edit`；标准 `grok-imagine-video` 的编辑继承源视频时长、画幅及最高
720p 输出，源视频最长 8.7 秒。提交不接受 `duration`／`seconds`、
`aspect_ratio`／`aspectRatio`、`resolution`，也不会静默丢弃这些用户要求。
共享精确 ID 限制同时作用于安装 profile、实时查询和提交前校验，旧目录或
缺失本地 profile 不能恢复关闭的能力。原任务查询、恢复和交付不受影响。

上线后目录已对标准 Grok 和 1.5 声明生成画幅 `21:9`／`5:2`，
0.19.5 已同步安装 profile；仅允许生成使用，不扩大编辑能力。
Wan 两款视频模型按目录声明使用 JSON 参考或独立 multipart 附件操作。
公开元数据、源码和夹具通过仍不能替代真实 Key 权限、计费调用和宿主交付验收。

## 专用接口与输出合同

专用调用固定在 `https://api.puretokensx.com`，不使用聊天条目的 URL 作请求目标：

| 能力 | 精确 ID 例子／权威来源 | 专用入口 | 交付 |
| --- | --- | --- | --- |
| 图片 | `gpt-image-2.5-flare`（默认）；已安装 image profile | `/v1/images/generations`、multipart `/v1/images/edits` | 同任务图片字节，再真实宿主附件 |
| 视频 | `MiniMax-H3`（默认）、`grok-imagine-video-1.5`；video profile | `/v1/videos`、声明的 `/v1/videos/edits` | 同任务 MP4/WebM，再实际播放／附件 |
| 配音／转写／声音 | `stepaudio-2.5-tts`、`stepaudio-2.5-asr`、`stepaudio-3-gen-preview`；[音频合同](audio-execution-contract.md) | `/v1/audio/speech`、`/v1/audio/transcriptions`、`/v1/audio/generate` | 同步音频字节或最终 text |
| 音乐 | `stepaudio-3-music-preview`；音频合同 | `/v1/audio/music/submit`，仅原任务 status/content | 异步 MP3/WAV，非同步音频命令 |
| ElevenLabs 音频源码增量 | 9个精确 ID，见音频合同；含 `music_v2_5`、`eleven_multilingual_sts_v2` | 配音／转写同上；同步音乐及变声 `/v1/audio/generate` | 同步 MP3／转写 text；不使用异步音乐任务 |
| Jev | `jev-latest`、`jev-1.13.0`、`jev-preview`；[Jev合同](jev-evaluation-contract.md) | `/typesafe/v1/systemone` | 同步 Choice/Score/Noul 类型结果 |

精确 ID 必须同时满足安装 profile 与真实 Key 分组／接口权限；
示例和源码允许列表不是对当前账户的推荐／授权。音频／Jev仍未实机通过。

## VS Code 修复范围与会话覆盖

0.18.6 在出现第二条 Pure Tokens 模型时停止。0.18.7 只在同一个
`customendpoint` 组内验证每条完整 URL、协议、唯一模型 ID 和内联 Bearer；
所有条目必须对应同一受支持服务和同一 Key。不同 Key、多匹配组、混合服务、
额外认证来源、动态值、重复 ID 和不支持的发现／路由覆盖都停止。
组名、Switch 标记不用于推断服务身份；不猜聊天框当前模型。
比较只在执行器内存中进行，回执没有 Key、摘要指纹或比较结果。

支持范围仍限 macOS／Windows **本地默认 profile**。便携模式已主动拒绝；
命名 profile、远程／WSL、宿主未反映在声明文件中的会话覆盖不支持。
无法从文件判断一个不可观察的会话覆盖，因此不能宣称“所有覆盖已检测”。
若宿主上下文与声明文件不一致，Skill 必须停止；覆盖隔离的真实验收仍待做。
OpenCode/Pi 的已声明覆盖顺序有离线回归，但也不代表所有宿主覆盖均通过。

## MiniMax Code 范围（0.18.7 起分发）

静态桌面包3.1.0证据确认 `<runtime-data>/skills` 和 `config.yaml` 的默认选择。Switch可写三种协议提供方；执行器只取 `defaultModel` 指向的启用条目，不要求删除其他模型，不比较不同提供方Key。CLI／未知会话覆盖不支持；独立安装器遇到桌面偏好文件但没有运行目录环境时要求已确认的绝对目标，不猜默认路径。具体选择、文件安全和拒绝项见凭证契约。跨仓库同一合成writer夹具覆盖两系统，尚无实机验收。

## TRAE

TRAE 官方支持全局与项目 Skills。国际版使用 `~/.trae/skills`，国内版使用
`~/.trae-cn/skills`，项目使用 `.trae/skills`。2026-10-20 核对官方
`https://docs.trae.cn/ide_skills`、`https://docs.trae.ai/ide/skills` 及本机
TRAE 3.6.4 的技能发现和模型缓存代码。版本号只作证据，不是配置门槛。

本轮源码新增单一 Switch 已保存连接适配器：只读唯一默认桌面数据库中
`puretokens.traework.status.v1` 的 active 回执和两个同账号模型缓存键，校验
所有回执模型的完整元数据与单一凭据指纹后，才在内存取匹配端点的 `ak`。
多安装版本数据库、多分组不同凭据、未完成事务、编辑或缺失模型均停止，不猜分组。
该契约证明保存连接，不证明当前聊天或 TRAE 登录账号的会话选择；手工／便携／远程
配置不在范围内。它不读取其他账号缓存，不解密登录信息或宿主恢复秘密，不新增凭据副本。
Switch 内部 CDP 不构成 Skills 传输许可；Skills 仅使用原生执行器的固定 API。

Switch 新写入的 `display_name` 仅为模型名，不含品牌或事务号；受管身份仍由
active 回执、模型 ID、账户和完整元数据匹配确认，不依赖可见名称前缀。
两仓库的合成 `trae-switch-skill-connection.json`／`traework/skill-connection.json`
夹具保持字节一致；旧事务前缀回执只要仍完整匹配，也继续可读。此项展示名同步
不改变执行器凭据逻辑；0.19.6 同时包含六平台执行器，但不证明真实宿主验收。

安装器按唯一已存在的 `.trae`／`.trae-cn` 根目录选择；两者并存要求明确绝对目标，
不迁移或删除旧目录。安装后在 TRAE 技能设置启用／刷新并新建会话。
源码、执行器与安装器夹具通过不等于已发布；真实技能加载、鉴权、分组权限、
图片打开、音视频播放和文本／文件交付仍须分别验收。

Paseo 的 Switch 源码写独立 `agents.providers` 配置，不能假定被启动的 Codex
把它落盘到 `~/.codex`。须先核对其会话传递与 Skill 发现，当前无直接宿主支持。
Rakazo 使用宿主服务管理的连接与加密凭据，当前没有可供本执行器读取的已验证
有效连接文件；不增加本地服务查询、解密或另存 Key。
OpenMausBot 的模型配置和独立 MCP 集成不能作为直接 Skill 发现／凭据／交付
证据；Bot 的 MCP 能力由对应项目独立维护，不复用为本仓直接 API 传输。
Orca／Herdr已从当前Switch／Web公开目录移除，保留历史记录但不再列入当前矩阵；任何壳客户端都不能借用内部Agent身份宣称独立支持。

## 0.19.5 向量与重排

本版新增 `puretokens-embedding`、`puretokens-rerank`，共十个 Skill 和十一份
受管清单。0.19.5 的 18 个凭据适配器复用其原有连接边界；0.19.6 新增
第 19 个 TRAE 适配器。Paseo、Rakazo、OpenMausBot 不因此获得直接 Skill 支持。新命令的源码、
离线 API 夹具、安装校验与真实宿主／计费验收分开记录，①④⑤未实机通过。

`embedding|rerank --host <当前宿主> --request <绝对JSON文件>` 仅接受两个
flags。先校验显式文本和绝对新文件路径，再读取当前宿主已验证的连接凭据，
同步单次90秒，无重试、轮询、隐式文件读取、目录扫描或建库。
请求最多1 MiB、响应最多2 MiB，1–64条文本、单条最多32768个 Unicode 字符，
重排另需最多8192字的非空 query；top_n 可省略并默认全部。

| 任务 | 精确审核 ID | 固定请求 | 明确用户产物 |
| --- | --- | --- | --- |
| 文本向量 | `qwen3.7-text-embedding`、`qwen3.7-text-embedding-flash` | `POST /v1/embeddings`，`encoding_format=float` | 新 `.jsonl`，每行 `{index,dimensions,embedding}`，按原输入序号排序；聊天只报条数／维度／路径 |
| 文本重排 | `qwen3.7-text-rerank` | `POST /v1/rerank`，`return_documents=false` | 新 `.json`，`{results:[{rank,index,relevance_score}]}`；rank从1起、原始index从0起，保留有限原始分，可展示相同排序摘要 |

分数不是概率。成功须 `output_verified=true`；接受后文件保存失败保留
`accepted`、`output_verified=false`，未知结果或交付失败不证明未扣费，
不自动重提。输出不是隐藏缓存或隐式向量库，也不承诺附件／实机已验收。
仅明确查询当前可见模型才用 `models kind=embedding|rerank`，一次
`GET /v1/models` 与审核精确 ID 相交；不作为常规预检、不代表专用调用成功。
完整[执行与回执契约](embedding-rerank-execution-contract.md)随本版维护。

Switch 中的 Jev／音频／向量／重排引导均应让用户保留现有分组、按明确的线上
分组信息补选权限、应用聊天连接并单独安装 Skills。用法示例只帮助表达输入和
结果，不能代替真实请求；专用命令以实际安装版本为准，不把专用模型
塞进聊天列表，也不将连接写入或目录可见报告为执行通过。

## 脱敏证据与验收

- E8：[2026-10-18 当前目录与客户端同步](acceptance/client-model-sync-20261018.json)：
  匿名快照摘要、22／19／18 范围、Grok 精确限制及新画幅上线依赖；`capabilityCompletion` 记录向量／重排的源码、契约、分发及 Switch 回归，原 `validation` 明确保留为此前批次。无真实 Key、
  计费调用或真实宿主交付证据。向量／重排各新增独立实机验收项，总计21项；旧媒体通过不自动推广为新能力通过。
- E1：`runtime/executor/{connection_diagnostics,client_file_credentials,desktop_credentials,opencode_credentials,pi_credentials,zcode_credentials,new_host_credentials}_test.go`：
  合成宿主配置、声明路径、选择与覆盖边界；不含用户凭据。
- E2：`test/fixtures/switch-vscode-connection.json` 与 Switch
  `tests/fixtures/client-configs/vscode/skill-connection.json` 字节一致。
  Switch `skill_connection_fixture_matches_actual_multi_model_writer` 在两个系统路径夹具
  中把实际 writer 输出与文件比较；Skill `TestSwitchVSCodeConnectionBoundary`
  覆盖三种聊天协议、同 Key、多 Key、多组、覆盖与不可变配置。
- E3：`TestSwitchVSCodeDedicatedPermissionsAndFreshCatalog` 运行实际 dispatcher；
  图片／视频／音频／Jev 的模拟 200、401、403，各仅一次专用 POST；
  目录有→无→有反映最新响应而不使用聊天模型列表。不是生产网关权限验收。
- E4：Web `services/bff/test/switch-skill-model-permissions.test.mjs`：
  完整专用模型并集刷新、移除旧名单、排除组外模型、保存未确认失败，
  不取 Key、不旋转 Key、不请求推理。Switch 客户端分组／确认快照测试提供上游证据。
- E5：`combined_workflow_test.go`、`chain_regression_test.go`、
  `audio_test.go`、`music_test.go`、`evaluation_test.go`：
  原字节附件、同任务续接、未知结果不重提、音频文件／转写及 Jev 类型。
  宿主附件交付在这些测试中模拟，⑤不因此改为实机通过。
- E7：[2026-10-04增量证据](acceptance/client-model-sync-20261004.json)：MiniMax静态桌面源码摘要、双仓库writer夹具及媒体目录变化。
- E6：[2026-10-03公开目录差异和宿主证据](acceptance/client-model-sync-20261003.json)：包含源摘要、精确增删ID和候选边界；9月29日快照仅保留作历史，不作为当前模型可用性或价格依据。
- 实际系统／客户端验收沿用 `host-acceptance.json`，不能复用不同版本、
  不同宿主或 Web/Pic 的成功结果。测试执行记录见
  [本轮离线证据](acceptance/switch-skill-20260929.md)。
- 下一阶段按[实机方案与成本](switch-skill-test-plan.md)单独授权。
  本轮没有读取真实客户端认证文件、修改真实配置或发起付费请求。

## 0.19.6 增量及线上目录复核（2026-10-21）

0.19.6 包含 19 个安装宿主及 19 个凭据夹具（含 TRAE）。Switch/Web
展示 ChatGPT，内部仍用 codex；不更改配置路径、启动器或凭据边界。
新增11个视频profile和9个ElevenLabs音频型号为仅源码支持／夹具验证，
不升级矩阵中的分组、专用API或实际交付状态。02:27 UTC 复核共 124 个型号，
其中 31 个明确图片／视频型号；MiniMax-H3、两款 Vidu Q3 已补齐公开视频参数，
另外七款已审核视频仍只有通用 openai 标记，Vidu Q4 未列出。公开元数据和安装配置
分别保留来源；公开补参不能覆盖已审核边界，不把可见性当专用接口验收。
旧 minimax_h3、seedance2.0 已移出目录；新任务视频默认改为精确 MiniMax-H3，
参考输入仅公开 HTTPS URL，旧任务身份不变，图片默认不变。
同步音乐music_v2_5使用audio/music_sync，StepFun保留kind=music异步任务；
新增变声voice_change须原录音及有权限的音色ID。API、音色权限和真实播放均待验收。
源证据和验证结果见 [本轮记录](acceptance/client-model-sync-20261021.json)。
