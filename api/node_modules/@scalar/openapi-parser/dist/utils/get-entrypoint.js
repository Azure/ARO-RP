function getEntrypoint(filesystem) {
  return filesystem?.find((file) => file.isEntrypoint);
}
export {
  getEntrypoint
};
//# sourceMappingURL=get-entrypoint.js.map
