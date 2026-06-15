import { parse } from "yaml";
function normalize(content) {
  if (content === null) {
    return void 0;
  }
  if (typeof content === "string") {
    if (content.trim() === "") {
      return void 0;
    }
    try {
      return JSON.parse(content);
    } catch (_error) {
      const hasColon = /^[^:]+:/.test(content);
      const isJson = content.slice(0, 50).trimStart().startsWith("{");
      if (!hasColon || isJson) {
        return void 0;
      }
      return parse(content, {
        maxAliasCount: 1e4,
        merge: true
      });
    }
  }
  return content;
}
export {
  normalize
};
//# sourceMappingURL=normalize.js.map
