import { CompilerHost, Type } from "../core/types.js";
import { TestFileSystem, TypeSpecTestLibrary } from "./types.js";
export declare const StandardTestLibrary: TypeSpecTestLibrary;
export interface TestHostOptions {
    caseInsensitiveFileSystem?: boolean;
    excludeTestLib?: boolean;
    compilerHostOverrides?: Partial<CompilerHost>;
}
export declare function createTestCompilerHost(virtualFs: Map<string, string>, jsImports: Map<string, Record<string, any>>, options?: TestHostOptions): CompilerHost;
export declare function addTestLib(fs: TestFileSystem): Record<string, Type>;
//# sourceMappingURL=test-compiler-host.d.ts.map