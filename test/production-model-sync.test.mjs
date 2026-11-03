import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { mediaDefaultModels } from "../scripts/skill-registry.mjs";
import { sourceRows } from "../scripts/sync-media-model-catalog-from-service.mjs";

const read = async file => JSON.parse(await readFile(new URL(`../${file}`, import.meta.url)));

test("new video selections use the current exact H3 ID without reviving retired catalog IDs", async () => {
  const catalog = await read("references/media-model-catalog.json");
  const index = await read("skills/puretokens-video/references/model-index.json");
  assert.equal(mediaDefaultModels.image, "gpt-image-2.5-flare");
  assert.equal(mediaDefaultModels.video, "MiniMax-H3");
  assert.equal(index.defaultModel, "MiniMax-H3");
  for (const id of ["minimax_h3", "seedance2.0"]) {
    assert.ok(!catalog.models.some(model => model.id === id));
    assert.ok(!index.models.some(model => model.id === id));
    await assert.rejects(read(`skills/puretokens-video/references/profiles/${id}.json`), { code: "ENOENT" });
  }
  const profile = await read("skills/puretokens-video/references/profiles/MiniMax-H3.json");
  assert.equal(profile.id, index.defaultModel);
  assert.deepEqual(profile.parameterSchema.operations.video_generation.request, {
    method: "POST", path: "/v1/videos", contentType: "application/json"
  });
  assert.deepEqual(profile.parameterSchema.properties.resolution.enum, ["768p"]);
});

test("generic public visibility does not invent a dedicated media schema or chat alias", () => {
  assert.deepEqual(sourceRows({ data: [{ model_name: "MiniMax-H3", vendor_id: 1,
    provider: "MiniMax", model_type: "unknown", supported_endpoint_types: ["openai"] }] }), []);
});

test("Wan 3.0 installed profiles follow the reviewed production aspect ratios", async () => {
  const catalog = await read("references/media-model-catalog.json");
  const ratios = ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9", "adaptive"];
  for (const id of ["wan3.0-video", "wan3.0-video-prime"]) {
    const profile = await read(`skills/puretokens-video/references/profiles/${id}.json`);
    assert.deepEqual(profile.parameterSchema.properties.aspect_ratio.enum, ratios);
    assert.deepEqual(catalog.models.find(model => model.id === id).parameterSchema.properties.aspect_ratio.enum, ratios);
  }
});
