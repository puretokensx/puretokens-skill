# vscode 结果交付

沿用当前宿主绑定和 [通用交付规则](../host-delivery.md)，仅使用符合当前模式和前提的协议。

## VS Code 的原生 artifacts 工具

适用模式：本地 IDE；媒体：图片、视频。

前提：当前聊天确实提供 artifacts 工具且接受当前工作区的本地文件。官方页面将其标为 Preview；功能入口或工具不可用时，不虚构调用。

动作：按实际工具 schema 提交校验后的本地图片或视频。工具每次更新会替换当前 artifacts 列表，更新时保留本次任务已经展示的其他文件；不把用户输入图片功能当作输出交付能力。

确认：以工具确认文件已进入 artifacts／媒体展示为交接依据；聊天中的裸路径和未验证预览不算。音频仍使用当前会话明确提供的原生能力，不能从图片／视频文档推断。

依据：VS Code 官方 Artifacts 页面：图片和视频 carousel、内置 artifacts 工具、每次更新替换列表。文档说明不是本项目实机验收。
- 官方资料：https://code.visualstudio.com/docs/agents/run/artifacts
