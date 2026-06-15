import { describe, expectTypeOf, it } from "vitest";
describe("OpenAPI", () => {
  it("has a generic type", () => {
    const document = {
      // anything
    };
    expectTypeOf(document).toEqualTypeOf();
  });
  it("narrows it down to Swagger 2.0", () => {
    const document = {
      swagger: "2.0"
    };
    expectTypeOf(document).toEqualTypeOf();
  });
  it("narrows it down to OpenAPI 3.0.0", () => {
    const document = {
      openapi: "3.0.0"
    };
    expectTypeOf(document).toEqualTypeOf();
  });
  it("narrows it down to OpenAPI 3.0.4", () => {
    const document = {
      openapi: "3.0.4"
    };
    expectTypeOf(document).toEqualTypeOf();
  });
  it("narrows it down to OpenAPI 3.1.0", () => {
    const document = {
      openapi: "3.1.0"
    };
    expectTypeOf(document).toEqualTypeOf();
  });
  it("narrows it down to OpenAPI 3.1.1", () => {
    const document = {
      openapi: "3.1.1"
    };
    expectTypeOf(document).toEqualTypeOf();
  });
  it("narrows it down to OpenAPI 3.1.2", () => {
    const document = {
      openapi: "3.1.2"
    };
    expectTypeOf(document).toEqualTypeOf();
  });
  it("narrows it down to OpenAPI 3.2.0", () => {
    const document = {
      openapi: "3.2.0"
    };
    expectTypeOf(document).toEqualTypeOf();
  });
  it("types a custom extension", () => {
    const document = {};
    expectTypeOf(document["random-attribute"]).toEqualTypeOf();
    expectTypeOf(document["x-custom"]).toEqualTypeOf();
  });
  it("has a HttpMethod type", () => {
    expectTypeOf("get").toEqualTypeOf();
    assertType("NOT_A_METHOD");
  });
});
//# sourceMappingURL=openapi-types.test-d.js.map
