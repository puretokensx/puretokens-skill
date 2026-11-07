# Kling 高级结构化请求

只在用户明确需要高级素材角色、已有主体／音色或多镜头设置时读取；普通文生视频继续用所选 profile 的平铺参数。

## 固定入口

六个精确 ID：`kling-video-v4`、`kling-video-v3`、`kling-video-v3-turbo`、`kling-video-v3-omni`、`kling-video-motion-control-v3.0`、`kling-video-motion-control-v2.6`。请求 `kind=video`、`operation=generate`、`media_operation=native_video`，执行器发送一次 `POST https://api.puretokensx.com/v1/videos`。即使 Omni 使用 base_video，也不改走 edits。

`parameters` 只能包含 `contents` 和可选 `settings`、`options`。不得混入任何平铺字段或 attachments；不把本地附件变成公网 URL。需要本地文件时使用原有已声明的平铺操作，不静默丢素材。

## 素材与参数

- contents 是 1–16 个对象。提示词用 `{"type":"prompt","text":"用户提示词"}`，最多一项；顶层 prompt 可省略，若同时给出必须逐字相同。motion 可无提示词。
- URL 素材只用用户给定的公网 HTTPS：`{"type":"first_frame","url":"..."}`，不探测或转存。其他角色按下表。
- 已有主体是 `{"type":"element","element_id":"已有 ID"}`；motion v3.0 还须给独立 `id`。已有音色是 `{"type":"voice","voice_id":"已有 ID","id":"独立引用 ID"}`。ID 最长 128 字节，不含空白／控制字符；不猜 ID、不创建资源、不克隆音色。

| 模型 | 额外角色与限制 | settings |
| --- | --- | --- |
| v4 | first_frame/last_frame/refer_image 合计最多 10；feature_video 最多 5；element 最多 7；voice 最多 3；素材总计最多 15 | duration 3–30；resolution 720p/1080p；aspect_ratio 21:9/16:9/9:16/1:1，参考模式可 adaptive |
| v3 | first_frame、last_frame、element；总 contents 最多 16 | duration 3–15；resolution 720p/1080p/4k；audio native/off；multi_shot 布尔 |
| v3-turbo | first_frame，最多一张 | duration 3–15；resolution 720p/1080p；纯文字时 aspect_ratio 16:9/9:16/1:1 |
| v3-omni | first_frame/last_frame、refer_image、element、voice；feature_video/base_video 二选一；参考图与主体合计最多 7，有视频时最多 4；voice 最多 2 | duration 3–15；resolution 720p/1080p/4k；audio 与 multi_shot 受模式约束 |
| motion v3.0/v2.6 | 必须一项 image 和一项 video；仅 v3.0 可额外一项 element | resolution 720p/1080p；character_orientation 必填 image/video；audio original/off；有 element 时方向必须 video |

尾帧输入必须伴随首帧。来源已决定画幅时不传会被忽略的 aspect_ratio。Omni feature_video 不配尾帧、只允许 audio=off 和 multi_shot=true；base_video 不配首尾帧、只允许 audio=original/off 和 multi_shot=false。音色不能配静音。其余细节以执行器的精确型号校验为准，不扩大字段或静默删减。

options 仅允许有界 `external_task_id` 和 `watermark_info:{"enabled":false}` 等布尔设置；external_task_id 不是幂等保证，不用于自动重提。

## 例子与恢复

```json
{
  "kind": "video",
  "operation": "generate",
  "media_operation": "native_video",
  "model": "kling-video-v3",
  "parameters": {
    "contents": [{"type": "prompt", "text": "镜头缓慢穿过空旷大厅"}],
    "settings": {"duration": 5, "resolution": "720p", "audio": "off", "multi_shot": false}
  }
}
```

先返回提交回执，再等待／取回同一视频；未知提交、失败和交付问题不自动重提。任务记录只保存安全身份和进度，不保存 contents/settings/options、提示词、素材 URL、主体／音色 ID。夹具通过不等于真实 Key 权限、计费或宿主播放验收。
