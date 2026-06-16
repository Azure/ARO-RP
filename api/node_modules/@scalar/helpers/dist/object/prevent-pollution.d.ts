/**
 * Validates that a key is safe to use and does not pose a prototype pollution risk.
 * Throws an error if a dangerous key is detected.
 *
 * @param key - The key to validate
 * @param context - Optional context string to help identify where the validation failed
 * @throws {Error} If the key matches a known prototype pollution vector
 *
 * @example
 * ```ts
 * preventPollution('__proto__') // throws Error
 * preventPollution('safeName') // passes
 * preventPollution('constructor', 'operation update') // throws Error with context
 * ```
 */
export declare const preventPollution: (key: string, context?: string) => void;
//# sourceMappingURL=prevent-pollution.d.ts.map