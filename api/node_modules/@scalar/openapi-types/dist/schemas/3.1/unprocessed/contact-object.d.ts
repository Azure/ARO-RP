/**
 * Contact Object
 *
 * Contact information for the exposed API.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#contact-object
 */
export declare const ContactObjectSchema: import("zod").ZodObject<{
    name: import("zod").ZodOptional<import("zod").ZodString>;
    url: import("zod").ZodCatch<import("zod").ZodOptional<import("zod").ZodString>>;
    email: import("zod").ZodCatch<import("zod").ZodOptional<import("zod").ZodString>>;
}, import("zod/v4/core").$strip>;
//# sourceMappingURL=contact-object.d.ts.map