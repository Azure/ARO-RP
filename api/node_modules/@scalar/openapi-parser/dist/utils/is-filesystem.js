function isFilesystem(value) {
  return typeof value !== "undefined" && Array.isArray(value) && value.length > 0 && value.some((file) => file.isEntrypoint === true);
}
export {
  isFilesystem
};
//# sourceMappingURL=is-filesystem.js.map
