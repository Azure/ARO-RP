import type { ParserOptions } from "prettier";
import { Diagnostic, Node, TypeSpecScriptNode } from "../core/types.js";
export declare function parse(text: string, options: ParserOptions<any>): TypeSpecScriptNode;
/**
 * We are patching the syntax tree to flatten the namespace nodes that are created from namespace Foo.Bar; which have the same pos, end
 * This causes prettier to not know where comments belong.
 * https://github.com/microsoft/typespec/pull/2061
 */
export declare function flattenNamespaces(base: Node): void;
export declare class PrettierParserError extends Error {
    readonly error: Diagnostic;
    loc: {
        start: number;
        end: number;
    };
    constructor(error: Diagnostic);
}
//# sourceMappingURL=parser.d.ts.map