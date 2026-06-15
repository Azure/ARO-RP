function traverse(content, transform, path = []) {
  const result = {};
  for (const [key, value] of Object.entries(content)) {
    const currentPath = [...path, key];
    if (Array.isArray(value)) {
      result[key] = value.map((item, index) => {
        if (typeof item === "object" && !Array.isArray(item) && item !== null) {
          return traverse(item, transform, [...currentPath, index.toString()]);
        }
        return item;
      });
      continue;
    }
    if (typeof value === "object" && value !== null) {
      result[key] = traverse(value, transform, currentPath);
      continue;
    }
    result[key] = value;
  }
  return transform(result, path);
}
export {
  traverse
};
//# sourceMappingURL=traverse.js.map
