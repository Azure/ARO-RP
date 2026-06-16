function parseJsonPointer(pointer) {
  return pointer.split("/").filter((segment, index) => (index !== 0 || segment !== "#") && segment);
}
function createPathFromSegments(obj, segments) {
  return segments.reduce((acc, part) => {
    if (acc[part] === void 0) {
      if (isNaN(Number(part))) {
        acc[part] = {};
      } else {
        acc[part] = [];
      }
    }
    return acc[part];
  }, obj);
}
export {
  createPathFromSegments,
  parseJsonPointer
};
//# sourceMappingURL=json-path-utils.js.map
