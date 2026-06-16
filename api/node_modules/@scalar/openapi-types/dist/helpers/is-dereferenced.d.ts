import type { OpenAPIV3_1 } from '../openapi-types.js';
/**
 * Type guard to check if an object is not a ReferenceObject.
 * A ReferenceObject is defined by having a $ref property that is a string.
 */
export declare const isDereferenced: <T>(obj: T | OpenAPIV3_1.ReferenceObject) => obj is T;
//# sourceMappingURL=is-dereferenced.d.ts.map