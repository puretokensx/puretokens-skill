// Local compatibility metadata, not an authenticated/live availability claim.
// Evidence and scope: references/media-parameter-review.md.
import { readFileSync } from "node:fs";

const policy = JSON.parse(readFileSync(new URL("../runtime/executor/reviewed-media-policy.json", import.meta.url), "utf8"));
if (policy.schemaVersion !== 1 || !policy.models) throw new Error("Invalid reviewed media policy.");
const seedanceModels = new Set(["seedance-2.0", "seedance-2.0-fast", "seedance-2.0-mini", "seedance-2.5"]);

export function supplementReviewedParameters(model) {
  const schema = structuredClone(model.parameterSchema);
  const boundary = policy.models[model.id];
  if (schema?.properties?.return_last_frame && !boundary?.tailFrameOutput) {
    schema.properties.return_last_frame.enum = [false];
    delete schema.properties.return_last_frame.default;
  }
  if (schema?.properties?.output_format?.enum?.includes("svg") && !boundary?.svgOutput) {
    schema.properties.output_format.enum = schema.properties.output_format.enum.filter(value => value !== "svg");
  }
  if (schema && boundary) {
    schema.constraints ??= {};
    if (boundary.unsupportedInputs?.length) {
      schema.constraints.unsupported_inputs ??= {};
      for (const field of boundary.unsupportedInputs) {
        if (schema.properties) delete schema.properties[field];
        schema.constraints.unsupported_inputs[field] = true;
      }
    }
    if (boundary.unsupportedTransports?.length || boundary.localFileEncoding || boundary.localMaskEncoding) {
      for (const [name, operation] of Object.entries(schema.operations || {})) {
        for (const [field, input] of Object.entries(operation.inputs || {})) {
          input.transports = (input.transports || []).map(value => {
            if (name === "masked_image_edit" && field === "mask" && value === "data_url" && boundary.localMaskEncoding === "data_url") return "local_file_data_url";
            return value === "multipart_file" && boundary.localFileEncoding === "data_url" ? "local_file_data_url" : value;
          }).filter(value => !boundary.unsupportedTransports?.includes(value));
        }
      }
    }
    if (boundary.unsupportedOperations?.length) {
      schema.constraints.unsupported_operations ??= {};
      for (const operation of boundary.unsupportedOperations) {
        if (schema.operations) delete schema.operations[operation];
        schema.constraints.unsupported_operations[operation] = true;
      }
    }
    if (boundary.unsupportedParametersByOperation) {
      schema.constraints.unsupported_parameters_by_operation ??= {};
      for (const [operation, fields] of Object.entries(boundary.unsupportedParametersByOperation)) {
        schema.constraints.unsupported_parameters_by_operation[operation] = [
          ...new Set([...(schema.constraints.unsupported_parameters_by_operation[operation] || []), ...fields])
        ];
      }
    }
    if (boundary.klingNative) {
      schema.properties ??= {};
      for (const field of ["contents", "settings", "options"]) {
        schema.properties[field] = { type: "json" };
        if (schema.constraints.unsupported_inputs) delete schema.constraints.unsupported_inputs[field];
      }
      schema.operations ??= {};
      schema.operations.native_video = {
        request: { method: "POST", path: "/v1/videos", contentType: "application/json" },
        requiredBodyFields: ["model", "contents"]
      };
    }
  }
  if (!seedanceModels.has(model.id) || !schema ||
      schema.properties?.first_frame_image?.type !== "string" ||
      schema.properties?.last_frame_image?.type !== "string") return schema;
  schema.operations ??= {};
  const sets = {
    first_frame_video: ["first_frame_image"],
    last_frame_video: ["last_frame_image"],
    first_last_frame_video: ["first_frame_image", "last_frame_image"]
  };
  for (const [name, fields] of Object.entries(sets)) {
    // A live definition wins; only fill the documented missing operation.
    schema.operations[name] ??= {
      request: { method: "POST", path: "/v1/videos", contentType: "multipart/form-data" },
      requiredBodyFields: ["model", "prompt", ...fields],
      inputs: Object.fromEntries(fields.map(field => [field, {
        field, required: true, minItems: 1, maxItems: 1, transports: ["multipart_file"]
      }]))
    };
  }
  schema.constraints ??= {};
  schema.constraints.reference_transport ??= {
    first_frame_image: ["public_https_url"], last_frame_image: ["public_https_url"]
  };
  schema.constraints.exclusive_reference_sets ??= {
    frames: ["first_frame_image", "last_frame_image"],
    references: ["image", "reference_images", "reference_videos", "reference_audios"]
  };
  return schema;
}
