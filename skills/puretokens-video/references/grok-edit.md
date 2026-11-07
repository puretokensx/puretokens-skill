# Grok 视频编辑

Grok 编辑限制：`grok-imagine-video-1.5`／`grok-imagine-video-1.5-preview` 暂不支持视频编辑，不按旧目录尝试。标准 `grok-imagine-video` 的编辑继承源视频时长、画幅与分辨率（输出最高 720p），源视频最长 8.7 秒；不发送 `duration`／`seconds`、`aspect_ratio`／`aspectRatio` 或 `resolution`。用户要求改变这些规格时先解释限制，不能静默删字段、把编辑改成生成或自动换模型。源视频的实际格式／时长由网关校验，不另加下载或探测；旧任务仍沿用原 task ID 查询与交付。
