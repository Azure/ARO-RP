export function startTimer() {
    const start = performance.now();
    return {
        end: () => {
            return performance.now() - start;
        },
    };
}
export function time(fn) {
    const timer = startTimer();
    fn();
    return timer.end();
}
export async function timeAsync(fn) {
    const timer = startTimer();
    await fn();
    return timer.end();
}
/** Perf utils  */
export const perf = {
    startTimer,
    time,
    timeAsync,
};
export function createPerfReporter() {
    const measures = {};
    function startReportingTimer(label) {
        const timer = startTimer();
        return {
            end: () => {
                const time = timer.end();
                measures[label] = time;
                return time;
            },
        };
    }
    return {
        startTimer: startReportingTimer,
        time: (label, fn) => {
            const timer = startReportingTimer(label);
            const result = fn();
            timer.end();
            return result;
        },
        timeAsync: async (label, fn) => {
            const timer = startReportingTimer(label);
            const result = await fn();
            timer.end();
            return result;
        },
        report: (label, duration) => {
            measures[label] = duration;
        },
        measures,
    };
}
//# sourceMappingURL=perf.js.map