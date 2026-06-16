import { Queue } from "../queue/queue.js";
function createLimiter(maxConcurrent) {
  let activeCount = 0;
  const queue = new Queue();
  const next = () => {
    if (queue.isEmpty() || activeCount >= maxConcurrent) {
      return;
    }
    const resolve = queue.dequeue();
    if (resolve) {
      resolve();
    }
  };
  const run = async (fn) => {
    if (activeCount >= maxConcurrent) {
      await new Promise((resolve) => queue.enqueue(resolve));
    }
    activeCount++;
    try {
      const result = await fn();
      return result;
    } finally {
      activeCount--;
      next();
    }
  };
  return run;
}
export {
  createLimiter
};
//# sourceMappingURL=create-limiter.js.map
