/**
 * Media Type Object (without encoding)
 *
 * Each Media Type Object provides schema and examples for the media type identified by its key.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#media-type-object
 */
export declare const MediaTypeObjectSchemaWithoutEncoding: import("zod").ZodObject<{
    schema: import("zod").ZodOptional<import("zod").ZodType<Record<string, any>, unknown, import("zod/v4/core").$ZodTypeInternals<Record<string, any>, unknown>>>;
    example: import("zod").ZodOptional<import("zod").ZodAny>;
    examples: import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodObject<{
        summary: import("zod").ZodOptional<import("zod").ZodString>;
        description: import("zod").ZodOptional<import("zod").ZodString>;
        value: import("zod").ZodOptional<import("zod").ZodAny>;
        externalValue: import("zod").ZodOptional<import("zod").ZodString>;
    }, import("zod/v4/core").$strip>>>;
}, import("zod/v4/core").$strip>;
//# sourceMappingURL=media-type-object-without-encoding.d.ts.map