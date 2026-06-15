const isDereferenced = (obj) => typeof obj === "object" && obj !== null && !("$ref" in obj && typeof obj.$ref === "string");
export {
  isDereferenced
};
//# sourceMappingURL=is-dereferenced.js.map
