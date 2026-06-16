import { details } from "./details.js";
import { getEntrypoint } from "./get-entrypoint.js";
import { makeFilesystem } from "./make-filesystem.js";
import { resolveReferences } from "./resolve-references.js";
function dereference(value, options) {
  const filesystem = makeFilesystem(value);
  const entrypoint = getEntrypoint(filesystem);
  const result = resolveReferences(filesystem, options);
  return {
    specification: entrypoint.specification,
    errors: result.errors,
    schema: result.schema,
    ...details(entrypoint.specification)
  };
}
export {
  dereference
};
//# sourceMappingURL=dereference.js.map
