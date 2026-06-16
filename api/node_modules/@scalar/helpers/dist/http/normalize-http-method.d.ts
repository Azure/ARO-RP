import type { HttpMethod } from './http-methods.js';
/**
 * Get a normalized request method (e.g. get, post, etc.)
 * Lowercases the method and returns the default if it is not a valid method so you will always have a valid method
 */
export declare const normalizeHttpMethod: (method?: string) => HttpMethod;
//# sourceMappingURL=normalize-http-method.d.ts.map