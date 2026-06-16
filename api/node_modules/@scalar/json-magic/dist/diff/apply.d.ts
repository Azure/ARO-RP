import type { Difference } from '../diff/diff.js';
export declare class InvalidChangesDetectedError extends Error {
    constructor(message: string);
}
/**
 * Applies a set of differences to a document object.
 * The function traverses the document structure following the paths specified in the differences
 * and applies the corresponding changes (add, update, or delete) at each location.
 *
 * @param document - The original document to apply changes to
 * @param diff - Array of differences to apply, each containing a path and change type
 * @returns The modified document with all changes applied
 *
 * @example
 * const original = {
 *   paths: {
 *     '/users': {
 *       get: { responses: { '200': { description: 'OK' } } }
 *     }
 *   }
 * }
 *
 * const changes = [
 *   {
 *     path: ['paths', '/users', 'get', 'responses', '200', 'content'],
 *     type: 'add',
 *     changes: { 'application/json': { schema: { type: 'object' } } }
 *   }
 * ]
 *
 * const updated = apply(original, changes)
 * // Result: original document with content added to the 200 response
 */
export declare const apply: <T extends Record<string, unknown>>(document: Record<string, unknown>, diff: Difference<T>[]) => T;
//# sourceMappingURL=apply.d.ts.map