import type { LoaderPlugin } from '../../../bundle/index.js';
/**
 * Creates a plugin that parses YAML strings into JavaScript objects.
 * @returns A plugin object with validate and exec functions
 * @example
 * ```ts
 * const yamlPlugin = parseYaml()
 * const result = yamlPlugin.exec('name: John\nage: 30')
 * // result = { name: 'John', age: 30 }
 * ```
 */
export declare function parseYaml(): LoaderPlugin;
//# sourceMappingURL=index.d.ts.map