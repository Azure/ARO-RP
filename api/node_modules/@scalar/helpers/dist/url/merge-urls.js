import { REGEX } from "../regex/regex-helpers.js";
import { isRelativePath } from "./is-relative-path.js";
import { ensureProtocol } from "./ensure-protocol.js";
const mergeSearchParams = (...params) => {
  const merged = {};
  params.forEach((p) => {
    const keys = Array.from(p.keys());
    const uniqueKeys = new Set(keys);
    uniqueKeys.forEach((key) => {
      const values = p.getAll(key);
      const value = values.length > 1 ? values : values[0] ?? "";
      merged[key] = value;
    });
  });
  const result = new URLSearchParams();
  Object.entries(merged).forEach(([key, value]) => {
    if (Array.isArray(value)) {
      value.forEach((v) => result.append(key, v));
    } else {
      result.append(key, value);
    }
  });
  return result;
};
const combineUrlAndPath = (url, path) => {
  if (!path || url === path) {
    return url.trim();
  }
  if (!url) {
    return path.trim();
  }
  return `${url.trim()}/${path.trim()}`.replace(REGEX.MULTIPLE_SLASHES, "/");
};
const mergeUrls = (url, path, urlParams = new URLSearchParams(), disableOriginPrefix = false) => {
  if (url && (!isRelativePath(url) || typeof window !== "undefined")) {
    const base = disableOriginPrefix ? url : isRelativePath(url) ? combineUrlAndPath(window.location.origin, url) : ensureProtocol(url);
    const [baseUrl = "", baseQuery] = base.split("?");
    const baseParams = new URLSearchParams(baseQuery || "");
    const [pathWithoutQuery = "", pathQuery] = path.split("?");
    const pathParams = new URLSearchParams(pathQuery || "");
    const mergedUrl = url === path ? baseUrl : combineUrlAndPath(baseUrl, pathWithoutQuery);
    const mergedSearchParams = mergeSearchParams(baseParams, pathParams, urlParams);
    const search = mergedSearchParams.toString();
    return search ? `${mergedUrl}?${search}` : mergedUrl;
  }
  if (path) {
    return combineUrlAndPath(url, path);
  }
  return "";
};
export {
  combineUrlAndPath,
  mergeSearchParams,
  mergeUrls
};
//# sourceMappingURL=merge-urls.js.map
