const isObject = (value) => {
  if (value === null || typeof value !== "object") {
    return false;
  }
  const proto = Object.getPrototypeOf(value);
  return proto === Object.prototype || proto === null;
};
export {
  isObject
};
//# sourceMappingURL=is-object.js.map
