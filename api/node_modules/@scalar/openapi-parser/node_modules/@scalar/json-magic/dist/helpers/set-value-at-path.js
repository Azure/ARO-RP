import { preventPollution } from "@scalar/helpers/object/prevent-pollution";
import { getSegmentsFromPath } from "../helpers/get-segments-from-path.js";
function setValueAtPath(obj, path, value) {
  if (path === "") {
    throw new Error("Cannot set value at root ('') pointer");
  }
  const parts = getSegmentsFromPath(path);
  parts.forEach((part) => preventPollution(part));
  let current = obj;
  for (let i = 0; i < parts.length; i++) {
    const key = parts[i];
    const isLast = i === parts.length - 1;
    const nextKey = parts[i + 1];
    const shouldBeArray = /^\d+$/.test(nextKey ?? "");
    if (isLast) {
      current[key] = value;
    } else {
      if (!(key in current) || typeof current[key] !== "object") {
        current[key] = shouldBeArray ? [] : {};
      }
      current = current[key];
    }
  }
}
export {
  setValueAtPath
};
//# sourceMappingURL=set-value-at-path.js.map
