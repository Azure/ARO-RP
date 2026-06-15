import { ok, strictEqual } from "assert";
import { applyCodeFix, applyCodeFixes } from "../core/code-fixes.js";
import { getNodeAtPosition, parse, visitChildren } from "../core/parser.js";
import { createSourceFile } from "../core/source-file.js";
import { mutate } from "../utils/misc.js";
import { extractCursor, extractCursors } from "./source-utils.js";
import { createTestHost } from "./test-host.js";
import { trimBlankLines } from "./test-utils.js";
/**
 * Test a code fix that only needs the ast as input.
 * @param code Code to parse. Use ┆ to mark the cursor position.
 * @param callback Callback to create the code fix it takes the node at the cursor position.
 *
 * @example
 *
 * ```ts
 *  await expectCodeFixOnAst(
 *    `
 *    model Foo {
 *      a: ┆number;
 *    }
 *  `,
 *    (node) => {
 *      strictEqual(node.kind, SyntaxKind.Identifier);
 *      return createChangeIdentifierCodeFix(node, "int32");
 *    }
 *  ).toChangeTo(`
 *    model Foo {
 *      a: int32;
 *    }
 *  `);
 * ```
 */
export function expectCodeFixOnAst(code, callback) {
    return { toChangeTo };
    async function toChangeTo(expectedCode) {
        const { pos, source } = extractCursor(code);
        const virtualFile = createSourceFile(source, "test.tsp");
        const script = parse(virtualFile);
        linkAstParents(script);
        const node = getNodeAtPosition(script, pos);
        ok(node, "Expected node at cursor. Make sure to have ┆ to mark which node.");
        const codefix = callback(node);
        const host = await createTestHost();
        let updatedContent;
        await applyCodeFix({
            ...host.compilerHost,
            writeFile: async (path, value) => {
                strictEqual(path, "test.tsp");
                updatedContent = value;
            },
        }, codefix);
        ok(updatedContent);
        strictEqual(trimBlankLines(updatedContent), trimBlankLines(expectedCode));
    }
}
export function expectCodeFixesOnAst(code, warningCode, callback) {
    return { toChangeTo };
    async function toChangeTo(expectedCode) {
        const { pos, source } = extractCursors(code);
        const virtualFile = createSourceFile(source, "test.tsp");
        const script = parse(virtualFile);
        linkAstParents(script);
        const diagnostics = [];
        for (const position of pos) {
            const node = getNodeAtPosition(script, position);
            ok(node, "Expected node at cursor. Make sure to have ┆ to mark which node.");
            diagnostics.push({
                code: warningCode,
                message: "",
                severity: "warning",
                target: { file: virtualFile, pos: node.pos, end: node.end },
            });
        }
        ok(diagnostics.length > 0, "Expected node at cursor. Make sure to have ┆ to mark which node.");
        const codeFixes = callback(diagnostics);
        const host = await createTestHost();
        let updatedContent;
        await applyCodeFixes({
            ...host.compilerHost,
            writeFile: async (path, value) => {
                strictEqual(path, "test.tsp");
                updatedContent = value;
            },
        }, codeFixes);
        ok(updatedContent);
        strictEqual(trimBlankLines(updatedContent), trimBlankLines(expectedCode));
    }
}
function linkAstParents(base) {
    visitChildren(base, (node) => {
        mutate(node).parent = base;
        linkAstParents(node);
    });
}
//# sourceMappingURL=code-fix-testing.js.map