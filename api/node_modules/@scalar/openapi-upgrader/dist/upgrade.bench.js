import { bench, describe, expect } from "vitest";
import { upgradeFromTwoToThree } from "./2.0-to-3.0/upgrade-from-two-to-three.js";
import { upgrade } from "./upgrade.js";
const STRIPE = await fetch(
  "https://raw.githubusercontent.com/stripe/openapi/refs/heads/master/openapi/spec3.json"
).then((r) => r.json());
const PETSTORE = await fetch("https://petstore.swagger.io/v2/swagger.json").then((r) => r.json());
describe("upgrade", () => {
  describe("Petstore: Swagger 2.0 to OpenAPI 3.0", () => {
    bench("new", () => {
      expect(PETSTORE.swagger).toBe("2.0");
      const result = upgradeFromTwoToThree({ ...PETSTORE });
      expect(result.openapi).toBe("3.0.4");
    });
  });
  describe("Stripe: OpenAPI 3.0 to 3.1", () => {
    bench("new", () => {
      expect(STRIPE.openapi).toBe("3.0.0");
      const document = upgrade(STRIPE, "3.1");
      expect(document?.openapi).toBe("3.1.1");
    });
  });
  describe("Petstore: Swagger 2.0 to OpenAPI 3.1", () => {
    bench("new", () => {
      expect(PETSTORE.swagger).toBe("2.0");
      const document = upgrade(PETSTORE, "3.1");
      expect(document?.openapi).toBe("3.1.1");
    });
  });
});
//# sourceMappingURL=upgrade.bench.js.map
