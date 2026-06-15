import { TestHostOptions } from "./test-compiler-host.js";
import type { JsFile, TestFileSystem } from "./types.js";
export declare function resolveVirtualPath(path: string, ...paths: string[]): string;
/**
 * Constructor for various mock files.
 */
export declare const mockFile: {
    /** Define a JS file with the given named exports */
    js: (exports: Record<string, unknown>) => JsFile;
};
export declare function createTestFileSystem(options?: TestHostOptions): TestFileSystem;
//# sourceMappingURL=fs.d.ts.map