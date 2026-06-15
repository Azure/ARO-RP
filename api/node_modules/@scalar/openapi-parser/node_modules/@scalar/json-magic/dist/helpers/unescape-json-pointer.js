function unescapeJsonPointer(uri) {
  return decodeURI(uri.replace(/~1/g, "/").replace(/~0/g, "~"));
}
export {
  unescapeJsonPointer
};
//# sourceMappingURL=unescape-json-pointer.js.map
