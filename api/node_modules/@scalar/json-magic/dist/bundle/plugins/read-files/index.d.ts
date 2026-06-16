import type { LoaderPlugin, ResolveResult } from '../../../bundle/index.js';
/**
 * Reads and normalizes data from a local file
 * @param path - The file path to read from
 * @returns A promise that resolves to either the normalized data or an error result
 * @example
 * ```ts
 * const result = await readFile('./schemas/user.json')
 * if (result.ok) {
 *   console.log(result.data) // The normalized data
 * } else {
 *   console.log('Failed to read file')
 * }
 * ```
 */
export declare function readFile(path: string): Promise<ResolveResult>;
/**
 * Creates a plugin for handling local file references.
 * This plugin validates and reads data from local filesystem paths.
 *
 * @returns A plugin object with validate and exec functions
 * @example
 * const filePlugin = readFiles()
 * if (filePlugin.validate('./local-schema.json')) {
 *   const result = await filePlugin.exec('./local-schema.json')
 * }
 */
export declare function readFiles(): LoaderPlugin;
//# sourceMappingURL=index.d.ts.map