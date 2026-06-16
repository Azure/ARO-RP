export { 
/** @deprecated Using this should be a noop. Prefer new test framework*/
StandardTestLibrary, } from "./test-compiler-host.js";
export { expectCodeFixOnAst } from "./code-fix-testing.js";
export { expectDiagnosticEmpty, expectDiagnostics } from "./expect.js";
export { createTestFileSystem, mockFile } from "./fs.js";
export { t } from "./marked-template.js";
export { createLinterRuleTester, } from "./rule-tester.js";
export { extractCursor, extractSquiggles } from "./source-utils.js";
export { createTestHost, createTestRunner, findFilesFromPattern } from "./test-host.js";
export { createTestLibrary, createTestWrapper, expectTypeEquals, findTestPackageRoot, resolveVirtualPath, trimBlankLines, } from "./test-utils.js";
export { createTester } from "./tester.js";
//# sourceMappingURL=index.js.map