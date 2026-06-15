const isKeyCollisions = (a, b) => {
  if (typeof a !== typeof b) {
    return true;
  }
  if (typeof a === "object" && typeof b === "object" && a !== null && b !== null) {
    const keys = /* @__PURE__ */ new Set([...Object.keys(a), ...Object.keys(b)]);
    for (const key of keys) {
      if (a[key] !== void 0 && b[key] !== void 0) {
        if (isKeyCollisions(a[key], b[key])) {
          return true;
        }
      }
    }
    return false;
  }
  return a !== b;
};
const mergeObjects = (a, b) => {
  for (const key in b) {
    if (!(key in a)) {
      a[key] = b[key];
    } else {
      const aValue = a[key];
      const bValue = b[key];
      if (typeof aValue === "object" && aValue !== null && typeof bValue === "object" && bValue !== null) {
        a[key] = mergeObjects(aValue, bValue);
      }
    }
  }
  return a;
};
const isArrayEqual = (a, b) => {
  if (a.length !== b.length) {
    return false;
  }
  for (let i = 0; i <= a.length; ++i) {
    if (a[i] !== b[i]) {
      return false;
    }
  }
  return true;
};
export {
  isArrayEqual,
  isKeyCollisions,
  mergeObjects
};
//# sourceMappingURL=utils.js.map
