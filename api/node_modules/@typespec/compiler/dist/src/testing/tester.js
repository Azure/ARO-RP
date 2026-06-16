import { realpath } from "fs/promises";
import { pathToFileURL } from "url";
import { compilerAssert } from "../core/diagnostics.js";
import { getEntityName } from "../core/helpers/type-name-utils.js";
import { NodeHost } from "../core/node-host.js";
import { getIdentifierContext, getNodeAtPosition } from "../core/parser.js";
import { getRelativePathFromDirectory, joinPaths, resolvePath } from "../core/path-utils.js";
import { compile as coreCompile } from "../core/program.js";
import { createSourceLoader } from "../core/source-loader.js";
import { NoTarget } from "../core/types.js";
import { resolveModule } from "../module-resolver/module-resolver.js";
import { NodePackageResolver } from "../module-resolver/node-package-resolver.js";
import { parseNodeModuleSpecifier } from "../module-resolver/utils.js";
import { $ } from "../typekit/index.js";
import { expectDiagnosticEmpty } from "./expect.js";
import { extractMarkers } from "./fourslash.js";
import { createTestFileSystem } from "./fs.js";
import { TemplateWithMarkers } from "./marked-template.js";
import { StandardTestLibrary, addTestLib } from "./test-compiler-host.js";
import { resolveVirtualPath } from "./test-utils.js";
export function createTester(base, options) {
    return createTesterInternal({
        fs: once(() => createTesterFs(base, options)),
        libraries: options.libraries,
    });
}
function once(fn) {
    let load;
    return () => {
        if (load)
            return load;
        load = fn();
        return load;
    };
}
async function createTesterFs(base, options) {
    const fs = createTestFileSystem();
    const host = options.host ?? {
        ...NodeHost,
        // We want to keep the original path in the file map but we do still want to resolve the full path when loading JS to prevent duplicate imports.
        realpath: async (x) => x,
        getJsImport: async (path) => {
            return await import(pathToFileURL(await realpath(path)).href);
        },
    };
    const sl = await createSourceLoader(host);
    const selfName = JSON.parse((await host.readFile(resolvePath(base, "package.json"))).text).name;
    const moduleHost = {
        realpath: async (x) => x,
        stat: host.stat,
        readFile: async (path) => {
            const file = await host.readFile(path);
            return file.text;
        },
    };
    const nodePackageResolver = new NodePackageResolver(moduleHost);
    for (const lib of options.libraries) {
        const specifier = parseNodeModuleSpecifier(lib);
        if (specifier === null) {
            throw new Error(`Library imports must be bare module specifiers. Got: ${lib}`);
        }
        if (specifier.subPath !== "") {
            // eslint-disable-next-line no-console
            console.warn(`Warning: Defining a subpath '${lib}' of a library is unnecessary. Just import the library. Ignoring the subpath in '${lib}'`);
            continue;
        }
        const pkg = await nodePackageResolver.resolve(specifier.packageName, base);
        for (const key of Object.keys(pkg?.exports ?? {})) {
            const spec = resolvePath(specifier.packageName, key);
            await sl.importPath(spec, NoTarget, base);
        }
        // We also need to load the library js entrypoint for emitters and linters.
        const resolved = await resolveModule(moduleHost, lib, {
            baseDir: base,
            conditions: ["import", "default"],
        });
        if (resolved.type === "module") {
            const virtualPath = computeRelativePath(lib, resolved.mainFile);
            fs.addJsFile(virtualPath, await host.getJsImport(resolved.mainFile));
            fs.add(resolvePath("node_modules", lib, "package.json"), resolved.manifest.file.text);
        }
    }
    await fs.addTypeSpecLibrary(StandardTestLibrary);
    function computeVirtualPath(file) {
        const context = sl.resolution.locationContexts.get(file);
        compilerAssert(context?.type === "library", `Unexpected: all source files should be in a library but ${file.path} was in '${context?.type}'`);
        return computeRelativePath(context.metadata.name, file.path);
    }
    function computeRelativePath(libName, realPath) {
        const relativePath = getRelativePathFromDirectory(base, realPath, false);
        if (libName === selfName) {
            return joinPaths("node_modules", selfName, relativePath);
        }
        else {
            return relativePath;
        }
    }
    for (const file of sl.resolution.sourceFiles.values()) {
        const relativePath = computeVirtualPath(file.file);
        fs.add(resolveVirtualPath(relativePath), file.file.text);
    }
    for (const file of sl.resolution.jsSourceFiles.values()) {
        const relativePath = computeVirtualPath(file.file);
        fs.addJsFile(resolveVirtualPath(relativePath), file.esmExports);
    }
    for (const [path, lib] of sl.resolution.loadedLibraries) {
        fs.add(resolvePath("node_modules", path, "package.json"), lib.manifest.file.text);
    }
    fs.freeze();
    return fs;
}
function createTesterBuilder(params, create) {
    return {
        files,
        wrap,
        importLibraries,
        import: importFn,
        using,
    };
    function files(files) {
        const fs = async () => {
            const fs = (await params.fs()).clone();
            for (const [name, value] of Object.entries(files)) {
                fs.add(name, value);
            }
            fs.freeze();
            return fs;
        };
        return create({
            ...params,
            fs,
        });
    }
    function wrap(fn) {
        return create({
            ...params,
            wraps: [...(params.wraps ?? []), fn],
        });
    }
    function importLibraries() {
        return create({
            ...params,
            imports: [...(params.imports ?? []), ...params.libraries],
        });
    }
    function importFn(...imports) {
        return create({
            ...params,
            imports: [...(params.imports ?? []), ...imports],
        });
    }
    function using(...usings) {
        return create({
            ...params,
            usings: [...(params.usings ?? []), ...usings],
        });
    }
}
function createTesterInternal(params) {
    return {
        ...createCompilable(async (...args) => {
            const instance = await createTesterInstance(params);
            return instance.compileAndDiagnose(...args);
        }),
        ...createTesterBuilder(params, createTesterInternal),
        emit,
        createInstance,
    };
    function emit(emitter, options) {
        return createEmitterTesterInternal({
            ...params,
            emitter,
            compilerOptions: options
                ? {
                    ...params.compilerOptions,
                    options: {
                        ...params.compilerOptions?.options,
                        [emitter]: options,
                    },
                }
                : params.compilerOptions,
        });
    }
    function createInstance() {
        return createTesterInstance(params);
    }
}
function createEmitterTesterInternal(params) {
    return {
        ...createCompilable(async (...args) => {
            const instance = await createEmitterTesterInstance(params);
            return instance.compileAndDiagnose(...args);
        }),
        ...createTesterBuilder(params, createEmitterTesterInternal),
        pipe: (cb) => {
            return createEmitterTesterInternal({
                ...params,
                outputProcess: async (result) => {
                    return params.outputProcess ? cb(params.outputProcess(result)) : cb(result);
                },
            });
        },
        createInstance: () => createEmitterTesterInstance(params),
    };
}
async function createEmitterTesterInstance(params) {
    const tester = await createTesterInstance(params);
    return {
        fs: tester.fs,
        ...createCompilable(compileAndDiagnose),
        get program() {
            return tester.program;
        },
        get $() {
            return tester.$;
        },
    };
    async function compileAndDiagnose(code, options) {
        if (options?.compilerOptions?.emit !== undefined) {
            throw new Error("Cannot set emit in options.");
        }
        const resolvedOptions = {
            ...options,
            compilerOptions: {
                ...params.compilerOptions,
                ...options?.compilerOptions,
                outputDir: "tsp-output",
                emit: [params.emitter],
            },
        };
        const [result, diagnostics] = await tester.compileAndDiagnose(code, resolvedOptions);
        const outputs = {};
        const outputDir = resolvedOptions.compilerOptions?.options?.[params.emitter]?.["emitter-output-dir"] ??
            resolveVirtualPath(resolvePath("tsp-output", params.emitter));
        for (const [name, value] of result.fs.fs) {
            if (name.startsWith(outputDir)) {
                const relativePath = name.slice(outputDir.length + 1);
                outputs[relativePath] = value;
            }
        }
        const prep = {
            ...result,
            outputs,
        };
        const final = params.outputProcess ? params.outputProcess(prep) : prep;
        return [final, diagnostics];
    }
}
async function createTesterInstance(params) {
    let savedProgram;
    let saved$;
    const fs = (await params.fs()).clone();
    return {
        ...createCompilable(compileAndDiagnose),
        fs,
        get program() {
            if (!savedProgram) {
                throw new Error("Program not initialized. Call compile first.");
            }
            return savedProgram;
        },
        get $() {
            if (!saved$) {
                throw new Error("Typekit not initialized. Call compile first.");
            }
            return saved$;
        },
    };
    function applyWraps(code, wraps) {
        for (const wrap of wraps) {
            code = wrap(code);
        }
        return code;
    }
    function addCode(fs, code) {
        const markerPositions = [];
        const markerConfigs = {};
        function addTsp(filename, value) {
            const codeStr = TemplateWithMarkers.is(value) ? value.code : value;
            const actualCode = filename === "main.tsp" ? wrapMain(codeStr) : codeStr;
            const markers = extractMarkers(actualCode);
            for (const marker of markers) {
                markerPositions.push({ ...marker, filename });
            }
            if (TemplateWithMarkers.is(value)) {
                for (const [markerName, markerConfig] of Object.entries(value.markers)) {
                    if (markerConfig) {
                        markerConfigs[markerName] = markerConfig;
                    }
                }
            }
            fs.addTypeSpecFile(filename, actualCode);
        }
        const files = typeof code === "string" || TemplateWithMarkers.is(code) ? { "main.tsp": code } : code;
        for (const [name, value] of Object.entries(files)) {
            addTsp(name, value);
        }
        return { markerPositions, markerConfigs };
    }
    function wrapMain(code) {
        const imports = (params.imports ?? []).map((x) => `import "${x}";`);
        const usings = (params.usings ?? []).map((x) => `using ${x};`);
        const actualCode = [
            ...imports,
            ...usings,
            params.wraps ? applyWraps(code, params.wraps) : code,
        ].join("\n");
        return actualCode;
    }
    async function compileAndDiagnose(code, options) {
        const typesCollected = addTestLib(fs);
        const { markerPositions, markerConfigs } = addCode(fs, code);
        const program = await coreCompile(fs.compilerHost, resolveVirtualPath("main.tsp"), options?.compilerOptions);
        savedProgram = program;
        saved$ = $(program);
        const entities = extractMarkedEntities(program, markerPositions, markerConfigs);
        return [
            {
                program,
                $: saved$,
                fs,
                pos: Object.fromEntries(markerPositions.map((x) => [x.name, x])),
                ...typesCollected,
                ...entities,
            },
            program.diagnostics,
        ];
    }
}
function extractMarkedEntities(program, markerPositions, markerConfigs) {
    const entities = {};
    for (const marker of markerPositions) {
        const file = program.sourceFiles.get(resolveVirtualPath(marker.filename));
        if (!file) {
            throw new Error(`Couldn't find ${resolveVirtualPath(marker.filename)} in program`);
        }
        const { name, pos } = marker;
        const markerConfig = markerConfigs[name];
        const node = getNodeAtPosition(file, pos);
        if (!node) {
            throw new Error(`Could not find node at ${pos}`);
        }
        const { node: contextNode } = getIdentifierContext(node);
        if (contextNode === undefined) {
            throw new Error(`Could not find context node for ${name} at ${pos}. File content: ${file.file.text}`);
        }
        const entity = program.checker.getTypeOrValueForNode(contextNode);
        if (entity === null) {
            throw new Error(`Expected ${name} to be of entity kind ${markerConfig?.entityKind} but got null (Means a value failed to resolve) at ${pos}`);
        }
        if (markerConfig) {
            const { entityKind, kind, valueKind } = markerConfig;
            if (entity.entityKind !== entityKind) {
                throw new Error(`Expected ${name} to be of entity kind ${entityKind} but got (${entity?.entityKind}) ${getEntityName(entity)} at ${pos}`);
            }
            if (entity.entityKind === "Type" && kind !== undefined && entity.kind !== kind) {
                throw new Error(`Expected ${name} to be of kind ${kind} but got (${entity.kind}) ${getEntityName(entity)} at ${pos}`);
            }
            else if (entity?.entityKind === "Value" &&
                valueKind !== undefined &&
                entity.valueKind !== valueKind) {
                throw new Error(`Expected ${name} to be of value kind ${valueKind} but got (${entity.valueKind}) ${getEntityName(entity)} at ${pos}`);
            }
        }
        entities[name] = entity;
    }
    return entities;
}
function createCompilable(fn) {
    return {
        compileAndDiagnose: fn,
        compile: async (...args) => {
            const [result, diagnostics] = await fn(...args);
            expectDiagnosticEmpty(diagnostics);
            return result;
        },
        diagnose: async (...args) => {
            const [_, diagnostics] = await fn(...args);
            return diagnostics;
        },
    };
}
//# sourceMappingURL=tester.js.map