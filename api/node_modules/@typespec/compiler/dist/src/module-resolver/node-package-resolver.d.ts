import { NodePackage, ResolveModuleHost } from "./types.js";
/**
 * Utility to resolve node packages.
 */
export declare class NodePackageResolver {
    #private;
    constructor(host: ResolveModuleHost);
    /**
     * Resolve a node package with the given specifier from the baseDir.
     * @param specifier Node package specifier
     * @returns NodePackage if found or undefined otherwise
     */
    resolve(specifier: string, baseDir: string): Promise<NodePackage | undefined>;
    /**
     * Resolve the NodePackage for the given specifier
     * Implementation from LOAD_PACKAGE_SELF minus the exports resolution which is called separately.
     */
    resolveSelf(packageName: string, baseDir: string): Promise<NodePackage | undefined>;
    /**
     * Resolve a node package from `node_modules`. Follows the implementation of LOAD_NODE_MODULES minus following the exports field.
     */
    resolveFromNodeModules(packageName: string, baseDir: string): Promise<NodePackage | undefined>;
}
//# sourceMappingURL=node-package-resolver.d.ts.map