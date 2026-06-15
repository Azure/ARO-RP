const truncate = (str, maxLength = 18) => {
  if (str.length <= maxLength) {
    return str;
  }
  return str.slice(0, maxLength) + "\u2026";
};
export {
  truncate
};
//# sourceMappingURL=truncate.js.map
