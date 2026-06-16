/**
 * Server Object
 *
 * An object representing a Server.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#server-object
 */
export declare const ServerObjectSchema: import("zod").ZodObject<{
    url: import("zod").ZodString;
    description: import("zod").ZodOptional<import("zod").ZodString>;
    variables: import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodObject<{
        enum: import("zod").ZodOptional<import("zod").ZodArray<import("zod").ZodString>>;
        default: import("zod").ZodOptional<import("zod").ZodString>;
        description: import("zod").ZodOptional<import("zod").ZodString>;
    }, import("zod/v4/core").$strip>>>;
}, import("zod/v4/core").$strip>;
//# sourceMappingURL=server-object.d.ts.map