/**
 * Parses a JSON Pointer string into an array of path segments
 *
 * @example
 * ```ts
 * parseJsonPointer('#/components/schemas/User')
 *
 * ['components', 'schemas', 'User']
 * ```
 */
export declare function parseJsonPointer(pointer: string): string[];
/**
 * Creates a nested path in an object from an array of path segments.
 * Only creates intermediate objects/arrays if they don't already exist.
 *
 * @param obj - The target object to create the path in
 * @param segments - Array of path segments to create
 * @returns The final nested object/array at the end of the path
 *
 * @example
 * ```ts
 * const obj = {}
 * createPathFromSegments(obj, ['components', 'schemas', 'User'])
 * // Creates: { components: { schemas: { User: {} } } }
 *
 * createPathFromSegments(obj, ['items', '0', 'name'])
 * // Creates: { items: [{ name: {} }] }
 * ```
 */
export declare function createPathFromSegments(obj: any, segments: string[]): any;
//# sourceMappingURL=json-path-utils.d.ts.map