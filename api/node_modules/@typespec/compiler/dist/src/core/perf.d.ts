import type { PerfReporter, Timer } from "./types.js";
export declare function startTimer(): Timer;
export declare function time(fn: () => void): number;
export declare function timeAsync(fn: () => Promise<void>): Promise<number>;
/** Perf utils  */
export declare const perf: {
    startTimer: typeof startTimer;
    time: typeof time;
    timeAsync: typeof timeAsync;
};
export declare function createPerfReporter(): PerfReporter;
//# sourceMappingURL=perf.d.ts.map