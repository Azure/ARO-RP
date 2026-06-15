const getId = (input) => {
  if (input && typeof input === "object" && input["$id"] && typeof input["$id"] === "string") {
    return input["$id"];
  }
  return void 0;
};
const getPath = (segments) => {
  return segments.join("/");
};
const getSchemas = (input, base = "", segments = [], map = /* @__PURE__ */ new Map(), visited = /* @__PURE__ */ new WeakSet()) => {
  if (typeof input !== "object" || input === null) {
    return map;
  }
  if (visited.has(input)) {
    return map;
  }
  visited.add(input);
  const id = getId(input);
  if (id) {
    map.set(id, getPath(segments));
  }
  const newBase = id ?? base;
  if (input["$anchor"] && typeof input["$anchor"] === "string") {
    map.set(`${newBase}#${input["$anchor"]}`, getPath(segments));
  }
  for (const key in input) {
    if (typeof input[key] === "object" && input[key] !== null) {
      getSchemas(input[key], newBase, [...segments, key], map, visited);
    }
  }
  return map;
};
export {
  getId,
  getSchemas
};
//# sourceMappingURL=get-schemas.js.map
