import { readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { repositoryRoot } from "./skill-registry.mjs";
import { supplementReviewedParameters } from "./reviewed-media-supplements.mjs";
import { assertModelID } from "./model-id.mjs";

const catalogPath = path.join(
  repositoryRoot,
  "references",
  "media-model-catalog.json"
);

const supportedCapabilities = new Set(["image", "video"]);
const imageEndpointTypes = new Set(["image-generation", "image_generation"]);
const videoEndpointTypes = new Set(["openai-video", "openai_video"]);

function parseArgs(argv) {
  const args = {
    input: "",
    sourceUrl: process.env.PURETOKENS_BASE_MODEL_CATALOG_URL || "",
    sourcePath: "/api/product/docs/model-catalog",
    write: false,
    check: false
  };
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--write") args.write = true;
    else if (arg === "--check") args.check = true;
    else if (arg === "--input") args.input = argv[++index] || "";
    else if (arg === "--source-url") args.sourceUrl = argv[++index] || "";
    else if (arg === "--source-path") args.sourcePath = argv[++index] || "";
    else throw new Error(`unknown argument: ${arg}`);
  }
  if (args.write === args.check) throw new Error("use exactly one of --write or --check");
  if (args.input && args.sourceUrl) throw new Error("use either --input or --source-url, not both");
  if (!args.input && !args.sourceUrl) {
    throw new Error("set PURETOKENS_BASE_MODEL_CATALOG_URL or pass --source-url/--input");
  }
  return args;
}

async function readCatalogSource(args) {
  if (args.input) {
    const source = args.input === "-"
      ? await readStdin()
      : await readFile(path.resolve(args.input), "utf8");
    return JSON.parse(source);
  }

  const headers = { Accept: "application/json" };
  const token = process.env.PURETOKENS_BASE_MODEL_CATALOG_TOKEN;
  if (token) headers.Authorization = `Bearer ${token}`;
  const response = await fetch(args.sourceUrl, { headers });
  if (!response.ok) throw new Error(`base model catalog request failed with HTTP ${response.status}`);
  return response.json();
}

function readStdin() {
  return new Promise((resolve, reject) => {
    let source = "";
    process.stdin.setEncoding("utf8");
    process.stdin.on("data", (chunk) => { source += chunk; });
    process.stdin.on("end", () => resolve(source));
    process.stdin.on("error", reject);
  });
}

function text(value) {
  return typeof value === "string" ? value.trim() : String(value ?? "").trim();
}

function normalizedEndpointTypes(model) {
  const values = model?.supported_endpoint_types
    ?? model?.supportedEndpointTypes
    ?? model?.endpoint_types
    ?? model?.endpointTypes
    ?? [];
  const types = Array.isArray(values) ? values : values ? [values] : [];
  return new Set(types.map((value) => text(value).toLowerCase()).filter(Boolean));
}

function explicitCapabilities(model) {
  const capabilities = new Set();
  const declared = Array.isArray(model?.capabilities) ? model.capabilities : [];
  for (const value of declared) {
    const capability = text(value).toLowerCase();
    if (supportedCapabilities.has(capability)) capabilities.add(capability);
  }
  const endpointTypes = normalizedEndpointTypes(model);
  if ([...endpointTypes].some((type) => imageEndpointTypes.has(type))) capabilities.add("image");
  if ([...endpointTypes].some((type) => videoEndpointTypes.has(type))) capabilities.add("video");
  return [...capabilities].sort();
}

function explicitParameterSchema(model) {
  const candidate = model?.media_input_schema
    ?? model?.mediaInputSchema
    ?? model?.input_schema
    ?? model?.inputSchema
    ?? model?.media_parameters
    ?? model?.mediaParameters;
  if (!candidate || typeof candidate !== "object" || Array.isArray(candidate)) return undefined;
  return candidate;
}

export function sourceRows(payload) {
  if (!Array.isArray(payload?.data)) throw new Error("base model catalog response is invalid");
  const vendors = new Map(
    (Array.isArray(payload?.vendors) ? payload.vendors : [])
      .map((vendor) => [text(vendor?.id), text(vendor?.name)])
      .filter(([id, name]) => id && name)
  );
  return payload.data.map((row) => {
    const id = text(row?.model_name ?? row?.id);
    const vendorId = Number(row?.vendor_id ?? row?.vendorId);
    const provider = text(row?.provider ?? row?.owned_by ?? row?.ownedBy) || vendors.get(text(vendorId)) || "";
    return {
      id,
      provider,
      vendorId,
      capabilities: explicitCapabilities(row),
      parameterSchema: explicitParameterSchema(row)
    };
  }).filter((model) => model.id && model.capabilities.length);
}

