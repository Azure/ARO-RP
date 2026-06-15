import { httpMethods } from "./http-methods.js";
const isHttpMethod = (method) => method && typeof method === "string" ? httpMethods.has(method.toLowerCase()) : false;
export {
  isHttpMethod
};
//# sourceMappingURL=is-http-method.js.map
