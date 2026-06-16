import { traverse } from "./traverse.js";
function getListOfReferences(specification) {
  const references = [];
  if (!specification || typeof specification !== "object") {
    return references;
  }
  traverse(specification, (value) => {
    if (value.$ref && typeof value.$ref === "string" && !value.$ref.startsWith("#")) {
      references.push(value.$ref.split("#")[0]);
    }
    return value;
  });
  return [...new Set(references)];
}
export {
  getListOfReferences
};
//# sourceMappingURL=get-list-of-references.js.map
