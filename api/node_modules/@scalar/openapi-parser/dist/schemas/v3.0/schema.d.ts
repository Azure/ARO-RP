declare const _default: {
    id: string;
    $schema: string;
    description: string;
    type: string;
    required: string[];
    properties: {
        openapi: {
            type: string;
            pattern: string;
        };
        info: {
            $ref: string;
        };
        externalDocs: {
            $ref: string;
        };
        servers: {
            type: string;
            items: {
                $ref: string;
            };
        };
        security: {
            type: string;
            items: {
                $ref: string;
            };
        };
        tags: {
            type: string;
            items: {
                $ref: string;
            };
            uniqueItems: boolean;
        };
        paths: {
            $ref: string;
        };
        components: {
            $ref: string;
        };
    };
    patternProperties: {
        "^x-": {};
    };
    additionalProperties: boolean;
    definitions: {
        Reference: {
            type: string;
            required: string[];
            patternProperties: {
                "^\\$ref$": {
                    type: string;
                    format: string;
                };
            };
        };
        Info: {
            type: string;
            required: string[];
            properties: {
                title: {
                    type: string;
                };
                description: {
                    type: string;
                };
                termsOfService: {
                    type: string;
                    format: string;
                };
                contact: {
                    $ref: string;
                };
                license: {
                    $ref: string;
                };
                version: {
                    type: string;
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        Contact: {
            type: string;
            properties: {
                name: {
                    type: string;
                };
                url: {
                    type: string;
                    format: string;
                };
                email: {
                    type: string;
                    format: string;
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        License: {
            type: string;
            required: string[];
            properties: {
                name: {
                    type: string;
                };
                url: {
                    type: string;
                    format: string;
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        Server: {
            type: string;
            required: string[];
            properties: {
                url: {
                    type: string;
                };
                description: {
                    type: string;
                };
                variables: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        ServerVariable: {
            type: string;
            required: string[];
            properties: {
                enum: {
                    type: string;
                    items: {
                        type: string;
                    };
                };
                default: {
                    type: string;
                };
                description: {
                    type: string;
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        Components: {
            type: string;
            properties: {
                schemas: {
                    type: string;
                    patternProperties: {
                        "^[a-zA-Z0-9\\.\\-_]+$": {
                            oneOf: {
                                $ref: string;
                            }[];
                        };
                    };
                };
                responses: {
                    type: string;
                    patternProperties: {
                        "^[a-zA-Z0-9\\.\\-_]+$": {
                            oneOf: {
                                $ref: string;
                            }[];
                        };
                    };
                };
                parameters: {
                    type: string;
                    patternProperties: {
                        "^[a-zA-Z0-9\\.\\-_]+$": {
                            oneOf: {
                                $ref: string;
                            }[];
                        };
                    };
                };
                examples: {
                    type: string;
                    patternProperties: {
                        "^[a-zA-Z0-9\\.\\-_]+$": {
                            oneOf: {
                                $ref: string;
                            }[];
                        };
                    };
                };
                requestBodies: {
                    type: string;
                    patternProperties: {
                        "^[a-zA-Z0-9\\.\\-_]+$": {
                            oneOf: {
                                $ref: string;
                            }[];
                        };
                    };
                };
                headers: {
                    type: string;
                    patternProperties: {
                        "^[a-zA-Z0-9\\.\\-_]+$": {
                            oneOf: {
                                $ref: string;
                            }[];
                        };
                    };
                };
                securitySchemes: {
                    type: string;
                    patternProperties: {
                        "^[a-zA-Z0-9\\.\\-_]+$": {
                            oneOf: {
                                $ref: string;
                            }[];
                        };
                    };
                };
                links: {
                    type: string;
                    patternProperties: {
                        "^[a-zA-Z0-9\\.\\-_]+$": {
                            oneOf: {
                                $ref: string;
                            }[];
                        };
                    };
                };
                callbacks: {
                    type: string;
                    patternProperties: {
                        "^[a-zA-Z0-9\\.\\-_]+$": {
                            oneOf: {
                                $ref: string;
                            }[];
                        };
                    };
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        Schema: {
            type: string;
            properties: {
                title: {
                    type: string;
                };
                multipleOf: {
                    type: string;
                    minimum: number;
                    exclusiveMinimum: boolean;
                };
                maximum: {
                    type: string;
                };
                exclusiveMaximum: {
                    type: string;
                    default: boolean;
                };
                minimum: {
                    type: string;
                };
                exclusiveMinimum: {
                    type: string;
                    default: boolean;
                };
                maxLength: {
                    type: string;
                    minimum: number;
                };
                minLength: {
                    type: string;
                    minimum: number;
                    default: number;
                };
                pattern: {
                    type: string;
                    format: string;
                };
                maxItems: {
                    type: string;
                    minimum: number;
                };
                minItems: {
                    type: string;
                    minimum: number;
                    default: number;
                };
                uniqueItems: {
                    type: string;
                    default: boolean;
                };
                maxProperties: {
                    type: string;
                    minimum: number;
                };
                minProperties: {
                    type: string;
                    minimum: number;
                    default: number;
                };
                required: {
                    type: string;
                    items: {
                        type: string;
                    };
                    minItems: number;
                    uniqueItems: boolean;
                };
                enum: {
                    type: string;
                    items: {};
                    minItems: number;
                    uniqueItems: boolean;
                };
                type: {
                    type: string;
                    enum: string[];
                };
                not: {
                    oneOf: {
                        $ref: string;
                    }[];
                };
                allOf: {
                    type: string;
                    items: {
                        oneOf: {
                            $ref: string;
                        }[];
                    };
                };
                oneOf: {
                    type: string;
                    items: {
                        oneOf: {
                            $ref: string;
                        }[];
                    };
                };
                anyOf: {
                    type: string;
                    items: {
                        oneOf: {
                            $ref: string;
                        }[];
                    };
                };
                items: {
                    oneOf: {
                        $ref: string;
                    }[];
                };
                properties: {
                    type: string;
                    additionalProperties: {
                        oneOf: {
                            $ref: string;
                        }[];
                    };
                };
                additionalProperties: {
                    oneOf: ({
                        $ref: string;
                        type?: undefined;
                    } | {
                        type: string;
                        $ref?: undefined;
                    })[];
                    default: boolean;
                };
                description: {
                    type: string;
                };
                format: {
                    type: string;
                };
                default: {};
                nullable: {
                    type: string;
                    default: boolean;
                };
                discriminator: {
                    $ref: string;
                };
                readOnly: {
                    type: string;
                    default: boolean;
                };
                writeOnly: {
                    type: string;
                    default: boolean;
                };
                example: {};
                externalDocs: {
                    $ref: string;
                };
                deprecated: {
                    type: string;
                    default: boolean;
                };
                xml: {
                    $ref: string;
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        Discriminator: {
            type: string;
            required: string[];
            properties: {
                propertyName: {
                    type: string;
                };
                mapping: {
                    type: string;
                    additionalProperties: {
                        type: string;
                    };
                };
            };
        };
        XML: {
            type: string;
            properties: {
                name: {
                    type: string;
                };
                namespace: {
                    type: string;
                    format: string;
                };
                prefix: {
                    type: string;
                };
                attribute: {
                    type: string;
                    default: boolean;
                };
                wrapped: {
                    type: string;
                    default: boolean;
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        Response: {
            type: string;
            required: string[];
            properties: {
                description: {
                    type: string;
                };
                headers: {
                    type: string;
                    additionalProperties: {
                        oneOf: {
                            $ref: string;
                        }[];
                    };
                };
                content: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                };
                links: {
                    type: string;
                    additionalProperties: {
                        oneOf: {
                            $ref: string;
                        }[];
                    };
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        MediaType: {
            type: string;
            properties: {
                schema: {
                    oneOf: {
                        $ref: string;
                    }[];
                };
                example: {};
                examples: {
                    type: string;
                    additionalProperties: {
                        oneOf: {
                            $ref: string;
                        }[];
                    };
                };
                encoding: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
            allOf: {
                $ref: string;
            }[];
        };
        Example: {
            type: string;
            properties: {
                summary: {
                    type: string;
                };
                description: {
                    type: string;
                };
                value: {};
                externalValue: {
                    type: string;
                    format: string;
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        Header: {
            type: string;
            properties: {
                description: {
                    type: string;
                };
                required: {
                    type: string;
                    default: boolean;
                };
                deprecated: {
                    type: string;
                    default: boolean;
                };
                allowEmptyValue: {
                    type: string;
                    default: boolean;
                };
                style: {
                    type: string;
                    enum: string[];
                    default: string;
                };
                explode: {
                    type: string;
                };
                allowReserved: {
                    type: string;
                    default: boolean;
                };
                schema: {
                    oneOf: {
                        $ref: string;
                    }[];
                };
                content: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                    minProperties: number;
                    maxProperties: number;
                };
                example: {};
                examples: {
                    type: string;
                    additionalProperties: {
                        oneOf: {
                            $ref: string;
                        }[];
                    };
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
            allOf: {
                $ref: string;
            }[];
        };
        Paths: {
            type: string;
            patternProperties: {
                "^\\/": {
                    $ref: string;
                };
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        PathItem: {
            type: string;
            properties: {
                summary: {
                    type: string;
                };
                description: {
                    type: string;
                };
                servers: {
                    type: string;
                    items: {
                        $ref: string;
                    };
                };
                parameters: {
                    type: string;
                    items: {
                        oneOf: {
                            $ref: string;
                        }[];
                    };
                    uniqueItems: boolean;
                };
            };
            patternProperties: {
                "^(get|put|post|delete|options|head|patch|trace)$": {
                    $ref: string;
                };
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        PathItemOrReference: {
            oneOf: {
                $ref: string;
            }[];
        };
        Operation: {
            type: string;
            required: string[];
            properties: {
                tags: {
                    type: string;
                    items: {
                        type: string;
                    };
                };
                summary: {
                    type: string;
                };
                description: {
                    type: string;
                };
                externalDocs: {
                    $ref: string;
                };
                operationId: {
                    type: string;
                };
                parameters: {
                    type: string;
                    items: {
                        oneOf: {
                            $ref: string;
                        }[];
                    };
                    uniqueItems: boolean;
                };
                requestBody: {
                    oneOf: {
                        $ref: string;
                    }[];
                };
                responses: {
                    $ref: string;
                };
                callbacks: {
                    type: string;
                    additionalProperties: {
                        oneOf: {
                            $ref: string;
                        }[];
                    };
                };
                deprecated: {
                    type: string;
                    default: boolean;
                };
                security: {
                    type: string;
                    items: {
                        $ref: string;
                    };
                };
                servers: {
                    type: string;
                    items: {
                        $ref: string;
                    };
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        Responses: {
            type: string;
            properties: {
                default: {
                    oneOf: {
                        $ref: string;
                    }[];
                };
            };
            patternProperties: {
                "^[1-5](?:\\d{2}|XX)$": {
                    oneOf: {
                        $ref: string;
                    }[];
                };
                "^x-": {};
            };
            minProperties: number;
            additionalProperties: boolean;
        };
        SecurityRequirement: {
            type: string;
            additionalProperties: {
                type: string;
                items: {
                    type: string;
                };
            };
        };
        Tag: {
            type: string;
            required: string[];
            properties: {
                name: {
                    type: string;
                };
                description: {
                    type: string;
                };
                externalDocs: {
                    $ref: string;
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        ExternalDocumentation: {
            type: string;
            required: string[];
            properties: {
                description: {
                    type: string;
                };
                url: {
                    type: string;
                    format: string;
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        ExampleXORExamples: {
            description: string;
            not: {
                required: string[];
            };
        };
        SchemaXORContent: {
            description: string;
            not: {
                required: string[];
            };
            oneOf: ({
                required: string[];
                description?: undefined;
                allOf?: undefined;
            } | {
                required: string[];
                description: string;
                allOf: {
                    not: {
                        required: string[];
                    };
                }[];
            })[];
        };
        Parameter: {
            type: string;
            properties: {
                name: {
                    type: string;
                };
                in: {
                    type: string;
                };
                description: {
                    type: string;
                };
                required: {
                    type: string;
                    default: boolean;
                };
                deprecated: {
                    type: string;
                    default: boolean;
                };
                allowEmptyValue: {
                    type: string;
                    default: boolean;
                };
                style: {
                    type: string;
                };
                explode: {
                    type: string;
                };
                allowReserved: {
                    type: string;
                    default: boolean;
                };
                schema: {
                    oneOf: {
                        $ref: string;
                    }[];
                };
                content: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                    minProperties: number;
                    maxProperties: number;
                };
                example: {};
                examples: {
                    type: string;
                    additionalProperties: {
                        oneOf: {
                            $ref: string;
                        }[];
                    };
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
            required: string[];
            allOf: {
                $ref: string;
            }[];
        };
        ParameterLocation: {
            description: string;
            oneOf: ({
                description: string;
                required: string[];
                properties: {
                    in: {
                        enum: string[];
                    };
                    style: {
                        enum: string[];
                        default: string;
                    };
                    required: {
                        enum: boolean[];
                    };
                };
            } | {
                description: string;
                properties: {
                    in: {
                        enum: string[];
                    };
                    style: {
                        enum: string[];
                        default: string;
                    };
                    required?: undefined;
                };
                required?: undefined;
            })[];
        };
        RequestBody: {
            type: string;
            required: string[];
            properties: {
                description: {
                    type: string;
                };
                content: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                };
                required: {
                    type: string;
                    default: boolean;
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        SecurityScheme: {
            oneOf: {
                $ref: string;
            }[];
            discriminator: {
                propertyName: string;
            };
        };
        APIKeySecurityScheme: {
            type: string;
            required: string[];
            properties: {
                type: {
                    type: string;
                    enum: string[];
                };
                name: {
                    type: string;
                };
                in: {
                    type: string;
                    enum: string[];
                };
                description: {
                    type: string;
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        HTTPSecurityScheme: {
            type: string;
            required: string[];
            properties: {
                scheme: {
                    type: string;
                };
                bearerFormat: {
                    type: string;
                };
                description: {
                    type: string;
                };
                type: {
                    type: string;
                    enum: string[];
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
            oneOf: ({
                description: string;
                properties: {
                    scheme: {
                        type: string;
                        pattern: string;
                        not?: undefined;
                    };
                };
                not?: undefined;
            } | {
                description: string;
                not: {
                    required: string[];
                };
                properties: {
                    scheme: {
                        not: {
                            type: string;
                            pattern: string;
                        };
                        type?: undefined;
                        pattern?: undefined;
                    };
                };
            })[];
        };
        OAuth2SecurityScheme: {
            type: string;
            required: string[];
            properties: {
                type: {
                    type: string;
                    enum: string[];
                };
                flows: {
                    $ref: string;
                };
                description: {
                    type: string;
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        OpenIdConnectSecurityScheme: {
            type: string;
            required: string[];
            properties: {
                type: {
                    type: string;
                    enum: string[];
                };
                openIdConnectUrl: {
                    type: string;
                    format: string;
                };
                description: {
                    type: string;
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        OAuthFlows: {
            type: string;
            properties: {
                implicit: {
                    $ref: string;
                };
                password: {
                    $ref: string;
                };
                clientCredentials: {
                    $ref: string;
                };
                authorizationCode: {
                    $ref: string;
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        ImplicitOAuthFlow: {
            type: string;
            required: string[];
            properties: {
                authorizationUrl: {
                    type: string;
                    format: string;
                };
                refreshUrl: {
                    type: string;
                    format: string;
                };
                scopes: {
                    type: string;
                    additionalProperties: {
                        type: string;
                    };
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        PasswordOAuthFlow: {
            type: string;
            required: string[];
            properties: {
                tokenUrl: {
                    type: string;
                    format: string;
                };
                refreshUrl: {
                    type: string;
                    format: string;
                };
                scopes: {
                    type: string;
                    additionalProperties: {
                        type: string;
                    };
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        ClientCredentialsFlow: {
            type: string;
            required: string[];
            properties: {
                tokenUrl: {
                    type: string;
                    format: string;
                };
                refreshUrl: {
                    type: string;
                    format: string;
                };
                scopes: {
                    type: string;
                    additionalProperties: {
                        type: string;
                    };
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        AuthorizationCodeOAuthFlow: {
            type: string;
            required: string[];
            properties: {
                authorizationUrl: {
                    type: string;
                    format: string;
                };
                tokenUrl: {
                    type: string;
                    format: string;
                };
                refreshUrl: {
                    type: string;
                    format: string;
                };
                scopes: {
                    type: string;
                    additionalProperties: {
                        type: string;
                    };
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
        };
        Link: {
            type: string;
            properties: {
                operationId: {
                    type: string;
                };
                operationRef: {
                    type: string;
                    format: string;
                };
                parameters: {
                    type: string;
                    additionalProperties: {};
                };
                requestBody: {};
                description: {
                    type: string;
                };
                server: {
                    $ref: string;
                };
            };
            patternProperties: {
                "^x-": {};
            };
            additionalProperties: boolean;
            not: {
                description: string;
                required: string[];
            };
        };
        Callback: {
            type: string;
            additionalProperties: {
                $ref: string;
            };
            patternProperties: {
                "^x-": {};
            };
        };
        Encoding: {
            type: string;
            properties: {
                contentType: {
                    type: string;
                };
                headers: {
                    type: string;
                    additionalProperties: {
                        oneOf: {
                            $ref: string;
                        }[];
                    };
                };
                style: {
                    type: string;
                    enum: string[];
                };
                explode: {
                    type: string;
                };
                allowReserved: {
                    type: string;
                    default: boolean;
                };
            };
            additionalProperties: boolean;
        };
    };
};
export default _default;
//# sourceMappingURL=schema.d.ts.map