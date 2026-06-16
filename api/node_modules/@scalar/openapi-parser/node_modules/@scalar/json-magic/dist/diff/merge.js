import { Trie } from "../diff/trie.js";
import { isArrayEqual, isKeyCollisions, mergeObjects } from "../diff/utils.js";
const merge = (diff1, diff2) => {
  const trie = new Trie();
  for (const [index, diff] of diff1.entries()) {
    trie.addPath(diff.path, { index, changes: diff });
  }
  const skipDiff1 = /* @__PURE__ */ new Set();
  const skipDiff2 = /* @__PURE__ */ new Set();
  const conflictsMap1 = /* @__PURE__ */ new Map();
  const conflictsMap2 = /* @__PURE__ */ new Map();
  for (const [index, diff] of diff2.entries()) {
    trie.findMatch(diff.path, (value) => {
      if (diff.type === "delete") {
        if (value.changes.type === "delete") {
          if (value.changes.path.length > diff.path.length) {
            skipDiff1.add(value.index);
          } else {
            skipDiff2.add(value.index);
          }
        } else {
          skipDiff1.add(value.index);
          skipDiff2.add(index);
          const conflictEntry = conflictsMap2.get(index);
          if (conflictEntry !== void 0) {
            conflictEntry[0].push(value.changes);
          } else {
            conflictsMap2.set(index, [[value.changes], [diff]]);
          }
        }
      }
      if (diff.type === "add" || diff.type === "update") {
        if (isArrayEqual(diff.path, value.changes.path) && value.changes.type !== "delete" && !isKeyCollisions(diff.changes, value.changes.changes)) {
          skipDiff1.add(value.index);
          if (typeof diff.changes === "object") {
            mergeObjects(diff.changes, value.changes.changes);
          }
          return;
        }
        skipDiff1.add(value.index);
        skipDiff2.add(index);
        const conflictEntry = conflictsMap1.get(value.index);
        if (conflictEntry !== void 0) {
          conflictEntry[1].push(diff);
        } else {
          conflictsMap1.set(value.index, [[value.changes], [diff]]);
        }
      }
    });
  }
  const conflicts = [...conflictsMap1.values(), ...conflictsMap2.values()];
  const diffs = [
    ...diff1.filter((_, index) => !skipDiff1.has(index)),
    ...diff2.filter((_, index) => !skipDiff2.has(index))
  ];
  return { diffs, conflicts };
};
export {
  merge
};
//# sourceMappingURL=merge.js.map
