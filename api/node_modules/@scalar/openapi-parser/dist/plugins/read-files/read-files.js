import fs from "node:fs";
import { dirname, join } from "node:path";
import { ERRORS } from "../../configuration/index.js";
import { isJson } from "../../utils/is-json.js";
import { isYaml } from "../../utils/is-yaml.js";
const readFiles = () => {
  return {
    check(value) {
      if (typeof value !== "string") {
        return false;
      }
      if (value.startsWith("http://") || value.startsWith("https://")) {
        return false;
      }
      if (value.includes("\n")) {
        return false;
      }
      if (isJson(value)) {
        return false;
      }
      if (isYaml(value)) {
        return false;
      }
      return true;
    },
    get(value) {
      if (!fs.existsSync(value)) {
        throw new Error(ERRORS.FILE_DOES_NOT_EXIST.replace("%s", value));
      }
      try {
        return fs.readFileSync(value, "utf-8");
      } catch (error) {
        console.error("[readFiles]", error);
        return false;
      }
    },
    resolvePath(value, reference) {
      const dir = dirname(value);
      return join(dir, reference);
    },
    getDir(value) {
      return dirname(value);
    },
    getFilename(value) {
      return value.split("/").pop();
    }
  };
};
export {
  readFiles
};
//# sourceMappingURL=read-files.js.map
