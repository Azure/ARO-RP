import type { UnknownObject } from '../types.js';
/**
 * Creates a "magic" proxy for a given object or array, enabling transparent access to
 * JSON Reference ($ref) values as if they were directly present on the object.
 *
 * Features:
 * - If an object contains a `$ref` property, accessing the special `$ref-value` property will resolve and return the referenced value from the root object.
 * - All nested objects and arrays are recursively wrapped in proxies, so reference resolution works at any depth.
 * - Properties starting with `__scalar_` are considered internal and are hidden by default: they return undefined on access, are excluded from enumeration, and `'in'` checks return false. This can be overridden with the `showInternal` option.
 * - Setting, deleting, and enumerating properties works as expected, including for proxied references.
 * - Ensures referential stability by caching proxies for the same target object.
 *
 * @param target - The object or array to wrap in a magic proxy
 * @param options - Optional settings (e.g., showInternal to expose internal properties)
 * @param args - Internal arguments for advanced usage (root object, proxy/cache maps, current context)
 * @returns A proxied version of the input object/array with magic $ref-value support
 *
 * @example
 * const input = {
 *   definitions: {
 *     foo: { bar: 123 }
 *   },
 *   refObj: { $ref: '#/definitions/foo' },
 *   __scalar_internal: 'hidden property'
 * }
 * const proxy = createMagicProxy(input)
 *
 * // Accessing proxy.refObj['$ref-value'] will resolve to { bar: 123 }
 * console.log(proxy.refObj['$ref-value']) // { bar: 123 }
 *
 * // Properties starting with __scalar_ are hidden
 * console.log(proxy.__scalar_internal) // undefined
 * console.log('__scalar_internal' in proxy) // false
 * console.log(Object.keys(proxy)) // ['definitions', 'refObj'] (no '__scalar_internal')
 *
 * // Setting and deleting properties works as expected
 * proxy.refObj.extra = 'hello'
 * delete proxy.refObj.extra
 */
export declare const createMagicProxy: <T extends Record<keyof T & symbol, unknown>, S extends UnknownObject>(target: T, options?: Partial<{
    showInternal: boolean;
}>, args?: {
    /**
     * The root object for resolving local JSON references.
     */
    root: S | T;
    /**
     * Cache to store already created proxies for target objects to ensure referential stability.
     *
     * It is helpful when dealing with reactive frameworks like Vue,
     */
    proxyCache: WeakMap<object, T>;
    /**
     * Cache to store resolved JSON references.
     */
    cache: Map<string, unknown>;
    /**
     * Map of all schemas by their $id or $anchor for cross-document reference resolution.
     */
    schemas: Map<string, string>;
    /**
     * The current JSON path context within the root object.
     *
     * Used to resolve $anchor references correctly.
     */
    currentContext: string;
}) => T;
/**
 * Gets the raw (non-proxied) version of an object created by createMagicProxy.
 * This is useful when you need to access the original object without the magic proxy wrapper.
 *
 * @param obj - The magic proxy object to get the raw version of
 * @returns The raw version of the object
 * @example
 * const proxy = createMagicProxy({ foo: { $ref: '#/bar' } })
 * const raw = getRaw(proxy) // { foo: { $ref: '#/bar' } }
 */
export declare function getRaw<T>(obj: T): T;
//# sourceMappingURL=proxy.d.ts.map