import { DiagnosticSeverity, Range } from "vscode-languageserver";
import { defaultConfig, findTypeSpecConfigPath, loadTypeSpecConfigFile, TypeSpecConfigFilename, } from "../config/config-loader.js";
import { resolveOptionsFromConfig } from "../config/config-to-options.js";
import { builtInLinterRule_UnusedTemplateParameter } from "../core/linter-rules/unused-template-parameter.rule.js";
import { builtInLinterRule_UnusedUsing } from "../core/linter-rules/unused-using.rule.js";
import { builtInLinterLibraryName } from "../core/linter.js";
import { parse } from "../core/parser.js";
import { getBaseFileName, getDirectoryPath } from "../core/path-utils.js";
import { deepClone, distinctArray } from "../utils/misc.js";
import { getLocationInYamlScript } from "../yaml/diagnostics.js";
import { parseYaml } from "../yaml/parser.js";
import { serverOptions } from "./constants.js";
import { debugLoggers } from "./debug.js";
import { resolveEntrypointFile } from "./entrypoint-resolver.js";
import { ServerCompileManager, } from "./server-compile-manager.js";
export function createCompileService({ compilerHost, serverHost, fileService, fileSystemCache, updateManager, log, clientConfigsProvider, }) {
    const eventListeners = new Map();
    const compileManager = new ServerCompileManager(updateManager, compilerHost, log);
    let configFilePath;
    const debug = debugLoggers.compileConfig;
    const logDebug = debug.enabled ? log : () => { };
    return { compile, getScript, on, notifyChange, getMainFileForDocument };
    function on(event, listener) {
        eventListeners.set(event, listener);
    }
    function notify(event, ...args) {
        const listener = eventListeners.get(event);
        if (listener) {
            void listener(...args);
        }
    }
    function notifyChange(document, updateType) {
        void updateManager.scheduleUpdate(document, updateType);
    }
    /**
     * Compile the given document.
     * First, the main.tsp file will be obtained for compilation.
     * If the current document is not the main.tsp file or not included in the compilation starting from the main file found,
     * the current document will be recompiled and returned as part of the result.
     * Otherwise, the compilation of main.tsp will be returned as part of the result.
     * @param document The document to compile. tsp file that is open or not opened in workspace.
     * @returns see {@link CompileResult} for more details.
     */
    async function compile(document, additionalOptions, serverCompileOptions) {
        const path = await fileService.getPath(document);
        const pathBaseName = getBaseFileName(path);
        if (!path.endsWith(".tsp") && pathBaseName !== TypeSpecConfigFilename) {
            return undefined;
        }
        const mainFile = await getMainFileForDocument(path);
        if (mainFile === undefined) {
            logDebug({ level: "debug", message: `failed to resolve main file for ${path}` });
            return undefined;
        }
        if (!mainFile.endsWith(".tsp")) {
            return undefined;
        }
        const config = await getConfig(mainFile);
        configFilePath = config.filename;
        logDebug({ level: "debug", message: `config resolved`, detail: config });
        const [optionsFromConfig, _] = resolveOptionsFromConfig(config, {
            cwd: getDirectoryPath(path),
        });
        // we need to keep the optionsFromConfig unchanged which will be returned in CompileResult
        const clone = deepClone(optionsFromConfig);
        const options = {
            ...clone,
            ...serverOptions,
            ...(additionalOptions ?? {}),
        };
        // If emit is set in additionalOptions, use this setting first
        // otherwise, obtain the `typespec.lsp.emit` configuration from clientConfigsProvider
        if (additionalOptions?.emit === undefined) {
            const configEmits = clientConfigsProvider?.config?.lsp?.emit;
            const CONFIG_DEFAULTS = "<config:defaults>";
            if (configEmits) {
                if (configEmits.find((e) => e === CONFIG_DEFAULTS)) {
                    // keep the emits in tspconfig only when user configs "<config:defaults>", and append other emits from vscode settings if there is any
                    options.emit = distinctArray([...(options.emit ?? []), ...configEmits.filter((e) => e !== CONFIG_DEFAULTS)], (s) => s);
                }
                else {
                    // use the configured emits if no "<config:defaults>" is found
                    options.emit = configEmits;
                }
            }
            else {
                // by default, exclude emits from compile which are not useful in most case but may cause perf issue
                // User can set ['<config:defaults>'] to opt-in
                options.emit = [];
            }
        }
        // add linter rule for unused using if user didn't configure it explicitly
        const unusedUsingRule = `${builtInLinterLibraryName}/${builtInLinterRule_UnusedUsing}`;
        if (options.linterRuleSet?.enable?.[unusedUsingRule] === undefined &&
            options.linterRuleSet?.disable?.[unusedUsingRule] === undefined) {
            options.linterRuleSet ??= {};
            options.linterRuleSet.enable ??= {};
            options.linterRuleSet.enable[unusedUsingRule] = true;
        }
        // add linter rule for unused template parameter if user didn't configure it explicitly
        const unusedTemplateParameterRule = `${builtInLinterLibraryName}/${builtInLinterRule_UnusedTemplateParameter}`;
        if (options.linterRuleSet?.enable?.[unusedTemplateParameterRule] === undefined &&
            options.linterRuleSet?.disable?.[unusedTemplateParameterRule] === undefined) {
            options.linterRuleSet ??= {};
            options.linterRuleSet.enable ??= {};
            options.linterRuleSet.enable[unusedTemplateParameterRule] = true;
        }
        const isCancelledOrOutOfDate = () => {
            return serverCompileOptions.isCancelled?.() || !fileService.upToDate(document);
        };
        if (isCancelledOrOutOfDate()) {
            return undefined;
        }
        let tracker;
        try {
            tracker = await compileManager.compile(mainFile, options, serverCompileOptions);
            let program = await tracker.getCompileResult();
            if (isCancelledOrOutOfDate()) {
                return undefined;
            }
            if (mainFile !== path &&
                !program.sourceFiles.has(path) &&
                pathBaseName !== TypeSpecConfigFilename) {
                // If the file that changed wasn't imported by anything from the main
                // file, retry using the file itself as the main file.
                logDebug({
                    level: "debug",
                    message: `target file was not included in compiling, try to compile ${path} as main file directly`,
                });
                tracker = await compileManager.compile(path, options, serverCompileOptions);
                program = await tracker.getCompileResult();
            }
            if (isCancelledOrOutOfDate()) {
                return undefined;
            }
            const doc = "version" in document ? document : serverHost.getOpenDocumentByURL(document.uri);
            const script = program.sourceFiles.get(path);
            const result = { program, document: doc, script, optionsFromConfig, tracker };
            notify("compileEnd", result);
            return result;
        }
        catch (err) {
            if (serverHost.throwInternalErrors) {
                throw err;
            }
            let uri = document.uri;
            let range = Range.create(0, 0, 0, 0);
            if (err.name === "ExternalError" && err.info.kind === "emitter" && configFilePath) {
                const emitterName = err.info.metadata.name;
                const [yamlScript] = parseYaml(await serverHost.compilerHost.readFile(configFilePath));
                const target = getLocationInYamlScript(yamlScript, ["emit", emitterName], "key");
                if (target.pos === 0) {
                    logDebug({
                        level: "debug",
                        message: `Unexpected situation, can't find emitter '${emitterName}' in config file '${configFilePath}'`,
                    });
                }
                uri = fileService.getURL(configFilePath);
                const lineAndChar = target.file.getLineAndCharacterOfPosition(target.pos);
                range = Range.create(lineAndChar.line, lineAndChar.character, lineAndChar.line, lineAndChar.character + emitterName.length);
            }
            serverHost.sendDiagnostics({
                uri,
                diagnostics: [
                    {
                        severity: DiagnosticSeverity.Error,
                        range,
                        message: (err.name === "ExternalError"
                            ? "External compiler error!\n"
                            : `Internal compiler error!\nFile issue at https://github.com/microsoft/typespec\n\n`) +
                            err.stack,
                    },
                ],
            });
            return undefined;
        }
    }
    async function getConfig(mainFile) {
        const entrypointStat = await compilerHost.stat(mainFile);
        const lookupDir = entrypointStat.isDirectory() ? mainFile : getDirectoryPath(mainFile);
        const configPath = await findTypeSpecConfigPath(compilerHost, lookupDir, true);
        if (!configPath) {
            logDebug({
                level: "debug",
                message: `can't find path with config file, try to use default config`,
            });
            return { ...defaultConfig, projectRoot: getDirectoryPath(mainFile) };
        }
        const cached = await fileSystemCache.get(configPath);
        const deepCopy = (obj) => JSON.parse(JSON.stringify(obj));
        if (cached?.data) {
            return deepCopy(cached.data);
        }
        const config = await loadTypeSpecConfigFile(compilerHost, configPath);
        return deepCopy(config);
    }
    async function getScript(document) {
        const file = await compilerHost.readFile(await fileService.getPath(document));
        const cached = compilerHost.parseCache?.get(file);
        if (cached === undefined) {
            const parsed = parse(file, { docs: true, comments: true });
            compilerHost.parseCache?.set(file, parsed);
            return parsed;
        }
        else {
            return cached;
        }
    }
    /**
     * Infer the appropriate entry point (a.k.a. "main file") for analyzing a
     * change to the file at the given path. This is necessary because different
     * results can be obtained from compiling the same file with different entry
     * points.
     *
     * Priority is given to processing user-defined files as the entry point,
     * and it has the highest priority.
     *
     * Walk directory structure upwards looking for package.json with tspMain or
     * main.tsp file. Stop search when reaching a workspace root. If a root is
     * reached without finding an entry point, use the given path as its own
     * entry point.
     *
     * Untitled documents are always treated as their own entry points as they
     * do not exist in a directory that could pull them in via another entry
     * point.
     */
    async function getMainFileForDocument(path) {
        if (path.startsWith("untitled:")) {
            logDebug({
                level: "debug",
                message: `untitled document treated as its own main file: ${path}`,
            });
            return path;
        }
        const entrypoints = clientConfigsProvider?.config?.entrypoint;
        return resolveEntrypointFile(compilerHost, entrypoints, path, fileSystemCache, log);
    }
}
//# sourceMappingURL=compile-service.js.map