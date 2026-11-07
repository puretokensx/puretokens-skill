import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const read = file => readFile(new URL(`../${file}`, import.meta.url), "utf8");

test("direct API acceptance includes explicit formats and reviewed masks in order", async () => {
  const contract = JSON.parse(await read("references/direct-api-execution-contract.json"));
  assert.deepEqual(contract.acceptanceScenarios.map(scenario => scenario.id), [
    "api-identity-read", "catalog-read", "media-submit", "same-task-status",
    "native-media-delivery", "explicit-image-format", "reviewed-local-mask",
    "executor-request-start",
  ]);
});

test("image guidance preserves GIF intent and declared native mask transport", async () => {
  const image = await read("skills/puretokens-image/SKILL.md");
  assert.match(image, /GIF/);
  assert.match(image, /output_format/);
  assert.match(image, /masked_image_edit/);
  assert.match(image, /local_file_data_url/);
  assert.match(image, /next_step.*不扩大.*索引/);
  assert.match(image, /缺失或不满足原请求格式／完整性/);
  const formats = await read("skills/puretokens-image/references/image-formats.md");
  assert.match(image, /references\/image-formats\.md/);
  assert.match(formats, /GIF 输出仅支持生成，不支持编辑/);
  assert.doesNotMatch(image, /本次本地附件只随声明的 multipart 发送/);
  const video = await read("skills/puretokens-video/SKILL.md");
  assert.match(video, /local_file_data_url/);
  assert.match(video, /return_last_frame=true/);
  assert.match(video, /索引 0 是视频、1 是尾帧/);
  assert.match(video, /media_operation=native_video/);
  assert.match(formats, /SVG.*8 MiB|8 MiB.*SVG/);
  assert.doesNotMatch(video, /视频只有索引 0/);
  assert.doesNotMatch(video, /独立尾帧图片尚不能交付/);
  assert.doesNotMatch(video, /本次本地附件只随声明的 multipart 发送/);
});

test("retrieval chooses a new filename only inside an already authorized directory", async () => {
  for (const skill of ["embedding", "rerank"]) {
    const entry = await read(`skills/puretokens-${skill}/SKILL.md`);
    assert.match(entry, /已明确且获准写入的工作区／输出目录/);
    assert.match(entry, /目录不明确时才询问/);
    assert.match(entry, /不把.*当前工作目录.*安装目录.*授权/);
    assert.match(entry, /不覆盖/);
  }
});

test("media correction guidance remains a one-time new-request exception", async () => {
  const guide = await read("references/skill-fragments/receipt-guide.md");
  for (const token of ["correct_request", "not_submitted", "media_request_invalid", "无 task_id", "一次", "unknown", "已有记录"]) assert.ok(guide.includes(token), token);
  for (const kind of ["image", "video"]) {
    const entry = await read(`skills/puretokens-${kind}/SKILL.md`);
    assert.match(entry, /correct_request.*not_submitted.*media_request_invalid/);
    assert.match(entry, /不要求用户编辑 JSON/);
  }
});

test("host delivery preserves the original GIF when only file attachments support it", async () => {
  const guide = await read("references/skill-fragments/host-delivery.md");
  assert.match(guide, /GIF.*原始.*附件/);
  assert.match(guide, /静态.*不.*动图|不.*静态.*替代/);
  assert.match(guide, /播放.*验收|验收.*播放/);
});
