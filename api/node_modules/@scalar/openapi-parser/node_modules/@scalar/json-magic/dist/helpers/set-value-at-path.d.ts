/**
 * Sets a value at a specified path in an object, creating intermediate objects/arrays as needed.
 * This function traverses the object structure and creates any missing intermediate objects
 * or arrays based on the path segments. If the next segment is a numeric string, it creates
 * an array instead of an object.
 *
 * ⚠️ Warning: Be careful with object keys that look like numbers (e.g. "123") as this function
 * will interpret them as array indices and create arrays instead of objects. If you need to
 * use numeric-looking keys, consider prefixing them with a non-numeric character.
 *
 * @param obj - The target object to set the value in
 * @param path - The JSON pointer path where the value should be set
 * @param value - The value to set at the specified path
 * @throws {Error} If attempting to set a value at the root path ('')
 *
 * @example
 * const obj = {}
 * setValueAtPath(obj, '/foo/bar/0', 'value')
 * // Result:
 * // {
 * //   foo: {
 * //     bar: ['value']
 * //   }
 * // }
 *
 * @example
 * const obj = { existing: { path: 'old' } }
 * setValueAtPath(obj, '/existing/path', 'new')
 * // Result:
 * // {
 * //   existing: {
 * //     path: 'new'
 * //   }
 * // }
 *
 * @example
 * // ⚠️ Warning: This will create an array instead of an object with key "123"
 * setValueAtPath(obj, '/foo/123/bar', 'value')
 * // Result:
 * // {
 * //   foo: [
 * //     undefined,
 * //     undefined,
 * //     undefined,
 * //     { bar: 'value' }
 * //   ]
 * // }
 */
export declare function setValueAtPath(obj: any, path: string, value: any): void;
//# sourceMappingURL=set-value-at-path.d.ts.map