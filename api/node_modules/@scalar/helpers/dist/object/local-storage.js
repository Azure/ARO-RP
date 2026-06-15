const LS_KEYS = {
  COLLECTION: "collection",
  COOKIE: "cookie",
  ENVIRONMENT: "environment",
  REQUEST: "request",
  REQUEST_EXAMPLE: "requestExample",
  SECURITY_SCHEME: "securityScheme",
  SERVER: "server",
  TAG: "tag",
  WORKSPACE: "workspace"
};
const REFERENCE_LS_KEYS = {
  /**
   * Store the selected client as a string in localStorage
   */
  SELECTED_CLIENT: "scalar-reference-selected-client-v2",
  /**
   * Store the auth as a string in localStorage
   */
  AUTH: "scalar-reference-auth"
};
const CLIENT_LS_KEYS = {
  /**
   * @deprecated This key is deprecated and will be removed in a future release.
   * We are now storing the entire document for the api-client instead.
   */
  AUTH: "scalar-client-auth",
  /**
   * @deprecated This key is deprecated and will be removed in a future release.
   * We are now storing the entire document for the api-client instead.
   */
  SELECTED_SECURITY_SCHEMES: "scalar-client-selected-security-schemes"
};
const safeLocalStorage = () => typeof window === "undefined" ? {
  getItem: () => null,
  setItem: () => null,
  removeItem: () => null
} : localStorage;
export {
  CLIENT_LS_KEYS,
  LS_KEYS,
  REFERENCE_LS_KEYS,
  safeLocalStorage
};
//# sourceMappingURL=local-storage.js.map
