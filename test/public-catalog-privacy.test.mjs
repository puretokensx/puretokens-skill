import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { assertPublicCatalogSchema, projectPublicCatalogSchema } from "../scripts/public-catalog-projection.mjs";
import { buildPublishedCatalog } from "../scripts/sync-media-model-catalog-from-service.mjs";
import { buildModelProfile } from "../scripts/sync-skill-model-profiles.mjs";
import { hasPrivateIdentity, validatePublicAcceptance } from "../scripts/validate-publication-privacy.mjs";
import { repositoryRoot } from "../scripts/skill-registry.mjs";

test("public schema boundary preserves every published and reviewed model", async () => {
  for (const file of ["references/media-model-catalog.json", "runtime/executor/reviewed-video-profiles.json"]) {
    const catalog = JSON.parse(await readFile(new URL(`../${file}`, import.meta.url)));
    for (const model of catalog.models) {
      assert.deepEqual(projectPublicCatalogSchema(model.parameterSchema), model.parameterSchema, model.id);
      assert.doesNotThrow(() => assertPublicCatalogSchema(model.parameterSchema));
    }
  }
});

test("unknown metadata and nested objects cannot enter public model profiles", () => {
  const schema = {
    properties: { meta_data: { type: "json" }, prompt: { type: "string",
      description: "synthetic-private-identity", enum: ["valid", { supplierName: "synthetic-private-identity" }] } },
    constraints: { reference_image: { max_bytes: [100], internalRouting: "synthetic-private-identity" } },
    operations: { image_generation: { request: { method: "POST", path: "/v1/images/generations", internalRevision: "synthetic-private-identity" } } }
  };
  const before = structuredClone(schema);
  const projected = projectPublicCatalogSchema(schema);
  assert.doesNotMatch(JSON.stringify(projected), /synthetic-private-identity|supplierName|internalRouting|internalRevision|description/);
  assert.deepEqual(projected.properties, { meta_data: { type: "json" }, prompt: { type: "string", enum: ["valid"] } });
  assert.deepEqual(projected.constraints.reference_image, { max_bytes: [100] });
  const model = { id: "fixture-image", provider: "OpenAI", vendorId: 1, capabilities: ["image"], parameterSchema: schema };
  for (const publish of [() => assertPublicCatalogSchema(schema),
    () => buildPublishedCatalog({}, [model], { capturedAt: "2026-11-08T00:00:00Z" }),
    () => buildModelProfile({}, "image", model)]) {
    assert.throws(publish, error => /unreviewed fields/.test(error.message) && !error.message.includes("synthetic-private-identity"));
  }
  assert.deepEqual(schema, before);
});

test("new execution constraints stop publication instead of being silently discarded", () => {
  assert.throws(() => assertPublicCatalogSchema({ constraints: { future_safety_limit: { max: 1 } } }), /unreviewed fields/);
  const schema = { properties: { future_control: { type: "integer", minimum: 1, maximum: 3, minLength: 1 } },
    constraints: { parameter_modes: { future_control: ["image_generation"] }, unsupported_operations: { future_operation: true } } };
  assert.deepEqual(projectPublicCatalogSchema(schema), schema);
});

test("public acceptance records omit removed private identities without rejecting model brands", async () => {
  assert.deepEqual(await validatePublicAcceptance(repositoryRoot), []);
  assert.equal(hasPrivateIdentity("OpenAI GPT Gemini Lyria Seedream Kling MiniMax"), false);
  const contract = JSON.parse(await readFile(new URL("../references/direct-api-execution-contract.json", import.meta.url)));
  const walk = value => typeof value === "string" ? [value] : value && typeof value === "object" ? Object.values(value).flatMap(walk) : [];
  const legacy = walk(contract).filter(hasPrivateIdentity);
  assert.ok(legacy.length, "wire compatibility dependency must remain visible until gateway migration");
  assert.equal(hasPrivateIdentity(legacy[0].toUpperCase()), true);
});

test("catalog projection rejects unsafe aliases, nonpublic routes and nested value objects", () => {
  const schema = { properties: {
    api_key: { type: "string" }, "API.KEY": { type: "string" },
    prompt: { type: "string", enum: ["ok", "Bearer synthetic-token", "https://private.invalid", { arbitrary: "synthetic-private" }], default: { arbitrary: "synthetic-private" } }
  }, operations: { image_generation: { request: { method: "POST", path: "/private/implementation" } } } };
  assert.deepEqual(projectPublicCatalogSchema(schema), {
    properties: { prompt: { type: "string", enum: ["ok"] } },
    operations: { image_generation: { request: { method: "POST" } } }
  });
});

test("unknown parameter types and request formats cannot carry private identities", () => {
  const schema = {
    properties: { prompt: { type: "private_supplier_fixture" } },
    operations: { image_generation: { request: { contentType: "private_protocol_fixture" } } }
  };
  assert.throws(() => assertPublicCatalogSchema(schema), /unreviewed fields/);
  assert.deepEqual(projectPublicCatalogSchema(schema), {
    properties: { prompt: {} }, operations: { image_generation: { request: {} } }
  });
  for (const type of ["integer", "number", "boolean", "string", "string[]", "json"]) {
    for (const contentType of ["application/json", "multipart/form-data"]) {
      assert.doesNotThrow(() => assertPublicCatalogSchema({ properties: { custom: { type } },
        operations: { image_edit: { request: { contentType } } } }));
    }
  }
});
