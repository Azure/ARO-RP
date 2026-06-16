import { z } from 'zod';
/**
 * Operation Object (without callbacks, used in callbacks)
 *
 * Describes a single API operation on a path.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#operation-object
 */
export declare const OperationObjectSchemaWithoutCallbacks: z.ZodObject<{
    tags: z.ZodOptional<z.ZodArray<z.ZodString>>;
    summary: z.ZodOptional<z.ZodString>;
    description: z.ZodOptional<z.ZodString>;
    operationId: z.ZodOptional<z.ZodString>;
    responses: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
        description: z.ZodString;
        headers: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
            description: z.ZodOptional<z.ZodString>;
            required: z.ZodOptional<z.ZodBoolean>;
            deprecated: z.ZodOptional<z.ZodBoolean>;
            style: z.ZodOptional<z.ZodEnum<{
                matrix: "matrix";
                label: "label";
                form: "form";
                simple: "simple";
                spaceDelimited: "spaceDelimited";
                pipeDelimited: "pipeDelimited";
                deepObject: "deepObject";
            }>>;
            explode: z.ZodOptional<z.ZodBoolean>;
            schema: z.ZodOptional<z.ZodType<Record<string, any>, unknown, z.core.$ZodTypeInternals<Record<string, any>, unknown>>>;
            example: z.ZodOptional<z.ZodAny>;
            examples: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                summary: z.ZodOptional<z.ZodString>;
                description: z.ZodOptional<z.ZodString>;
                value: z.ZodOptional<z.ZodAny>;
                externalValue: z.ZodOptional<z.ZodString>;
            }, z.core.$strip>>>;
            content: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                schema: z.ZodOptional<z.ZodType<Record<string, any>, unknown, z.core.$ZodTypeInternals<Record<string, any>, unknown>>>;
                example: z.ZodOptional<z.ZodAny>;
                examples: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                    summary: z.ZodOptional<z.ZodString>;
                    description: z.ZodOptional<z.ZodString>;
                    value: z.ZodOptional<z.ZodAny>;
                    externalValue: z.ZodOptional<z.ZodString>;
                }, z.core.$strip>>>;
            }, z.core.$strip>>>;
        }, z.core.$strip>>>;
        content: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
            schema: z.ZodOptional<z.ZodType<Record<string, any>, unknown, z.core.$ZodTypeInternals<Record<string, any>, unknown>>>;
            example: z.ZodOptional<z.ZodAny>;
            examples: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                summary: z.ZodOptional<z.ZodString>;
                description: z.ZodOptional<z.ZodString>;
                value: z.ZodOptional<z.ZodAny>;
                externalValue: z.ZodOptional<z.ZodString>;
            }, z.core.$strip>>>;
            encoding: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                contentType: z.ZodString;
                headers: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                    description: z.ZodOptional<z.ZodString>;
                    required: z.ZodOptional<z.ZodBoolean>;
                    deprecated: z.ZodOptional<z.ZodBoolean>;
                    style: z.ZodOptional<z.ZodEnum<{
                        matrix: "matrix";
                        label: "label";
                        form: "form";
                        simple: "simple";
                        spaceDelimited: "spaceDelimited";
                        pipeDelimited: "pipeDelimited";
                        deepObject: "deepObject";
                    }>>;
                    explode: z.ZodOptional<z.ZodBoolean>;
                    schema: z.ZodOptional<z.ZodType<Record<string, any>, unknown, z.core.$ZodTypeInternals<Record<string, any>, unknown>>>;
                    example: z.ZodOptional<z.ZodAny>;
                    examples: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                        summary: z.ZodOptional<z.ZodString>;
                        description: z.ZodOptional<z.ZodString>;
                        value: z.ZodOptional<z.ZodAny>;
                        externalValue: z.ZodOptional<z.ZodString>;
                    }, z.core.$strip>>>;
                    content: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                        schema: z.ZodOptional<z.ZodType<Record<string, any>, unknown, z.core.$ZodTypeInternals<Record<string, any>, unknown>>>;
                        example: z.ZodOptional<z.ZodAny>;
                        examples: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                            summary: z.ZodOptional<z.ZodString>;
                            description: z.ZodOptional<z.ZodString>;
                            value: z.ZodOptional<z.ZodAny>;
                            externalValue: z.ZodOptional<z.ZodString>;
                        }, z.core.$strip>>>;
                    }, z.core.$strip>>>;
                }, z.core.$strip>>>;
            }, z.core.$strip>>>;
        }, z.core.$strip>>>;
        links: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
            operationRef: z.ZodOptional<z.ZodString>;
            operationId: z.ZodOptional<z.ZodString>;
            parameters: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodString>>;
            requestBody: z.ZodOptional<z.ZodString>;
            description: z.ZodOptional<z.ZodString>;
            server: z.ZodOptional<z.ZodObject<{
                url: z.ZodString;
                description: z.ZodOptional<z.ZodString>;
                variables: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                    enum: z.ZodOptional<z.ZodArray<z.ZodString>>;
                    default: z.ZodOptional<z.ZodString>;
                    description: z.ZodOptional<z.ZodString>;
                }, z.core.$strip>>>;
            }, z.core.$strip>>;
        }, z.core.$strip>>>;
    }, z.core.$strip>>>;
    deprecated: z.ZodOptional<z.ZodBoolean>;
    externalDocs: z.ZodOptional<z.ZodObject<{
        description: z.ZodOptional<z.ZodString>;
        url: z.ZodString;
    }, z.core.$strip>>;
    parameters: z.ZodOptional<z.ZodArray<z.ZodUnion<readonly [z.ZodObject<{
        $ref: z.ZodString;
        summary: z.ZodOptional<z.ZodString>;
        description: z.ZodOptional<z.ZodString>;
    }, z.core.$strip>, z.ZodObject<{
        name: z.ZodString;
        in: z.ZodEnum<{
            query: "query";
            cookie: "cookie";
            header: "header";
            path: "path";
        }>;
        description: z.ZodOptional<z.ZodString>;
        required: z.ZodOptional<z.ZodBoolean>;
        deprecated: z.ZodOptional<z.ZodBoolean>;
        allowEmptyValue: z.ZodOptional<z.ZodBoolean>;
        style: z.ZodOptional<z.ZodEnum<{
            matrix: "matrix";
            label: "label";
            form: "form";
            simple: "simple";
            spaceDelimited: "spaceDelimited";
            pipeDelimited: "pipeDelimited";
            deepObject: "deepObject";
        }>>;
        explode: z.ZodOptional<z.ZodBoolean>;
        allowReserved: z.ZodOptional<z.ZodBoolean>;
        schema: z.ZodOptional<z.ZodType<Record<string, any>, unknown, z.core.$ZodTypeInternals<Record<string, any>, unknown>>>;
        example: z.ZodOptional<z.ZodAny>;
        examples: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodUnion<readonly [z.ZodObject<{
            $ref: z.ZodString;
            summary: z.ZodOptional<z.ZodString>;
            description: z.ZodOptional<z.ZodString>;
        }, z.core.$strip>, z.ZodObject<{
            summary: z.ZodOptional<z.ZodString>;
            description: z.ZodOptional<z.ZodString>;
            value: z.ZodOptional<z.ZodAny>;
            externalValue: z.ZodOptional<z.ZodString>;
        }, z.core.$strip>]>>>;
        content: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
            schema: z.ZodOptional<z.ZodType<any, unknown, z.core.$ZodTypeInternals<any, unknown>>>;
            example: z.ZodOptional<z.ZodAny>;
            examples: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodUnion<readonly [z.ZodObject<{
                $ref: z.ZodString;
                summary: z.ZodOptional<z.ZodString>;
                description: z.ZodOptional<z.ZodString>;
            }, z.core.$strip>, z.ZodObject<{
                summary: z.ZodOptional<z.ZodString>;
                description: z.ZodOptional<z.ZodString>;
                value: z.ZodOptional<z.ZodAny>;
                externalValue: z.ZodOptional<z.ZodString>;
            }, z.core.$strip>]>>>;
            encoding: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                contentType: z.ZodString;
                headers: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodUnion<readonly [z.ZodObject<{
                    $ref: z.ZodString;
                    summary: z.ZodOptional<z.ZodString>;
                    description: z.ZodOptional<z.ZodString>;
                }, z.core.$strip>, z.ZodObject<{
                    description: z.ZodOptional<z.ZodString>;
                    required: z.ZodOptional<z.ZodBoolean>;
                    deprecated: z.ZodOptional<z.ZodBoolean>;
                    style: z.ZodOptional<z.ZodEnum<{
                        matrix: "matrix";
                        label: "label";
                        form: "form";
                        simple: "simple";
                        spaceDelimited: "spaceDelimited";
                        pipeDelimited: "pipeDelimited";
                        deepObject: "deepObject";
                    }>>;
                    explode: z.ZodOptional<z.ZodBoolean>;
                    example: z.ZodOptional<z.ZodAny>;
                    schema: z.ZodOptional<z.ZodType<any, unknown, z.core.$ZodTypeInternals<any, unknown>>>;
                    examples: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodUnion<readonly [z.ZodObject<{
                        $ref: z.ZodString;
                        summary: z.ZodOptional<z.ZodString>;
                        description: z.ZodOptional<z.ZodString>;
                    }, z.core.$strip>, z.ZodObject<{
                        summary: z.ZodOptional<z.ZodString>;
                        description: z.ZodOptional<z.ZodString>;
                        value: z.ZodOptional<z.ZodAny>;
                        externalValue: z.ZodOptional<z.ZodString>;
                    }, z.core.$strip>]>>>;
                    content: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                        schema: z.ZodOptional<z.ZodType<Record<string, any>, unknown, z.core.$ZodTypeInternals<Record<string, any>, unknown>>>;
                        example: z.ZodOptional<z.ZodAny>;
                        examples: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                            summary: z.ZodOptional<z.ZodString>;
                            description: z.ZodOptional<z.ZodString>;
                            value: z.ZodOptional<z.ZodAny>;
                            externalValue: z.ZodOptional<z.ZodString>;
                        }, z.core.$strip>>>;
                    }, z.core.$strip>>>;
                }, z.core.$strip>]>>>;
            }, z.core.$strip>>>;
        }, z.core.$strip>>>;
    }, z.core.$strip>]>>>;
    requestBody: z.ZodOptional<z.ZodUnion<readonly [z.ZodObject<{
        $ref: z.ZodString;
        summary: z.ZodOptional<z.ZodString>;
        description: z.ZodOptional<z.ZodString>;
    }, z.core.$strip>, z.ZodObject<{
        description: z.ZodOptional<z.ZodString>;
        content: z.ZodRecord<z.ZodString, z.ZodObject<{
            schema: z.ZodOptional<z.ZodType<Record<string, any>, unknown, z.core.$ZodTypeInternals<Record<string, any>, unknown>>>;
            example: z.ZodOptional<z.ZodAny>;
            examples: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                summary: z.ZodOptional<z.ZodString>;
                description: z.ZodOptional<z.ZodString>;
                value: z.ZodOptional<z.ZodAny>;
                externalValue: z.ZodOptional<z.ZodString>;
            }, z.core.$strip>>>;
            encoding: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                contentType: z.ZodString;
                headers: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                    description: z.ZodOptional<z.ZodString>;
                    required: z.ZodOptional<z.ZodBoolean>;
                    deprecated: z.ZodOptional<z.ZodBoolean>;
                    style: z.ZodOptional<z.ZodEnum<{
                        matrix: "matrix";
                        label: "label";
                        form: "form";
                        simple: "simple";
                        spaceDelimited: "spaceDelimited";
                        pipeDelimited: "pipeDelimited";
                        deepObject: "deepObject";
                    }>>;
                    explode: z.ZodOptional<z.ZodBoolean>;
                    schema: z.ZodOptional<z.ZodType<Record<string, any>, unknown, z.core.$ZodTypeInternals<Record<string, any>, unknown>>>;
                    example: z.ZodOptional<z.ZodAny>;
                    examples: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                        summary: z.ZodOptional<z.ZodString>;
                        description: z.ZodOptional<z.ZodString>;
                        value: z.ZodOptional<z.ZodAny>;
                        externalValue: z.ZodOptional<z.ZodString>;
                    }, z.core.$strip>>>;
                    content: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                        schema: z.ZodOptional<z.ZodType<Record<string, any>, unknown, z.core.$ZodTypeInternals<Record<string, any>, unknown>>>;
                        example: z.ZodOptional<z.ZodAny>;
                        examples: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                            summary: z.ZodOptional<z.ZodString>;
                            description: z.ZodOptional<z.ZodString>;
                            value: z.ZodOptional<z.ZodAny>;
                            externalValue: z.ZodOptional<z.ZodString>;
                        }, z.core.$strip>>>;
                    }, z.core.$strip>>>;
                }, z.core.$strip>>>;
            }, z.core.$strip>>>;
        }, z.core.$strip>>;
        required: z.ZodOptional<z.ZodBoolean>;
        encoding: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
            contentType: z.ZodString;
            headers: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                description: z.ZodOptional<z.ZodString>;
                required: z.ZodOptional<z.ZodBoolean>;
                deprecated: z.ZodOptional<z.ZodBoolean>;
                style: z.ZodOptional<z.ZodEnum<{
                    matrix: "matrix";
                    label: "label";
                    form: "form";
                    simple: "simple";
                    spaceDelimited: "spaceDelimited";
                    pipeDelimited: "pipeDelimited";
                    deepObject: "deepObject";
                }>>;
                explode: z.ZodOptional<z.ZodBoolean>;
                schema: z.ZodOptional<z.ZodType<Record<string, any>, unknown, z.core.$ZodTypeInternals<Record<string, any>, unknown>>>;
                example: z.ZodOptional<z.ZodAny>;
                examples: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                    summary: z.ZodOptional<z.ZodString>;
                    description: z.ZodOptional<z.ZodString>;
                    value: z.ZodOptional<z.ZodAny>;
                    externalValue: z.ZodOptional<z.ZodString>;
                }, z.core.$strip>>>;
                content: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                    schema: z.ZodOptional<z.ZodType<Record<string, any>, unknown, z.core.$ZodTypeInternals<Record<string, any>, unknown>>>;
                    example: z.ZodOptional<z.ZodAny>;
                    examples: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                        summary: z.ZodOptional<z.ZodString>;
                        description: z.ZodOptional<z.ZodString>;
                        value: z.ZodOptional<z.ZodAny>;
                        externalValue: z.ZodOptional<z.ZodString>;
                    }, z.core.$strip>>>;
                }, z.core.$strip>>>;
            }, z.core.$strip>>>;
        }, z.core.$strip>>>;
    }, z.core.$strip>]>>;
    security: z.ZodOptional<z.ZodArray<z.ZodRecord<z.ZodString, z.ZodArray<z.ZodString>>>>;
}, z.core.$strip>;
//# sourceMappingURL=operation-object-without-callbacks.d.ts.map