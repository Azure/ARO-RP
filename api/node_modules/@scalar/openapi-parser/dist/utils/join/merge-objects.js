const mergeObjects = (a, b) => {
  for (const key in b) {
    if (!(key in a)) {
      a[key] = b[key];
    } else {
      const aValue = a[key];
      const bValue = b[key];
      if (typeof aValue === "object" && aValue !== null && typeof bValue === "object" && bValue !== null) {
        mergeObjects(aValue, bValue);
      } else {
        a[key] = bValue;
      }
    }
  }
  return a;
};
export {
  mergeObjects
};
//# sourceMappingURL=merge-objects.js.map
