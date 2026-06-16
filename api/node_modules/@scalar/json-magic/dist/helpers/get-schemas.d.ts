/**
 * Retrieves the $id property from the input object if it exists and is a string.
 *
 * @param input - The object to extract the $id from.
 * @returns The $id string if present, otherwise undefined.
 */
export declare const getId: (input: unknown) => string | undefined;
/**
 * Recursively traverses the input object to collect all schemas identified by $id and $anchor properties.
 *
 * - If an object has a $id property, it is added to the map with its $id as the key.
 * - If an object has a $anchor property, it is added to the map with a key composed of the current base and the anchor.
 * - The function performs a depth-first search (DFS) through all nested objects.
 *
 * @param input - The input object to traverse.
 * @param base - The current base URI, used for resolving anchors.
 * @param map - The map collecting found schemas.
 * @returns A map of schema identifiers to their corresponding objects.
 */
export declare const getSchemas: (input: unknown, base?: string, segments?: string[], map?: Map<string, string>, visited?: WeakSet<object>) => Map<string, string>;
//# sourceMappingURL=get-schemas.d.ts.map