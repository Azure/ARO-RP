import { z } from 'zod';
/**
 * Webhooks Object
 *
 * The incoming webhooks that MAY be received as part of this API and that the API consumer MAY choose to implement.
 * Closely related to the callbacks feature, this section describes requests initiated other than by an API call, for
 * example by an out of band registration.
 *
 * The key name is a unique string to refer to each webhook, while the
 * (optionally referenced) Path Item Object describes a request that may be initiated by the API provider and the
 * expected responses. An example is available.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#oas-webhooks
 */
export declare const WebhooksObjectSchema: z.ZodRecord<z.ZodString, z.ZodType<{
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
} & {
    get?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    put?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    post?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    delete?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    options?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    head?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    patch?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    trace?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
}, unknown, z.core.$ZodTypeInternals<{
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
} & {
    get?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    put?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    post?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    delete?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    options?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    head?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    patch?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    trace?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
}, unknown>>>;
//# sourceMappingURL=webhooks-object.d.ts.map