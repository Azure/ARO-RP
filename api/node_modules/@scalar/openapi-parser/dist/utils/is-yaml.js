import { parse } from "yaml";
function isYaml(value) {
  if (!value.includes("\n")) {
    return false;
  }
  try {
    parse(value, {
      maxAliasCount: 1e4
    });
    return true;
  } catch (_error) {
    return false;
  }
}
export {
  isYaml
};
//# sourceMappingURL=is-yaml.js.map
