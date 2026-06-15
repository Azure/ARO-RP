/**
 *
 * XML Object
 *
 * A metadata object that allows for more fine-tuned XML model definitions.
 *
 * When using arrays, XML element names are not inferred (for singular/plural forms) and the name field SHOULD be used
 * to add that information. See examples for expected behavior.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#xml-object
 */
export declare const XmlObjectSchema: import("zod").ZodObject<{
    name: import("zod").ZodOptional<import("zod").ZodString>;
    namespace: import("zod").ZodOptional<import("zod").ZodString>;
    prefix: import("zod").ZodOptional<import("zod").ZodString>;
    attribute: import("zod").ZodOptional<import("zod").ZodBoolean>;
    wrapped: import("zod").ZodOptional<import("zod").ZodBoolean>;
}, import("zod/v4/core").$strip>;
//# sourceMappingURL=xml-object.d.ts.map