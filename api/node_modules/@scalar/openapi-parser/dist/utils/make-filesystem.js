import { getListOfReferences } from "./get-list-of-references.js";
import { isFilesystem } from "./is-filesystem.js";
import { normalize } from "./normalize.js";
function makeFilesystem(value, overwrites = {}) {
  if (isFilesystem(value)) {
    return value;
  }
  const specification = normalize(value);
  return [
    {
      isEntrypoint: true,
      specification,
      filename: null,
      dir: "./",
      references: getListOfReferences(specification),
      ...overwrites
    }
  ];
}
export {
  makeFilesystem
};
//# sourceMappingURL=make-filesystem.js.map
