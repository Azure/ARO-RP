import { CompilerHost, CompilerOptions, Program, ServerLog } from "../index.js";
import { UpdateManager } from "./update-manager.js";
/**
 * core: linter and emitter will be set to [] when trigger compilation
 * full: compile as it is
 */
export type ServerCompileMode = "core" | "full";
export interface ServerCompileOptions {
    skipCache?: boolean;
    skipOldProgramFromCache?: boolean;
    /** Make this non-optional on purpose so that the caller needs to determine the correct mode to compile explicitly */
    mode: ServerCompileMode;
    /** A simple func to check if the compilation is cancelled. After compiler supports cancellation, we may want to change to use it */
    isCancelled?: () => boolean;
}
/**
 * This class purely manages compilations triggered and the caches used underneath. It doesn't have or care about any extra knowledge beyond compile itself.
 * Instead compiler service would have more knowledge about the lsp scenarios to provide higher level service. It will be responsible to make sure compilation request
 * is built properly (i.e. figure out the correct entrypoint file, double check whether compilation result covers given document, and so on) and then trigger the actual
 * compilation with proper options through this class.
 */
export declare class ServerCompileManager {
    private updateManager;
    private compilerHost;
    private log;
    private trackerCache;
    private compileId;
    private logDebug;
    constructor(updateManager: UpdateManager, compilerHost: CompilerHost, log: (log: ServerLog) => void);
    compile(mainFile: string, compileOptions: CompilerOptions | undefined, serverCompileOptions: ServerCompileOptions): Promise<CompileTracker>;
}
export declare class CompileTracker {
    private id;
    private updateManager;
    private entrypoint;
    private compileResultPromise;
    private version;
    private startTime;
    private mode;
    private logs;
    private endTime;
    static compile(id: number, updateManager: UpdateManager, host: CompilerHost, mainFile: string, options: CompilerOptions | undefined, mode: ServerCompileMode, oldProgram: Program | undefined, log: (msg: string) => void): CompileTracker;
    private constructor();
    getCompileId(): number;
    getEntryPoint(): string;
    getCompileResult(): Promise<Program>;
    getVersion(): number;
    getStartTime(): Date;
    getEndTime(): Date | undefined;
    isCompleted(): boolean;
    isUpToDate(): boolean;
    getMode(): ServerCompileMode;
    getLogs(): ServerLog[];
}
//# sourceMappingURL=server-compile-manager.d.ts.map