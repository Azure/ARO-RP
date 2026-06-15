import { dereference } from "../../../utils/dereference.js";
import { filter } from "../../../utils/filter.js";
import { load } from "../../../utils/load/load.js";
import { upgrade } from "../../../utils/upgrade.js";
import { validate } from "../../../utils/validate.js";
async function workThroughQueue(queue) {
  const { input } = {
    ...queue
  };
  let result = {};
  for (const task of queue.tasks) {
    const name = task.name;
    const options = "options" in task ? task.options : void 0;
    const currentSpecification = result.specification ? result.specification : typeof input === "object" ? (
      // Detach from the original object
      structuredClone(input)
    ) : input;
    if (name === "load") {
      result = {
        ...result,
        ...await load(input, options)
      };
    } else if (name === "filter") {
      result = {
        ...result,
        ...filter(currentSpecification, options)
      };
    } else if (name === "dereference") {
      result = {
        ...result,
        ...dereference(currentSpecification, options)
      };
    } else if (name === "upgrade") {
      result = {
        ...result,
        ...upgrade(currentSpecification)
      };
    } else if (name === "validate") {
      result = {
        ...result,
        ...await validate(currentSpecification, options)
      };
    } else {
      name;
    }
  }
  return result;
}
export {
  workThroughQueue
};
//# sourceMappingURL=workThroughQueue.js.map
