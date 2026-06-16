import { workThroughQueue } from "../utils/workThroughQueue.js";
async function get(queue) {
  return {
    filesystem: [],
    ...await workThroughQueue(queue)
  };
}
export {
  get
};
//# sourceMappingURL=get.js.map
