/**
 * Traverses an object using an array of string segments (path keys) and returns
 * the value at the specified path along with its context (id if available).
 *
 * @param target - The root object to traverse.
 * @param segments - An array of string keys representing the path to traverse.
 * @returns An object containing the final context (id or previous context) and the value at the path.
 *
 * @example
 * const obj = {
 *   foo: {
 *     bar: {
 *       baz: 42
 *     }
 *   }
 * };
 * // Returns: { context: '', value: 42 }
 * getValueByPath(obj, ['foo', 'bar', 'baz']);
 */
export declare function getValueByPath(target: unknown, segments: string[]): {
    context: string;
    value: any;
};
//# sourceMappingURL=get-value-by-path.d.ts.map