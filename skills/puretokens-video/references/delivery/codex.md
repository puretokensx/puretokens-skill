# codex 结果交付

沿用当前宿主绑定和 [通用交付规则](../host-delivery.md)，仅使用符合当前模式和前提的协议。

## Codex Desktop 的本地媒体消息

适用模式：本地桌面；媒体：图片、视频、音频。

前提：当前 Codex Desktop 会话明确声明以 Markdown 图片语法展示本地图片、视频或音频；这是当前应用的条件协议，不推广至 Codex CLI、IDE 或远程执行模式。

动作：在用户可见回复中发出 ![图片](/absolute/output.png)、![视频](/absolute/output.mp4) 或 ![音频](/absolute/output.mp3)，路径必须来自本次已校验结果，使用真实绝对路径；有原生预览或附件工具时按其 schema 使用。

确认：先发送当前应用声明支持的媒体项，再按该协议判断是否完成交接；不能在尚未发送回复时预先标记 delivered，也不能据此声称用户已经打开或播放。

依据：Codex Desktop 当前会话注入的 Images/Visuals/Files 协议；未得到这一声明的会话使用通用原生文件协议。公开通用文档不作为此格式的静态保证。
