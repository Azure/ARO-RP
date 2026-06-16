import { fileURLToPath, pathToFileURL } from "url";
import { CompilerPackageRoot, NodeHost } from "../core/node-host.js";
import { createSourceFile, getSourceFileKindFromExt } from "../core/source-file.js";
import { resolveVirtualPath } from "./fs.js";
import { TestHostError } from "./types.js";
export const StandardTestLibrary = {
    name: "@typespec/compiler",
    packageRoot: CompilerPackageRoot,
    files: [
        {
            virtualPath: "./node_modules/@typespec/compiler/dist/src",
            realDir: "./dist/src",
            pattern: "index.js",
        },
        {
            virtualPath: "./node_modules/@typespec/compiler/dist/src/lib",
            realDir: "./dist/src/lib",
            pattern: "**",
        },
        { virtualPath: "./node_modules/@typespec/compiler/lib", realDir: "./lib", pattern: "**" },
    ],
};
export function createTestCompilerHost(virtualFs, jsImports, options) {
    const libDirs = [resolveVirtualPath("./node_modules/@typespec/compiler/lib/std")];
    if (!options?.excludeTestLib) {
        libDirs.push(resolveVirtualPath("./node_modules/@typespec/compiler/test-lib"));
    }
    return {
        async readUrl(url) {
            const contents = virtualFs.get(url);
            if (contents === undefined) {
                throw new TestHostError(`File ${url} not found.`, "ENOENT");
            }
            return createSourceFile(contents, url);
        },
        async readFile(path) {
            path = resolveVirtualPath(path);
            const contents = virtualFs.get(path);
            if (contents === undefined) {
                throw new TestHostError(`File ${path} not found.`, "ENOENT");
            }
            return createSourceFile(contents, path);
        },
        async writeFile(path, content) {
            path = resolveVirtualPath(path);
            virtualFs.set(path, content);
        },
        async readDir(path) {
            path = resolveVirtualPath(path);
            const fileFolder = [...virtualFs.keys()]
                .filter((x) => x.startsWith(`${path}/`))
                .map((x) => x.replace(`${path}/`, ""))
                .map((x) => {
                const index = x.indexOf("/");
                return index !== -1 ? x.substring(0, index) : x;
            });
            return [...new Set(fileFolder)];
        },
        async rm(path, options) {
            path = resolveVirtualPath(path);
            if (options.recursive && !virtualFs.has(path)) {
                for (const key of virtualFs.keys()) {
                    if (key.startsWith(`${path}/`)) {
                        virtualFs.delete(key);
                    }
                }
            }
            else {
                virtualFs.delete(path);
            }
        },
        getLibDirs() {
            return libDirs;
        },
        getExecutionRoot() {
            return resolveVirtualPath("./node_modules/@typespec/compiler");
        },
        async getJsImport(path) {
            path = resolveVirtualPath(path);
            const module = jsImports.get(path);
            if (module === undefined) {
                throw new TestHostError(`Module ${path} not found`, "ERR_MODULE_NOT_FOUND");
            }
            return module;
        },
        async stat(path) {
            path = resolveVirtualPath(path);
            if (virtualFs.has(path)) {
                return {
                    isDirectory() {
                        return false;
                    },
                    isFile() {
                        return true;
                    },
                };
            }
            for (const fsPath of virtualFs.keys()) {
                if (fsPath.startsWith(path) && fsPath !== path) {
                    return {
                        isDirectory() {
                            return true;
                        },
                        isFile() {
                            return false;
                        },
                    };
                }
            }
            throw new TestHostError(`File ${path} not found`, "ENOENT");
        },
        // symlinks not supported in test-host
        async realpath(path) {
            return path;
        },
        getSourceFileKind: getSourceFileKindFromExt,
        logSink: { log: NodeHost.logSink.log },
        mkdirp: async (path) => path,
        fileURLToPath,
        pathToFileURL(path) {
            return pathToFileURL(path).href;
        },
        ...options?.compilerHostOverrides,
    };
}
export function addTestLib(fs) {
    const testTypes = {};
    // add test decorators
    fs.add("./node_modules/@typespec/compiler/test-lib/main.tsp", 'import "./test.js";');
    fs.addJsFile("./node_modules/@typespec/compiler/test-lib/test.js", {
        namespace: "TypeSpec",
        $test(_, target, nameLiteral) {
            let name = nameLiteral?.value;
            if (!name) {
                if ("name" in target && typeof target.name === "string") {
                    name = target.name;
                }
                else {
                    throw new Error("Need to specify a name for test type");
                }
            }
            testTypes[name] = target;
        },
    });
    return testTypes;
}
//# sourceMappingURL=test-compiler-host.js.map