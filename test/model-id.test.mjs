import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { validModelID } from "../scripts/model-id.mjs";
import { buildModelIndex, buildModelProfile } from "../scripts/sync-skill-model-profiles.mjs";
import { buildPublishedCatalog } from "../scripts/sync-media-model-catalog-from-service.mjs";
import { compileSchema } from "./support/schema-validator.mjs";

const catalog = JSON.parse(await readFile(new URL("../references/media-model-catalog.json", import.meta.url)));
const indexSchema = JSON.parse(await readFile(new URL("../schemas/model-index.schema.json", import.meta.url)));
const profileSchema = JSON.parse(await readFile(new URL("../schemas/model-profile.schema.json", import.meta.url)));
const querySchema = JSON.parse(await readFile(new URL("../schemas/model-query.schema.json", import.meta.url)));
const schemas = [indexSchema, profileSchema, querySchema];
const validateIndex = compileSchema(indexSchema, schemas);
const validateProfile = compileSchema(profileSchema, schemas);
const validateQuery = compileSchema(querySchema, schemas);
const model = catalog.models.find(model => model.id === "gpt-image-2(Sub)");

test("exact parenthesized model identity survives every generated document", () => {
 assert.ok(model);
 const index = buildModelIndex(catalog, "image", "gpt-image-2");
 const entry = index.models.find(entry => entry.id === model.id);
 assert.equal(entry.profile, "profiles/gpt-image-2(Sub).json");
 assert.deepEqual(validateIndex(index), []);
 assert.deepEqual(validateProfile(buildModelProfile(catalog, "image", model)), []);
 assert.deepEqual(validateQuery({model: model.id}), []);
 assert.equal(buildPublishedCatalog(catalog, [model], {capturedAt: catalog.serviceCatalog.capturedAt}).models[0].id, model.id);
});

test("generators and schemas reject unsafe IDs before producing paths", () => {
 for (const id of ["", ".", "..", "../escape", "x/../../outside", "x\\y", "/absolute", "x()", "x(a)(b)", "x(a/b)", "x\n", "x\r", "x\u0000", "x".repeat(161), " Super Vega", "Super Vega ", "Super  Vega", "Super\tVega", "Super\u00a0Vega"]) {
  const changed = {...model, id};
  const input = {...catalog, models:[changed]};
  assert.equal(validModelID(id), false, JSON.stringify(id));
  assert.throws(() => buildModelIndex(input, "image", "gpt-image-2"), /model ID/);
  assert.throws(() => buildModelProfile(input, "image", changed), /model ID/);
  assert.throws(() => buildPublishedCatalog(catalog, [changed], {}), /model ID/);
  assert.ok(validateQuery({model:id}).length, JSON.stringify(id));
  const profile = {...buildModelProfile(catalog, "image", model), id};
  assert.ok(validateProfile(profile).length, JSON.stringify(id));
  const index = buildModelIndex(catalog, "image", "gpt-image-2");
  index.models = [{id, aliases:[], profile:`profiles/${id}.json`}];
  assert.ok(validateIndex(index).length, JSON.stringify(id));
 }
 for (const id of ["gpt-image-2(Sub)", "x".repeat(160), "standard-v1.5", "Super Vega 1.5-pro"]) {
  assert.equal(validModelID(id), true);
  assert.deepEqual(validateQuery({model:id}), []);
 }
});

test("spaced exact IDs survive catalog generation, profile paths and schemas", () => {
 const changed = {...model, id:"Super Vega 1.5-pro", capabilities:["video"]};
 const input = {...catalog, models:[changed]};
 const index = buildModelIndex(input, "video", changed.id);
 assert.equal(index.models[0].profile, "profiles/Super Vega 1.5-pro.json");
 assert.deepEqual(validateIndex(index), []);
 assert.deepEqual(validateProfile(buildModelProfile(input, "video", changed)), []);
 assert.equal(buildPublishedCatalog(catalog, [changed], {}).models[0].id, changed.id);
});