function genericCopy(id, capabilities) {
  const isVideo = capabilities.includes("video") && !capabilities.includes("image");
  return {
    aliases: [],
    goodFor: {
      en: isVideo ? "Video generation" : "Image generation",
      zh: isVideo ? "视频生成" : "图片生成"
    },
    example: {
      en: isVideo ? `Use ${id} to generate a short video.` : `Use ${id} to generate an image.`,
      zh: isVideo ? `用 ${id} 生成一条短视频。` : `用 ${id} 生成一张图片。`
    }
  };
}

export function buildPublishedCatalog(previous, models, args) {
  const previousById = new Map((Array.isArray(previous?.models) ? previous.models : []).map((model) => [model.id, model]));
  const normalized = models.map((model) => {
    assertModelID(model.id);
    if (!model.provider || !Number.isInteger(model.vendorId)) {
      throw new Error(`${model.id}: base model catalog must provide provider and vendor ID`);
    }
    const editorial = previousById.get(model.id) || genericCopy(model.id, model.capabilities);
    return {
      id: model.id,
      name: model.id,
      provider: model.provider,
      vendorId: model.vendorId,
      capabilities: model.capabilities,
      aliases: editorial.aliases || [],
      goodFor: editorial.goodFor || genericCopy(model.id, model.capabilities).goodFor,
      example: editorial.example || genericCopy(model.id, model.capabilities).example,
      ...(model.parameterSchema ? { parameterSchema: supplementReviewedParameters(model) } : {})
    };
  }).sort((left, right) => left.id.localeCompare(right.id, "en", { numeric: true }));
  const capturedAt = args.capturedAt || new Date().toISOString();
  return {
    schemaVersion: 2,
    updatedAt: capturedAt.slice(0, 10),
    serviceCatalog: {
      source: "Pure Tokens base model catalog",
      path: args.sourcePath || args.sourceUrl,
      capturedAt
    },
    availabilityNotice: {
      en: "This list is an installed selection aid, not a per-request authorization check. Ordinary generation reads only the selected profile; live discovery is limited to explicit requests, profile gaps or rejection diagnosis. Reviewed local compatibility supplements are separately sourced.",
      zh: "这份清单用于安装后的模型选择，不是每次请求的认证检查。普通生成只读选中 profile；仅明确查询、profile 缺口或拒绝诊断时读取实时目录。经审查的本地兼容补充定义单独注明来源。"
    },
    models: normalized,
    localMetadataRevision: {
      date: "2026-10-18",
      source: "references/media-parameter-review.md; scripts/reviewed-media-supplements.mjs; runtime/executor/reviewed-media-policy.json",
      retainedAfterCatalogRefresh: ["Seedance first/last frame operations and exclusive reference constraints when both frame fields remain declared"],
      conservativeRestrictions: ["Grok 1.5/preview video editing is closed; standard Grok edits inherit source duration, ratio and resolution"],
      note: "Local compatibility additions and conservative restrictions are not claimed as returned by the public catalog or as live availability proof. Reviewed restrictions narrow stale declarations; they never enable an operation or parameter."
    }
  };
}

function serialized(value) {
  return JSON.stringify(value, null, 2) + "\n";
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const [previous, payload] = await Promise.all([
    readFile(catalogPath, "utf8").then(JSON.parse),
    readCatalogSource(args)
  ]);
  const next = buildPublishedCatalog(previous, sourceRows(payload), {
    ...args,
    capturedAt: previous?.serviceCatalog?.capturedAt
  });
  const currentText = serialized(previous);
  const nextText = serialized(next);
  if (currentText === nextText) return;
  if (args.check) {
    throw new Error("published media model catalog is out of sync with the base model catalog; run npm run docs:sync-media-models-from-service");
  }
  const refreshed = buildPublishedCatalog(previous, sourceRows(payload), {
    ...args,
    capturedAt: new Date().toISOString()
  });
  await writeFile(catalogPath, serialized(refreshed));
  process.stdout.write(`Published ${refreshed.models.length} base media models.\n`);
}

if (process.argv[1] === fileURLToPath(import.meta.url)) await main();
