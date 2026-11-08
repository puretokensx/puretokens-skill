import { readFileSync } from "node:fs";
import { isDeepStrictEqual } from "node:util";

const policy = JSON.parse(readFileSync(new URL("../runtime/executor/public-catalog-projection.json", import.meta.url)));
const privateKeys = new Set(["apikey", "token", "secret", "authorization", "headers", "provider", "upstream",
  "supplier", "channel", "credentials", "constructor", "prototype", "proto"]);
const paths = new Set(["/v1/images/generations", "/v1/images/edits", "/v1/videos", "/v1/videos/edits",
  "/v1/images/{image_id}", "/v1/videos/{task_id}"]);
const unsafeText = /https?:|sk-[a-z0-9_-]+|bearer\s+\S+|(?:api[_ -]?key|token|secret)\s*[:=]\s*\S+|stack trace|\b[a-z0-9-]+\.(?:com|net|org|internal|local|ai)\b|\p{Cc}/iu;
const safeString = value => Buffer.byteLength(value) <= 500 && value.trim() === value && !unsafeText.test(value);
const safeKey = key => /^[a-zA-Z0-9_.-]{1,100}$/.test(key) && safeString(key) &&
  !privateKeys.has(key.replace(/[_.-]/g, "").toLowerCase());
const scalar = value => value === null || typeof value === "boolean" ||
  (typeof value === "number" && Number.isFinite(value)) || (typeof value === "string" && safeString(value));

function project(value, rule) {
  if (Array.isArray(rule)) return rule.includes(value) ? value : undefined;
  if (rule && typeof rule === "object") {
    if (!value || typeof value !== "object" || Array.isArray(value)) return undefined;
    return Object.fromEntries(Object.entries(value).flatMap(([key, item]) => {
      if (!safeKey(key)) return [];
      const child = Object.hasOwn(rule, key) ? rule[key] : rule["*"];
      const selected = child === undefined ? undefined : project(item, child);
      return selected === undefined ? [] : [[key, selected]];
    }));
  }
  if (rule === "values" || rule === "value") {
    if (Array.isArray(value)) return value.filter(scalar);
    if (rule === "value" && scalar(value)) return value;
  }
  if (rule === "string" && typeof value === "string" && safeString(value)) return value;
  if (rule === "number" && typeof value === "number" && Number.isFinite(value)) return value;
  if (rule === "boolean" && typeof value === "boolean") return value;
  if (rule === "method" && (value === "GET" || value === "POST")) return value;
  if (rule === "path" && paths.has(value)) return value;
  return undefined;
}

export function projectPublicCatalogSchema(value) {
  return project(value, policy);
}

// Publication must stop for review, not silently remove a new safety constraint
// or persist previously unknown metadata in installed profiles. Never echo it.
export function assertPublicCatalogSchema(value) {
  if (!isDeepStrictEqual(value, projectPublicCatalogSchema(value))) {
    throw new Error("Media schema contains unreviewed fields or values; review the public projection contract before publishing. No raw metadata is reported.");
  }
}
