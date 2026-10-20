// Keep aligned with runtime/executor/model_id.go and the public JSON schemas.
export function validModelID(id) {
  return typeof id === "string" && id.length <= 160 && id !== "." && id !== ".." &&
    /^[A-Za-z0-9_.-]+(?: [A-Za-z0-9_.-]+)*(\([A-Za-z0-9_.-]+\))?$(?![\s\S])/.test(id);
}

export function assertModelID(id) {
  if (!validModelID(id)) throw new Error("Catalog contains an unsafe or unsupported exact model ID.");
}
