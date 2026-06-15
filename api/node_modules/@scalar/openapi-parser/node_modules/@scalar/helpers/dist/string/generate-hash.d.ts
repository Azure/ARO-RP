/**
 * MurmurHash3 implementation
 *
 * Generate a hash from a string using the MurmurHash3 algorithm
 * Provides 64-bit hash output with excellent speed and distribution
 *
 * We had to move away from xxhash-wasm since it was causing issues with content security policy (CSP) violations.
 *
 * We cannot use crypto.subtle because it is only available in secure contexts (HTTPS) or on localhost.
 *
 * @param input - The string to hash
 * @returns The 64-bit hash of the input string as a 16-character hex string
 */
export declare const generateHash: (input: string) => string;
//# sourceMappingURL=generate-hash.d.ts.map