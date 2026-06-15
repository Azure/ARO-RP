/**
 * Options for configuring the debounce behavior.
 */
export type DebounceOptions = {
    /** The delay in milliseconds before executing the function. Defaults to 100ms. */
    delay?: number;
    /** Maximum time in milliseconds to wait before forcing execution, even with continuous calls. */
    maxWait?: number;
};
/**
 * Creates a debounced function executor that delays execution until after a specified time.
 * Multiple calls with the same key will cancel previous pending executions.
 *
 * This is useful for batching rapid updates (like auto-save or API calls) to avoid
 * unnecessary processing or network requests.
 *
 * @param options - Configuration options for delay, maxWait, and key separator
 * @returns A function that accepts a key array and callback to execute
 *
 * @example
 * const debouncedSave = debounce({ delay: 328 })
 * debouncedSave.execute(['user', '123'], () => saveUser(user))
 *
 * @example
 * // With maxWait to guarantee execution even with continuous calls
 * const debouncedSave = debounce({ delay: 328, maxWait: 2000 })
 * debouncedSave.execute(['user', '123'], () => saveUser(user))
 */
export declare const debounce: (options?: DebounceOptions) => {
    execute: (key: string, fn: () => unknown | Promise<unknown>) => void;
    cleanup: () => void;
};
//# sourceMappingURL=debounce.d.ts.map