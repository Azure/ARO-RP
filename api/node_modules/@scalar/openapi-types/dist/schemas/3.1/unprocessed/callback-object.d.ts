/**
 * Callback Object
 *
 * A map of possible out-of-band callbacks related to the parent operation. Each value in the map is a
 * Path Item Object that describes a set of requests that may be initiated by the API provider and the
 * expected responses. The key value used to identify the callback object is an expression, evaluated
 * at runtime, that identifies a URL to be used for the callback operation.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#callback-object
 */
export declare const CallbackObjectSchema: import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodType<{
    summary?: string | undefined;
    description?: string | undefined;
    servers?: {
        url: string;
        description?: string | undefined;
        variables?: Record<string, {
            enum?: string[] | undefined;
            default?: string | undefined;
            description?: string | undefined;
        }> | undefined;
    }[] | undefined;
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
} & {
    get?: import("zod").infer<typeof import("../processed/operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    put?: import("zod").infer<typeof import("../processed/operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    post?: import("zod").infer<typeof import("../processed/operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    delete?: import("zod").infer<typeof import("../processed/operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    options?: import("zod").infer<typeof import("../processed/operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    head?: import("zod").infer<typeof import("../processed/operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    patch?: import("zod").infer<typeof import("../processed/operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    trace?: import("zod").infer<typeof import("../processed/operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
}, unknown, import("zod/v4/core").$ZodTypeInternals<{
    summary?: string | undefined;
    description?: string | undefined;
    servers?: {
        url: string;
        description?: string | undefined;
        variables?: Record<string, {
            enum?: string[] | undefined;
            default?: string | undefined;
            description?: string | undefined;
        }> | undefined;
    }[] | undefined;
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
} & {
    get?: import("zod").infer<typeof import("../processed/operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    put?: import("zod").infer<typeof import("../processed/operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    post?: import("zod").infer<typeof import("../processed/operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    delete?: import("zod").infer<typeof import("../processed/operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    options?: import("zod").infer<typeof import("../processed/operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    head?: import("zod").infer<typeof import("../processed/operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    patch?: import("zod").infer<typeof import("../processed/operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    trace?: import("zod").infer<typeof import("../processed/operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
}, unknown>>>;
//# sourceMappingURL=callback-object.d.ts.map