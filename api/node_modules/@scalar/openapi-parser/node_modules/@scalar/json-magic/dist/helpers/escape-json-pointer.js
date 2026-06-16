function escapeJsonPointer(str) {
  return str.replace(/~/g, "~0").replace(/\//g, "~1");
}
export {
  escapeJsonPointer
};
//# sourceMappingURL=escape-json-pointer.js.map
