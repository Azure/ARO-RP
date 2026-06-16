import {
  escapeJsonPointer
} from "@scalar/json-magic/helpers/escape-json-pointer";
import {
  upgradeFromTwoToThree
} from "@scalar/openapi-upgrader/2.0-to-3.0";
import {
  upgradeFromThreeToThreeOne
} from "@scalar/openapi-upgrader/3.0-to-3.1";
import { dereference } from "./utils/dereference.js";
import { filter } from "./utils/filter.js";
import { isJson } from "./utils/is-json.js";
import { isYaml } from "./utils/is-yaml.js";
import { join } from "./utils/join/index.js";
import { load } from "./utils/load/index.js";
import { normalize } from "./utils/normalize.js";
import { openapi } from "./utils/openapi/openapi.js";
import { toJson } from "./utils/to-json.js";
import { toYaml } from "./utils/to-yaml.js";
import { sanitize } from "./utils/transform/sanitize.js";
import { traverse } from "./utils/traverse.js";
import { unescapeJsonPointer } from "./utils/unescape-json-pointer.js";
import { upgrade } from "./utils/upgrade.js";
import { validate } from "./utils/validate.js";
export {
  dereference,
  escapeJsonPointer,
  filter,
  isJson,
  isYaml,
  join,
  load,
  normalize,
  openapi,
  sanitize,
  toJson,
  toYaml,
  traverse,
  unescapeJsonPointer,
  upgrade,
  upgradeFromThreeToThreeOne,
  upgradeFromTwoToThree,
  validate
};
//# sourceMappingURL=index.js.map
