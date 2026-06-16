import { DiagnosticMessages, Entity, LinterRuleDefinition } from "../core/types.js";
import { DiagnosticMatch } from "./expect.js";
import { GetMarkedEntities, TemplateWithMarkers } from "./marked-template.js";
import { BasicTestRunner, TestCompileResult, TesterInstance } from "./types.js";
export interface LinterRuleTester {
    expect<T extends string | TemplateWithMarkers<any> | Record<string, string | TemplateWithMarkers<any>>>(code: T): LinterRuleTestExpect<GetMarkedEntities<T>>;
}
export interface LinterRuleTestExpect<T extends Record<string, Entity> = any> {
    toBeValid(): Promise<void>;
    toEmitDiagnostics(diagnostics: DiagnosticMatch | DiagnosticMatch[] | ((res: TestCompileResult<T>) => DiagnosticMatch | DiagnosticMatch[])): Promise<void>;
    applyCodeFix(codeFixId: string): ApplyCodeFixExpect;
}
export interface ApplyCodeFixExpect {
    toEqual(code: string): Promise<void>;
}
export declare function createLinterRuleTester(runner: BasicTestRunner | TesterInstance, ruleDef: LinterRuleDefinition<string, DiagnosticMessages>, libraryName: string): LinterRuleTester;
//# sourceMappingURL=rule-tester.d.ts.map