import type { LoaderPlugin } from '../../../bundle/index.js';
/**
 * Creates a plugin that parses JSON strings into JavaScript objects.
 * @returns A plugin object with validate and exec functions
 * @example
 * ```ts
 * const jsonPlugin = parseJson()
 * const result = jsonPlugin.exec('{"name": "John", "age": 30}')
 * // result = { name: 'John', age: 30 }
 * ```
 */
export declare function parseJson(): LoaderPlugin;
//# sourceMappingURL=index.d.ts.map