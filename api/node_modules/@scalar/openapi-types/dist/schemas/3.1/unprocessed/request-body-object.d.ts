/**
 * Request Body Object
 *
 * Describes a single request body.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#request-body-object
 */
export declare const RequestBodyObjectSchema: import("zod").ZodObject<{
    description: import("zod").ZodOptional<import("zod").ZodString>;
    content: import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodObject<{
        schema: import("zod").ZodOptional<import("zod").ZodType<Record<string, any>, unknown, import("zod/v4/core").$ZodTypeInternals<Record<string, any>, unknown>>>;
        example: import("zod").ZodOptional<import("zod").ZodAny>;
        examples: import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodObject<{
            summary: import("zod").ZodOptional<import("zod").ZodString>;
            description: import("zod").ZodOptional<import("zod").ZodString>;
            value: import("zod").ZodOptional<import("zod").ZodAny>;
            externalValue: import("zod").ZodOptional<import("zod").ZodString>;
        }, import("zod/v4/core").$strip>>>;
        encoding: import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodObject<{
            contentType: import("zod").ZodString;
            headers: import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodObject<{
                description: import("zod").ZodOptional<import("zod").ZodString>;
                required: import("zod").ZodOptional<import("zod").ZodBoolean>;
                deprecated: import("zod").ZodOptional<import("zod").ZodBoolean>;
                style: import("zod").ZodOptional<import("zod").ZodEnum<{
                    matrix: "matrix";
                    label: "label";
                    form: "form";
                    simple: "simple";
                    spaceDelimited: "spaceDelimited";
                    pipeDelimited: "pipeDelimited";
                    deepObject: "deepObject";
                }>>;
                explode: import("zod").ZodOptional<import("zod").ZodBoolean>;
                schema: import("zod").ZodOptional<import("zod").ZodType<Record<string, any>, unknown, import("zod/v4/core").$ZodTypeInternals<Record<string, any>, unknown>>>;
                example: import("zod").ZodOptional<import("zod").ZodAny>;
                examples: import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodObject<{
                    summary: import("zod").ZodOptional<import("zod").ZodString>;
                    description: import("zod").ZodOptional<import("zod").ZodString>;
                    value: import("zod").ZodOptional<import("zod").ZodAny>;
                    externalValue: import("zod").ZodOptional<import("zod").ZodString>;
                }, import("zod/v4/core").$strip>>>;
                content: import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodObject<{
                    schema: import("zod").ZodOptional<import("zod").ZodType<Record<string, any>, unknown, import("zod/v4/core").$ZodTypeInternals<Record<string, any>, unknown>>>;
                    example: import("zod").ZodOptional<import("zod").ZodAny>;
                    examples: import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodObject<{
                        summary: import("zod").ZodOptional<import("zod").ZodString>;
                        description: import("zod").ZodOptional<import("zod").ZodString>;
                        value: import("zod").ZodOptional<import("zod").ZodAny>;
                        externalValue: import("zod").ZodOptional<import("zod").ZodString>;
                    }, import("zod/v4/core").$strip>>>;
                }, import("zod/v4/core").$strip>>>;
            }, import("zod/v4/core").$strip>>>;
        }, import("zod/v4/core").$strip>>>;
    }, import("zod/v4/core").$strip>>;
    required: import("zod").ZodOptional<import("zod").ZodBoolean>;
    encoding: import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodObject<{
        contentType: import("zod").ZodString;
        headers: import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodObject<{
            description: import("zod").ZodOptional<import("zod").ZodString>;
            required: import("zod").ZodOptional<import("zod").ZodBoolean>;
            deprecated: import("zod").ZodOptional<import("zod").ZodBoolean>;
            style: import("zod").ZodOptional<import("zod").ZodEnum<{
                matrix: "matrix";
                label: "label";
                form: "form";
                simple: "simple";
                spaceDelimited: "spaceDelimited";
                pipeDelimited: "pipeDelimited";
                deepObject: "deepObject";
            }>>;
            explode: import("zod").ZodOptional<import("zod").ZodBoolean>;
            schema: import("zod").ZodOptional<import("zod").ZodType<Record<string, any>, unknown, import("zod/v4/core").$ZodTypeInternals<Record<string, any>, unknown>>>;
            example: import("zod").ZodOptional<import("zod").ZodAny>;
            examples: import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodObject<{
                summary: import("zod").ZodOptional<import("zod").ZodString>;
                description: import("zod").ZodOptional<import("zod").ZodString>;
                value: import("zod").ZodOptional<import("zod").ZodAny>;
                externalValue: import("zod").ZodOptional<import("zod").ZodString>;
            }, import("zod/v4/core").$strip>>>;
            content: import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodObject<{
                schema: import("zod").ZodOptional<import("zod").ZodType<Record<string, any>, unknown, import("zod/v4/core").$ZodTypeInternals<Record<string, any>, unknown>>>;
                example: import("zod").ZodOptional<import("zod").ZodAny>;
                examples: import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodObject<{
                    summary: import("zod").ZodOptional<import("zod").ZodString>;
                    description: import("zod").ZodOptional<import("zod").ZodString>;
                    value: import("zod").ZodOptional<import("zod").ZodAny>;
                    externalValue: import("zod").ZodOptional<import("zod").ZodString>;
                }, import("zod/v4/core").$strip>>>;
            }, import("zod/v4/core").$strip>>>;
        }, import("zod/v4/core").$strip>>>;
    }, import("zod/v4/core").$strip>>>;
}, import("zod/v4/core").$strip>;
//# sourceMappingURL=request-body-object.d.ts.map