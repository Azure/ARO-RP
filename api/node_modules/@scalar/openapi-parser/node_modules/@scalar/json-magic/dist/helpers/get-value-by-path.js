import { getId } from "../helpers/get-schemas.js";
function getValueByPath(target, segments) {
  return segments.reduce(
    (acc, key) => {
      if (acc.value === void 0) {
        return { context: "", value: void 0 };
      }
      if (typeof acc.value !== "object" || acc.value === null) {
        return { context: "", value: void 0 };
      }
      const id = getId(acc.value);
      return { context: id ?? acc.context, value: acc.value?.[key] };
    },
    {
      context: "",
      value: target
    }
  );
}
export {
  getValueByPath
};
//# sourceMappingURL=get-value-by-path.js.map
