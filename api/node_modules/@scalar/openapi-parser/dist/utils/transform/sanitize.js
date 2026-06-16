import { addInfoObject } from "./utils/addInfoObject.js";
import { addLatestOpenApiVersion } from "./utils/addLatestOpenApiVersion.js";
import { addMissingTags } from "./utils/addMissingTags.js";
import { normalizeSecuritySchemes } from "./utils/normalizeSecuritySchemes.js";
import { rejectSwaggerDocuments } from "./utils/rejectSwaggerDocuments.js";
import { DEFAULT_TITLE } from "./utils/addInfoObject.js";
import { DEFAULT_OPENAPI_VERSION } from "./utils/addLatestOpenApiVersion.js";
function sanitize(definition) {
  const transformers = [
    rejectSwaggerDocuments,
    addLatestOpenApiVersion,
    addInfoObject,
    addMissingTags,
    normalizeSecuritySchemes
  ];
  return transformers.reduce((doc, transformer) => transformer(doc), definition);
}
export {
  DEFAULT_OPENAPI_VERSION,
  DEFAULT_TITLE,
  sanitize
};
//# sourceMappingURL=sanitize.js.map
