import { isJsonObject } from "../../../helpers/is-json-object.js";
function parseJson() {
  return {
    type: "loader",
    validate: isJsonObject,
    exec: (value) => {
      try {
        return Promise.resolve({
          ok: true,
          data: JSON.parse(value),
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
  parseJson
};
//# sourceMappingURL=index.js.map
