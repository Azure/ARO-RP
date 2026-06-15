function getValueAtPath(obj, pointer) {
  return pointer.reduce((acc, part) => {
    if (acc === void 0 || acc === null) {
      return void 0;
    }
    return acc[part];
  }, obj);
}
export {
  getValueAtPath
};
//# sourceMappingURL=get-value-at-path.js.map
