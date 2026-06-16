import { CodeFix, DiagnosticTarget } from "../../types.js";
/**
 * Create a codefix to add a decorator at the target location.
 * @param diagnosticTarget Diagnostic target
 * @param decoratorName Decorator name(e.g. `doc`)
 * @param decoratorParamText Decorator args(e.g. `"This is a doc string."`)
 */
export declare function createAddDecoratorCodeFix(diagnosticTarget: DiagnosticTarget, name: string, args?: string[]): CodeFix;
//# sourceMappingURL=create-add-decorator.codefix.d.ts.map