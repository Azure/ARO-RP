import { debugLoggers } from "./debug.js";
/**
 * Track file updates and recompile the affected files after some debounce time.
 * T will be returned if the scheduled update is triggered eventually, but if a newer scheduleUpdate is triggered, the previous ones will be cancelled and return undefined.
 */
export class UpdateManager {
    name;
    #pendingUpdates = new Map();
    #updateCb;
    // overall version which should be bumped for any actual doc change
    #docChangedVersion = 0;
    #scheduleBatchUpdate;
    #docChangedTimesteps = [];
    #isStarted = false;
    _log;
    getDebounceDelay;
    /**
     *
     * @param name For logging purpose, identify different update manager if there are multiple
     * @param log
     */
    constructor(name, log, getDebounceDelay) {
        this.name = name;
        const debug = debugLoggers.updateManager;
        this._log = debug.enabled
            ? (sl) => {
                log({ ...sl, message: `#FromUpdateManager(${this.name}): ${sl.message}` });
            }
            : () => { };
        // Set the debounce delay function once during construction
        this.getDebounceDelay = getDebounceDelay ?? this.getAdaptiveDebounceDelay;
        this.#scheduleBatchUpdate = debounceThrottle(async (arg) => {
            const updates = this.#pendingUpdates;
            this.#pendingUpdates = new Map();
            return await this.#update(Array.from(updates.values()), arg);
        }, () => (this.#isStarted ? "ready" : "pending"), this.getDebounceDelay, this._log);
    }
    /**
     * Callback will only be invoked after start() is called.
     * We need to start explicitly to avoid compiling with incorrect settings when the lsp hasn't fully initialized (the client settings are not loaded yet.)
     */
    start() {
        this.#isStarted = true;
    }
    setCallback(callback) {
        this.#updateCb = callback;
    }
    get docChangedVersion() {
        return this.#docChangedVersion;
    }
    bumpDocChangedVersion() {
        this.#docChangedVersion++;
    }
    pushDocChangedTimestamp() {
        const now = Date.now();
        this.#docChangedTimesteps = [...this.getWindowedDocChangedTimesteps(), now];
    }
    WINDOW = 5000;
    DEFAULT_DELAY = 500;
    // Provider different debounce delay according to whether usr are actively typing, increase the delay if so to avoid unnecessary invoke
    // The category below is suggested from AI, may adjust as needed in the future
    DELAY_CANDIDATES = [
        // IMPORTANT: sort by frequencyInWindow desc, we will pick the first match
        {
            // active typing
            frequencyInWindow: 20,
            delay: 1000,
        },
        {
            // moderate typing
            frequencyInWindow: 10,
            delay: 800,
        },
        {
            // light typing
            frequencyInWindow: 0,
            delay: this.DEFAULT_DELAY,
        },
    ];
    getWindowedDocChangedTimesteps() {
        const now = Date.now();
        return this.#docChangedTimesteps.filter((timestamp) => {
            const age = now - timestamp;
            return age < this.WINDOW;
        });
    }
    getAdaptiveDebounceDelay = () => {
        const frequent = this.getWindowedDocChangedTimesteps().length;
        for (const c of this.DELAY_CANDIDATES) {
            if (frequent >= c.frequencyInWindow) {
                return c.delay;
            }
        }
        return this.DEFAULT_DELAY;
    };
    /**
     * T will be returned if the schedule is triggered eventually, if a newer scheduleUpdate
     *  occurs before the debounce time, the previous ones will be cancelled and return undefined.
     */
    scheduleUpdate(document, updateType) {
        if (updateType === "changed" || updateType === "renamed") {
            // only bump this when the file is actually changed
            // skip open
            this.bumpDocChangedVersion();
            this.pushDocChangedTimestamp();
        }
        const existing = this.#pendingUpdates.get(document.uri);
        if (existing === undefined) {
            this.#pendingUpdates.set(document.uri, {
                latest: document,
                latestUpdateTimestamp: Date.now(),
            });
        }
        else {
            existing.latest = document;
            existing.latestUpdateTimestamp = Date.now();
        }
        return this.#scheduleBatchUpdate(document);
    }
    async #update(updates, arg) {
        if (this.#updateCb === undefined) {
            this._log({
                level: "warning",
                message: `No update callback registered, skip invoking update.`,
            });
            return undefined;
        }
        return await this.#updateCb(updates, arg);
    }
}
/**
 * Debounces a function but also waits at minimum the specified number of milliseconds until
 * the next invocation. This avoids needless calls when a synchronous call (like diagnostics)
 * took too long and the whole timeout of the next call was eaten up already.
 *
 * @param fn The function
 * @param getFnStatus Fn will only be called when this returns "ready"
 * @param milliseconds Number of milliseconds to debounce/throttle
 */
export function debounceThrottle(fn, getFnStatus, getDelay, log) {
    let timeout;
    let lastInvocation = undefined;
    let executingCount = 0;
    let debounceExecutionId = 0;
    const executionPromises = new Map();
    const UPDATE_PARALLEL_LIMIT = 2;
    function maybeCall(arg) {
        const promise = new DeferredPromise();
        const curId = debounceExecutionId++;
        executionPromises.set(curId, promise);
        maybeCallInternal(curId, arg, promise);
        return promise.getPromise();
    }
    /** Clear all promises before the given id to make sure we are not leaking anything */
    function clearPromisesBefore(id) {
        // clear all promises before with id < the given id
        for (const k of executionPromises.keys()) {
            if (k < id) {
                executionPromises.get(k)?.resolvePromise(undefined);
                executionPromises.delete(k);
            }
        }
    }
    function maybeCallInternal(id, arg, promise) {
        clearTimeout(timeout);
        clearPromisesBefore(id);
        timeout = setTimeout(async () => {
            const delay = getDelay();
            const tooSoon = lastInvocation !== undefined && Date.now() - lastInvocation < delay;
            const notReady = getFnStatus() !== "ready";
            const tooManyParallel = executingCount >= UPDATE_PARALLEL_LIMIT;
            if (notReady || tooSoon || tooManyParallel) {
                maybeCallInternal(id, arg, promise);
                return;
            }
            const s = new Date();
            try {
                executingCount++;
                log({
                    level: "debug",
                    message: `Starting debounce execution #${id} at ${s.toISOString()}. Current parallel count: ${executingCount}`,
                });
                const r = await fn(arg);
                promise.resolvePromise(r);
            }
            catch (e) {
                promise.rejectPromise(e);
            }
            finally {
                executionPromises.delete(id);
                executingCount--;
                const e = new Date();
                log({
                    level: "debug",
                    message: `Finish debounce execution #${id} at ${e.toISOString()}, duration=${e.getTime() - s.getTime()}. Current parallel count: ${executingCount}`,
                });
            }
            lastInvocation = Date.now();
        }, getDelay());
    }
    return maybeCall;
}
class DeferredPromise {
    #promise;
    #resolve;
    #reject;
    constructor() {
        this.#promise = new Promise((res, rej) => {
            this.#resolve = res;
            this.#reject = rej;
        });
    }
    getPromise() {
        return this.#promise;
    }
    resolvePromise(value) {
        this.#resolve(value);
    }
    rejectPromise(reason) {
        this.#reject(reason);
    }
}
//# sourceMappingURL=update-manager.js.map