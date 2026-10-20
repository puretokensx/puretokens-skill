import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { compileSchema } from "./support/schema-validator.mjs";

const read = async file => JSON.parse(await readFile(new URL(`../${file}`, import.meta.url)));

test("reviewed video profiles preserve source scope separately from the public snapshot", async () => {
  const [source, catalog, index, profileSchema, indexSchema] = await Promise.all([
    read("runtime/executor/reviewed-video-profiles.json"), read("references/media-model-catalog.json"),
    read("skills/puretokens-video/references/model-index.json"),
    read("schemas/model-profile.schema.json"), read("schemas/model-index.schema.json")
  ]);
  const validateProfile = compileSchema(profileSchema, [profileSchema]);
  const validateIndex = compileSchema(indexSchema, [indexSchema]);
  assert.deepEqual(validateIndex(index), []);
  assert.equal(source.models.length, 11);
  for (const model of source.models) {
    const row = index.models.find(row => row.id === model.id);
    assert.equal(row.supportScope, "reviewed_source");
    const installed = await read(`skills/puretokens-video/references/${row.profile}`);
    assert.equal(installed.sourceReview.scope, "reviewed_source_not_live_availability");
    assert.equal(installed.sourceReview.reviewedAt, source.reviewedAt);
    assert.deepEqual(installed.parameterSchema, model.parameterSchema);
    assert.deepEqual(validateProfile(installed), []);
    for (const operation of Object.values(model.parameterSchema.operations)) {
      assert.equal(operation.request.method, "POST");
      assert.ok(["/v1/videos", "/v1/videos/edits"].includes(operation.request.path));
      assert.equal(operation.request.contentType, "application/json");
    }
  }
  for (const id of ["MiniMax-H3", "viduq3-pro", "viduq3-turbo"]) {
    const live = catalog.models.find(row => row.id === id);
    const installed = await read(`skills/puretokens-video/references/profiles/${id}.json`);
    assert.ok(live, "current public schemas must remain in the captured catalog");
    assert.ok(!live.sourceReview, "the raw public snapshot must not claim local source additions");
    assert.notDeepEqual(live.parameterSchema, installed.parameterSchema,
      "installed and embedded reviewed restrictions must survive partial live declarations");
    assert.equal(index.models.filter(row => row.id === id).length, 1);
  }
  assert.equal(index.defaultModel, "MiniMax-H3");
});

test("synchronous music duration schema enforces millisecond precision and route separation", async () => {
  const schema = await read("schemas/audio-request.schema.json");
  const validate = compileSchema(schema, [schema]);
  const request = {operation:"music_sync", model:"music_v2_5", instruction:"原样歌词",
    duration_seconds:3.125, instrumental:false, response_format:"mp3", output_dir:"/output"};
  assert.deepEqual(validate(request), []);
  assert.deepEqual(validate({...request, duration_seconds:4.001}), []);
  for (const patch of [{duration_seconds:3.0001}, {duration_seconds:601}, {duration_seconds:2.999},
    {instrumental:null}, {lyrics:"额外字段"}, {response_format:"wav"}, {operation:"music"}]) {
    assert.ok(validate({...request, ...patch}).length);
  }
});
