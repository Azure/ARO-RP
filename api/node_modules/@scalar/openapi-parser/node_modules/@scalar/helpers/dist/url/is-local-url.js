const LOCAL_HOSTNAMES = ["localhost", "127.0.0.1", "[::1]", "0.0.0.0"];
const RESERVED_TLDS = ["test", "example", "invalid", "localhost"];
function isLocalUrl(url) {
  try {
    const { hostname } = new URL(url);
    if (LOCAL_HOSTNAMES.includes(hostname)) {
      return true;
    }
    const tld = hostname.split(".").pop();
    if (tld && RESERVED_TLDS.includes(tld)) {
      return true;
    }
    return false;
  } catch {
    return true;
  }
}
export {
  isLocalUrl
};
//# sourceMappingURL=is-local-url.js.map
