import { escapeJsonPointer } from "../json/escape-json-pointer.js";
import { Queue } from "../queue/queue.js";
const toJsonCompatible = (obj, options = {}) => {
  const { prefix = "", cache = /* @__PURE__ */ new WeakMap() } = options;
  const toRef = (path) => ({ $ref: `#${path ?? ""}` });
  if (typeof obj !== "object" || obj === null) {
    return obj;
  }
  const rootPath = prefix;
  cache.set(obj, rootPath);
  const rootResult = Array.isArray(obj) ? new Array(obj.length) : {};
  const queue = new Queue();
  queue.enqueue({ node: obj, result: rootResult, path: rootPath });
  while (!queue.isEmpty()) {
    const frame = queue.dequeue();
    if (!frame) {
      continue;
    }
    const { node, result, path } = frame;
    if (Array.isArray(node)) {
      const input = node;
      const out2 = result;
      for (let index = 0; index < input.length; index++) {
        if (!(index in input)) {
          continue;
        }
        const item = input[index];
        const itemPath = `${path}/${index}`;
        if (typeof item !== "object" || item === null) {
          out2[index] = item;
          continue;
        }
        const existingPath = cache.get(item);
        if (existingPath !== void 0) {
          out2[index] = toRef(existingPath);
          continue;
        }
        cache.set(item, itemPath);
        const childResult = Array.isArray(item) ? new Array(item.length) : {};
        out2[index] = childResult;
        queue.enqueue({ node: item, result: childResult, path: itemPath });
      }
      continue;
    }
    const out = result;
    for (const [key, value] of Object.entries(node)) {
      const valuePath = `${path}/${escapeJsonPointer(key)}`;
      if (typeof value !== "object" || value === null) {
        out[key] = value;
        continue;
      }
      const existingPath = cache.get(value);
      if (existingPath !== void 0) {
        out[key] = toRef(existingPath);
        continue;
      }
      cache.set(value, valuePath);
      const childResult = Array.isArray(value) ? new Array(value.length) : {};
      out[key] = childResult;
      queue.enqueue({ node: value, result: childResult, path: valuePath });
    }
  }
  return rootResult;
};
export {
  toJsonCompatible
};
//# sourceMappingURL=to-json-compatible.js.map
