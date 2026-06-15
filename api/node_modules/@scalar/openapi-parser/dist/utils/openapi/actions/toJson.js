import { getEntrypoint } from "../../../utils/get-entrypoint.js";
import { toJson as toJsonUtility } from "../../../utils/to-json.js";
import { workThroughQueue } from "../utils/workThroughQueue.js";
async function toJson(queue) {
  const { filesystem } = await workThroughQueue(queue);
  return toJsonUtility(getEntrypoint(filesystem).specification);
}
export {
  toJson
};
//# sourceMappingURL=toJson.js.map
