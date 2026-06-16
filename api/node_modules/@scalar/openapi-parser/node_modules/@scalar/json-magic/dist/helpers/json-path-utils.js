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
  createPathFromSegments
};
//# sourceMappingURL=json-path-utils.js.map
