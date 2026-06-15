const isMacOS = () => {
  if (typeof navigator === "undefined") {
    return false;
  }
  if (navigator.userAgentData?.platform) {
    return navigator.userAgentData.platform.toLowerCase().includes("mac");
  }
  return /Mac/.test(navigator.userAgent);
};
export {
  isMacOS
};
//# sourceMappingURL=is-mac-os.js.map
