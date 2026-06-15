import type { NodePackage, ResolveModuleHost } from "./types.js";
export interface NodeModuleSpecifier {
    readonly packageName: string;
    readonly subPath: string;
}
export declare function parseNodeModuleSpecifier(id: string): NodeModuleSpecifier | null;
export declare function readPackage(host: ResolveModuleHost, pkgfile: string): Promise<NodePackage>;
export declare function isFile(host: ResolveModuleHost, path: string): Promise<boolean>;
export declare function pathToFileURL(path: string): string;
export declare function fileURLToPath(url: string): string;
/**
 * Returns a list of all the parent directory and the given one.
 */
export declare function listDirHierarchy(baseDir: string): string[];
//# sourceMappingURL=utils.d.ts.map