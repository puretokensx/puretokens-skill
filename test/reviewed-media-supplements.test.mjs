import assert from "node:assert/strict";
import test from "node:test";
import { supplementReviewedParameters } from "../scripts/reviewed-media-supplements.mjs";

test("reviewed frame supplements preserve live authority and do not restore absent fields", () => {
  const model = { id: "seedance-2.0", parameterSchema: {
    properties: { first_frame_image: { type: "string" }, last_frame_image: { type: "string" } }
  } };
  const result = supplementReviewedParameters(model);
  assert.equal(Object.keys(result.operations).length, 3);
  assert.deepEqual(result.operations.first_last_frame_video.requiredBodyFields, ["model", "prompt", "first_frame_image", "last_frame_image"]);
  assert.equal(model.parameterSchema.operations, undefined);
  const live = structuredClone(model);
  live.parameterSchema.operations = { first_frame_video: { disabled: true } };
  assert.deepEqual(supplementReviewedParameters(live).operations.first_frame_video, { disabled: true });
  delete live.parameterSchema.properties.last_frame_image;
  assert.deepEqual(supplementReviewedParameters(live), live.parameterSchema);
  assert.equal(supplementReviewedParameters({ ...model, id: "unknown" }).operations, undefined);
  assert.equal(supplementReviewedParameters({ id: model.id }), undefined);
});

test("reviewed Grok edit boundaries narrow stale declarations without opening generation options", () => {
  const schema = {
    properties: { aspect_ratio: { type: "string", enum: ["16:9"] } },
    operations: { video_edit: { request: { path: "/v1/videos/edits" } }, image_to_video: { declared: true } }
  };
  for (const id of ["grok-imagine-video-1.5", "grok-imagine-video-1.5-preview"]) {
    const result = supplementReviewedParameters({ id, parameterSchema: schema });
    assert.equal(result.operations.video_edit, undefined);
    assert.equal(result.constraints.unsupported_operations.video_edit, true);
    assert.deepEqual(result.operations.image_to_video, schema.operations.image_to_video);
    assert.deepEqual(result.properties, schema.properties);
  }
  const standard = supplementReviewedParameters({ id: "grok-imagine-video", parameterSchema: schema });
  assert.deepEqual(standard.constraints.unsupported_parameters_by_operation.video_edit,
    ["duration", "seconds", "aspect_ratio", "aspectRatio", "resolution"]);
  assert.deepEqual(standard.properties.aspect_ratio.enum, ["16:9"]);
  assert.ok(schema.operations.video_edit, "source capture must not be mutated");
  assert.deepEqual(supplementReviewedParameters({ id: "grok-imagine-video-other", parameterSchema: schema }), schema);
});
