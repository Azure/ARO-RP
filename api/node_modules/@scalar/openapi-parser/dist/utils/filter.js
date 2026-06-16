import { getEntrypoint } from "./get-entrypoint.js";
import { makeFilesystem } from "./make-filesystem.js";
import { traverse } from "./traverse.js";
function filter(specification, callback) {
  const filesystem = makeFilesystem(specification);
  return {
    specification: traverse(getEntrypoint(filesystem).specification, (schema) => {
      return callback(schema) ? schema : void 0;
    })
  };
}
export {
  filter
};
//# sourceMappingURL=filter.js.map
