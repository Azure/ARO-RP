import { formatLog } from "../core/logger/index.js";
import { compile as compileProgram, normalizePath, } from "../index.js";
import { debugLoggers } from "./debug.js";
import { trackActionFunc } from "./server-track-action-task.js";
/**
 * This class purely manages compilations triggered and the caches used underneath. It doesn't have or care about any extra knowledge beyond compile itself.
 * Instead compiler service would have more knowledge about the lsp scenarios to provide higher level service. It will be responsible to make sure compilation request
 * is built properly (i.e. figure out the correct entrypoint file, double check whether compilation result covers given document, and so on) and then trigger the actual
 * compilation with proper options through this class.
 */
export class ServerCompileManager {
    updateManager;
    compilerHost;
    log;
    // We may want a ttl for this
    trackerCache = new CompileCache((msg) => this.logDebug(msg));
    compileId = 0;
    logDebug;
    constructor(updateManager, compilerHost, log) {
        this.updateManager = updateManager;
        this.compilerHost = compilerHost;
        this.log = log;
        const debug = debugLoggers.serverCompile;
        this.logDebug = debug.enabled ? (msg) => this.log({ level: "debug", message: msg }) : () => { };
    }
    async compile(mainFile, compileOptions = {}, serverCompileOptions) {
        let cache;
        const curId = this.compileId++;
        const err = new Error();
        const lines = err.stack?.split("\n") ?? [];
        // log where the compiler is triggered, skip the first 2 frame and only log the next 2 if exists
        const stackLines = lines.slice(3, 5).join("\n");
        this.logDebug(`Server compile #${curId}: Triggered, version=${this.updateManager.docChangedVersion}, mode=${serverCompileOptions.mode}, mainFile=${mainFile}, from\n${stackLines}`);
        if (!serverCompileOptions.skipCache) {
            cache = this.trackerCache.get(mainFile, compileOptions, false /*hasCompleted*/, serverCompileOptions.mode);
            if (cache && cache.isUpToDate()) {
                this.logDebug(`Server compile #${curId}: Return cache at #${cache.getCompileId()}(${cache.getMode()})`);
                return cache;
            }
            else {
                this.logDebug(`Server compile #${curId}: Cache miss because ${cache ? `it's outdated with version ${cache.getVersion()}` : "no cache available"}`);
            }
        }
        let oldProgram = undefined;
        if (!serverCompileOptions.skipOldProgramFromCache) {
            const completedTracker = this.trackerCache.get(mainFile, compileOptions, true /* hasCompleted */, serverCompileOptions.mode);
            oldProgram = await completedTracker?.getCompileResult();
            this.logDebug(`Server compile #${curId}: Use old program from cache: ${oldProgram ? `from #${completedTracker?.getCompileId() ?? "undefined"}` : "n/a"}`);
        }
        const tracker = CompileTracker.compile(curId, this.updateManager, this.compilerHost, mainFile, compileOptions, serverCompileOptions.mode, oldProgram, (msg) => this.logDebug(msg));
        this.trackerCache.set(mainFile, compileOptions, tracker);
        return tracker;
    }
}
class CompileCache {
    log;
    coreCache;
    fullCache;
    constructor(log) {
        this.log = log;
        this.coreCache = new CompileCacheInternal(log);
        this.fullCache = new CompileCacheInternal(log);
    }
    get(entrypoint, compileOption, completedOnly, mode) {
        switch (mode) {
            case "core":
                // full cache can also be used for core, just return the latest one
                const core = this.coreCache.get(entrypoint, compileOption, completedOnly);
                const full = this.fullCache.get(entrypoint, compileOption, completedOnly);
                // only consider using full when it's already completed, otherwise, full compilation may take longer time
                if (core && full && full.isCompleted()) {
                    if (full.getVersion() > core.getVersion()) {
                        this.log(`Server compile: Using full cache (version ${full.getVersion()}) over core cache (version ${core.getVersion()})`);
                        return full;
                    }
                    else {
                        return core;
                    }
                }
                else if (core) {
                    return core;
                }
                else if (full && full.isCompleted()) {
                    this.log(`Server compile: Using full cache (version ${full.getVersion()}) over core cache which is unavailable`);
                    return full;
                }
                else {
                    return undefined;
                }
            case "full":
                return this.fullCache.get(entrypoint, compileOption, completedOnly);
            default:
                // not expected, just in case, and we don't want to terminate because of cache in prod
                if (process.env.NODE_ENV === "development") {
                    throw new Error(`Unexpected compile mode: ${mode}`);
                }
                return undefined;
        }
    }
    set(entrypoint, compileOption, tracker) {
        const mode = tracker.getMode();
        switch (mode) {
            case "core":
                this.coreCache.set(entrypoint, compileOption, tracker);
                break;
            case "full":
                this.fullCache.set(entrypoint, compileOption, tracker);
                break;
            default:
                if (process.env.NODE_ENV === "development") {
                    throw new Error(`Unexpected compile mode: ${mode}`);
                }
                return undefined;
        }
    }
}
class CompileCacheInternal {
    log;
    cacheLatest = new Map();
    /** Cache for completed compilation which is needed when we need an old program but the latest one is still in progress */
    cacheCompleted = new Map();
    constructor(log) {
        this.log = log;
    }
    getCacheKey(entrypoint, compileOption) {
        const normalizedEntrypoint = normalizePath(entrypoint);
        const normalizedOptions = {
            ...compileOption,
            outputDir: undefined,
        };
        return `${normalizedEntrypoint}\n${normalizedOptions}`;
    }
    /** Get the latest completed compilation */
    get(entrypoint, compileOption, 
    /**
     * Whether to only return completed compilations
     */
    completedOnly) {
        const key = this.getCacheKey(entrypoint, compileOption);
        const tracker = this.cacheLatest.get(key);
        // completed cache is from latest cache, so if latest is undefined, no need to check completed cache any more
        if (!completedOnly || !tracker) {
            return tracker;
        }
        if (tracker.isCompleted() === true) {
            return tracker;
        }
        return this.cacheCompleted.get(key);
    }
    set(entrypoint, compileOption, tracker) {
        const key = this.getCacheKey(entrypoint, compileOption);
        this.cacheLatest.set(key, tracker);
        const onComplete = () => {
            const cur = this.cacheCompleted.get(key);
            if (!cur || cur.getVersion() < tracker.getVersion()) {
                // There may be a race condition here when two onComplete occur at the same time(both of them pass the check and try to set the cache)
                // But the chance is very low, the cache status is still good (just set to a newer but not latest version), and the next compile can
                // likely fix it, so don't do special handling here for it
                this.cacheCompleted.set(key, tracker);
                this.log(`Server compile #${tracker.getCompileId()}: Completed Cache updated ( ${cur?.getVersion() ?? "n/a"} -> ${tracker.getVersion()} )`);
            }
        };
        tracker.getCompileResult().then(() => {
            onComplete();
        }, (err) => {
            onComplete();
        });
    }
}
export class CompileTracker {
    id;
    updateManager;
    entrypoint;
    compileResultPromise;
    version;
    startTime;
    mode;
    logs;
    endTime;
    static compile(id, updateManager, host, mainFile, options = {}, mode, oldProgram, log) {
        const sLogs = [];
        // Clone an compilerhost instance with my logSink so that we can collect compiler logs
        // for each compilation to it's own tracker
        // Usually we don't want to send out the log because we are compiling aggressively in the lsp
        // but in some case like the 'emit-code', we will want to send back the logs if there is any
        const myHost = {
            ...host,
            logSink: {
                log: (log) => {
                    const msg = formatLog(log, { excludeLogLevel: true });
                    const sLog = {
                        level: log.level,
                        message: msg,
                    };
                    sLogs.push(sLog);
                },
                trackAction: (message, finalMessage, action) => trackActionFunc((log) => sLogs.push(log), message, finalMessage, action),
            },
        };
        const myOption = mode === "core"
            ? {
                ...options,
                emit: [],
                linterRuleSet: undefined,
            }
            : {
                ...options,
            };
        const version = updateManager.docChangedVersion;
        const startTime = new Date();
        const p = compileProgram(myHost, mainFile, myOption, oldProgram);
        log(`Server compile #${id}: Start compilation at ${startTime.toISOString()}, version = ${version}, mainFile = ${mainFile}, mode = ${mode}`);
        return new CompileTracker(id, updateManager, mainFile, p, version, startTime, mode, sLogs, log);
    }
    constructor(id, updateManager, entrypoint, compileResultPromise, version, startTime, mode, logs, log) {
        this.id = id;
        this.updateManager = updateManager;
        this.entrypoint = entrypoint;
        this.compileResultPromise = compileResultPromise;
        this.version = version;
        this.startTime = startTime;
        this.mode = mode;
        this.logs = logs;
        this.startTime = startTime;
        const onComplete = () => {
            this.endTime = new Date();
            log(`Server compile #${this.getCompileId()}: Compilation finished at ${this.endTime.toISOString()}. Duration = ${this.endTime.getTime() - this.startTime.getTime()}ms`);
        };
        compileResultPromise.then((r) => {
            onComplete();
        }, (err) => {
            onComplete();
        });
    }
    getCompileId() {
        return this.id;
    }
    getEntryPoint() {
        return this.entrypoint;
    }
    async getCompileResult() {
        return await this.compileResultPromise;
    }
    getVersion() {
        return this.version;
    }
    getStartTime() {
        return this.startTime;
    }
    getEndTime() {
        return this.endTime;
    }
    isCompleted() {
        return this.endTime !== undefined;
    }
    isUpToDate() {
        return this.version === this.updateManager.docChangedVersion;
    }
    getMode() {
        return this.mode;
    }
    getLogs() {
        return this.logs;
    }
}
//# sourceMappingURL=server-compile-manager.js.map