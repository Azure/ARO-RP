import { Diagnostic } from "../core/types.js";
import type { Tester } from "./types.js";
export interface TesterOptions {
    libraries: string[];
}
export declare function createTester(base: string, options: TesterOptions): Tester;
export interface Compilable<A extends unknown[], R> {
    compileAndDiagnose(...args: A): Promise<[R, readonly Diagnostic[]]>;
    compile(...args: A): Promise<R>;
    diagnose(...args: A): Promise<readonly Diagnostic[]>;
}
//# sourceMappingURL=tester.d.ts.map