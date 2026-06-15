import { TextDocumentIdentifier } from "vscode-languageserver";
import { TextDocument } from "vscode-languageserver-textdocument";
import { ServerLog } from "./types.js";
interface PendingUpdate {
    latest: TextDocument | TextDocumentIdentifier;
    latestUpdateTimestamp: number;
}
export type UpdateType = "opened" | "changed" | "closed" | "renamed";
type UpdateCallback<T> = (updates: PendingUpdate[], triggeredBy: TextDocument | TextDocumentIdentifier) => Promise<T>;
/**
 * Track file updates and recompile the affected files after some debounce time.
 * T will be returned if the scheduled update is triggered eventually, but if a newer scheduleUpdate is triggered, the previous ones will be cancelled and return undefined.
 */
export declare class UpdateManager<T = void> {
    #private;
    private name;
    private _log;
    readonly getDebounceDelay: () => number;
    /**
     *
     * @param name For logging purpose, identify different update manager if there are multiple
     * @param log
     */
    constructor(name: string, log: (sl: ServerLog) => void, getDebounceDelay?: () => number);
    /**
     * Callback will only be invoked after start() is called.
     * We need to start explicitly to avoid compiling with incorrect settings when the lsp hasn't fully initialized (the client settings are not loaded yet.)
     */
    start(): void;
    setCallback(callback: UpdateCallback<T>): void;
    get docChangedVersion(): number;
    private bumpDocChangedVersion;
    private pushDocChangedTimestamp;
    private readonly WINDOW;
    private readonly DEFAULT_DELAY;
    private readonly DELAY_CANDIDATES;
    private getWindowedDocChangedTimesteps;
    private getAdaptiveDebounceDelay;
    /**
     * T will be returned if the schedule is triggered eventually, if a newer scheduleUpdate
     *  occurs before the debounce time, the previous ones will be cancelled and return undefined.
     */
    scheduleUpdate(document: TextDocument | TextDocumentIdentifier, updateType: UpdateType): Promise<T | undefined>;
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
export declare function debounceThrottle<T, P>(fn: (arg: P) => T | Promise<T>, getFnStatus: () => "ready" | "pending", getDelay: () => number, log: (sl: ServerLog) => void): (arg: P) => Promise<T | undefined>;
export {};
//# sourceMappingURL=update-manager.d.ts.map