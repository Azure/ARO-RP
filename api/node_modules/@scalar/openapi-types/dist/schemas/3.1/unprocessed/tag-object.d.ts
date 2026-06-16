/**
 * Tag Object
 *
 * Adds metadata to a single tag that is used by the Operation Object. It is not mandatory to have a Tag Object per tag
 * defined in the Operation Object instances.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#tag-object
 */
export declare const TagObjectSchema: import("zod").ZodObject<{
    name: import("zod").ZodString;
    description: import("zod").ZodCatch<import("zod").ZodOptional<import("zod").ZodString>>;
    externalDocs: import("zod").ZodOptional<import("zod").ZodObject<{
        description: import("zod").ZodOptional<import("zod").ZodString>;
        url: import("zod").ZodString;
    }, import("zod/v4/core").$strip>>;
}, import("zod/v4/core").$strip>;
//# sourceMappingURL=tag-object.d.ts.map