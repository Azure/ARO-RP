import path from "pathe";
import { isHttpUrl } from "../helpers/is-http-url.js";
const toRelativePath = (input, base) => {
  if (isHttpUrl(input) && isHttpUrl(base)) {
    const inputUrl = new URL(input);
    const baseUrl = new URL(base);
    if (inputUrl.origin !== baseUrl.origin) {
      return input;
    }
    const baseDir2 = path.dirname(path.posix.resolve("/", baseUrl.pathname));
    const inputPath2 = path.posix.resolve("/", inputUrl.pathname);
    return path.posix.relative(baseDir2, inputPath2);
  }
  if (isHttpUrl(base)) {
    const baseUrl = new URL(base);
    const baseDir2 = path.dirname(path.posix.resolve("/", baseUrl.pathname));
    baseUrl.pathname = path.posix.relative(baseDir2, path.posix.resolve("/", input));
    return baseUrl.toString();
  }
  if (isHttpUrl(input)) {
    return input;
  }
  const baseDir = path.dirname(path.resolve(base));
  const inputPath = path.resolve(input);
  return path.relative(baseDir, inputPath);
};
export {
  toRelativePath
};
//# sourceMappingURL=to-relative-path.js.map
