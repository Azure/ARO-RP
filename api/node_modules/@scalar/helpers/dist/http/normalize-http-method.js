import { isHttpMethod } from "./is-http-method.js";
const DEFAULT_REQUEST_METHOD = "get";
const normalizeHttpMethod = (method) => {
  if (typeof method !== "string") {
    console.warn(`Request method is not a string. Using ${DEFAULT_REQUEST_METHOD} as the default.`);
    return DEFAULT_REQUEST_METHOD;
  }
  const normalizedMethod = method.trim().toLowerCase();
  if (!isHttpMethod(normalizedMethod)) {
    console.warn(
      `${method || "Request method"} is not a valid request method. Using ${DEFAULT_REQUEST_METHOD} as the default.`
    );
    return DEFAULT_REQUEST_METHOD;
  }
  return normalizedMethod;
};
export {
  normalizeHttpMethod
};
//# sourceMappingURL=normalize-http-method.js.map
