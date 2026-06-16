import { REGEX } from "../regex/regex-helpers.js";
function replaceVariables(value, variablesOrCallback) {
  const doubleCurlyBrackets = /{{\s*([\w.-]+)\s*}}/g;
  const singleCurlyBrackets = /{\s*([\w.-]+)\s*}/g;
  const callback = (_, match) => {
    if (typeof variablesOrCallback === "function") {
      return variablesOrCallback(match);
    }
    return variablesOrCallback[match]?.toString() || `{${match}}`;
  };
  return value.replace(doubleCurlyBrackets, callback).replace(singleCurlyBrackets, callback);
}
const replacePathVariables = (path, variables = {}) => path.replace(REGEX.PATH, (match, key) => variables[key] ?? match);
const replaceEnvVariables = (path, variables = {}) => path.replace(REGEX.VARIABLES, (match, key) => variables[key] ?? match);
export {
  replaceEnvVariables,
  replacePathVariables,
  replaceVariables
};
//# sourceMappingURL=replace-variables.js.map
