import { vi } from "vitest";
const consoleWarnSpy = vi.spyOn(console, "warn");
let isConsoleWarnEnabled = false;
const consoleErrorSpy = vi.spyOn(console, "error");
let isConsoleErrorEnabled = false;
const resetConsoleSpies = () => {
  consoleWarnSpy.mockClear();
  consoleErrorSpy.mockClear();
};
const enableConsoleWarn = () => isConsoleWarnEnabled = true;
const disableConsoleWarn = () => isConsoleWarnEnabled = false;
const enableConsoleError = () => isConsoleErrorEnabled = true;
const disableConsoleError = () => isConsoleErrorEnabled = false;
export {
  consoleErrorSpy,
  consoleWarnSpy,
  disableConsoleError,
  disableConsoleWarn,
  enableConsoleError,
  enableConsoleWarn,
  isConsoleErrorEnabled,
  isConsoleWarnEnabled,
  resetConsoleSpies
};
//# sourceMappingURL=console-spies.js.map
