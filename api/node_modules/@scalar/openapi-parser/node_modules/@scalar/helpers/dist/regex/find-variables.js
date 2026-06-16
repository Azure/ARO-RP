import { REGEX } from "./regex-helpers.js";
const findVariables = (value, { includePath = true, includeEnv = true } = {}) => [includePath && REGEX.PATH, includeEnv && REGEX.VARIABLES].flatMap(
  (regex) => regex ? [...value.matchAll(regex)].map((match) => match[1]?.trim()).filter((variable) => variable !== void 0) : []
);
export {
  findVariables
};
//# sourceMappingURL=find-variables.js.map
