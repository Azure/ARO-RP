import { combineUrlAndPath } from "../url/merge-urls.js";
const makeUrlAbsolute = (url, {
  /** Optional base URL to resolve against (defaults to window.location.href) */
  baseUrl,
  /** If we have a basePath then we resolve against window.location.origin + basePath */
  basePath
} = {}) => {
  if (typeof window === "undefined" && !baseUrl) {
    return url;
  }
  try {
    new URL(url);
    return url;
  } catch {
  }
  try {
    let base = baseUrl || window.location.href;
    if (basePath) {
      const origin = baseUrl ? new URL(baseUrl).origin : window.location.origin;
      base = combineUrlAndPath(origin, basePath + "/");
    }
    return new URL(url, base).toString();
  } catch {
    return url;
  }
};
export {
  makeUrlAbsolute
};
//# sourceMappingURL=make-url-absolute.js.map
