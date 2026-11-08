# Wan 原生参考输入

仅适用于 `wan3.0-video`、`wan3.0-video-prime` 的 `generate`，精确选择
`media_operation=native_generation`。固定 `POST /v1/videos`，沿用一次提交、
立即回执、同任务 status/wait/content 和真实附件交付，不调用独立 edit API。
普通文本、首尾帧、图片／音视频参考继续使用已有 profile 的简单入口。

请求的 `parameters` 内仅允许 `input` 和可选的 `parameters` 设置对象，
不与平面参考或设置混用。`input` 仅有 `prompt` 和 `media`；提示词最多
20000 字符，顶层 prompt 可省略，若同时提供必须完全相同。提示词与媒体
不能同时为空。media 每项仅 `{type,url}`，URL 必须是明确提供的公网 HTTPS：

- `first_frame`／`last_frame` 各最多一项；尾帧依赖首帧，不与其他素材混用。
- `reference_image`／`reference_video`／`reference_audio` 分别最多 10／5／5 项，合计最多 20 项。
- `file`／`link` 合计最多一项，必须明确设置 `prompt_extend=true`。不是磁盘文件路径，也不读取、抓取或托管本地文件。

设置仅允许 duration（2–30 的整数或 -1 自动）、resolution（480P/720P/1080P，大小写均可）、
ratio（adaptive、21:9、16:9、4:3、1:1、3:4、9:16），以及布尔 audio、
prompt_extend、watermark。保留明确 false，不添加用户未请求的设置。
file/link 不允许关闭或省略 prompt_extend，也不承诺原样提示词控制生成。

```json
{
  "kind": "video",
  "operation": "generate",
  "model": "wan3.0-video",
  "media_operation": "native_generation",
  "parameters": {
    "input": {
      "prompt": "根据提供的参考制作一段视频",
      "media": [{"type": "link", "url": "https://example.com/reference"}]
    },
    "parameters": {"duration": 5, "prompt_extend": true}
  }
}
```

示例 URL 仅为占位，不得作为用户素材提交。参考音视频的实际时长／内容由服务端
校验；本地角色和数量通过不证明素材或账户可用。拒绝本地附件、data URL、私有
地址、自定义 headers、未知字段及客户端幂等声明。任务记录和回执不保存 prompt、
参考 URL 或嵌套输入。失败不重提，也不自动改模型。
