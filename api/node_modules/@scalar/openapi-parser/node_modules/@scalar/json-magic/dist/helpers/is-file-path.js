import { isHttpUrl } from "../helpers/is-http-url.js";
import { isJsonObject } from "../helpers/is-json-object.js";
import { isYaml } from "../helpers/is-yaml.js";
function isFilePath(value) {
  return !isHttpUrl(value) && !isYaml(value) && !isJsonObject(value);
}
export {
  isFilePath
};
//# sourceMappingURL=is-file-path.js.map
