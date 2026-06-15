import { Diagnostic, type CodeFix, type DiagnosticTarget } from "../types.js";
export declare function createSuppressCodeFix(diagnosticTarget: DiagnosticTarget, warningCode: string, suppressionMessage?: string): CodeFix;
/**
 * Create code fixes to suppress the given diagnostics.
 * This function groups diagnostics by their suppression target to avoid duplicate suppressions.
 * @param diagnostics The diagnostics to suppress.
 * @param suppressionMessage The suppression message to use.
 * @returns An array of code fixes to apply the suppressions.
 */
export declare function createSuppressCodeFixes(diagnostics: readonly Diagnostic[], suppressionMessage?: string): readonly CodeFix[];
//# sourceMappingURL=suppress.codefix.d.ts.map