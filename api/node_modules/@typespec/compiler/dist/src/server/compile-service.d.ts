import { TextDocumentIdentifier } from "vscode-languageserver";
import { TextDocument } from "vscode-languageserver-textdocument";
import { CompilerOptions } from "../core/options.js";
import type { CompilerHost, TypeSpecScriptNode } from "../core/types.js";
import { ClientConfigProvider } from "./client-config-provider.js";
import { FileService } from "./file-service.js";
import { FileSystemCache } from "./file-system-cache.js";
import { ServerCompileOptions } from "./server-compile-manager.js";
import { CompileResult, ServerHost, ServerLog } from "./types.js";
import { UpdateManager, UpdateType } from "./update-manager.js";
/**
 * Service managing compilation/caching of different TypeSpec projects
 */
export interface CompileService {
    /**
     * Compile the given document.
     *
     * Compilation can be aborted for various reasons:
     * - By the time the compilation start the document is already out of date.
     *
     * @param document The document to compile. This is not necessarily the entrypoint, compile will try to guess which entrypoint to compile to include this one.
     * @returns the compiled result or undefined if compilation was aborted.
     */
    compile(document: TextDocument | TextDocumentIdentifier, additionalOptions: CompilerOptions | undefined, serverCompileOptions: ServerCompileOptions): Promise<CompileResult | undefined>;
    /**
     * Load the AST for the given document.
     * @param document The document to load the AST for.
     */
    getScript(document: TextDocument | TextDocumentIdentifier): Promise<TypeSpecScriptNode>;
    /**
     * Notify the service that the given document has changed and a compilation should be requested.
     * It will recompile after a debounce timer so we don't recompile on every keystroke.
     * @param document Document that changed.
     */
    notifyChange(document: TextDocument | TextDocumentIdentifier, updateType: UpdateType): void;
    on(event: "compileEnd", listener: (result: CompileResult) => void): void;
    getMainFileForDocument(path: string): Promise<string | undefined>;
}
export interface CompileServiceOptions {
    readonly fileSystemCache: FileSystemCache;
    readonly fileService: FileService;
    readonly serverHost: ServerHost;
    readonly compilerHost: CompilerHost;
    readonly updateManager: UpdateManager;
    readonly log: (log: ServerLog) => void;
    readonly clientConfigsProvider?: ClientConfigProvider;
}
export declare function createCompileService({ compilerHost, serverHost, fileService, fileSystemCache, updateManager, log, clientConfigsProvider, }: CompileServiceOptions): CompileService;
//# sourceMappingURL=compile-service.d.ts.map