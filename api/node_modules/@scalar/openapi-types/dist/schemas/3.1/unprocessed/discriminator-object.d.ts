/**
 * Discriminator Object
 *
 * When request bodies or response payloads may be one of a number of different schemas, a Discriminator Object gives a
 * hint about the expected schema of the document. This hint can be used to aid in serialization, deserialization, and
 * validation. The Discriminator Object does this by implicitly or explicitly associating the possible values of a named
 * property with alternative schemas.
 *
 * Note that discriminator MUST NOT change the validation outcome of the schema.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#discriminator-object
 */
export declare const DiscriminatorObjectSchema: import("zod").ZodObject<{
    propertyName: import("zod").ZodString;
    mapping: import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodString>>;
}, import("zod/v4/core").$strip>;
//# sourceMappingURL=discriminator-object.d.ts.map