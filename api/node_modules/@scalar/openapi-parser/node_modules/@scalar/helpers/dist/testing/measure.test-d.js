import { describe, vi } from "vitest";
import { measureAsync, measureSync } from "./measure.js";
const asyncFunction = vi.fn();
describe("measureAsync types", async () => {
  await measureAsync("test", asyncFunction);
  await measureAsync("test", () => 1);
});
describe("measureSync types", () => {
  measureSync("test", () => 1);
  measureSync("test", asyncFunction);
});
//# sourceMappingURL=measure.test-d.js.map
