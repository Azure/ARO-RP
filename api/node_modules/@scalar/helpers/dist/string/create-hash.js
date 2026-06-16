const createHash = (input) => {
  let chr = 0;
  let hash = 0;
  let i = 0;
  if (!input?.length) {
    return hash;
  }
  for (i = 0; i < input.length; i++) {
    chr = input.charCodeAt(i);
    hash = (hash << 5) - hash + chr;
    hash |= 0;
  }
  return hash;
};
export {
  createHash
};
//# sourceMappingURL=create-hash.js.map
