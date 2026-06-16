const extractServerFromPath = (path = "") => {
  if (!path.trim()) {
    return null;
  }
  if (path.startsWith("//")) {
    try {
      const url = new URL(`https:${path}`);
      if (url.origin === "null") {
        return null;
      }
      const origin = url.origin.replace(/^https?:/, "");
      const remainingPath = decodeURIComponent(url.pathname) + url.search + url.hash;
      return [origin, remainingPath];
    } catch {
      return null;
    }
  }
  try {
    const url = new URL(path);
    if (url.origin === "null") {
      return null;
    }
    const remainingPath = decodeURIComponent(url.pathname) + url.search + url.hash;
    return [url.origin, remainingPath];
  } catch {
    return null;
  }
};
export {
  extractServerFromPath
};
//# sourceMappingURL=extract-server-from-path.js.map
