import { ModuleResolutionResult, NodePackage, ResolveModuleHost } from "./types.js";
export interface ResolveModuleOptions {
    baseDir: string;
    /**
     * When resolution reach package.json returns the path to the file relative to it.
     * @default pkg.main
     */
    resolveMain?: (pkg: any) => string;
    /**
     * When resolution reach a directory without package.json look for those files to load in order.
     * @default `["index.mjs", "index.js"]`
     */
    directoryIndexFiles?: string[];
    /** List of conditions to match in package exports */
    readonly conditions?: string[];
    /**
     * If exports is defined ignore if the none of the given condition is found and fallback to using main field resolution.
     * By default it will throw an error.
     */
    readonly fallbackOnMissingCondition?: boolean;
}
type ResolveModuleErrorCode = "MODULE_NOT_FOUND" | "INVALID_MAIN" | "INVALID_MODULE"
/** When an imports points to an invalid file. */
 | "INVALID_MODULE_IMPORT_TARGET"
/** When an exports points to an invalid file. */
 | "INVALID_MODULE_EXPORT_TARGET";
export declare class ResolveModuleError extends Error {
    code: ResolveModuleErrorCode;
    pkgJson?: NodePackage | undefined;
    constructor(code: ResolveModuleErrorCode, message: string, pkgJson?: NodePackage | undefined);
}
/**
 * Resolve a module
 * @param host
 * @param specifier
 * @param options
 * @returns
 * @throws {ResolveModuleError} When the module cannot be resolved.
 */
export declare function resolveModule(host: ResolveModuleHost, specifier: string, options: ResolveModuleOptions): Promise<ModuleResolutionResult>;
export {};
//# sourceMappingURL=module-resolver.d.ts.map