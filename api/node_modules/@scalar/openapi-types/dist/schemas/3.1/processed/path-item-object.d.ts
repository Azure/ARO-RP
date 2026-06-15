/**
 * Path Item Object
 *
 * Describes the operations available on a single path. A Path Item MAY be empty, due to ACL constraints. The path
 * itself is still exposed to the documentation viewer but they will not know which operations and parameters are
 * available.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#path-item-object
 */
export declare const PathItemObjectSchema: import("zod").ZodObject<{
    summary: import("zod").ZodOptional<import("zod").ZodString>;
    description: import("zod").ZodOptional<import("zod").ZodString>;
    servers: import("zod").ZodOptional<import("zod").ZodArray<import("zod").ZodObject<{
        url: import("zod").ZodString;
        description: import("zod").ZodOptional<import("zod").ZodString>;
        variables: import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodObject<{
            enum: import("zod").ZodOptional<import("zod").ZodArray<import("zod").ZodString>>;
            default: import("zod").ZodOptional<import("zod").ZodString>;
            description: import("zod").ZodOptional<import("zod").ZodString>;
        }, import("zod/v4/core").$strip>>>;
    }, import("zod/v4/core").$strip>>>;
    parameters: import("zod").ZodOptional<import("zod").ZodArray<import("zod").ZodObject<{
        name: import("zod").ZodString;
        in: import("zod").ZodEnum<{
            query: "query";
            cookie: "cookie";
            header: "header";
            path: "path";
        }>;
        description: import("zod").ZodOptional<import("zod").ZodString>;
        required: import("zod").ZodOptional<import("zod").ZodBoolean>;
        deprecated: import("zod").ZodOptional<import("zod").ZodBoolean>;
        allowEmptyValue: import("zod").ZodOptional<import("zod").ZodBoolean>;
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
        allowReserved: import("zod").ZodOptional<import("zod").ZodBoolean>;
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
        }, import("zod/v4/core").$strip>>>;
    }, import("zod/v4/core").$strip>>>;
    get: import("zod").ZodOptional<import("zod").ZodType<{
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
        callbacks?: Record<string, import("zod").infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, import("zod/v4/core").$ZodTypeInternals<{
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
        callbacks?: Record<string, import("zod").infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
    put: import("zod").ZodOptional<import("zod").ZodType<{
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
        callbacks?: Record<string, import("zod").infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, import("zod/v4/core").$ZodTypeInternals<{
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
        callbacks?: Record<string, import("zod").infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
    post: import("zod").ZodOptional<import("zod").ZodType<{
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
        callbacks?: Record<string, import("zod").infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, import("zod/v4/core").$ZodTypeInternals<{
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
        callbacks?: Record<string, import("zod").infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
    delete: import("zod").ZodOptional<import("zod").ZodType<{
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
        callbacks?: Record<string, import("zod").infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, import("zod/v4/core").$ZodTypeInternals<{
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
        callbacks?: Record<string, import("zod").infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
    options: import("zod").ZodOptional<import("zod").ZodType<{
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
        callbacks?: Record<string, import("zod").infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, import("zod/v4/core").$ZodTypeInternals<{
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
        callbacks?: Record<string, import("zod").infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
    head: import("zod").ZodOptional<import("zod").ZodType<{
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
        callbacks?: Record<string, import("zod").infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, import("zod/v4/core").$ZodTypeInternals<{
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
        callbacks?: Record<string, import("zod").infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
    patch: import("zod").ZodOptional<import("zod").ZodType<{
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
        callbacks?: Record<string, import("zod").infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, import("zod/v4/core").$ZodTypeInternals<{
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
        callbacks?: Record<string, import("zod").infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
    trace: import("zod").ZodOptional<import("zod").ZodType<{
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
        callbacks?: Record<string, import("zod").infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, import("zod/v4/core").$ZodTypeInternals<{
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
        callbacks?: Record<string, import("zod").infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
}, import("zod/v4/core").$strip>;
//# sourceMappingURL=path-item-object.d.ts.map