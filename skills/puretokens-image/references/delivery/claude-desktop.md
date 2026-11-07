# claude-desktop 结果交付

沿用当前宿主绑定和 [通用交付规则](../host-delivery.md)，仅使用符合当前模式和前提的协议。

## Claude Desktop 的本地媒体预览入口

适用模式：本地桌面；媒体：图片、视频。

前提：当前是 Claude Desktop 本地 Code 会话。官方文档只声明用户点击聊天中的本地图片／视频路径可在 Browser pane 打开。

动作：可提供已校验文件的本地入口供用户点击；如当前会话另有原生附件工具，优先按通用协议交付。只靠文件入口不能承诺聊天已附加媒体，音频能力不从这段文档推断。

确认：文件入口本身不完成附件交付，不据此执行 delivered。只有当前宿主确认实际文件交接，或用户明确确认该原文件已经收到，才更新交付记录；不代用户控制浏览器验证。

依据：Claude Code Desktop 官方 Preview your app：点击图片／视频文件路径打开预览。该入口不是新的 API 传输或本项目附件验收。
- 官方资料：https://code.claude.com/docs/en/desktop#preview-your-app
