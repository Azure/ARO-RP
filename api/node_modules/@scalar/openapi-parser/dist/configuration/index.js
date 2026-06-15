import Swagger20 from "../schemas/v2.0/schema.js";
import OpenApi30 from "../schemas/v3.0/schema.js";
import OpenApi31 from "../schemas/v3.1/schema.js";
import OpenApi32 from "../schemas/v3.2/schema.js";
const OpenApiSpecifications = {
  "2.0": Swagger20,
  "3.0": OpenApi30,
  "3.1": OpenApi31,
  "3.2": OpenApi32
};
const OpenApiVersions = Object.keys(OpenApiSpecifications);
const ERRORS = {
  EMPTY_OR_INVALID: "Can't find JSON, YAML or filename in data.",
  OPENAPI_VERSION_NOT_SUPPORTED: "Can't find supported Swagger/OpenAPI version in the provided document, version must be a string.",
  INVALID_REFERENCE: "Can't resolve reference: %s",
  EXTERNAL_REFERENCE_NOT_FOUND: "Can't resolve external reference: %s",
  SELF_REFERENCE: "Can't resolve reference to itself: %s",
  FILE_DOES_NOT_EXIST: "File does not exist: %s",
  NO_CONTENT: "No content found"
};
export {
  ERRORS,
  OpenApiSpecifications,
  OpenApiVersions
};
//# sourceMappingURL=index.js.map
