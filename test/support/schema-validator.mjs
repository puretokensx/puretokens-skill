import { isDeepStrictEqual } from "node:util";

// Repository-test evaluator for the explicitly supported draft 2020-12
// vocabulary below. This is not a general-purpose JSON Schema implementation.
// Every schema branch is compiled before validating data, and unknown keywords,
// formats and references fail the test instead of silently skipping constraints.
const annotations = new Set(["$schema", "$id", "$comment", "title", "description", "default", "examples"]);
const keywords = new Set([
  ...annotations, "$ref", "$defs", "type", "const", "enum", "required", "properties",
  "patternProperties", "additionalProperties", "items", "minItems", "maxItems",
  "uniqueItems", "minLength", "maxLength", "minProperties", "maxProperties", "pattern", "format", "minimum", "maximum",
  "exclusiveMinimum", "exclusiveMaximum", "multipleOf", "anyOf", "oneOf", "allOf", "not", "if", "then", "else"
]);
const types = {
  object: (value) => value !== null && typeof value === "object" && !Array.isArray(value),
  array: Array.isArray,
  string: (value) => typeof value === "string",
  boolean: (value) => typeof value === "boolean",
  null: (value) => value === null,
  number: (value) => typeof value === "number" && Number.isFinite(value),
  integer: (value) => Number.isInteger(value)
};

function dateTime(value) {
  const parts = /^(\d{4})-(\d{2})-(\d{2})[Tt](\d{2}):(\d{2}):(\d{2})(?:\.\d+)?(?:[Zz]|([+-])(\d{2}):(\d{2}))$/.exec(value);
  if (!parts) return false;
  const [, year, month, day, hour, minute, second, , offsetHour = "0", offsetMinute = "0"] = parts;
  const days = new Date(Date.UTC(Number(year), Number(month), 0)).getUTCDate();
  return Number(month) >= 1 && Number(month) <= 12 && Number(day) >= 1 && Number(day) <= days &&
    Number(hour) <= 23 && Number(minute) <= 59 && Number(second) <= 60 &&
    Number(offsetHour) <= 23 && Number(offsetMinute) <= 59;
}

