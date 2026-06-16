import { BasicTestRunner, TestHost, TestHostConfig } from "./types.js";
/** Use {@link createTester} */
export declare function createTestHost(config?: TestHostConfig): Promise<TestHost>;
/** Use {@link createTester} */
export declare function createTestRunner(host?: TestHost): Promise<BasicTestRunner>;
export declare function findFilesFromPattern(directory: string, pattern: string): Promise<string[]>;
//# sourceMappingURL=test-host.d.ts.map