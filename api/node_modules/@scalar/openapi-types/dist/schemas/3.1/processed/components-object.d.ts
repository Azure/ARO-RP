import { z } from 'zod';
import { CallbackObjectSchema } from './callback-object.js';
/**
 * Components Object
 *
 * Holds a set of reusable objects for different aspects of the OAS. All objects defined within the Components Object
 * will have no effect on the API unless they are explicitly referenced from outside the Components Object.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#components-object
 */
export declare const ComponentsObjectSchema: z.ZodObject<{
    schemas: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodType<Record<string, any>, unknown, z.core.$ZodTypeInternals<Record<string, any>, unknown>>>>;
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
    parameters: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
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
    examples: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
        summary: z.ZodOptional<z.ZodString>;
        description: z.ZodOptional<z.ZodString>;
        value: z.ZodOptional<z.ZodAny>;
        externalValue: z.ZodOptional<z.ZodString>;
    }, z.core.$strip>>>;
    requestBodies: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
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
    }, z.core.$strip>>>;
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
    securitySchemes: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodUnion<readonly [z.ZodObject<{
        description: z.ZodOptional<z.ZodString>;
        type: z.ZodLiteral<"apiKey">;
        name: z.ZodDefault<z.ZodOptional<z.ZodString>>;
        in: z.ZodCatch<z.ZodDefault<z.ZodOptional<z.ZodEnum<{
            query: "query";
            cookie: "cookie";
            header: "header";
        }>>>>;
    }, z.core.$strip>, z.ZodObject<{
        description: z.ZodOptional<z.ZodString>;
        type: z.ZodLiteral<"http">;
        scheme: z.ZodDefault<z.ZodOptional<z.ZodPipe<z.ZodString, z.ZodEnum<{
            basic: "basic";
            bearer: "bearer";
        }>>>>;
        bearerFormat: z.ZodOptional<z.ZodUnion<readonly [z.ZodLiteral<"JWT">, z.ZodLiteral<"bearer">, z.ZodString]>>;
    }, z.core.$strip>, z.ZodObject<{
        description: z.ZodOptional<z.ZodString>;
        type: z.ZodLiteral<"mutualTLS">;
    }, z.core.$strip>, z.ZodObject<{
        description: z.ZodOptional<z.ZodString>;
        type: z.ZodLiteral<"oauth2">;
        flows: z.ZodObject<{
            implicit: z.ZodOptional<z.ZodOptional<z.ZodObject<{
                refreshUrl: z.ZodOptional<z.ZodString>;
                scopes: z.ZodCatch<z.ZodDefault<z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodOptional<z.ZodString>>>>>;
                type: z.ZodOptional<z.ZodLiteral<"implicit">>;
                authorizationUrl: z.ZodDefault<z.ZodString>;
            }, z.core.$strip>>>;
            password: z.ZodOptional<z.ZodOptional<z.ZodObject<{
                refreshUrl: z.ZodOptional<z.ZodString>;
                scopes: z.ZodCatch<z.ZodDefault<z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodOptional<z.ZodString>>>>>;
                type: z.ZodOptional<z.ZodLiteral<"password">>;
                tokenUrl: z.ZodDefault<z.ZodString>;
            }, z.core.$strip>>>;
            clientCredentials: z.ZodOptional<z.ZodOptional<z.ZodObject<{
                refreshUrl: z.ZodOptional<z.ZodString>;
                scopes: z.ZodCatch<z.ZodDefault<z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodOptional<z.ZodString>>>>>;
                type: z.ZodOptional<z.ZodLiteral<"clientCredentials">>;
                tokenUrl: z.ZodDefault<z.ZodString>;
            }, z.core.$strip>>>;
            authorizationCode: z.ZodOptional<z.ZodOptional<z.ZodObject<{
                refreshUrl: z.ZodOptional<z.ZodString>;
                scopes: z.ZodCatch<z.ZodDefault<z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodOptional<z.ZodString>>>>>;
                type: z.ZodOptional<z.ZodLiteral<"authorizationCode">>;
                authorizationUrl: z.ZodDefault<z.ZodString>;
                tokenUrl: z.ZodDefault<z.ZodString>;
            }, z.core.$strip>>>;
        }, z.core.$strip>;
    }, z.core.$strip>, z.ZodObject<{
        description: z.ZodOptional<z.ZodString>;
        type: z.ZodLiteral<"openIdConnect">;
        openIdConnectUrl: z.ZodDefault<z.ZodOptional<z.ZodString>>;
    }, z.core.$strip>]>>>;
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
    callbacks: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodRecord<z.ZodString, z.ZodType<{
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
        get?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
        put?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
        post?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
        delete?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
        options?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
        head?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
        patch?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
        trace?: z.infer<typeof import("./operation-object-without-callbacks.js").OperationObjectSchemaWithoutCallbacks>;
    }, unknown>>>>>;
    pathItems: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
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
            callbacks?: Record<string, z.infer<typeof CallbackObjectSchema>>;
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
            callbacks?: Record<string, z.infer<typeof CallbackObjectSchema>>;
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
            callbacks?: Record<string, z.infer<typeof CallbackObjectSchema>>;
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
            callbacks?: Record<string, z.infer<typeof CallbackObjectSchema>>;
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
            callbacks?: Record<string, z.infer<typeof CallbackObjectSchema>>;
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
            callbacks?: Record<string, z.infer<typeof CallbackObjectSchema>>;
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
            callbacks?: Record<string, z.infer<typeof CallbackObjectSchema>>;
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
            callbacks?: Record<string, z.infer<typeof CallbackObjectSchema>>;
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
            callbacks?: Record<string, z.infer<typeof CallbackObjectSchema>>;
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
            callbacks?: Record<string, z.infer<typeof CallbackObjectSchema>>;
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
            callbacks?: Record<string, z.infer<typeof CallbackObjectSchema>>;
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
            callbacks?: Record<string, z.infer<typeof CallbackObjectSchema>>;
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
            callbacks?: Record<string, z.infer<typeof CallbackObjectSchema>>;
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
            callbacks?: Record<string, z.infer<typeof CallbackObjectSchema>>;
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
            callbacks?: Record<string, z.infer<typeof CallbackObjectSchema>>;
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
            callbacks?: Record<string, z.infer<typeof CallbackObjectSchema>>;
        }, unknown>>>;
    }, z.core.$strip>>>;
}, z.core.$strip>;
//# sourceMappingURL=components-object.d.ts.map