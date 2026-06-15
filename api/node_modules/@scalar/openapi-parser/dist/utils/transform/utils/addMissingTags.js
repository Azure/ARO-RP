const addMissingTags = (definition) => {
  if (!definition.paths) {
    return definition;
  }
  const usedTags = /* @__PURE__ */ new Set();
  for (const path of Object.values(definition.paths)) {
    if (typeof path === "object" && path !== null) {
      for (const operation of Object.values(path)) {
        if (typeof operation === "object" && operation !== null && "tags" in operation) {
          const tags = operation.tags;
          if (Array.isArray(tags)) {
            tags.forEach((tag) => usedTags.add(String(tag)));
          }
        }
      }
    }
  }
  const existingTags = new Set(
    (Array.isArray(definition.tags) ? definition.tags : []).map((tag) => typeof tag === "object" && tag !== null ? String(tag.name) : null).filter(Boolean)
  );
  const missingTags = [...usedTags].filter((tag) => !existingTags.has(tag)).map((name) => ({ name }));
  return {
    ...definition,
    tags: [...Array.isArray(definition.tags) ? definition.tags : [], ...missingTags]
  };
};
export {
  addMissingTags
};
//# sourceMappingURL=addMissingTags.js.map
