/**
 * Deep check for objects for collisions
 * Check primitives if their values are different
 *
 * @param a - First value to compare
 * @param b - Second value to compare
 * @returns true if there is a collision, false otherwise
 *
 * @example
 * // Objects with different values for same key
 * isKeyCollisions({ a: 1 }, { a: 2 }) // true
 *
 * // Objects with different types
 * isKeyCollisions({ a: 1 }, { a: '1' }) // true
 *
 * // Objects with no collisions
 * isKeyCollisions({ a: 1 }, { b: 2 }) // false
 *
 * // Nested objects with collision
 * isKeyCollisions({ a: { b: 1 } }, { a: { b: 2 } }) // true
 */
export declare const isKeyCollisions: (a: unknown, b: unknown) => boolean;
/**
 * Deep merges two objects, combining their properties recursively.
 *
 * ⚠️ Note: This operation assumes there are no key collisions between the objects.
 * Use isKeyCollisions() to check for collisions before merging.
 *
 * @param a - Target object to merge into
 * @param b - Source object to merge from
 * @returns The merged object (mutates and returns a)
 *
 * @example
 * // Simple merge
 * const a = { name: 'John' }
 * const b = { age: 30 }
 * mergeObjects(a, b) // { name: 'John', age: 30 }
 *
 * // Nested merge
 * const a = { user: { name: 'John' } }
 * const b = { user: { age: 30 } }
 * mergeObjects(a, b) // { user: { name: 'John', age: 30 } }
 */
export declare const mergeObjects: (a: Record<string, unknown>, b: Record<string, unknown>) => Record<string, unknown>;
/**
 * Checks if two arrays have the same elements in the same order.
 *
 * @param a - First array to compare
 * @param b - Second array to compare
 * @returns True if arrays have same length and elements, false otherwise
 *
 * @example
 * // Arrays with same elements
 * isArrayEqual([1, 2, 3], [1, 2, 3]) // true
 *
 * // Arrays with different elements
 * isArrayEqual([1, 2, 3], [1, 2, 4]) // false
 *
 * // Arrays with different lengths
 * isArrayEqual([1, 2], [1, 2, 3]) // false
 */
export declare const isArrayEqual: <T>(a: T[], b: T[]) => boolean;
//# sourceMappingURL=utils.d.ts.map