export function compileSchema(schema, documents) {
  const validators = new WeakMap();
  const registry = new Map([...documents].map((document) => [document.$id, document]));

  function compile(rule, document) {
    if (typeof rule === "boolean") return (value, at) => rule ? [] : [`${at}: false schema`];
    if (!types.object(rule)) throw new Error("Schema must be an object or boolean");
    if (validators.has(rule)) return validators.get(rule);
    for (const key of Object.keys(rule)) {
      if (!keywords.has(key)) throw new Error(`Unsupported schema keyword: ${key}`);
    }
    if (rule.$schema && rule.$schema !== "https://json-schema.org/draft/2020-12/schema") {
      throw new Error(`Unsupported schema dialect: ${rule.$schema}`);
    }
    if (rule.type !== undefined && !(Array.isArray(rule.type) ? rule.type : [rule.type]).every((type) => Object.hasOwn(types, type))) {
      throw new Error(`Unsupported schema type: ${rule.type}`);
    }
    if (rule.format !== undefined && rule.format !== "date-time") {
      throw new Error(`Unsupported schema format: ${rule.format}`);
    }
    if (rule.multipleOf !== undefined && (!types.number(rule.multipleOf) || rule.multipleOf <= 0)) {
      throw new Error("multipleOf must be a positive finite number");
    }
    let evaluate;
    const validator = (value, at = "$") => evaluate(value, at);
    validators.set(rule, validator);
    const properties = Object.entries(rule.properties ?? {}).map(([key, child]) => [key, compile(child, document)]);
    const patterns = Object.entries(rule.patternProperties ?? {}).map(([key, child]) => [new RegExp(key, "u"), compile(child, document)]);
    for (const child of Object.values(rule.$defs ?? {})) compile(child, document);
    const extra = rule.additionalProperties === undefined ? undefined : compile(rule.additionalProperties, document);
    const item = rule.items === undefined ? undefined : compile(rule.items, document);
    const pattern = rule.pattern === undefined ? undefined : new RegExp(rule.pattern, "u");
    const alternatives = Object.fromEntries(["anyOf", "oneOf", "allOf"].map((key) => {
      if (rule[key] !== undefined && (!Array.isArray(rule[key]) || rule[key].length === 0)) throw new Error(`${key} must have schema branches`);
      return [key, rule[key]?.map((child) => compile(child, document))];
    }));
    const conditional = Object.fromEntries(["if", "then", "else", "not"].map((key) =>
      [key, rule[key] === undefined ? undefined : compile(rule[key], document)]));
    let referenced;
    if (rule.$ref !== undefined) {
      const url = new URL(rule.$ref, document.$id);
      const fragment = url.hash.slice(1);
      url.hash = "";
      const target = registry.get(url.href);
      if (!target) throw new Error(`Unregistered schema reference: ${rule.$ref}`);
      let child = target;
      if (fragment) {
        if (!fragment.startsWith("/")) throw new Error(`Unsupported schema anchor: ${fragment}`);
        for (const part of decodeURIComponent(fragment).slice(1).split("/")) {
          const key = part.replace(/~1/g, "/").replace(/~0/g, "~");
          if (!types.object(child) || !Object.hasOwn(child, key)) throw new Error(`Unresolved schema reference: ${rule.$ref}`);
          child = child[key];
        }
      }
      referenced = compile(child, target);
    }

    evaluate = (value, at) => {
      const errors = referenced ? referenced(value, at) : [];
      const fail = (keyword) => errors.push(`${at}: ${keyword}`);
      if (rule.type !== undefined && !(Array.isArray(rule.type) ? rule.type : [rule.type]).some((type) => types[type](value))) fail("type");
      if (Object.hasOwn(rule, "const") && !isDeepStrictEqual(value, rule.const)) fail("const");
      if (rule.enum && !rule.enum.some((option) => isDeepStrictEqual(value, option))) fail("enum");
      if (types.number(value)) {
        if (rule.multipleOf !== undefined) {
          const quotient = value / rule.multipleOf;
          if (!Number.isFinite(quotient) || Math.abs(quotient - Math.round(quotient)) > Number.EPSILON * Math.max(1, Math.abs(quotient)) * 8) fail("multipleOf");
        }
        if (rule.minimum !== undefined && value < rule.minimum) fail("minimum");
        if (rule.maximum !== undefined && value > rule.maximum) fail("maximum");
        if (rule.exclusiveMinimum !== undefined && value <= rule.exclusiveMinimum) fail("exclusiveMinimum");
        if (rule.exclusiveMaximum !== undefined && value >= rule.exclusiveMaximum) fail("exclusiveMaximum");
      }
      if (typeof value === "string") {
        const length = [...value].length;
        if (rule.minLength !== undefined && length < rule.minLength) fail("minLength");
        if (rule.maxLength !== undefined && length > rule.maxLength) fail("maxLength");
        if (pattern && !pattern.test(value)) fail("pattern");
        if (rule.format === "date-time" && !dateTime(value)) fail("format date-time");
      }
      if (Array.isArray(value)) {
        if (rule.minItems !== undefined && value.length < rule.minItems) fail("minItems");
        if (rule.maxItems !== undefined && value.length > rule.maxItems) fail("maxItems");
        if (rule.uniqueItems && value.some((entry, index) => value.slice(0, index).some((other) => isDeepStrictEqual(entry, other)))) fail("uniqueItems");
        if (item) value.forEach((entry, index) => errors.push(...item(entry, `${at}/${index}`)));
      }
      if (types.object(value)) {
        if (rule.minProperties !== undefined && Object.keys(value).length < rule.minProperties) fail("minProperties");
        if (rule.maxProperties !== undefined && Object.keys(value).length > rule.maxProperties) fail("maxProperties");
        for (const key of rule.required ?? []) if (!Object.hasOwn(value, key)) fail(`required ${key}`);
        for (const [key, child] of properties) if (Object.hasOwn(value, key)) errors.push(...child(value[key], `${at}/${key}`));
        for (const [key, entry] of Object.entries(value)) {
          let matched = properties.some(([name]) => name === key);
          for (const [regex, child] of patterns) {
            if (regex.test(key)) {
              matched = true;
              errors.push(...child(entry, `${at}/${key}`));
            }
          }
          if (!matched && extra) errors.push(...extra(entry, `${at}/${key}`));
        }
      }
      if (alternatives.allOf) for (const child of alternatives.allOf) errors.push(...child(value, at));
      if (alternatives.anyOf && !alternatives.anyOf.some((child) => child(value, at).length === 0)) fail("anyOf");
      if (alternatives.oneOf && alternatives.oneOf.filter((child) => child(value, at).length === 0).length !== 1) fail("oneOf");
      if (conditional.not && conditional.not(value, at).length === 0) fail("not");
      if (conditional.if) {
        const branch = conditional.if(value, at).length === 0 ? conditional.then : conditional.else;
        if (branch) errors.push(...branch(value, at));
      }
      return errors;
    };
    return validator;
  }

  return compile(schema, schema);
}
