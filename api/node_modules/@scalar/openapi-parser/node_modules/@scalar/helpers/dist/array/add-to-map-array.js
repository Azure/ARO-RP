const addToMapArray = (map, key, value) => {
  const prev = map.get(key) ?? [];
  prev.push(value);
  map.set(key, prev);
};
export {
  addToMapArray
};
//# sourceMappingURL=add-to-map-array.js.map
