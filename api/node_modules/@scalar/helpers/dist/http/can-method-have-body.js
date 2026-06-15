const BODY_METHODS = ["post", "put", "patch", "delete"];
const canMethodHaveBody = (method) => BODY_METHODS.includes(method.toLowerCase());
export {
  canMethodHaveBody
};
//# sourceMappingURL=can-method-have-body.js.map
