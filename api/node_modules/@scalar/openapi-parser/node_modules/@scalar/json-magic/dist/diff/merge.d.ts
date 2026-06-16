import type { Difference } from '../diff/diff.js';
/**
 * Merges two sets of differences from the same document and resolves conflicts.
 * This function combines changes from two diff lists while handling potential conflicts
 * that arise when both diffs modify the same paths. It uses a trie data structure for
 * efficient path matching and conflict detection.
 *
 * @param diff1 - First list of differences
 * @param diff2 - Second list of differences
 * @returns Object containing:
 *   - diffs: Combined list of non-conflicting differences
 *   - conflicts: Array of conflicting difference pairs that need manual resolution
 *
 * @example
 * // Merge two sets of changes to a user profile
 * const diff1 = [
 *   { path: ['name'], changes: 'John', type: 'update' },
 *   { path: ['age'], changes: 30, type: 'add' }
 * ]
 * const diff2 = [
 *   { path: ['name'], changes: 'Johnny', type: 'update' },
 *   { path: ['address'], changes: { city: 'NY' }, type: 'add' }
 * ]
 * const { diffs, conflicts } = merge(diff1, diff2)
 * // Returns:
 * // {
 * //   diffs: [
 * //     { path: ['age'], changes: 30, type: 'add' },
 * //     { path: ['address'], changes: { city: 'NY' }, type: 'add' }
 * //   ],
 * //   conflicts: [
 * //     [
 * //       [{ path: ['name'], changes: 'John', type: 'update' }],
 * //       [{ path: ['name'], changes: 'Johnny', type: 'update' }]
 * //     ]
 * //   ]
 * // }
 */
export declare const merge: <T>(diff1: Difference<T>[], diff2: Difference<T>[]) => {
    diffs: Difference<T>[];
    conflicts: [Difference<T>[], Difference<T>[]][];
};
//# sourceMappingURL=merge.d.ts.map