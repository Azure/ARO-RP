/** Spy on console.warn */
export declare const consoleWarnSpy: import("vitest").Mock<{
    (...data: any[]): void;
    (message?: any, ...optionalParams: any[]): void;
}>;
export declare let isConsoleWarnEnabled: boolean;
/** Spy on console.error */
export declare const consoleErrorSpy: import("vitest").Mock<{
    (...data: any[]): void;
    (message?: any, ...optionalParams: any[]): void;
}>;
export declare let isConsoleErrorEnabled: boolean;
/** Reset the spies */
export declare const resetConsoleSpies: () => void;
/** Helper to re-enable console warn checks */
export declare const enableConsoleWarn: () => boolean;
/** Helper to disable console warn checks */
export declare const disableConsoleWarn: () => boolean;
/** Helper to enable console error checks */
export declare const enableConsoleError: () => boolean;
/** Helper to disable console error checks */
export declare const disableConsoleError: () => boolean;
//# sourceMappingURL=console-spies.d.ts.map