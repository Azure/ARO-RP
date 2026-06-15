const debounce = (options = {}) => {
  const { delay = 328, maxWait } = options;
  const timeouts = /* @__PURE__ */ new Map();
  const maxWaitTimeouts = /* @__PURE__ */ new Map();
  const latestFunctions = /* @__PURE__ */ new Map();
  const cleanup = () => {
    timeouts.forEach(clearTimeout);
    maxWaitTimeouts.forEach(clearTimeout);
    timeouts.clear();
    maxWaitTimeouts.clear();
    latestFunctions.clear();
  };
  const executeAndCleanup = (key) => {
    const fn = latestFunctions.get(key);
    const timeout = timeouts.get(key);
    if (timeout !== void 0) {
      clearTimeout(timeout);
      timeouts.delete(key);
    }
    const maxWaitTimeout = maxWaitTimeouts.get(key);
    if (maxWaitTimeout !== void 0) {
      clearTimeout(maxWaitTimeout);
      maxWaitTimeouts.delete(key);
    }
    latestFunctions.delete(key);
    if (fn !== void 0) {
      try {
        fn();
      } catch {
      }
    }
  };
  const execute = (key, fn) => {
    latestFunctions.set(key, fn);
    const existingTimeout = timeouts.get(key);
    if (existingTimeout !== void 0) {
      clearTimeout(existingTimeout);
    }
    timeouts.set(
      key,
      setTimeout(() => executeAndCleanup(key), delay)
    );
    if (maxWait !== void 0 && !maxWaitTimeouts.has(key)) {
      maxWaitTimeouts.set(
        key,
        setTimeout(() => executeAndCleanup(key), maxWait)
      );
    }
  };
  return { execute, cleanup };
};
export {
  debounce
};
//# sourceMappingURL=debounce.js.map
