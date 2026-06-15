import { getDirectoryPath } from "./path-utils.js";
const emittedFilesPerProgramKey = Symbol.for("TYPESPEC_EMITTED_FILES_PATHS");
if (globalThis[emittedFilesPerProgramKey] === undefined) {
    globalThis[emittedFilesPerProgramKey] = new WeakMap();
}
export function getEmittedFilesForProgram(program) {
    const existing = globalThis[emittedFilesPerProgramKey].get(program);
    if (existing)
        return existing;
    const val = [];
    globalThis[emittedFilesPerProgramKey].set(program, val);
    return val;
}
/**
 * Helper to emit a file.
 * @param program TypeSpec Program
 * @param options File Emitter options
 */
export async function emitFile(program, options) {
    // ensure path exists
    const outputFolder = getDirectoryPath(options.path);
    await program.host.mkdirp(outputFolder);
    const content = options.newLine && options.newLine === "crlf"
        ? options.content.replace(/(\r\n|\n|\r)/gm, "\r\n")
        : options.content;
    getEmittedFilesForProgram(program).push(options.path);
    return await program.host.writeFile(options.path, content);
}
//# sourceMappingURL=emitter-utils.js.map