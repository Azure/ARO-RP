const omitUndefinedValues = (data) => {
  if (Array.isArray(data)) {
    return data.map(
      (item) => typeof item === "object" && item !== null ? omitUndefinedValues(item) : item
    );
  }
  return Object.fromEntries(
    Object.entries(data).filter(([_, value]) => value !== void 0).map(([key, value]) => {
      if (typeof value === "object" && value !== null) {
        return [key, omitUndefinedValues(value)];
      }
      return [key, value];
    })
  );
};
export {
  omitUndefinedValues
};
//# sourceMappingURL=omit-undefined-values.js.map
