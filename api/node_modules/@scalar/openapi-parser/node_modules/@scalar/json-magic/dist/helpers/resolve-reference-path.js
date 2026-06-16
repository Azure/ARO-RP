import path from "pathe";
import { isHttpUrl } from "../helpers/is-http-url.js";
const resolveReferencePath = (base, relativePath) => {
  if (isHttpUrl(relativePath)) {
    return relativePath;
  }
  if (isHttpUrl(base)) {
    const baseUrl = new URL(base);
    baseUrl.pathname = path.posix.resolve("/", path.dirname(baseUrl.pathname), relativePath);
    return baseUrl.toString();
  }
  return path.resolve(path.dirname(base), relativePath);
};
export {
  resolveReferencePath
};
//# sourceMappingURL=resolve-reference-path.js.map
