const diff = (doc1, doc2) => {
  const diff2 = [];
  const bfs = (el1, el2, prefix = []) => {
    if (typeof el1 !== typeof el2) {
      if (typeof el1 === "undefined") {
        diff2.push({ path: prefix, changes: el2, type: "add" });
        return;
      }
      if (typeof el2 === "undefined") {
        diff2.push({ path: prefix, changes: el1, type: "delete" });
        return;
      }
      diff2.push({ path: prefix, changes: el2, type: "update" });
      return;
    }
    if (typeof el1 === "object" && typeof el2 === "object" && el1 !== null && el2 !== null) {
      const keys = /* @__PURE__ */ new Set([...Object.keys(el1), ...Object.keys(el2)]);
      for (const key of keys) {
        bfs(el1[key], el2[key], [...prefix, key]);
      }
      return;
    }
    if (el1 !== el2) {
      diff2.push({ path: prefix, changes: el2, type: "update" });
    }
  };
  bfs(doc1, doc2);
  return diff2;
};
export {
  diff
};
//# sourceMappingURL=diff.js.map
