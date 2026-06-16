import type { CodeFix, CodeFixContext, CodeFixEdit, CompilerHost } from "./types.js";
export declare function applyCodeFixes(host: CompilerHost, codeFixes: readonly CodeFix[]): Promise<void>;
export declare function resolveCodeFix(codeFix: CodeFix): Promise<CodeFixEdit[]>;
export declare function applyCodeFix(host: CompilerHost, codeFix: CodeFix): Promise<void>;
export declare function applyCodeFixEditsOnText(content: string, edits: CodeFixEdit[]): string;
export declare function createCodeFixContext(): CodeFixContext;
//# sourceMappingURL=code-fixes.d.ts.map