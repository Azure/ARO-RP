type RemoveCircularOptions = {
    /** Prefix to add before the path in $ref values */
    prefix?: string;
    /** Cache of already processed objects */
    cache?: WeakMap<object, string>;
};
/**
 * Traverses an object or array, returning a deep copy in which circular references are replaced
 * by JSON Reference objects of the form: `{ $ref: "#/path/to/original" }`.
 * This allows safe serialization of objects with cycles, following the JSON Reference convention (RFC 6901).
 * An optional `prefix` for the `$ref` path can be provided via options.
 *
 * @param obj - The input object or array to process
 * @param options - Optional configuration; you can set a prefix for $ref pointers
 * @returns A new object or array, with all circular references replaced by $ref pointers
 */
export declare const toJsonCompatible: <T>(obj: T, options?: RemoveCircularOptions) => T;
export {};
//# sourceMappingURL=to-json-compatible.d.ts.map