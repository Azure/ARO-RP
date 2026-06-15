/**
 * Generates a short hash from a string value using xxhash.
 *
 * This function is used to create unique identifiers for external references
 * while keeping the hash length manageable. It uses xxhash-wasm instead of
 * crypto.subtle because crypto.subtle is only available in secure contexts (HTTPS) or on localhost.
 * Returns the first 7 characters of the hash string.
 * If the hash would be all numbers, it ensures at least one letter is included.
 *
 * @param value - The string to hash
 * @returns A Promise that resolves to a 7-character hexadecimal hash with at least one letter
 * @example
 * // Returns "2ae91d7"
 * getHash("https://example.com/schema.json")
 */
export declare function getHash(value: string): string;
/**
 * Generates a unique compressed value for a string, handling collisions by recursively compressing
 * until a unique value is found. This is used to create unique identifiers for external
 * references in the bundled OpenAPI document.
 *
 * @param compress - Function that generates a compressed value from a string
 * @param value - The original string value to compress
 * @param compressedToValue - Object mapping compressed values to their original values
 * @param prevCompressedValue - Optional previous compressed value to use as input for generating a new value
 * @param depth - Current recursion depth to prevent infinite loops
 * @returns A unique compressed value that doesn't conflict with existing values
 *
 * @example
 * const valueMap = {}
 * // First call generates compressed value for "example.com/schema.json"
 * const value1 = await generateUniqueValue(compress, "example.com/schema.json", valueMap)
 * // Returns something like "2ae91d7"
 *
 * // Second call with same value returns same compressed value
 * const value2 = await generateUniqueValue(compress, "example.com/schema.json", valueMap)
 * // Returns same value as value1
 *
 * // Call with different value generates new unique compressed value
 * const value3 = await generateUniqueValue(compress, "example.com/other.json", valueMap)
 * // Returns different value like "3bf82e9"
 */
export declare function generateUniqueValue(compress: (value: string) => Promise<string> | string, value: string, compressedToValue: Record<string, string>, prevCompressedValue?: string, depth?: number): Promise<string>;
/**
 * Factory function that creates a value generator with caching capabilities.
 * The generator maintains a bidirectional mapping between original values and their compressed forms.
 *
 * @param compress - Function that generates a compressed value from a string
 * @param compressedToValue - Initial mapping of compressed values to their original values
 * @returns An object with a generate method that produces unique compressed values
 *
 * @example
 * const compress = (value) => value.substring(0, 6) // Simple compression example
 * const initialMap = { 'abc123': 'example.com/schema.json' }
 * const generator = uniqueValueGeneratorFactory(compress, initialMap)
 *
 * // Generate compressed value for new string
 * const compressed = await generator.generate('example.com/other.json')
 * // Returns something like 'example'
 *
 * // Generate compressed value for existing string
 * const cached = await generator.generate('example.com/schema.json')
 * // Returns 'abc123' from cache
 */
export declare const uniqueValueGeneratorFactory: (compress: (value: string) => Promise<string> | string, compressedToValue: Record<string, string>) => {
    /**
     * Generates a unique compressed value for the given input string.
     * First checks if a compressed value already exists in the cache.
     * If not, generates a new unique compressed value and stores it in the cache.
     *
     * @param value - The original string value to compress
     * @returns A Promise that resolves to the compressed string value
     *
     * @example
     * const generator = uniqueValueGeneratorFactory(compress, {})
     * const compressed = await generator.generate('example.com/schema.json')
     * // Returns a unique compressed value like 'example'
     */
    generate: (value: string) => Promise<string>;
};
//# sourceMappingURL=value-generator.d.ts.map