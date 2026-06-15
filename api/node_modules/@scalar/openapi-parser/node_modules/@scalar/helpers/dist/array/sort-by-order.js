function sortByOrder(arr, order, getId) {
  const orderMap = /* @__PURE__ */ new Map();
  order.forEach((e, idx) => orderMap.set(e, idx));
  const sorted = [];
  const untagged = [];
  arr.forEach((e) => {
    const sortedIdx = orderMap.get(getId(e));
    if (sortedIdx === void 0) {
      untagged.push(e);
      return;
    }
    sorted[sortedIdx] = e;
  });
  return [...sorted.filter((it) => it !== void 0), ...untagged];
}
export {
  sortByOrder
};
//# sourceMappingURL=sort-by-order.js.map
