function createLimiter(maxConcurrent) {
  let activeCount = 0;
  const queue = [];
  const next = () => {
    if (queue.length === 0 || activeCount >= maxConcurrent) {
      return;
    }
    const resolve = queue.shift();
    if (resolve) {
      resolve();
    }
  };
  const run = async (fn) => {
    if (activeCount >= maxConcurrent) {
      await new Promise((resolve) => queue.push(resolve));
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
