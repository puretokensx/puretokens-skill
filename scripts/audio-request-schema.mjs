// Development-only projection of the same reviewed profiles embedded in Go.
export function buildAudioRequestSchema(catalog) {
  const absolute = { type: "string", pattern: "^(?:/|[A-Za-z]:[\\\\/]|\\\\\\\\)[^\\u0000]+$(?![\\s\\S])" };
  const text = maxLength => ({ type: "string", minLength: 1, maxLength, pattern: "\\S" });
  return {
    $schema: "https://json-schema.org/draft/2020-12/schema",
    $id: "https://puretokensx.com/schemas/audio-request.schema.json",
    title: "One synchronous reviewed audio request",
    oneOf: Object.values(catalog.models).filter(p => p.operation !== "music").map(p => {
      const properties = { operation: { const: p.operation }, model: { const: p.model } };
      const required = ["operation", "model"];
      const add = (key, rule, mandatory = true) => {
        properties[key] = rule;
        if (mandatory) required.push(key);
      };
      if (p.operation !== "transcribe") {
        add("response_format", { enum: p.formats });
        add("output_dir", absolute);
      }
      if (["speech", "voice_change"].includes(p.operation)) {
        add("voice", p.voice_pattern
          ? { type: "string", maxLength: 120, pattern: p.voice_pattern + "(?![\\s\\S])" }
          : { enum: Object.keys(p.voices) });
      }
      if (p.operation === "speech") {
        add("input", text(p.max_input_characters));
        if (p.speed_min) add("speed", { type: "number", minimum: p.speed_min, maximum: p.speed_max }, false);
        if (p.supports_instruction) add("instruction", text(500), false);
      }
      if (["generate", "music_sync"].includes(p.operation)) {
        add("instruction", text(p.max_instruction_characters));
        if (p.duration_max) add("duration_seconds", {
          type: "number", minimum: p.duration_min, maximum: p.duration_max,
          ...(p.duration_milliseconds ? { multipleOf: 0.001 } : {})
        });
        if (p.instrumental_required) add("instrumental", { type: "boolean" });
      }
      if (["transcribe", "voice_change"].includes(p.operation)) add("file", absolute);
      return { type: "object", required, properties, additionalProperties: false };
    })
  };
}
