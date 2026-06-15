/**
 * Extracts the server from a string, used to check for servers in paths during migration
 *
 * @param path - The URL string to parse. If no protocol is provided, the URL API will throw an error.
 * @returns A tuple of [origin, remainingPath] or null if the input is empty, whitespace-only, or invalid.
 *
 * @example
 * extractServer('https://api.example.com/v1/users?id=123')
 * // Returns: ['https://api.example.com', '/v1/users?id=123']
 *
 * @example
 * extractServer('/users')
 * // Returns: null
 *
 * @example
 * extractServer('/users')
 * // Returns: null
 *
 * @example
 * extractServer('//api.example.com/v1/users')
 * // Returns: ['//api.example.com', '/v1/users']
 */
export declare const extractServerFromPath: (path?: string) => [string, string] | null;
//# sourceMappingURL=extract-server-from-path.d.ts.map