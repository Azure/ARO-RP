import { REGEX } from "../regex/regex-helpers.js";
function ensureProtocol(url) {
  if (REGEX.PROTOCOL.test(url)) {
    return url;
  }
  return `http://${url.replace(/^\//, "")}`;
}
export {
  ensureProtocol
};
//# sourceMappingURL=ensure-protocol.js.map
