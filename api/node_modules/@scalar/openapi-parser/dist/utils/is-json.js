function isJson(value) {
  try {
    JSON.parse(value);
    return true;
  } catch {
    return false;
  }
}
export {
  isJson
};
//# sourceMappingURL=is-json.js.map
