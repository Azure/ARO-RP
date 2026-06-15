import { unescapeJsonPointer } from "./unescape-json-pointer.js";
function getSegmentsFromPath(path) {
  return (
    // /paths/~1test
    path.split("/").slice(1).map(unescapeJsonPointer)
  );
}
export {
  getSegmentsFromPath
};
//# sourceMappingURL=get-segments-from-path.js.map
