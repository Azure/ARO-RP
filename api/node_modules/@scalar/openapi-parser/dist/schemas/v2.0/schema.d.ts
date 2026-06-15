declare const _default: {
    title: string;
    id: string;
    $schema: string;
    type: string;
    required: string[];
    additionalProperties: boolean;
    patternProperties: {
        "^x-": {
            $ref: string;
        };
    };
    properties: {
        swagger: {
            type: string;
            enum: string[];
            description: string;
        };
        info: {
            $ref: string;
        };
        host: {
            type: string;
            pattern: string;
            description: string;
        };
        basePath: {
            type: string;
            pattern: string;
            description: string;
        };
        schemes: {
            $ref: string;
        };
        consumes: {
            description: string;
            allOf: {
                $ref: string;
            }[];
        };
        produces: {
            description: string;
            allOf: {
                $ref: string;
            }[];
        };
        paths: {
            $ref: string;
        };
        definitions: {
            $ref: string;
        };
        parameters: {
            $ref: string;
        };
        responses: {
            $ref: string;
        };
        security: {
            $ref: string;
        };
        securityDefinitions: {
            $ref: string;
        };
        tags: {
            type: string;
            items: {
                $ref: string;
            };
            uniqueItems: boolean;
        };
        externalDocs: {
            $ref: string;
        };
    };
    definitions: {
        info: {
            type: string;
            description: string;
            required: string[];
            additionalProperties: boolean;
            patternProperties: {
                "^x-": {
                    $ref: string;
                };
            };
            properties: {
                title: {
                    type: string;
                    description: string;
                };
                version: {
                    type: string;
                    description: string;
                };
                description: {
                    type: string;
                    description: string;
                };
                termsOfService: {
                    type: string;
                    description: string;
                };
                contact: {
                    $ref: string;
                };
                license: {
                    $ref: string;
                };
            };
        };
        contact: {
            type: string;
            description: string;
            additionalProperties: boolean;
            properties: {
                name: {
                    type: string;
                    description: string;
                };
                url: {
                    type: string;
                    description: string;
                    format: string;
                };
                email: {
                    type: string;
                    description: string;
                    format: string;
                };
            };
            patternProperties: {
                "^x-": {
                    $ref: string;
                };
            };
        };
        license: {
            type: string;
            required: string[];
            additionalProperties: boolean;
            properties: {
                name: {
                    type: string;
                    description: string;
                };
                url: {
                    type: string;
                    description: string;
                    format: string;
                };
            };
            patternProperties: {
                "^x-": {
                    $ref: string;
                };
            };
        };
        paths: {
            type: string;
            description: string;
            patternProperties: {
                "^x-": {
                    $ref: string;
                };
                "^/": {
                    oneOf: {
                        $ref: string;
                    }[];
                };
            };
            additionalProperties: boolean;
        };
        definitions: {
            type: string;
            additionalProperties: {
                $ref: string;
            };
            description: string;
        };
        parameterDefinitions: {
            type: string;
            additionalProperties: {
                $ref: string;
            };
            description: string;
        };
        responseDefinitions: {
            type: string;
            additionalProperties: {
                $ref: string;
            };
            description: string;
        };
        externalDocs: {
            type: string;
            additionalProperties: boolean;
            description: string;
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
                "^x-": {
                    $ref: string;
                };
            };
        };
        examples: {
            type: string;
            additionalProperties: boolean;
        };
        mimeType: {
            type: string;
            description: string;
        };
        operation: {
            type: string;
            required: string[];
            additionalProperties: boolean;
            patternProperties: {
                "^x-": {
                    $ref: string;
                };
            };
            properties: {
                tags: {
                    type: string;
                    items: {
                        type: string;
                    };
                    uniqueItems: boolean;
                };
                summary: {
                    type: string;
                    description: string;
                };
                description: {
                    type: string;
                    description: string;
                };
                externalDocs: {
                    $ref: string;
                };
                operationId: {
                    type: string;
                    description: string;
                };
                produces: {
                    description: string;
                    allOf: {
                        $ref: string;
                    }[];
                };
                consumes: {
                    description: string;
                    allOf: {
                        $ref: string;
                    }[];
                };
                parameters: {
                    $ref: string;
                };
                responses: {
                    $ref: string;
                };
                schemes: {
                    $ref: string;
                };
                deprecated: {
                    type: string;
                    default: boolean;
                };
                security: {
                    $ref: string;
                };
            };
        };
        pathItem: {
            type: string;
            additionalProperties: boolean;
            patternProperties: {
                "^x-": {
                    $ref: string;
                };
            };
            properties: {
                $ref: {
                    type: string;
                };
                get: {
                    $ref: string;
                };
                put: {
                    $ref: string;
                };
                post: {
                    $ref: string;
                };
                delete: {
                    $ref: string;
                };
                options: {
                    $ref: string;
                };
                head: {
                    $ref: string;
                };
                patch: {
                    $ref: string;
                };
                parameters: {
                    $ref: string;
                };
            };
        };
        responses: {
            type: string;
            description: string;
            minProperties: number;
            additionalProperties: boolean;
            patternProperties: {
                "^([0-9]{3})$|^(default)$": {
                    $ref: string;
                };
                "^x-": {
                    $ref: string;
                };
            };
            not: {
                type: string;
                additionalProperties: boolean;
                patternProperties: {
                    "^x-": {
                        $ref: string;
                    };
                };
            };
        };
        responseValue: {
            oneOf: {
                $ref: string;
            }[];
        };
        response: {
            type: string;
            required: string[];
            properties: {
                description: {
                    type: string;
                };
                schema: {
                    oneOf: {
                        $ref: string;
                    }[];
                };
                headers: {
                    $ref: string;
                };
                examples: {
                    $ref: string;
                };
            };
            additionalProperties: boolean;
            patternProperties: {
                "^x-": {
                    $ref: string;
                };
            };
        };
        headers: {
            type: string;
            additionalProperties: {
                $ref: string;
            };
        };
        header: {
            type: string;
            additionalProperties: boolean;
            required: string[];
            properties: {
                type: {
                    type: string;
                    enum: string[];
                };
                format: {
                    type: string;
                };
                items: {
                    $ref: string;
                };
                collectionFormat: {
                    $ref: string;
                };
                default: {
                    $ref: string;
                };
                maximum: {
                    $ref: string;
                };
                exclusiveMaximum: {
                    $ref: string;
                };
                minimum: {
                    $ref: string;
                };
                exclusiveMinimum: {
                    $ref: string;
                };
                maxLength: {
                    $ref: string;
                };
                minLength: {
                    $ref: string;
                };
                pattern: {
                    $ref: string;
                };
                maxItems: {
                    $ref: string;
                };
                minItems: {
                    $ref: string;
                };
                uniqueItems: {
                    $ref: string;
                };
                enum: {
                    $ref: string;
                };
                multipleOf: {
                    $ref: string;
                };
                description: {
                    type: string;
                };
            };
            patternProperties: {
                "^x-": {
                    $ref: string;
                };
            };
        };
        vendorExtension: {
            description: string;
            additionalProperties: boolean;
            additionalItems: boolean;
        };
        bodyParameter: {
            type: string;
            required: string[];
            patternProperties: {
                "^x-": {
                    $ref: string;
                };
            };
            properties: {
                description: {
                    type: string;
                    description: string;
                };
                name: {
                    type: string;
                    description: string;
                };
                in: {
                    type: string;
                    description: string;
                    enum: string[];
                };
                required: {
                    type: string;
                    description: string;
                    default: boolean;
                };
                schema: {
                    $ref: string;
                };
            };
            additionalProperties: boolean;
        };
        headerParameterSubSchema: {
            additionalProperties: boolean;
            patternProperties: {
                "^x-": {
                    $ref: string;
                };
            };
            properties: {
                required: {
                    type: string;
                    description: string;
                    default: boolean;
                };
                in: {
                    type: string;
                    description: string;
                    enum: string[];
                };
                description: {
                    type: string;
                    description: string;
                };
                name: {
                    type: string;
                    description: string;
                };
                type: {
                    type: string;
                    enum: string[];
                };
                format: {
                    type: string;
                };
                items: {
                    $ref: string;
                };
                collectionFormat: {
                    $ref: string;
                };
                default: {
                    $ref: string;
                };
                maximum: {
                    $ref: string;
                };
                exclusiveMaximum: {
                    $ref: string;
                };
                minimum: {
                    $ref: string;
                };
                exclusiveMinimum: {
                    $ref: string;
                };
                maxLength: {
                    $ref: string;
                };
                minLength: {
                    $ref: string;
                };
                pattern: {
                    $ref: string;
                };
                maxItems: {
                    $ref: string;
                };
                minItems: {
                    $ref: string;
                };
                uniqueItems: {
                    $ref: string;
                };
                enum: {
                    $ref: string;
                };
                multipleOf: {
                    $ref: string;
                };
            };
        };
        queryParameterSubSchema: {
            additionalProperties: boolean;
            patternProperties: {
                "^x-": {
                    $ref: string;
                };
            };
            properties: {
                required: {
                    type: string;
                    description: string;
                    default: boolean;
                };
                in: {
                    type: string;
                    description: string;
                    enum: string[];
                };
                description: {
                    type: string;
                    description: string;
                };
                name: {
                    type: string;
                    description: string;
                };
                allowEmptyValue: {
                    type: string;
                    default: boolean;
                    description: string;
                };
                type: {
                    type: string;
                    enum: string[];
                };
                format: {
                    type: string;
                };
                items: {
                    $ref: string;
                };
                collectionFormat: {
                    $ref: string;
                };
                default: {
                    $ref: string;
                };
                maximum: {
                    $ref: string;
                };
                exclusiveMaximum: {
                    $ref: string;
                };
                minimum: {
                    $ref: string;
                };
                exclusiveMinimum: {
                    $ref: string;
                };
                maxLength: {
                    $ref: string;
                };
                minLength: {
                    $ref: string;
                };
                pattern: {
                    $ref: string;
                };
                maxItems: {
                    $ref: string;
                };
                minItems: {
                    $ref: string;
                };
                uniqueItems: {
                    $ref: string;
                };
                enum: {
                    $ref: string;
                };
                multipleOf: {
                    $ref: string;
                };
            };
        };
        formDataParameterSubSchema: {
            additionalProperties: boolean;
            patternProperties: {
                "^x-": {
                    $ref: string;
                };
            };
            properties: {
                required: {
                    type: string;
                    description: string;
                    default: boolean;
                };
                in: {
                    type: string;
                    description: string;
                    enum: string[];
                };
                description: {
                    type: string;
                    description: string;
                };
                name: {
                    type: string;
                    description: string;
                };
                allowEmptyValue: {
                    type: string;
                    default: boolean;
                    description: string;
                };
                type: {
                    type: string;
                    enum: string[];
                };
                format: {
                    type: string;
                };
                items: {
                    $ref: string;
                };
                collectionFormat: {
                    $ref: string;
                };
                default: {
                    $ref: string;
                };
                maximum: {
                    $ref: string;
                };
                exclusiveMaximum: {
                    $ref: string;
                };
                minimum: {
                    $ref: string;
                };
                exclusiveMinimum: {
                    $ref: string;
                };
                maxLength: {
                    $ref: string;
                };
                minLength: {
                    $ref: string;
                };
                pattern: {
                    $ref: string;
                };
                maxItems: {
                    $ref: string;
                };
                minItems: {
                    $ref: string;
                };
                uniqueItems: {
                    $ref: string;
                };
                enum: {
                    $ref: string;
                };
                multipleOf: {
                    $ref: string;
                };
            };
        };
        pathParameterSubSchema: {
            additionalProperties: boolean;
            patternProperties: {
                "^x-": {
                    $ref: string;
                };
            };
            required: string[];
            properties: {
                required: {
                    type: string;
                    enum: boolean[];
                    description: string;
                };
                in: {
                    type: string;
                    description: string;
                    enum: string[];
                };
                description: {
                    type: string;
                    description: string;
                };
                name: {
                    type: string;
                    description: string;
                };
                type: {
                    type: string;
                    enum: string[];
                };
                format: {
                    type: string;
                };
                items: {
                    $ref: string;
                };
                collectionFormat: {
                    $ref: string;
                };
                default: {
                    $ref: string;
                };
                maximum: {
                    $ref: string;
                };
                exclusiveMaximum: {
                    $ref: string;
                };
                minimum: {
                    $ref: string;
                };
                exclusiveMinimum: {
                    $ref: string;
                };
                maxLength: {
                    $ref: string;
                };
                minLength: {
                    $ref: string;
                };
                pattern: {
                    $ref: string;
                };
                maxItems: {
                    $ref: string;
                };
                minItems: {
                    $ref: string;
                };
                uniqueItems: {
                    $ref: string;
                };
                enum: {
                    $ref: string;
                };
                multipleOf: {
                    $ref: string;
                };
            };
        };
        nonBodyParameter: {
            type: string;
            required: string[];
            oneOf: {
                $ref: string;
            }[];
        };
        parameter: {
            oneOf: {
                $ref: string;
            }[];
        };
        schema: {
            type: string;
            description: string;
            patternProperties: {
                "^x-": {
                    $ref: string;
                };
            };
            properties: {
                $ref: {
                    type: string;
                };
                format: {
                    type: string;
                };
                title: {
                    $ref: string;
                };
                description: {
                    $ref: string;
                };
                default: {
                    $ref: string;
                };
                multipleOf: {
                    $ref: string;
                };
                maximum: {
                    $ref: string;
                };
                exclusiveMaximum: {
                    $ref: string;
                };
                minimum: {
                    $ref: string;
                };
                exclusiveMinimum: {
                    $ref: string;
                };
                maxLength: {
                    $ref: string;
                };
                minLength: {
                    $ref: string;
                };
                pattern: {
                    $ref: string;
                };
                maxItems: {
                    $ref: string;
                };
                minItems: {
                    $ref: string;
                };
                uniqueItems: {
                    $ref: string;
                };
                maxProperties: {
                    $ref: string;
                };
                minProperties: {
                    $ref: string;
                };
                required: {
                    $ref: string;
                };
                enum: {
                    $ref: string;
                };
                additionalProperties: {
                    anyOf: ({
                        $ref: string;
                        type?: undefined;
                    } | {
                        type: string;
                        $ref?: undefined;
                    })[];
                    default: {};
                };
                type: {
                    $ref: string;
                };
                items: {
                    anyOf: ({
                        $ref: string;
                        type?: undefined;
                        minItems?: undefined;
                        items?: undefined;
                    } | {
                        type: string;
                        minItems: number;
                        items: {
                            $ref: string;
                        };
                        $ref?: undefined;
                    })[];
                    default: {};
                };
                allOf: {
                    type: string;
                    minItems: number;
                    items: {
                        $ref: string;
                    };
                };
                properties: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                    default: {};
                };
                discriminator: {
                    type: string;
                };
                readOnly: {
                    type: string;
                    default: boolean;
                };
                xml: {
                    $ref: string;
                };
                externalDocs: {
                    $ref: string;
                };
                example: {};
            };
            additionalProperties: boolean;
        };
        fileSchema: {
            type: string;
            description: string;
            patternProperties: {
                "^x-": {
                    $ref: string;
                };
            };
            required: string[];
            properties: {
                format: {
                    type: string;
                };
                title: {
                    $ref: string;
                };
                description: {
                    $ref: string;
                };
                default: {
                    $ref: string;
                };
                required: {
                    $ref: string;
                };
                type: {
                    type: string;
                    enum: string[];
                };
                readOnly: {
                    type: string;
                    default: boolean;
                };
                externalDocs: {
                    $ref: string;
                };
                example: {};
            };
            additionalProperties: boolean;
        };
        primitivesItems: {
            type: string;
            additionalProperties: boolean;
            properties: {
                type: {
                    type: string;
                    enum: string[];
                };
                format: {
                    type: string;
                };
                items: {
                    $ref: string;
                };
                collectionFormat: {
                    $ref: string;
                };
                default: {
                    $ref: string;
                };
                maximum: {
                    $ref: string;
                };
                exclusiveMaximum: {
                    $ref: string;
                };
                minimum: {
                    $ref: string;
                };
                exclusiveMinimum: {
                    $ref: string;
                };
                maxLength: {
                    $ref: string;
                };
                minLength: {
                    $ref: string;
                };
                pattern: {
                    $ref: string;
                };
                maxItems: {
                    $ref: string;
                };
                minItems: {
                    $ref: string;
                };
                uniqueItems: {
                    $ref: string;
                };
                enum: {
                    $ref: string;
                };
                multipleOf: {
                    $ref: string;
                };
            };
            patternProperties: {
                "^x-": {
                    $ref: string;
                };
            };
        };
        security: {
            type: string;
            items: {
                $ref: string;
            };
            uniqueItems: boolean;
        };
        securityRequirement: {
            type: string;
            additionalProperties: {
                type: string;
                items: {
                    type: string;
                };
                uniqueItems: boolean;
            };
        };
        xml: {
            type: string;
            additionalProperties: boolean;
            properties: {
                name: {
                    type: string;
                };
                namespace: {
                    type: string;
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
                "^x-": {
                    $ref: string;
                };
            };
        };
        tag: {
            type: string;
            additionalProperties: boolean;
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
                "^x-": {
                    $ref: string;
                };
            };
        };
        securityDefinitions: {
            type: string;
            additionalProperties: {
                oneOf: {
                    $ref: string;
                }[];
            };
        };
        basicAuthenticationSecurity: {
            type: string;
            additionalProperties: boolean;
            required: string[];
            properties: {
                type: {
                    type: string;
                    enum: string[];
                };
                description: {
                    type: string;
                };
            };
            patternProperties: {
                "^x-": {
                    $ref: string;
                };
            };
        };
        apiKeySecurity: {
            type: string;
            additionalProperties: boolean;
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
                "^x-": {
                    $ref: string;
                };
            };
        };
        oauth2ImplicitSecurity: {
            type: string;
            additionalProperties: boolean;
            required: string[];
            properties: {
                type: {
                    type: string;
                    enum: string[];
                };
                flow: {
                    type: string;
                    enum: string[];
                };
                scopes: {
                    $ref: string;
                };
                authorizationUrl: {
                    type: string;
                    format: string;
                };
                description: {
                    type: string;
                };
            };
            patternProperties: {
                "^x-": {
                    $ref: string;
                };
            };
        };
        oauth2PasswordSecurity: {
            type: string;
            additionalProperties: boolean;
            required: string[];
            properties: {
                type: {
                    type: string;
                    enum: string[];
                };
                flow: {
                    type: string;
                    enum: string[];
                };
                scopes: {
                    $ref: string;
                };
                tokenUrl: {
                    type: string;
                    format: string;
                };
                description: {
                    type: string;
                };
            };
            patternProperties: {
                "^x-": {
                    $ref: string;
                };
            };
        };
        oauth2ApplicationSecurity: {
            type: string;
            additionalProperties: boolean;
            required: string[];
            properties: {
                type: {
                    type: string;
                    enum: string[];
                };
                flow: {
                    type: string;
                    enum: string[];
                };
                scopes: {
                    $ref: string;
                };
                tokenUrl: {
                    type: string;
                    format: string;
                };
                description: {
                    type: string;
                };
            };
            patternProperties: {
                "^x-": {
                    $ref: string;
                };
            };
        };
        oauth2AccessCodeSecurity: {
            type: string;
            additionalProperties: boolean;
            required: string[];
            properties: {
                type: {
                    type: string;
                    enum: string[];
                };
                flow: {
                    type: string;
                    enum: string[];
                };
                scopes: {
                    $ref: string;
                };
                authorizationUrl: {
                    type: string;
                    format: string;
                };
                tokenUrl: {
                    type: string;
                    format: string;
                };
                description: {
                    type: string;
                };
            };
            patternProperties: {
                "^x-": {
                    $ref: string;
                };
            };
        };
        oauth2Scopes: {
            type: string;
            additionalProperties: {
                type: string;
            };
        };
        mediaTypeList: {
            type: string;
            items: {
                $ref: string;
            };
            uniqueItems: boolean;
        };
        parametersList: {
            type: string;
            description: string;
            additionalItems: boolean;
            items: {
                oneOf: {
                    $ref: string;
                }[];
            };
            uniqueItems: boolean;
        };
        schemesList: {
            type: string;
            description: string;
            items: {
                type: string;
                enum: string[];
            };
            uniqueItems: boolean;
        };
        collectionFormat: {
            type: string;
            enum: string[];
            default: string;
        };
        collectionFormatWithMulti: {
            type: string;
            enum: string[];
            default: string;
        };
        title: {
            $ref: string;
        };
        description: {
            $ref: string;
        };
        default: {
            $ref: string;
        };
        multipleOf: {
            $ref: string;
        };
        maximum: {
            $ref: string;
        };
        exclusiveMaximum: {
            $ref: string;
        };
        minimum: {
            $ref: string;
        };
        exclusiveMinimum: {
            $ref: string;
        };
        maxLength: {
            $ref: string;
        };
        minLength: {
            $ref: string;
        };
        pattern: {
            $ref: string;
        };
        maxItems: {
            $ref: string;
        };
        minItems: {
            $ref: string;
        };
        uniqueItems: {
            $ref: string;
        };
        enum: {
            $ref: string;
        };
        jsonReference: {
            type: string;
            required: string[];
            additionalProperties: boolean;
            properties: {
                $ref: {
                    type: string;
                };
            };
        };
    };
};
export default _default;
//# sourceMappingURL=schema.d.ts.map