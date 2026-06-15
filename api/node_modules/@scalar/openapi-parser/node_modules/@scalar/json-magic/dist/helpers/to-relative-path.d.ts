/**
 * Converts an input path or URL to a relative path based on the provided base.
 * Handles both remote URLs and local file system paths.
 * - If both input and base are remote URLs and share the same origin, computes the relative pathname.
 * - If base is a remote URL but input is local, returns a remote URL with a relative pathname.
 * - If input is a remote URL but base is local, returns input as is.
 * - Otherwise, computes the relative path between two local paths.
 */
export declare const toRelativePath: (input: string, base: string) => string;
//# sourceMappingURL=to-relative-path.d.ts.map