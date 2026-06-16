import { isFilePath } from "../../../bundle/bundle.js";
import { normalize } from "../../../helpers/normalize.js";
async function readFile(path) {
  const fs = typeof window === "undefined" ? await import("node:fs/promises") : void 0;
  if (fs === void 0) {
    throw "Can not use readFiles plugin outside of a node environment";
  }
  try {
    const fileContents = await fs.readFile(path, { encoding: "utf-8" });
    return {
      ok: true,
      data: normalize(fileContents),
      raw: fileContents
    };
  } catch {
    return {
      ok: false
    };
  }
}
function readFiles() {
  return {
    type: "loader",
    validate: isFilePath,
    exec: readFile
  };
}
export {
  readFile,
  readFiles
};
//# sourceMappingURL=index.js.map
