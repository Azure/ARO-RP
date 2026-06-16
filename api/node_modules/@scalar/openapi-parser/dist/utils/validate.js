import { Validator } from "../lib/Validator/Validator.js";
import { makeFilesystem } from "./make-filesystem.js";
function validate(value, options) {
  try {
    const filesystem = makeFilesystem(value);
    const validator = new Validator();
    const result = validator.validate(filesystem, options);
    return Promise.resolve({
      ...result,
      specification: validator.specification,
      version: validator.version
    });
  } catch (err) {
    return Promise.reject(err);
  }
}
export {
  validate
};
//# sourceMappingURL=validate.js.map
