# 新增视频型号：源码审核范围

本说明与带 `sourceReview` 的 profile 来自网关源码，尚非线上目录开放或付费实机验收。
普通请求仍只读选中 profile。沿用固定 `/v1/videos`、`/v1/videos/edits` 和原任务读取，
不连接供应商。此批声明 JSON 公网 HTTPS 参考，不支持本地附件上传；
用户给本地附件时停止，不能上传或重托管来补足传输。

- 旧 `kling-v3`／`kling-v3-turbo` 不再用于新提交，不与当前
  `kling-video-*` 精确 ID 混用。当前六个 Kling 型号按各自 profile 选择
  已声明的平面参数与参考角色；高级需求使用独立 native_video 和
  [Kling 高级参数](kling-native-usage.md) 的 contents/settings/options，
  不混入平铺参数、不创建资源／主体或克隆音色。旧任务仍用原 ID 恢复。
- `MiniMax-H3`：1–15整数秒，固定768p；与 `minimax_h3` 不互为别名。
  首尾帧与参考素材互斥。最多9图／3视频／3音频，总计12份；允许纯音频参考。
  adaptive 须视觉输入。
- 五个 `Super Vega` 精确名称以索引为准，保留内部空格。4–15整数秒，
  1.5-pro 为4–30；480p／720p／768p／1080p／2k／4k。
  自动时长写 `auto_duration=true`，不发送 duration=-1。
  首尾帧不带 aspect_ratio，不与通用参考混用；音频参考必须有视觉输入。
  1.0最多9图／3视频／3音频，总计12；1.5最多30图／10视频／10音频，总计50。
  Pro 可用 video_edit，Lite 不支持编辑和延长。
  四个 Pro 型号可显式 `return_last_frame=true`：同任务索引 0 为视频，
  索引 1 为尾帧图片；requested_count 保持 1，分别取回、校验并交付。
  缺少尾帧仅停止同任务恢复，不静默只交视频、不截帧或重新提交。
- `viduq3-pro`／`viduq3-turbo`：1–16整数秒，540p／720p／1080p，
  文本、首帧或首尾帧；尾帧必须有首帧。
  `viduq4-preview`：3–16秒，最高4k；必须一张首帧或1–15张参考图，
  参考图可带至多3份音频。无纯文本、视频参考、尾帧或编辑。
  仅这些型号明确声明的 `image_to_video` 操作可省略额外 prompt；其他操作仍需用户提示词。

profile 中显式 false 与0须原样保留。不要填入未声明控制或从其他型号继承参数。
参考素材的字节、时长和编码检查由网关负责；文件下载完成不证明 HEVC 等编码能在宿主播放。
任务、费用和交付停止条件沿用主 Skill，已有任务恢复不受新提交限制影响。
