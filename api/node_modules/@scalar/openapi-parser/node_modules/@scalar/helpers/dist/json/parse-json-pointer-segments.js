import { unescapeJsonPointer } from "./unescape-json-pointer.js";
const parseJsonPointerSegments = (path) => path.split("/").slice(1).map(unescapeJsonPointer);
export {
  parseJsonPointerSegments
};
//# sourceMappingURL=parse-json-pointer-segments.js.map
