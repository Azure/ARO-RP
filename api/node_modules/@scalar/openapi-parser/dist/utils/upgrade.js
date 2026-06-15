import { upgrade as originalUpgrade } from "@scalar/openapi-upgrader";
import { getEntrypoint } from "./get-entrypoint.js";
import { isFilesystem } from "./is-filesystem.js";
import { normalize } from "./normalize.js";
function upgrade(value) {
  if (!value) {
    return {
      specification: null,
      version: "3.1"
    };
  }
  const document = originalUpgrade(
    isFilesystem(value) ? getEntrypoint(value).specification : normalize(value),
    "3.1"
  );
  return {
    specification: document,
    // TODO: Make dynamic
    version: "3.1"
  };
}
export {
  upgrade
};
//# sourceMappingURL=upgrade.js.map
