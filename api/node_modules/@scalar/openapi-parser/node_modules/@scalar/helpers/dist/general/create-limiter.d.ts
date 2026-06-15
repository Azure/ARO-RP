/**
 * Creates a function that limits the number of concurrent executions of async functions.
 *
 * @param maxConcurrent - Maximum number of concurrent executions allowed
 * @returns A function that wraps async functions to limit their concurrent execution
 *
 * @example
 * ```ts
 * const limiter = createLimiter(2) // Allow max 2 concurrent executions
 *
 * // These will run with max 2 at a time
 * const results = await Promise.all([
 *   limiter(() => fetch('/api/1')),
 *   limiter(() => fetch('/api/2')),
 *   limiter(() => fetch('/api/3')),
 *   limiter(() => fetch('/api/4'))
 * ])
 * ```
 */
export declare function createLimiter(maxConcurrent: number): <T>(fn: () => Promise<T>) => Promise<T>;
//# sourceMappingURL=create-limiter.d.ts.map