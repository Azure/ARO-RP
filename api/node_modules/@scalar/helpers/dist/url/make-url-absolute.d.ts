/**
 * Converts a relative URL to an absolute URL using the provided base URL or current window location.
 * @param url - The URL to make absolute
 * @param options - Configuration options
 * @param options.baseUrl - Optional base URL to resolve against (defaults to window.location.href)
 * @param options.basePath - If provided, combines with baseUrl or window.location.origin before resolving
 * @returns The absolute URL, or the original URL if it's already absolute or invalid
 */
export declare const makeUrlAbsolute: (url: string, { baseUrl, basePath, }?: {
    baseUrl?: string;
    basePath?: string;
}) => string;
//# sourceMappingURL=make-url-absolute.d.ts.map