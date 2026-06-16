import { getEntrypoint } from "../../../utils/get-entrypoint.js";
import { details as detailsUtility } from "../../../utils/details.js";
import { workThroughQueue } from "../utils/workThroughQueue.js";
async function details(queue) {
  const { filesystem } = await workThroughQueue(queue);
  return detailsUtility(getEntrypoint(filesystem).specification);
}
export {
  details
};
//# sourceMappingURL=details.js.map
