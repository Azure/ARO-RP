import { describe, expectTypeOf, it } from "vitest";
import { openapi } from "./openapi.js";
describe("openapi", () => {
  it("returns the correct type for load()", async () => {
    const result = await openapi().load({}).get();
    expectTypeOf(result.filesystem).toMatchTypeOf();
    expectTypeOf(result.valid).toMatchTypeOf();
  });
  it("returns the correct type for validate()", async () => {
    const result = await openapi().load({}).validate().get();
    expectTypeOf(result.valid).toMatchTypeOf();
  });
});
//# sourceMappingURL=openapi.test-d.js.map
