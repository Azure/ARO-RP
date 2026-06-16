import { z } from 'zod';
/**
 * Paths Object
 *
 * Holds the relative paths to the individual endpoints and their operations. The path is appended to the URL from the
 * Server Object in order to construct the full URL. The Paths Object MAY be empty, due to Access Control List (ACL)
 * constraints.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#paths-object
 */
export declare const PathsObjectSchema: z.ZodRecord<z.ZodString, z.ZodObject<{
    summary: z.ZodOptional<z.ZodString>;
    description: z.ZodOptional<z.ZodString>;
    servers: z.ZodOptional<z.ZodArray<z.ZodObject<{
        url: z.ZodString;
        description: z.ZodOptional<z.ZodString>;
        variables: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
            enum: z.ZodOptional<z.ZodArray<z.ZodString>>;
            default: z.ZodOptional<z.ZodString>;
            description: z.ZodOptional<z.ZodString>;
        }, z.core.$strip>>>;
    }, z.core.$strip>>>;
    parameters: z.ZodOptional<z.ZodArray<z.ZodObject<{
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
    }, z.core.$strip>>>;
    get: z.ZodOptional<z.ZodType<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        operationId?: string | undefined;
        parameters?: {
            name: string;
            in: "query" | "cookie" | "header" | "path";
            description?: string | undefined;
            required?: boolean | undefined;
            deprecated?: boolean | undefined;
            allowEmptyValue?: boolean | undefined;
            style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
            explode?: boolean | undefined;
            allowReserved?: boolean | undefined;
            schema?: Record<string, any> | undefined;
            example?: any;
            examples?: Record<string, {
                summary?: string | undefined;
                description?: string | undefined;
                value?: any;
                externalValue?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        }[] | undefined;
        requestBody?: {
            content: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }>;
            description?: string | undefined;
            required?: boolean | undefined;
            encoding?: Record<string, {
                contentType: string;
                headers?: Record<string, {
                    description?: string | undefined;
                    required?: boolean | undefined;
                    deprecated?: boolean | undefined;
                    style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                    explode?: boolean | undefined;
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                    content?: Record<string, {
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        } | undefined;
        responses?: Record<string, {
            description: string;
            headers?: Record<string, {
                description?: string | undefined;
                required?: boolean | undefined;
                deprecated?: boolean | undefined;
                style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                explode?: boolean | undefined;
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                content?: Record<string, {
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            links?: Record<string, {
                operationRef?: string | undefined;
                operationId?: string | undefined;
                parameters?: Record<string, string> | undefined;
                requestBody?: string | undefined;
                description?: string | undefined;
                server?: {
                    url: string;
                    description?: string | undefined;
                    variables?: Record<string, {
                        enum?: string[] | undefined;
                        default?: string | undefined;
                        description?: string | undefined;
                    }> | undefined;
                } | undefined;
            }> | undefined;
        }> | undefined;
        security?: Record<string, string[]>[] | undefined;
        deprecated?: boolean | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, z.core.$ZodTypeInternals<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        operationId?: string | undefined;
        parameters?: {
            name: string;
            in: "query" | "cookie" | "header" | "path";
            description?: string | undefined;
            required?: boolean | undefined;
            deprecated?: boolean | undefined;
            allowEmptyValue?: boolean | undefined;
            style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
            explode?: boolean | undefined;
            allowReserved?: boolean | undefined;
            schema?: Record<string, any> | undefined;
            example?: any;
            examples?: Record<string, {
                summary?: string | undefined;
                description?: string | undefined;
                value?: any;
                externalValue?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        }[] | undefined;
        requestBody?: {
            content: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }>;
            description?: string | undefined;
            required?: boolean | undefined;
            encoding?: Record<string, {
                contentType: string;
                headers?: Record<string, {
                    description?: string | undefined;
                    required?: boolean | undefined;
                    deprecated?: boolean | undefined;
                    style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                    explode?: boolean | undefined;
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                    content?: Record<string, {
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        } | undefined;
        responses?: Record<string, {
            description: string;
            headers?: Record<string, {
                description?: string | undefined;
                required?: boolean | undefined;
                deprecated?: boolean | undefined;
                style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                explode?: boolean | undefined;
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                content?: Record<string, {
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            links?: Record<string, {
                operationRef?: string | undefined;
                operationId?: string | undefined;
                parameters?: Record<string, string> | undefined;
                requestBody?: string | undefined;
                description?: string | undefined;
                server?: {
                    url: string;
                    description?: string | undefined;
                    variables?: Record<string, {
                        enum?: string[] | undefined;
                        default?: string | undefined;
                        description?: string | undefined;
                    }> | undefined;
                } | undefined;
            }> | undefined;
        }> | undefined;
        security?: Record<string, string[]>[] | undefined;
        deprecated?: boolean | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
    put: z.ZodOptional<z.ZodType<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        operationId?: string | undefined;
        parameters?: {
            name: string;
            in: "query" | "cookie" | "header" | "path";
            description?: string | undefined;
            required?: boolean | undefined;
            deprecated?: boolean | undefined;
            allowEmptyValue?: boolean | undefined;
            style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
            explode?: boolean | undefined;
            allowReserved?: boolean | undefined;
            schema?: Record<string, any> | undefined;
            example?: any;
            examples?: Record<string, {
                summary?: string | undefined;
                description?: string | undefined;
                value?: any;
                externalValue?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        }[] | undefined;
        requestBody?: {
            content: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }>;
            description?: string | undefined;
            required?: boolean | undefined;
            encoding?: Record<string, {
                contentType: string;
                headers?: Record<string, {
                    description?: string | undefined;
                    required?: boolean | undefined;
                    deprecated?: boolean | undefined;
                    style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                    explode?: boolean | undefined;
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                    content?: Record<string, {
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        } | undefined;
        responses?: Record<string, {
            description: string;
            headers?: Record<string, {
                description?: string | undefined;
                required?: boolean | undefined;
                deprecated?: boolean | undefined;
                style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                explode?: boolean | undefined;
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                content?: Record<string, {
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            links?: Record<string, {
                operationRef?: string | undefined;
                operationId?: string | undefined;
                parameters?: Record<string, string> | undefined;
                requestBody?: string | undefined;
                description?: string | undefined;
                server?: {
                    url: string;
                    description?: string | undefined;
                    variables?: Record<string, {
                        enum?: string[] | undefined;
                        default?: string | undefined;
                        description?: string | undefined;
                    }> | undefined;
                } | undefined;
            }> | undefined;
        }> | undefined;
        security?: Record<string, string[]>[] | undefined;
        deprecated?: boolean | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, z.core.$ZodTypeInternals<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        operationId?: string | undefined;
        parameters?: {
            name: string;
            in: "query" | "cookie" | "header" | "path";
            description?: string | undefined;
            required?: boolean | undefined;
            deprecated?: boolean | undefined;
            allowEmptyValue?: boolean | undefined;
            style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
            explode?: boolean | undefined;
            allowReserved?: boolean | undefined;
            schema?: Record<string, any> | undefined;
            example?: any;
            examples?: Record<string, {
                summary?: string | undefined;
                description?: string | undefined;
                value?: any;
                externalValue?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        }[] | undefined;
        requestBody?: {
            content: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }>;
            description?: string | undefined;
            required?: boolean | undefined;
            encoding?: Record<string, {
                contentType: string;
                headers?: Record<string, {
                    description?: string | undefined;
                    required?: boolean | undefined;
                    deprecated?: boolean | undefined;
                    style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                    explode?: boolean | undefined;
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                    content?: Record<string, {
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        } | undefined;
        responses?: Record<string, {
            description: string;
            headers?: Record<string, {
                description?: string | undefined;
                required?: boolean | undefined;
                deprecated?: boolean | undefined;
                style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                explode?: boolean | undefined;
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                content?: Record<string, {
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            links?: Record<string, {
                operationRef?: string | undefined;
                operationId?: string | undefined;
                parameters?: Record<string, string> | undefined;
                requestBody?: string | undefined;
                description?: string | undefined;
                server?: {
                    url: string;
                    description?: string | undefined;
                    variables?: Record<string, {
                        enum?: string[] | undefined;
                        default?: string | undefined;
                        description?: string | undefined;
                    }> | undefined;
                } | undefined;
            }> | undefined;
        }> | undefined;
        security?: Record<string, string[]>[] | undefined;
        deprecated?: boolean | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
    post: z.ZodOptional<z.ZodType<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        operationId?: string | undefined;
        parameters?: {
            name: string;
            in: "query" | "cookie" | "header" | "path";
            description?: string | undefined;
            required?: boolean | undefined;
            deprecated?: boolean | undefined;
            allowEmptyValue?: boolean | undefined;
            style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
            explode?: boolean | undefined;
            allowReserved?: boolean | undefined;
            schema?: Record<string, any> | undefined;
            example?: any;
            examples?: Record<string, {
                summary?: string | undefined;
                description?: string | undefined;
                value?: any;
                externalValue?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        }[] | undefined;
        requestBody?: {
            content: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }>;
            description?: string | undefined;
            required?: boolean | undefined;
            encoding?: Record<string, {
                contentType: string;
                headers?: Record<string, {
                    description?: string | undefined;
                    required?: boolean | undefined;
                    deprecated?: boolean | undefined;
                    style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                    explode?: boolean | undefined;
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                    content?: Record<string, {
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        } | undefined;
        responses?: Record<string, {
            description: string;
            headers?: Record<string, {
                description?: string | undefined;
                required?: boolean | undefined;
                deprecated?: boolean | undefined;
                style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                explode?: boolean | undefined;
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                content?: Record<string, {
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            links?: Record<string, {
                operationRef?: string | undefined;
                operationId?: string | undefined;
                parameters?: Record<string, string> | undefined;
                requestBody?: string | undefined;
                description?: string | undefined;
                server?: {
                    url: string;
                    description?: string | undefined;
                    variables?: Record<string, {
                        enum?: string[] | undefined;
                        default?: string | undefined;
                        description?: string | undefined;
                    }> | undefined;
                } | undefined;
            }> | undefined;
        }> | undefined;
        security?: Record<string, string[]>[] | undefined;
        deprecated?: boolean | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, z.core.$ZodTypeInternals<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        operationId?: string | undefined;
        parameters?: {
            name: string;
            in: "query" | "cookie" | "header" | "path";
            description?: string | undefined;
            required?: boolean | undefined;
            deprecated?: boolean | undefined;
            allowEmptyValue?: boolean | undefined;
            style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
            explode?: boolean | undefined;
            allowReserved?: boolean | undefined;
            schema?: Record<string, any> | undefined;
            example?: any;
            examples?: Record<string, {
                summary?: string | undefined;
                description?: string | undefined;
                value?: any;
                externalValue?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        }[] | undefined;
        requestBody?: {
            content: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }>;
            description?: string | undefined;
            required?: boolean | undefined;
            encoding?: Record<string, {
                contentType: string;
                headers?: Record<string, {
                    description?: string | undefined;
                    required?: boolean | undefined;
                    deprecated?: boolean | undefined;
                    style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                    explode?: boolean | undefined;
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                    content?: Record<string, {
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        } | undefined;
        responses?: Record<string, {
            description: string;
            headers?: Record<string, {
                description?: string | undefined;
                required?: boolean | undefined;
                deprecated?: boolean | undefined;
                style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                explode?: boolean | undefined;
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                content?: Record<string, {
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            links?: Record<string, {
                operationRef?: string | undefined;
                operationId?: string | undefined;
                parameters?: Record<string, string> | undefined;
                requestBody?: string | undefined;
                description?: string | undefined;
                server?: {
                    url: string;
                    description?: string | undefined;
                    variables?: Record<string, {
                        enum?: string[] | undefined;
                        default?: string | undefined;
                        description?: string | undefined;
                    }> | undefined;
                } | undefined;
            }> | undefined;
        }> | undefined;
        security?: Record<string, string[]>[] | undefined;
        deprecated?: boolean | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
    delete: z.ZodOptional<z.ZodType<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        operationId?: string | undefined;
        parameters?: {
            name: string;
            in: "query" | "cookie" | "header" | "path";
            description?: string | undefined;
            required?: boolean | undefined;
            deprecated?: boolean | undefined;
            allowEmptyValue?: boolean | undefined;
            style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
            explode?: boolean | undefined;
            allowReserved?: boolean | undefined;
            schema?: Record<string, any> | undefined;
            example?: any;
            examples?: Record<string, {
                summary?: string | undefined;
                description?: string | undefined;
                value?: any;
                externalValue?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        }[] | undefined;
        requestBody?: {
            content: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }>;
            description?: string | undefined;
            required?: boolean | undefined;
            encoding?: Record<string, {
                contentType: string;
                headers?: Record<string, {
                    description?: string | undefined;
                    required?: boolean | undefined;
                    deprecated?: boolean | undefined;
                    style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                    explode?: boolean | undefined;
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                    content?: Record<string, {
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        } | undefined;
        responses?: Record<string, {
            description: string;
            headers?: Record<string, {
                description?: string | undefined;
                required?: boolean | undefined;
                deprecated?: boolean | undefined;
                style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                explode?: boolean | undefined;
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                content?: Record<string, {
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            links?: Record<string, {
                operationRef?: string | undefined;
                operationId?: string | undefined;
                parameters?: Record<string, string> | undefined;
                requestBody?: string | undefined;
                description?: string | undefined;
                server?: {
                    url: string;
                    description?: string | undefined;
                    variables?: Record<string, {
                        enum?: string[] | undefined;
                        default?: string | undefined;
                        description?: string | undefined;
                    }> | undefined;
                } | undefined;
            }> | undefined;
        }> | undefined;
        security?: Record<string, string[]>[] | undefined;
        deprecated?: boolean | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, z.core.$ZodTypeInternals<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        operationId?: string | undefined;
        parameters?: {
            name: string;
            in: "query" | "cookie" | "header" | "path";
            description?: string | undefined;
            required?: boolean | undefined;
            deprecated?: boolean | undefined;
            allowEmptyValue?: boolean | undefined;
            style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
            explode?: boolean | undefined;
            allowReserved?: boolean | undefined;
            schema?: Record<string, any> | undefined;
            example?: any;
            examples?: Record<string, {
                summary?: string | undefined;
                description?: string | undefined;
                value?: any;
                externalValue?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        }[] | undefined;
        requestBody?: {
            content: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }>;
            description?: string | undefined;
            required?: boolean | undefined;
            encoding?: Record<string, {
                contentType: string;
                headers?: Record<string, {
                    description?: string | undefined;
                    required?: boolean | undefined;
                    deprecated?: boolean | undefined;
                    style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                    explode?: boolean | undefined;
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                    content?: Record<string, {
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        } | undefined;
        responses?: Record<string, {
            description: string;
            headers?: Record<string, {
                description?: string | undefined;
                required?: boolean | undefined;
                deprecated?: boolean | undefined;
                style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                explode?: boolean | undefined;
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                content?: Record<string, {
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            links?: Record<string, {
                operationRef?: string | undefined;
                operationId?: string | undefined;
                parameters?: Record<string, string> | undefined;
                requestBody?: string | undefined;
                description?: string | undefined;
                server?: {
                    url: string;
                    description?: string | undefined;
                    variables?: Record<string, {
                        enum?: string[] | undefined;
                        default?: string | undefined;
                        description?: string | undefined;
                    }> | undefined;
                } | undefined;
            }> | undefined;
        }> | undefined;
        security?: Record<string, string[]>[] | undefined;
        deprecated?: boolean | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
    options: z.ZodOptional<z.ZodType<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        operationId?: string | undefined;
        parameters?: {
            name: string;
            in: "query" | "cookie" | "header" | "path";
            description?: string | undefined;
            required?: boolean | undefined;
            deprecated?: boolean | undefined;
            allowEmptyValue?: boolean | undefined;
            style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
            explode?: boolean | undefined;
            allowReserved?: boolean | undefined;
            schema?: Record<string, any> | undefined;
            example?: any;
            examples?: Record<string, {
                summary?: string | undefined;
                description?: string | undefined;
                value?: any;
                externalValue?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        }[] | undefined;
        requestBody?: {
            content: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }>;
            description?: string | undefined;
            required?: boolean | undefined;
            encoding?: Record<string, {
                contentType: string;
                headers?: Record<string, {
                    description?: string | undefined;
                    required?: boolean | undefined;
                    deprecated?: boolean | undefined;
                    style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                    explode?: boolean | undefined;
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                    content?: Record<string, {
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        } | undefined;
        responses?: Record<string, {
            description: string;
            headers?: Record<string, {
                description?: string | undefined;
                required?: boolean | undefined;
                deprecated?: boolean | undefined;
                style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                explode?: boolean | undefined;
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                content?: Record<string, {
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            links?: Record<string, {
                operationRef?: string | undefined;
                operationId?: string | undefined;
                parameters?: Record<string, string> | undefined;
                requestBody?: string | undefined;
                description?: string | undefined;
                server?: {
                    url: string;
                    description?: string | undefined;
                    variables?: Record<string, {
                        enum?: string[] | undefined;
                        default?: string | undefined;
                        description?: string | undefined;
                    }> | undefined;
                } | undefined;
            }> | undefined;
        }> | undefined;
        security?: Record<string, string[]>[] | undefined;
        deprecated?: boolean | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, z.core.$ZodTypeInternals<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        operationId?: string | undefined;
        parameters?: {
            name: string;
            in: "query" | "cookie" | "header" | "path";
            description?: string | undefined;
            required?: boolean | undefined;
            deprecated?: boolean | undefined;
            allowEmptyValue?: boolean | undefined;
            style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
            explode?: boolean | undefined;
            allowReserved?: boolean | undefined;
            schema?: Record<string, any> | undefined;
            example?: any;
            examples?: Record<string, {
                summary?: string | undefined;
                description?: string | undefined;
                value?: any;
                externalValue?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        }[] | undefined;
        requestBody?: {
            content: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }>;
            description?: string | undefined;
            required?: boolean | undefined;
            encoding?: Record<string, {
                contentType: string;
                headers?: Record<string, {
                    description?: string | undefined;
                    required?: boolean | undefined;
                    deprecated?: boolean | undefined;
                    style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                    explode?: boolean | undefined;
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                    content?: Record<string, {
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        } | undefined;
        responses?: Record<string, {
            description: string;
            headers?: Record<string, {
                description?: string | undefined;
                required?: boolean | undefined;
                deprecated?: boolean | undefined;
                style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                explode?: boolean | undefined;
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                content?: Record<string, {
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            links?: Record<string, {
                operationRef?: string | undefined;
                operationId?: string | undefined;
                parameters?: Record<string, string> | undefined;
                requestBody?: string | undefined;
                description?: string | undefined;
                server?: {
                    url: string;
                    description?: string | undefined;
                    variables?: Record<string, {
                        enum?: string[] | undefined;
                        default?: string | undefined;
                        description?: string | undefined;
                    }> | undefined;
                } | undefined;
            }> | undefined;
        }> | undefined;
        security?: Record<string, string[]>[] | undefined;
        deprecated?: boolean | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
    head: z.ZodOptional<z.ZodType<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        operationId?: string | undefined;
        parameters?: {
            name: string;
            in: "query" | "cookie" | "header" | "path";
            description?: string | undefined;
            required?: boolean | undefined;
            deprecated?: boolean | undefined;
            allowEmptyValue?: boolean | undefined;
            style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
            explode?: boolean | undefined;
            allowReserved?: boolean | undefined;
            schema?: Record<string, any> | undefined;
            example?: any;
            examples?: Record<string, {
                summary?: string | undefined;
                description?: string | undefined;
                value?: any;
                externalValue?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        }[] | undefined;
        requestBody?: {
            content: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }>;
            description?: string | undefined;
            required?: boolean | undefined;
            encoding?: Record<string, {
                contentType: string;
                headers?: Record<string, {
                    description?: string | undefined;
                    required?: boolean | undefined;
                    deprecated?: boolean | undefined;
                    style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                    explode?: boolean | undefined;
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                    content?: Record<string, {
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        } | undefined;
        responses?: Record<string, {
            description: string;
            headers?: Record<string, {
                description?: string | undefined;
                required?: boolean | undefined;
                deprecated?: boolean | undefined;
                style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                explode?: boolean | undefined;
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                content?: Record<string, {
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            links?: Record<string, {
                operationRef?: string | undefined;
                operationId?: string | undefined;
                parameters?: Record<string, string> | undefined;
                requestBody?: string | undefined;
                description?: string | undefined;
                server?: {
                    url: string;
                    description?: string | undefined;
                    variables?: Record<string, {
                        enum?: string[] | undefined;
                        default?: string | undefined;
                        description?: string | undefined;
                    }> | undefined;
                } | undefined;
            }> | undefined;
        }> | undefined;
        security?: Record<string, string[]>[] | undefined;
        deprecated?: boolean | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, z.core.$ZodTypeInternals<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        operationId?: string | undefined;
        parameters?: {
            name: string;
            in: "query" | "cookie" | "header" | "path";
            description?: string | undefined;
            required?: boolean | undefined;
            deprecated?: boolean | undefined;
            allowEmptyValue?: boolean | undefined;
            style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
            explode?: boolean | undefined;
            allowReserved?: boolean | undefined;
            schema?: Record<string, any> | undefined;
            example?: any;
            examples?: Record<string, {
                summary?: string | undefined;
                description?: string | undefined;
                value?: any;
                externalValue?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        }[] | undefined;
        requestBody?: {
            content: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }>;
            description?: string | undefined;
            required?: boolean | undefined;
            encoding?: Record<string, {
                contentType: string;
                headers?: Record<string, {
                    description?: string | undefined;
                    required?: boolean | undefined;
                    deprecated?: boolean | undefined;
                    style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                    explode?: boolean | undefined;
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                    content?: Record<string, {
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        } | undefined;
        responses?: Record<string, {
            description: string;
            headers?: Record<string, {
                description?: string | undefined;
                required?: boolean | undefined;
                deprecated?: boolean | undefined;
                style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                explode?: boolean | undefined;
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                content?: Record<string, {
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            links?: Record<string, {
                operationRef?: string | undefined;
                operationId?: string | undefined;
                parameters?: Record<string, string> | undefined;
                requestBody?: string | undefined;
                description?: string | undefined;
                server?: {
                    url: string;
                    description?: string | undefined;
                    variables?: Record<string, {
                        enum?: string[] | undefined;
                        default?: string | undefined;
                        description?: string | undefined;
                    }> | undefined;
                } | undefined;
            }> | undefined;
        }> | undefined;
        security?: Record<string, string[]>[] | undefined;
        deprecated?: boolean | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
    patch: z.ZodOptional<z.ZodType<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        operationId?: string | undefined;
        parameters?: {
            name: string;
            in: "query" | "cookie" | "header" | "path";
            description?: string | undefined;
            required?: boolean | undefined;
            deprecated?: boolean | undefined;
            allowEmptyValue?: boolean | undefined;
            style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
            explode?: boolean | undefined;
            allowReserved?: boolean | undefined;
            schema?: Record<string, any> | undefined;
            example?: any;
            examples?: Record<string, {
                summary?: string | undefined;
                description?: string | undefined;
                value?: any;
                externalValue?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        }[] | undefined;
        requestBody?: {
            content: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }>;
            description?: string | undefined;
            required?: boolean | undefined;
            encoding?: Record<string, {
                contentType: string;
                headers?: Record<string, {
                    description?: string | undefined;
                    required?: boolean | undefined;
                    deprecated?: boolean | undefined;
                    style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                    explode?: boolean | undefined;
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                    content?: Record<string, {
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        } | undefined;
        responses?: Record<string, {
            description: string;
            headers?: Record<string, {
                description?: string | undefined;
                required?: boolean | undefined;
                deprecated?: boolean | undefined;
                style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                explode?: boolean | undefined;
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                content?: Record<string, {
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            links?: Record<string, {
                operationRef?: string | undefined;
                operationId?: string | undefined;
                parameters?: Record<string, string> | undefined;
                requestBody?: string | undefined;
                description?: string | undefined;
                server?: {
                    url: string;
                    description?: string | undefined;
                    variables?: Record<string, {
                        enum?: string[] | undefined;
                        default?: string | undefined;
                        description?: string | undefined;
                    }> | undefined;
                } | undefined;
            }> | undefined;
        }> | undefined;
        security?: Record<string, string[]>[] | undefined;
        deprecated?: boolean | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, z.core.$ZodTypeInternals<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        operationId?: string | undefined;
        parameters?: {
            name: string;
            in: "query" | "cookie" | "header" | "path";
            description?: string | undefined;
            required?: boolean | undefined;
            deprecated?: boolean | undefined;
            allowEmptyValue?: boolean | undefined;
            style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
            explode?: boolean | undefined;
            allowReserved?: boolean | undefined;
            schema?: Record<string, any> | undefined;
            example?: any;
            examples?: Record<string, {
                summary?: string | undefined;
                description?: string | undefined;
                value?: any;
                externalValue?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        }[] | undefined;
        requestBody?: {
            content: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }>;
            description?: string | undefined;
            required?: boolean | undefined;
            encoding?: Record<string, {
                contentType: string;
                headers?: Record<string, {
                    description?: string | undefined;
                    required?: boolean | undefined;
                    deprecated?: boolean | undefined;
                    style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                    explode?: boolean | undefined;
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                    content?: Record<string, {
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        } | undefined;
        responses?: Record<string, {
            description: string;
            headers?: Record<string, {
                description?: string | undefined;
                required?: boolean | undefined;
                deprecated?: boolean | undefined;
                style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                explode?: boolean | undefined;
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                content?: Record<string, {
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            links?: Record<string, {
                operationRef?: string | undefined;
                operationId?: string | undefined;
                parameters?: Record<string, string> | undefined;
                requestBody?: string | undefined;
                description?: string | undefined;
                server?: {
                    url: string;
                    description?: string | undefined;
                    variables?: Record<string, {
                        enum?: string[] | undefined;
                        default?: string | undefined;
                        description?: string | undefined;
                    }> | undefined;
                } | undefined;
            }> | undefined;
        }> | undefined;
        security?: Record<string, string[]>[] | undefined;
        deprecated?: boolean | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
    trace: z.ZodOptional<z.ZodType<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        operationId?: string | undefined;
        parameters?: {
            name: string;
            in: "query" | "cookie" | "header" | "path";
            description?: string | undefined;
            required?: boolean | undefined;
            deprecated?: boolean | undefined;
            allowEmptyValue?: boolean | undefined;
            style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
            explode?: boolean | undefined;
            allowReserved?: boolean | undefined;
            schema?: Record<string, any> | undefined;
            example?: any;
            examples?: Record<string, {
                summary?: string | undefined;
                description?: string | undefined;
                value?: any;
                externalValue?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        }[] | undefined;
        requestBody?: {
            content: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }>;
            description?: string | undefined;
            required?: boolean | undefined;
            encoding?: Record<string, {
                contentType: string;
                headers?: Record<string, {
                    description?: string | undefined;
                    required?: boolean | undefined;
                    deprecated?: boolean | undefined;
                    style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                    explode?: boolean | undefined;
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                    content?: Record<string, {
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        } | undefined;
        responses?: Record<string, {
            description: string;
            headers?: Record<string, {
                description?: string | undefined;
                required?: boolean | undefined;
                deprecated?: boolean | undefined;
                style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                explode?: boolean | undefined;
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                content?: Record<string, {
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            links?: Record<string, {
                operationRef?: string | undefined;
                operationId?: string | undefined;
                parameters?: Record<string, string> | undefined;
                requestBody?: string | undefined;
                description?: string | undefined;
                server?: {
                    url: string;
                    description?: string | undefined;
                    variables?: Record<string, {
                        enum?: string[] | undefined;
                        default?: string | undefined;
                        description?: string | undefined;
                    }> | undefined;
                } | undefined;
            }> | undefined;
        }> | undefined;
        security?: Record<string, string[]>[] | undefined;
        deprecated?: boolean | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, z.core.$ZodTypeInternals<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        operationId?: string | undefined;
        parameters?: {
            name: string;
            in: "query" | "cookie" | "header" | "path";
            description?: string | undefined;
            required?: boolean | undefined;
            deprecated?: boolean | undefined;
            allowEmptyValue?: boolean | undefined;
            style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
            explode?: boolean | undefined;
            allowReserved?: boolean | undefined;
            schema?: Record<string, any> | undefined;
            example?: any;
            examples?: Record<string, {
                summary?: string | undefined;
                description?: string | undefined;
                value?: any;
                externalValue?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        }[] | undefined;
        requestBody?: {
            content: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }>;
            description?: string | undefined;
            required?: boolean | undefined;
            encoding?: Record<string, {
                contentType: string;
                headers?: Record<string, {
                    description?: string | undefined;
                    required?: boolean | undefined;
                    deprecated?: boolean | undefined;
                    style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                    explode?: boolean | undefined;
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                    content?: Record<string, {
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
        } | undefined;
        responses?: Record<string, {
            description: string;
            headers?: Record<string, {
                description?: string | undefined;
                required?: boolean | undefined;
                deprecated?: boolean | undefined;
                style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                explode?: boolean | undefined;
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                content?: Record<string, {
                    schema?: Record<string, any> | undefined;
                    example?: any;
                    examples?: Record<string, {
                        summary?: string | undefined;
                        description?: string | undefined;
                        value?: any;
                        externalValue?: string | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: Record<string, any> | undefined;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        schema?: Record<string, any> | undefined;
                        example?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        }> | undefined;
                        content?: Record<string, {
                            schema?: Record<string, any> | undefined;
                            example?: any;
                            examples?: Record<string, {
                                summary?: string | undefined;
                                description?: string | undefined;
                                value?: any;
                                externalValue?: string | undefined;
                            }> | undefined;
                        }> | undefined;
                    }> | undefined;
                }> | undefined;
            }> | undefined;
            links?: Record<string, {
                operationRef?: string | undefined;
                operationId?: string | undefined;
                parameters?: Record<string, string> | undefined;
                requestBody?: string | undefined;
                description?: string | undefined;
                server?: {
                    url: string;
                    description?: string | undefined;
                    variables?: Record<string, {
                        enum?: string[] | undefined;
                        default?: string | undefined;
                        description?: string | undefined;
                    }> | undefined;
                } | undefined;
            }> | undefined;
        }> | undefined;
        security?: Record<string, string[]>[] | undefined;
        deprecated?: boolean | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
}, z.core.$strip>>;
//# sourceMappingURL=paths-object.d.ts.map