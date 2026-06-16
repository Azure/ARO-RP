/**
 * Measures the execution time of a function and logs it.
 *
 * Works only with sync functions and returns the result of the measured function.
 *
 * @example
 *
 * ```ts
 * // Sync function
 * const result = measureSync('computation', () => {
 *   return heavyComputation()
 * })
 * ```
 */
export declare const measureSync: <F extends () => unknown>(name: string, fn: ReturnType<F> extends Promise<unknown> ? never : F) => ReturnType<F>;
/**
 * Measures the execution time of an async function and logs it.
 *
 * Works only with async functions and returns the result of the measured function.
 *
 * @example
 *
 * ```ts
 * // Async function
 * const result = await measure('api-call', async () => {
 *   return await fetchData()
 * })
 * ````
 */
export declare const measureAsync: <T>(name: string, fn: () => Promise<T>) => Promise<T>;
//# sourceMappingURL=measure.d.ts.map