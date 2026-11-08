import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { mediaDefaultModels } from "../scripts/skill-registry.mjs";
import { sourceRows } from "../scripts/sync-media-model-catalog-from-service.mjs";
import { supplementReviewedParameters } from "../scripts/reviewed-media-supplements.mjs";

const read = async file => JSON.parse(await readFile(new URL(`../${file}`, import.meta.url)));

test("published Kling models have exact dedicated profiles without replacing historical IDs", async () => {
  const index = await read("skills/puretokens-video/references/model-index.json");
  for (const id of ["kling-video-v4", "kling-video-v3", "kling-video-v3-turbo", "kling-video-v3-omni",
    "kling-video-motion-control-v3.0", "kling-video-motion-control-v2.6"]) {
    assert.ok(index.models.some(model => model.id === id), `${id} missing`);
    const profile = await read(`skills/puretokens-video/references/profiles/${id}.json`);
    assert.equal(profile.id, id);
    for (const operation of Object.values(profile.parameterSchema.operations)) {
      assert.equal(operation.request.path, "/v1/videos");
      assert.equal(operation.request.method, "POST");
    }
  }
});

test("new video selections use the current exact H3 ID without reviving retired catalog IDs", async () => {
  const catalog = await read("references/media-model-catalog.json");
  const index = await read("skills/puretokens-video/references/model-index.json");
  assert.equal(mediaDefaultModels.image, "gpt-image-2.6");
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

test("image default selects the exact GPT 2.6 profile without retiring explicit alternatives", async () => {
  const index = await read("skills/puretokens-image/references/model-index.json");
  const manifest = await read("skills/puretokens-image/skill.json");
  const contract = await read("skills/puretokens-image/references/execution-contract.json");
  assert.equal(index.defaultModel, "gpt-image-2.6");
  assert.equal(manifest.rules.defaultModel, index.defaultModel);
  assert.equal(contract.parameterValidation.defaultModel, index.defaultModel);
  const selected = index.models.find(model => model.id === index.defaultModel);
  assert.ok(selected);
  const profile = await read(`skills/puretokens-image/references/${selected.profile}`);
  assert.equal(profile.id, index.defaultModel);
  assert.equal(profile.parameterSchema.operations.image_edit.request.path, "/v1/images/edits");
  assert.ok(index.models.some(model => model.id === "gpt-image-2.5-flare"));
});

test("embedding default is declared without making the executor model implicit", async () => {
  const manifest = await read("skills/puretokens-embedding/skill.json");
  const contract = await read("skills/puretokens-embedding/references/execution-contract.json");
  const schema = await read("schemas/embedding-request.schema.json");
  assert.equal(manifest.rules.defaultModel, "qwen3.7-text-embedding");
  assert.equal(contract.parameterValidation.defaultModel, manifest.rules.defaultModel);
  assert.equal(schema.properties.model.default, manifest.rules.defaultModel);
  assert.ok(schema.required.includes("model"));
  assert.deepEqual(schema.properties.model.enum, ["qwen3.7-text-embedding", "qwen3.7-text-embedding-flash", "voyage-4-large"]);
  assert.deepEqual(contract.parameterValidation.reviewedModelIds, schema.properties.model.enum);
  const help = await readFile(new URL("../skills/puretokens-update/references/usage-guide.md", import.meta.url), "utf8");
  for (const id of schema.properties.model.enum) assert.ok(help.includes(id), `local help omits ${id}`);
  assert.match(help, /input_type=query\/document/);
  assert.match(help, /output_dimension=256\/512\/1024\/2048/);
  assert.match(help, /不把这些参数传给 Qwen/);
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
    assert.deepEqual([...profile.parameterSchema.properties.aspect_ratio.enum].sort(), [...ratios].sort());
    assert.deepEqual([...catalog.models.find(model => model.id === id).parameterSchema.properties.aspect_ratio.enum].sort(), [...ratios].sort());
  }
});

test("published model projections narrow unsafe transports without mutating API evidence", () => {
  const parameterSchema = {
    properties: { output_format: { enum: ["png", "gif", "svg"] }, input: { type: "object" }, return_last_frame: {type: "boolean", default: true} },
    operations: {
      image_edit: { inputs: { images: { transports: ["https_url", "multipart_file", "data_url"] } } },
      native_generation: { declared: true }
    }
  };
  const before = structuredClone(parameterSchema);
  for (const id of ["gemini-3.0-pro-image", "gemini-3.1-flash-image", "gemini-nano-banana-2.1",
    "veo-3.1", "veo-3.1-fast", "veo-3.1-lite", "omni-flash"]) {
    const projected = supplementReviewedParameters({ id, parameterSchema });
    assert.deepEqual(projected.operations.image_edit.inputs.images.transports, ["https_url", "local_file_data_url"]);
    assert.deepEqual(projected.properties.output_format.enum, ["png", "gif"]);
    assert.deepEqual(projected.properties.return_last_frame.enum, [false]);
    assert.equal(projected.properties.return_last_frame.default, undefined);
  }
  const wan = supplementReviewedParameters({ id: "wan3.0-video", parameterSchema });
  assert.deepEqual(wan.properties.input, { type: "json" });
  assert.deepEqual(wan.operations.native_generation.request, { method: "POST", path: "/v1/videos", contentType: "application/json" });
  assert.equal(wan.constraints.unsupported_inputs.idempotency_key, true);
  assert.equal(wan.constraints.unsupported_inputs.input, undefined);
  const unknown = supplementReviewedParameters({ id: "fixture-unrestricted-model", parameterSchema });
  assert.deepEqual(unknown.operations.image_edit.inputs.images.transports, before.operations.image_edit.inputs.images.transports);
  assert.deepEqual(unknown.properties.input, before.properties.input);
  assert.deepEqual(parameterSchema, before);
});

test("reviewed masks project only their JSON local mask without changing multipart edits", async () => {
  for (const id of ["gpt-image-2.6", "gpt-image-2.6-flare", "gpt-image-2.6-sunburst"]) {
    const parameterSchema = { operations: {
      masked_image_edit: { inputs: { image_urls: { transports: ["https_url"] }, mask: { transports: ["data_url"] } } },
      image_edit: { inputs: { image: { transports: ["multipart_file"] }, mask: { transports: ["multipart_file"] } } }
    } };
    const before = structuredClone(parameterSchema);
    const projected = supplementReviewedParameters({ id, parameterSchema });
    assert.deepEqual(projected.operations.masked_image_edit.inputs.mask.transports, ["local_file_data_url"]);
    assert.deepEqual(projected.operations.masked_image_edit.inputs.image_urls.transports, ["https_url"]);
    assert.deepEqual(projected.operations.image_edit, before.operations.image_edit);
    assert.deepEqual(parameterSchema, before);
    const installed = await read(`skills/puretokens-image/references/profiles/${id}.json`);
    assert.deepEqual(installed.parameterSchema.operations.masked_image_edit.inputs.mask.transports, ["local_file_data_url"]);
    assert.deepEqual(installed.parameterSchema.operations.image_edit.inputs.mask.transports, ["multipart_file"]);
  }
});
