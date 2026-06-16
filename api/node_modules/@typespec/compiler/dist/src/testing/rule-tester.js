import { ok, strictEqual } from "assert";
import { applyCodeFix as applyCodeFixReal } from "../core/code-fixes.js";
import { createDiagnosticCollector } from "../core/diagnostics.js";
import { createLinterRuleContext } from "../core/linter.js";
import { navigateProgram } from "../core/semantic-walker.js";
import { expectDiagnosticEmpty, expectDiagnostics } from "./expect.js";
import { resolveVirtualPath, trimBlankLines } from "./test-utils.js";
export function createLinterRuleTester(runner, ruleDef, libraryName) {
    return {
        expect,
    };
    function expect(code) {
        return {
            toBeValid,
            toEmitDiagnostics,
            applyCodeFix,
        };
        async function toBeValid() {
            const [_, diagnostics] = await compileAndDiagnose(code);
            expectDiagnosticEmpty(diagnostics);
        }
        async function toEmitDiagnostics(match) {
            const [result, diagnostics] = await compileAndDiagnose(code);
            let expected;
            if (typeof match === "function") {
                if ("autoCodeOffset" in runner) {
                    throw new Error(".toEmitDiagnostics with a function match can only be used with a TesterInstance");
                }
                expected = match(result);
            }
            else {
                expected = match;
            }
            expectDiagnostics(diagnostics, expected);
        }
        function applyCodeFix(fixId) {
            return { toEqual };
            async function toEqual(expectedCode) {
                const [_, diagnostics] = await compileAndDiagnose(code);
                const codefix = diagnostics[0].codefixes?.find((x) => x.id === fixId);
                ok(codefix, `Codefix with id "${fixId}" not found.`);
                let content;
                const host = {
                    ...runner.program.host,
                    writeFile: (name, newContent) => {
                        content = newContent;
                        return Promise.resolve();
                    },
                };
                await applyCodeFixReal(host, codefix);
                ok(content, "No content was written to the host.");
                const fs = "keys" in runner.fs ? runner.fs : runner.fs.fs;
                const offset = fs.get(resolveVirtualPath("./main.tsp"))?.indexOf(code);
                strictEqual(trimBlankLines(content.slice(offset)), trimBlankLines(expectedCode));
            }
        }
    }
    async function compileAndDiagnose(code) {
        const compilerOptions = { parseOptions: { comments: true } };
        let res;
        let codeDiagnostics;
        if (isLegacyTestRunner(runner)) {
            if (typeof code !== "string") {
                throw new Error("Only string code is supported with BasicTestRunner. Use Tester.createInstance()");
            }
            codeDiagnostics = await runner.diagnose(code, compilerOptions);
        }
        else {
            [res, codeDiagnostics] = await runner.compileAndDiagnose(code, { compilerOptions });
        }
        expectDiagnosticEmpty(codeDiagnostics);
        const diagnostics = createDiagnosticCollector();
        const rule = { ...ruleDef, id: `${libraryName}/${ruleDef.name}` };
        const context = createLinterRuleContext(runner.program, rule, diagnostics);
        const listener = ruleDef.create(context);
        navigateProgram(runner.program, listener);
        if (listener.exit) {
            await listener.exit(runner.program);
        }
        // No diagnostics should have been reported to the program. If it happened the rule is calling reportDiagnostic directly and should NOT be doing that.
        expectDiagnosticEmpty(runner.program.diagnostics);
        return [res, diagnostics.diagnostics];
    }
}
function isLegacyTestRunner(tester) {
    return "autoCodeOffset" in tester;
}
//# sourceMappingURL=rule-tester.js.map