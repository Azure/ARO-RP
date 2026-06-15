import { getEntrypoint } from "../../../utils/get-entrypoint.js";
import { toYaml as toYamlUtility } from "../../../utils/to-yaml.js";
import { workThroughQueue } from "../utils/workThroughQueue.js";
async function toYaml(queue) {
  const { filesystem } = await workThroughQueue(queue);
  return toYamlUtility(getEntrypoint(filesystem).specification);
}
export {
  toYaml
};
//# sourceMappingURL=toYaml.js.map
