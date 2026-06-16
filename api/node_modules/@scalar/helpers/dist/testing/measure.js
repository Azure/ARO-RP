const measureSync = (name, fn) => {
  const start = performance.now();
  const result = fn();
  const end = performance.now();
  const duration = Math.round(end - start);
  console.info(`${name}: ${duration} ms`);
  return result;
};
const measureAsync = async (name, fn) => {
  const start = performance.now();
  const result = await fn();
  const end = performance.now();
  const duration = Math.round(end - start);
  console.info(`${name}: ${duration} ms`);
  return result;
};
export {
  measureAsync,
  measureSync
};
//# sourceMappingURL=measure.js.map
