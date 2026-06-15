import { z } from 'zod';
/**
 * Path Item Object
 *
 * Describes the operations available on a single path. A Path Item MAY be empty, due to ACL constraints. The path
 * itself is still exposed to the documentation viewer but they will not know which operations and parameters are
 * available.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#path-item-object
 */
export declare const PathItemObjectSchema: z.ZodObject<{
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
    $ref: z.ZodOptional<z.ZodString>;
    get: z.ZodOptional<z.ZodType<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        operationId?: string | undefined;
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
        deprecated?: boolean | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        parameters?: ({
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | {
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
            } | {
                $ref: string;
                summary?: string | undefined;
                description?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: any;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                } | {
                    $ref: string;
                    summary?: string | undefined;
                    description?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        $ref: string;
                        summary?: string | undefined;
                        description?: string | undefined;
                    } | {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        example?: any;
                        schema?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        } | {
                            $ref: string;
                            summary?: string | undefined;
                            description?: string | undefined;
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
        })[] | undefined;
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
        } | {
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | undefined;
        security?: Record<string, string[]>[] | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./reference-object.js").ReferenceObjectSchema> | z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, z.core.$ZodTypeInternals<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        operationId?: string | undefined;
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
        deprecated?: boolean | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        parameters?: ({
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | {
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
            } | {
                $ref: string;
                summary?: string | undefined;
                description?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: any;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                } | {
                    $ref: string;
                    summary?: string | undefined;
                    description?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        $ref: string;
                        summary?: string | undefined;
                        description?: string | undefined;
                    } | {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        example?: any;
                        schema?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        } | {
                            $ref: string;
                            summary?: string | undefined;
                            description?: string | undefined;
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
        })[] | undefined;
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
        } | {
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | undefined;
        security?: Record<string, string[]>[] | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./reference-object.js").ReferenceObjectSchema> | z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
    put: z.ZodOptional<z.ZodType<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        operationId?: string | undefined;
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
        deprecated?: boolean | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        parameters?: ({
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | {
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
            } | {
                $ref: string;
                summary?: string | undefined;
                description?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: any;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                } | {
                    $ref: string;
                    summary?: string | undefined;
                    description?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        $ref: string;
                        summary?: string | undefined;
                        description?: string | undefined;
                    } | {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        example?: any;
                        schema?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        } | {
                            $ref: string;
                            summary?: string | undefined;
                            description?: string | undefined;
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
        })[] | undefined;
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
        } | {
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | undefined;
        security?: Record<string, string[]>[] | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./reference-object.js").ReferenceObjectSchema> | z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, z.core.$ZodTypeInternals<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        operationId?: string | undefined;
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
        deprecated?: boolean | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        parameters?: ({
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | {
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
            } | {
                $ref: string;
                summary?: string | undefined;
                description?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: any;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                } | {
                    $ref: string;
                    summary?: string | undefined;
                    description?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        $ref: string;
                        summary?: string | undefined;
                        description?: string | undefined;
                    } | {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        example?: any;
                        schema?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        } | {
                            $ref: string;
                            summary?: string | undefined;
                            description?: string | undefined;
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
        })[] | undefined;
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
        } | {
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | undefined;
        security?: Record<string, string[]>[] | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./reference-object.js").ReferenceObjectSchema> | z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
    post: z.ZodOptional<z.ZodType<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        operationId?: string | undefined;
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
        deprecated?: boolean | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        parameters?: ({
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | {
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
            } | {
                $ref: string;
                summary?: string | undefined;
                description?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: any;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                } | {
                    $ref: string;
                    summary?: string | undefined;
                    description?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        $ref: string;
                        summary?: string | undefined;
                        description?: string | undefined;
                    } | {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        example?: any;
                        schema?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        } | {
                            $ref: string;
                            summary?: string | undefined;
                            description?: string | undefined;
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
        })[] | undefined;
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
        } | {
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | undefined;
        security?: Record<string, string[]>[] | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./reference-object.js").ReferenceObjectSchema> | z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, z.core.$ZodTypeInternals<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        operationId?: string | undefined;
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
        deprecated?: boolean | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        parameters?: ({
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | {
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
            } | {
                $ref: string;
                summary?: string | undefined;
                description?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: any;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                } | {
                    $ref: string;
                    summary?: string | undefined;
                    description?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        $ref: string;
                        summary?: string | undefined;
                        description?: string | undefined;
                    } | {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        example?: any;
                        schema?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        } | {
                            $ref: string;
                            summary?: string | undefined;
                            description?: string | undefined;
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
        })[] | undefined;
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
        } | {
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | undefined;
        security?: Record<string, string[]>[] | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./reference-object.js").ReferenceObjectSchema> | z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
    delete: z.ZodOptional<z.ZodType<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        operationId?: string | undefined;
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
        deprecated?: boolean | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        parameters?: ({
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | {
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
            } | {
                $ref: string;
                summary?: string | undefined;
                description?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: any;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                } | {
                    $ref: string;
                    summary?: string | undefined;
                    description?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        $ref: string;
                        summary?: string | undefined;
                        description?: string | undefined;
                    } | {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        example?: any;
                        schema?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        } | {
                            $ref: string;
                            summary?: string | undefined;
                            description?: string | undefined;
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
        })[] | undefined;
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
        } | {
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | undefined;
        security?: Record<string, string[]>[] | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./reference-object.js").ReferenceObjectSchema> | z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, z.core.$ZodTypeInternals<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        operationId?: string | undefined;
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
        deprecated?: boolean | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        parameters?: ({
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | {
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
            } | {
                $ref: string;
                summary?: string | undefined;
                description?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: any;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                } | {
                    $ref: string;
                    summary?: string | undefined;
                    description?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        $ref: string;
                        summary?: string | undefined;
                        description?: string | undefined;
                    } | {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        example?: any;
                        schema?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        } | {
                            $ref: string;
                            summary?: string | undefined;
                            description?: string | undefined;
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
        })[] | undefined;
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
        } | {
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | undefined;
        security?: Record<string, string[]>[] | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./reference-object.js").ReferenceObjectSchema> | z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
    options: z.ZodOptional<z.ZodType<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        operationId?: string | undefined;
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
        deprecated?: boolean | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        parameters?: ({
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | {
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
            } | {
                $ref: string;
                summary?: string | undefined;
                description?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: any;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                } | {
                    $ref: string;
                    summary?: string | undefined;
                    description?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        $ref: string;
                        summary?: string | undefined;
                        description?: string | undefined;
                    } | {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        example?: any;
                        schema?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        } | {
                            $ref: string;
                            summary?: string | undefined;
                            description?: string | undefined;
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
        })[] | undefined;
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
        } | {
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | undefined;
        security?: Record<string, string[]>[] | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./reference-object.js").ReferenceObjectSchema> | z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, z.core.$ZodTypeInternals<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        operationId?: string | undefined;
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
        deprecated?: boolean | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        parameters?: ({
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | {
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
            } | {
                $ref: string;
                summary?: string | undefined;
                description?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: any;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                } | {
                    $ref: string;
                    summary?: string | undefined;
                    description?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        $ref: string;
                        summary?: string | undefined;
                        description?: string | undefined;
                    } | {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        example?: any;
                        schema?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        } | {
                            $ref: string;
                            summary?: string | undefined;
                            description?: string | undefined;
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
        })[] | undefined;
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
        } | {
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | undefined;
        security?: Record<string, string[]>[] | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./reference-object.js").ReferenceObjectSchema> | z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
    head: z.ZodOptional<z.ZodType<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        operationId?: string | undefined;
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
        deprecated?: boolean | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        parameters?: ({
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | {
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
            } | {
                $ref: string;
                summary?: string | undefined;
                description?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: any;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                } | {
                    $ref: string;
                    summary?: string | undefined;
                    description?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        $ref: string;
                        summary?: string | undefined;
                        description?: string | undefined;
                    } | {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        example?: any;
                        schema?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        } | {
                            $ref: string;
                            summary?: string | undefined;
                            description?: string | undefined;
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
        })[] | undefined;
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
        } | {
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | undefined;
        security?: Record<string, string[]>[] | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./reference-object.js").ReferenceObjectSchema> | z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, z.core.$ZodTypeInternals<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        operationId?: string | undefined;
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
        deprecated?: boolean | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        parameters?: ({
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | {
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
            } | {
                $ref: string;
                summary?: string | undefined;
                description?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: any;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                } | {
                    $ref: string;
                    summary?: string | undefined;
                    description?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        $ref: string;
                        summary?: string | undefined;
                        description?: string | undefined;
                    } | {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        example?: any;
                        schema?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        } | {
                            $ref: string;
                            summary?: string | undefined;
                            description?: string | undefined;
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
        })[] | undefined;
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
        } | {
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | undefined;
        security?: Record<string, string[]>[] | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./reference-object.js").ReferenceObjectSchema> | z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
    patch: z.ZodOptional<z.ZodType<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        operationId?: string | undefined;
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
        deprecated?: boolean | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        parameters?: ({
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | {
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
            } | {
                $ref: string;
                summary?: string | undefined;
                description?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: any;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                } | {
                    $ref: string;
                    summary?: string | undefined;
                    description?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        $ref: string;
                        summary?: string | undefined;
                        description?: string | undefined;
                    } | {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        example?: any;
                        schema?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        } | {
                            $ref: string;
                            summary?: string | undefined;
                            description?: string | undefined;
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
        })[] | undefined;
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
        } | {
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | undefined;
        security?: Record<string, string[]>[] | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./reference-object.js").ReferenceObjectSchema> | z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, z.core.$ZodTypeInternals<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        operationId?: string | undefined;
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
        deprecated?: boolean | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        parameters?: ({
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | {
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
            } | {
                $ref: string;
                summary?: string | undefined;
                description?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: any;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                } | {
                    $ref: string;
                    summary?: string | undefined;
                    description?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        $ref: string;
                        summary?: string | undefined;
                        description?: string | undefined;
                    } | {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        example?: any;
                        schema?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        } | {
                            $ref: string;
                            summary?: string | undefined;
                            description?: string | undefined;
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
        })[] | undefined;
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
        } | {
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | undefined;
        security?: Record<string, string[]>[] | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./reference-object.js").ReferenceObjectSchema> | z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
    trace: z.ZodOptional<z.ZodType<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        operationId?: string | undefined;
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
        deprecated?: boolean | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        parameters?: ({
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | {
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
            } | {
                $ref: string;
                summary?: string | undefined;
                description?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: any;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                } | {
                    $ref: string;
                    summary?: string | undefined;
                    description?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        $ref: string;
                        summary?: string | undefined;
                        description?: string | undefined;
                    } | {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        example?: any;
                        schema?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        } | {
                            $ref: string;
                            summary?: string | undefined;
                            description?: string | undefined;
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
        })[] | undefined;
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
        } | {
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | undefined;
        security?: Record<string, string[]>[] | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./reference-object.js").ReferenceObjectSchema> | z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown, z.core.$ZodTypeInternals<{
        tags?: string[] | undefined;
        summary?: string | undefined;
        description?: string | undefined;
        operationId?: string | undefined;
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
        deprecated?: boolean | undefined;
        externalDocs?: {
            url: string;
            description?: string | undefined;
        } | undefined;
        parameters?: ({
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | {
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
            } | {
                $ref: string;
                summary?: string | undefined;
                description?: string | undefined;
            }> | undefined;
            content?: Record<string, {
                schema?: any;
                example?: any;
                examples?: Record<string, {
                    summary?: string | undefined;
                    description?: string | undefined;
                    value?: any;
                    externalValue?: string | undefined;
                } | {
                    $ref: string;
                    summary?: string | undefined;
                    description?: string | undefined;
                }> | undefined;
                encoding?: Record<string, {
                    contentType: string;
                    headers?: Record<string, {
                        $ref: string;
                        summary?: string | undefined;
                        description?: string | undefined;
                    } | {
                        description?: string | undefined;
                        required?: boolean | undefined;
                        deprecated?: boolean | undefined;
                        style?: "matrix" | "label" | "form" | "simple" | "spaceDelimited" | "pipeDelimited" | "deepObject" | undefined;
                        explode?: boolean | undefined;
                        example?: any;
                        schema?: any;
                        examples?: Record<string, {
                            summary?: string | undefined;
                            description?: string | undefined;
                            value?: any;
                            externalValue?: string | undefined;
                        } | {
                            $ref: string;
                            summary?: string | undefined;
                            description?: string | undefined;
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
        })[] | undefined;
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
        } | {
            $ref: string;
            summary?: string | undefined;
            description?: string | undefined;
        } | undefined;
        security?: Record<string, string[]>[] | undefined;
    } & {
        callbacks?: Record<string, z.infer<typeof import("./reference-object.js").ReferenceObjectSchema> | z.infer<typeof import("./callback-object.js").CallbackObjectSchema>>;
    }, unknown>>>;
}, z.core.$strip>;
//# sourceMappingURL=path-item-object.d.ts.map