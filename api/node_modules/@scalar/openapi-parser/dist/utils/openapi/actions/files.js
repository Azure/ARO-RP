import { workThroughQueue } from "../utils/workThroughQueue.js";
async function files(queue) {
  const { filesystem } = await workThroughQueue(queue);
  return filesystem;
}
export {
  files
};
//# sourceMappingURL=files.js.map
