/**
 * localStorage keys for resources
 * DO NOT CHANGE THESE AS IT WILL BREAK THE MIGRATION
 */
export declare const LS_KEYS: {
    readonly COLLECTION: "collection";
    readonly COOKIE: "cookie";
    readonly ENVIRONMENT: "environment";
    readonly REQUEST: "request";
    readonly REQUEST_EXAMPLE: "requestExample";
    readonly SECURITY_SCHEME: "securityScheme";
    readonly SERVER: "server";
    readonly TAG: "tag";
    readonly WORKSPACE: "workspace";
};
/**
 * localStorage keys for all reference resources
 * to ensure we do not have any conflicts
 */
export declare const REFERENCE_LS_KEYS: {
    /**
     * Store the selected client as a string in localStorage
     */
    readonly SELECTED_CLIENT: "scalar-reference-selected-client-v2";
    /**
     * Store the auth as a string in localStorage
     */
    readonly AUTH: "scalar-reference-auth";
};
/**
 * localStorage keys for all client resources
 * to ensure we do not have any conflicts
 */
export declare const CLIENT_LS_KEYS: {
    /**
     * @deprecated This key is deprecated and will be removed in a future release.
     * We are now storing the entire document for the api-client instead.
     */
    readonly AUTH: "scalar-client-auth";
    /**
     * @deprecated This key is deprecated and will be removed in a future release.
     * We are now storing the entire document for the api-client instead.
     */
    readonly SELECTED_SECURITY_SCHEMES: "scalar-client-selected-security-schemes";
};
/** SSR safe alias for localStorage */
export declare const safeLocalStorage: () => Storage | {
    getItem: () => null;
    setItem: () => null;
    removeItem: () => null;
};
//# sourceMappingURL=local-storage.d.ts.map