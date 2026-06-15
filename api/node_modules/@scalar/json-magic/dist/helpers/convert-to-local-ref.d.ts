/**
 * Translates a JSON Reference ($ref) to a local object path within the root schema.
 *
 * @param ref - The JSON Reference string (e.g., "#/foo/bar", "other.json#/baz", "other.json#anchor")
 * @param currentContext - The current base context (usually the $id of the current schema or parent)
 * @param schemas - A map of schema identifiers ($id, $anchor) to their local object paths
 * @returns The local object path as a string, or undefined if the reference cannot be resolved
 */
export declare const convertToLocalRef: (ref: string, currentContext: string, schemas: Map<string, string>) => string | undefined;
//# sourceMappingURL=convert-to-local-ref.d.ts.map