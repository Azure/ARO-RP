const PROTOTYPE_POLLUTION_KEYS = /* @__PURE__ */ new Set(["__proto__", "prototype", "constructor"]);
const preventPollution = (key, context) => {
  if (PROTOTYPE_POLLUTION_KEYS.has(key)) {
    const errorMessage = context ? `Prototype pollution key detected: "${key}" in ${context}` : `Prototype pollution key detected: "${key}"`;
    throw new Error(errorMessage);
  }
};
export {
  preventPollution
};
//# sourceMappingURL=prevent-pollution.js.map
