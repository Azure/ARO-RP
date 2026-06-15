/**
 * Info Object
 *
 * The object provides metadata about the API. The metadata MAY be used by the clients if needed,
 * and MAY be presented in editing or documentation generation tools for convenience.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#info-object
 */
export declare const InfoObjectSchema: import("zod").ZodObject<{
    title: import("zod").ZodCatch<import("zod").ZodString>;
    summary: import("zod").ZodCatch<import("zod").ZodOptional<import("zod").ZodString>>;
    description: import("zod").ZodCatch<import("zod").ZodOptional<import("zod").ZodString>>;
    termsOfService: import("zod").ZodCatch<import("zod").ZodOptional<import("zod").ZodString>>;
    contact: import("zod").ZodCatch<import("zod").ZodOptional<import("zod").ZodObject<{
        name: import("zod").ZodOptional<import("zod").ZodString>;
        url: import("zod").ZodCatch<import("zod").ZodOptional<import("zod").ZodString>>;
        email: import("zod").ZodCatch<import("zod").ZodOptional<import("zod").ZodString>>;
    }, import("zod/v4/core").$strip>>>;
    license: import("zod").ZodCatch<import("zod").ZodOptional<import("zod").ZodObject<{
        name: import("zod").ZodCatch<import("zod").ZodNullable<import("zod").ZodOptional<import("zod").ZodString>>>;
        identifier: import("zod").ZodCatch<import("zod").ZodOptional<import("zod").ZodString>>;
        url: import("zod").ZodCatch<import("zod").ZodOptional<import("zod").ZodString>>;
    }, import("zod/v4/core").$strip>>>;
    version: import("zod").ZodCatch<import("zod").ZodString>;
}, import("zod/v4/core").$strip>;
//# sourceMappingURL=info-object.d.ts.map