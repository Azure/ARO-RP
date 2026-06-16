import type { HttpMethod } from './http-methods.js';
/** HTTP Methods which can have a body */
declare const BODY_METHODS: ["post", "put", "patch", "delete"];
type BodyMethod = (typeof BODY_METHODS)[number];
/** Makes a check to see if this method CAN have a body */
export declare const canMethodHaveBody: (method: HttpMethod) => method is BodyMethod;
export {};
//# sourceMappingURL=can-method-have-body.d.ts.map