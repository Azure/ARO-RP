import YAML from "yaml";
import { isYaml } from "../../../helpers/is-yaml.js";
function parseYaml() {
  return {
    type: "loader",
    validate: isYaml,
    exec: (value) => {
      try {
        return Promise.resolve({
          ok: true,
          data: YAML.parse(value, { merge: true, maxAliasCount: 1e4 }),
          raw: value
        });
      } catch {
        return Promise.resolve({
          ok: false
        });
      }
    }
  };
}
export {
  parseYaml
};
//# sourceMappingURL=index.js.map
