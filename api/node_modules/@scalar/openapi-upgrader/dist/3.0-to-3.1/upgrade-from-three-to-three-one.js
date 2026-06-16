import { traverse } from "../helpers/traverse.js";
const SCHEMA_SEGMENTS = /* @__PURE__ */ new Set([
  "properties",
  "items",
  "allOf",
  "anyOf",
  "oneOf",
  "not",
  "additionalProperties",
  "schema"
]);
function isSchemaPath(path) {
  if (!path) {
    return false;
  }
  if (path.some((segment) => SCHEMA_SEGMENTS.has(segment))) {
    return true;
  }
  if (path.some((segment) => segment.endsWith("Schema"))) {
    return true;
  }
  if (path.length >= 2 && path[0] === "components" && path[1] === "schemas") {
    return true;
  }
  return false;
}
function upgradeFromThreeToThreeOne(originalContent) {
  let content = originalContent;
  if (content === null || typeof content.openapi !== "string" || !content.openapi.startsWith("3.0")) {
    return content;
  }
  content.openapi = "3.1.1";
  content = traverse(content, applyChangesToDocument);
  return content;
}
const applyChangesToDocument = (schema, path) => {
  if (schema.type !== void 0 && schema.nullable === true) {
    schema.type = [schema.type, "null"];
    delete schema.nullable;
  }
  if (schema.exclusiveMinimum === true) {
    schema.exclusiveMinimum = schema.minimum;
    delete schema.minimum;
  } else if (schema.exclusiveMinimum === false) {
    delete schema.exclusiveMinimum;
  }
  if (schema.exclusiveMaximum === true) {
    schema.exclusiveMaximum = schema.maximum;
    delete schema.maximum;
  } else if (schema.exclusiveMaximum === false) {
    delete schema.exclusiveMaximum;
  }
  const isInsideExamplesMap = path?.some((segment, index) => {
    if (segment === "examples" && index > 0) {
      const parent = path[index - 1];
      return parent !== "properties";
    }
    return false;
  });
  if (schema.example !== void 0 && !isInsideExamplesMap) {
    if (isSchemaPath(path)) {
      schema.examples = [schema.example];
    } else {
      schema.examples = {
        default: {
          value: schema.example
        }
      };
    }
    delete schema.example;
  }
  if (schema.type === "object" && schema.properties !== void 0) {
    const parentPath = path?.slice(0, -1);
    const isMultipart = parentPath?.some((segment, index) => {
      return segment === "content" && path?.[index + 1] === "multipart/form-data";
    });
    if (isMultipart && schema.properties !== null) {
      for (const value of Object.values(schema.properties)) {
        if (typeof value === "object" && value !== null && "type" in value && "format" in value && value.type === "string" && value.format === "binary") {
          value.contentMediaType = "application/octet-stream";
          delete value.format;
        }
      }
    }
  }
  if (path?.includes("content") && path?.includes("application/octet-stream")) {
    return {};
  }
  const { format: _, ...rest } = schema;
  if (schema.type === "string") {
    if (schema.format === "binary") {
      return {
        ...rest,
        type: "string",
        contentMediaType: "application/octet-stream"
      };
    }
    if (schema.format === "base64") {
      return {
        ...rest,
        type: "string",
        contentEncoding: "base64"
      };
    }
    if (schema.format === "byte") {
      const parentPath = path?.slice(0, -1);
      const contentMediaType = parentPath?.find((_2, index) => path?.[index - 1] === "content");
      return {
        ...rest,
        type: "string",
        contentEncoding: "base64",
        contentMediaType
      };
    }
  }
  if (schema["x-webhooks"] !== void 0) {
    schema.webhooks = schema["x-webhooks"];
    delete schema["x-webhooks"];
  }
  return schema;
};
export {
  isSchemaPath,
  upgradeFromThreeToThreeOne
};
//# sourceMappingURL=upgrade-from-three-to-three-one.js.map
