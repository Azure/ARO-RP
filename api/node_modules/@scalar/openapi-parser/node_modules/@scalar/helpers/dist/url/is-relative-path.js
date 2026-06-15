import { REGEX } from "../regex/regex-helpers.js";
const isRelativePath = (url) => {
  if (REGEX.PROTOCOL.test(url)) {
    return false;
  }
  if (/^[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+(\/|$)/.test(url)) {
    return false;
  }
  return true;
};
export {
  isRelativePath
};
//# sourceMappingURL=is-relative-path.js.map
