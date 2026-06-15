function isValidUrl(url) {
  try {
    return Boolean(new URL(url));
  } catch {
    return false;
  }
}
export {
  isValidUrl
};
//# sourceMappingURL=is-valid-url.js.map
