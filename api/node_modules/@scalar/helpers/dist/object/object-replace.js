const objectReplace = (target, replacement) => {
  Object.keys(target).forEach((key) => {
    if (!Object.hasOwn(replacement, key)) {
      delete target[key];
    }
  });
  Object.assign(target, replacement);
  return target;
};
export {
  objectReplace
};
//# sourceMappingURL=object-replace.js.map
