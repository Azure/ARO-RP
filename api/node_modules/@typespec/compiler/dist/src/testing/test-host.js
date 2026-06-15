import assert from "assert";
import { globby } from "globby";
import { logDiagnostics, logVerboseTestOutput } from "../core/diagnostics.js";
import { createLogger } from "../core/logger/logger.js";
import { compile as compileProgram } from "../core/program.js";
import { expectDiagnosticEmpty } from "./expect.js";
import { createTestFileSystem } from "./fs.js";
import { addTestLib, StandardTestLibrary } from "./test-compiler-host.js";
import { createTestWrapper, resolveVirtualPath } from "./test-utils.js";
/** Use {@link createTester} */
export async function createTestHost(config = {}) {
    const testHost = await createTestHostInternal();
    await testHost.addTypeSpecLibrary(StandardTestLibrary);
    if (config.libraries) {
        for (const library of config.libraries) {
            await testHost.addTypeSpecLibrary(library);
        }
    }
    return testHost;
}
/** Use {@link createTester} */
export async function createTestRunner(host) {
    const testHost = host ?? (await createTestHost());
    return createTestWrapper(testHost);
}
async function createTestHostInternal() {
    let program;
    const libraries = [];
    const fileSystem = await createTestFileSystem();
    const testTypes = addTestLib(fileSystem);
    return {
        ...fileSystem,
        addTypeSpecLibrary: async (lib) => {
            if (lib !== StandardTestLibrary) {
                libraries.push(lib);
            }
            await fileSystem.addTypeSpecLibrary(lib);
        },
        compile,
        diagnose,
        compileAndDiagnose,
        testTypes,
        libraries,
        get program() {
            assert(program, "Program cannot be accessed without calling compile, diagnose, or compileAndDiagnose.");
            return program;
        },
    };
    async function compile(main, options = {}) {
        const [testTypes, diagnostics] = await compileAndDiagnose(main, options);
        expectDiagnosticEmpty(diagnostics);
        return testTypes;
    }
    async function diagnose(main, options = {}) {
        const [, diagnostics] = await compileAndDiagnose(main, options);
        return diagnostics;
    }
    async function compileAndDiagnose(mainFile, options = {}) {
        const p = await compileProgram(fileSystem.compilerHost, resolveVirtualPath(mainFile), options);
        program = p;
        logVerboseTestOutput((log) => logDiagnostics(p.diagnostics, createLogger({ sink: fileSystem.compilerHost.logSink })));
        return [testTypes, p.diagnostics];
    }
}
export async function findFilesFromPattern(directory, pattern) {
    return globby(pattern, {
        cwd: directory,
        onlyFiles: true,
        ignore: ["**/*.{test,spec}.{ts,tsx,js,jsx}"],
    });
}
//# sourceMappingURL=test-host.js.map