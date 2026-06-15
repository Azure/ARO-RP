import { isObject } from "@scalar/helpers/object/is-object";
import { OpenApiVersions } from "../configuration/index.js";
function details(specification) {
  if (specification === null) {
    return {
      version: void 0,
      specificationType: void 0,
      specificationVersion: void 0
    };
  }
  if (isObject(specification)) {
    for (const version of new Set(OpenApiVersions)) {
      const specificationType = version === "2.0" ? "swagger" : "openapi";
      const value = specification[specificationType];
      if (typeof value === "string" && value.startsWith(version)) {
        return {
          version,
          specificationType,
          specificationVersion: value
        };
      }
    }
  }
  return {
    version: void 0,
    specificationType: void 0,
    specificationVersion: void 0
  };
}
export {
  details
};
//# sourceMappingURL=details.js.map
