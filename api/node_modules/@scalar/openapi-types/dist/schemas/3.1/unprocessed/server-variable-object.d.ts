/**
 * Server Variable Object
 *
 * An object representing a Server Variable for server URL template substitution.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#server-variable-object
 */
export declare const ServerVariableObjectSchema: import("zod").ZodObject<{
    enum: import("zod").ZodOptional<import("zod").ZodArray<import("zod").ZodString>>;
    default: import("zod").ZodOptional<import("zod").ZodString>;
    description: import("zod").ZodOptional<import("zod").ZodString>;
}, import("zod/v4/core").$strip>;
//# sourceMappingURL=server-variable-object.d.ts.map