function isHttpUrl(value) {
  try {
    const url = new URL(value);
    return url.protocol === "http:" || url.protocol === "https:";
  } catch {
    return false;
  }
}
export {
  isHttpUrl
};
//# sourceMappingURL=is-http-url.js.map
