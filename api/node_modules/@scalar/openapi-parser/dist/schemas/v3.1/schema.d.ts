declare const _default: {
    $id: string;
    $schema: string;
    description: string;
    type: string;
    properties: {
        openapi: {
            type: string;
            pattern: string;
        };
        info: {
            $ref: string;
        };
        jsonSchemaDialect: {
            type: string;
            format: string;
            default: string;
        };
        servers: {
            type: string;
            items: {
                $ref: string;
            };
            default: {
                url: string;
            }[];
        };
        paths: {
            $ref: string;
        };
        webhooks: {
            type: string;
            additionalProperties: {
                $ref: string;
            };
        };
        components: {
            $ref: string;
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
        };
        externalDocs: {
            $ref: string;
        };
    };
    required: string[];
    anyOf: {
        required: string[];
    }[];
    $ref: string;
    unevaluatedProperties: boolean;
    $defs: {
        info: {
            $comment: string;
            type: string;
            properties: {
                title: {
                    type: string;
                };
                summary: {
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
            required: string[];
            $ref: string;
            unevaluatedProperties: boolean;
        };
        contact: {
            $comment: string;
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
            $ref: string;
            unevaluatedProperties: boolean;
        };
        license: {
            $comment: string;
            type: string;
            properties: {
                name: {
                    type: string;
                };
                identifier: {
                    type: string;
                };
                url: {
                    type: string;
                    format: string;
                };
            };
            required: string[];
            dependentSchemas: {
                identifier: {
                    not: {
                        required: string[];
                    };
                };
            };
            $ref: string;
            unevaluatedProperties: boolean;
        };
        server: {
            $comment: string;
            type: string;
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
            required: string[];
            $ref: string;
            unevaluatedProperties: boolean;
        };
        "server-variable": {
            $comment: string;
            type: string;
            properties: {
                enum: {
                    type: string;
                    items: {
                        type: string;
                    };
                    minItems: number;
                };
                default: {
                    type: string;
                };
                description: {
                    type: string;
                };
            };
            required: string[];
            $ref: string;
            unevaluatedProperties: boolean;
        };
        components: {
            $comment: string;
            type: string;
            properties: {
                schemas: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                };
                responses: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                };
                parameters: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                };
                examples: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                };
                requestBodies: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                };
                headers: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                };
                securitySchemes: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                };
                links: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                };
                callbacks: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                };
                pathItems: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                };
            };
            patternProperties: {
                "^(schemas|responses|parameters|examples|requestBodies|headers|securitySchemes|links|callbacks|pathItems)$": {
                    $comment: string;
                    propertyNames: {
                        pattern: string;
                    };
                };
            };
            $ref: string;
            unevaluatedProperties: boolean;
        };
        paths: {
            $comment: string;
            type: string;
            patternProperties: {
                "^/": {
                    $ref: string;
                };
            };
            $ref: string;
            unevaluatedProperties: boolean;
        };
        "path-item": {
            $comment: string;
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
                        $ref: string;
                    };
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
                trace: {
                    $ref: string;
                };
            };
            $ref: string;
            unevaluatedProperties: boolean;
        };
        "path-item-or-reference": {
            if: {
                type: string;
                required: string[];
            };
            then: {
                $ref: string;
            };
            else: {
                $ref: string;
            };
        };
        operation: {
            $comment: string;
            type: string;
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
                        $ref: string;
                    };
                };
                requestBody: {
                    $ref: string;
                };
                responses: {
                    $ref: string;
                };
                callbacks: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                };
                deprecated: {
                    default: boolean;
                    type: string;
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
            $ref: string;
            unevaluatedProperties: boolean;
        };
        "external-documentation": {
            $comment: string;
            type: string;
            properties: {
                description: {
                    type: string;
                };
                url: {
                    type: string;
                    format: string;
                };
            };
            required: string[];
            $ref: string;
            unevaluatedProperties: boolean;
        };
        parameter: {
            $comment: string;
            type: string;
            properties: {
                name: {
                    type: string;
                };
                in: {
                    enum: string[];
                };
                description: {
                    type: string;
                };
                required: {
                    default: boolean;
                    type: string;
                };
                deprecated: {
                    default: boolean;
                    type: string;
                };
                schema: {
                    $ref: string;
                };
                content: {
                    $ref: string;
                    minProperties: number;
                    maxProperties: number;
                };
            };
            required: string[];
            oneOf: {
                required: string[];
            }[];
            if: {
                properties: {
                    in: {
                        const: string;
                    };
                };
                required: string[];
            };
            then: {
                properties: {
                    allowEmptyValue: {
                        default: boolean;
                        type: string;
                    };
                };
            };
            dependentSchemas: {
                schema: {
                    properties: {
                        style: {
                            type: string;
                        };
                        explode: {
                            type: string;
                        };
                    };
                    allOf: {
                        $ref: string;
                    }[];
                    $defs: {
                        "styles-for-path": {
                            if: {
                                properties: {
                                    in: {
                                        const: string;
                                    };
                                };
                                required: string[];
                            };
                            then: {
                                properties: {
                                    name: {
                                        pattern: string;
                                    };
                                    style: {
                                        default: string;
                                        enum: string[];
                                    };
                                    required: {
                                        const: boolean;
                                    };
                                };
                                required: string[];
                            };
                        };
                        "styles-for-header": {
                            if: {
                                properties: {
                                    in: {
                                        const: string;
                                    };
                                };
                                required: string[];
                            };
                            then: {
                                properties: {
                                    style: {
                                        default: string;
                                        const: string;
                                    };
                                };
                            };
                        };
                        "styles-for-query": {
                            if: {
                                properties: {
                                    in: {
                                        const: string;
                                    };
                                };
                                required: string[];
                            };
                            then: {
                                properties: {
                                    style: {
                                        default: string;
                                        enum: string[];
                                    };
                                    allowReserved: {
                                        default: boolean;
                                        type: string;
                                    };
                                };
                            };
                        };
                        "styles-for-cookie": {
                            if: {
                                properties: {
                                    in: {
                                        const: string;
                                    };
                                };
                                required: string[];
                            };
                            then: {
                                properties: {
                                    style: {
                                        default: string;
                                        const: string;
                                    };
                                };
                            };
                        };
                        "styles-for-form": {
                            if: {
                                properties: {
                                    style: {
                                        const: string;
                                    };
                                };
                                required: string[];
                            };
                            then: {
                                properties: {
                                    explode: {
                                        default: boolean;
                                    };
                                };
                            };
                            else: {
                                properties: {
                                    explode: {
                                        default: boolean;
                                    };
                                };
                            };
                        };
                    };
                };
            };
            $ref: string;
            unevaluatedProperties: boolean;
        };
        "parameter-or-reference": {
            if: {
                type: string;
                required: string[];
            };
            then: {
                $ref: string;
            };
            else: {
                $ref: string;
            };
        };
        "request-body": {
            $comment: string;
            type: string;
            properties: {
                description: {
                    type: string;
                };
                content: {
                    $ref: string;
                };
                required: {
                    default: boolean;
                    type: string;
                };
            };
            required: string[];
            $ref: string;
            unevaluatedProperties: boolean;
        };
        "request-body-or-reference": {
            if: {
                type: string;
                required: string[];
            };
            then: {
                $ref: string;
            };
            else: {
                $ref: string;
            };
        };
        content: {
            $comment: string;
            type: string;
            additionalProperties: {
                $ref: string;
            };
            propertyNames: {
                format: string;
            };
        };
        "media-type": {
            $comment: string;
            type: string;
            properties: {
                schema: {
                    $ref: string;
                };
                encoding: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                };
            };
            allOf: {
                $ref: string;
            }[];
            unevaluatedProperties: boolean;
        };
        encoding: {
            $comment: string;
            type: string;
            properties: {
                contentType: {
                    type: string;
                    format: string;
                };
                headers: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                };
                style: {
                    default: string;
                    enum: string[];
                };
                explode: {
                    type: string;
                };
                allowReserved: {
                    default: boolean;
                    type: string;
                };
            };
            allOf: {
                $ref: string;
            }[];
            unevaluatedProperties: boolean;
            $defs: {
                "explode-default": {
                    if: {
                        properties: {
                            style: {
                                const: string;
                            };
                        };
                        required: string[];
                    };
                    then: {
                        properties: {
                            explode: {
                                default: boolean;
                            };
                        };
                    };
                    else: {
                        properties: {
                            explode: {
                                default: boolean;
                            };
                        };
                    };
                };
            };
        };
        responses: {
            $comment: string;
            type: string;
            properties: {
                default: {
                    $ref: string;
                };
            };
            patternProperties: {
                "^[1-5](?:[0-9]{2}|XX)$": {
                    $ref: string;
                };
            };
            minProperties: number;
            $ref: string;
            unevaluatedProperties: boolean;
        };
        response: {
            $comment: string;
            type: string;
            properties: {
                description: {
                    type: string;
                };
                headers: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                };
                content: {
                    $ref: string;
                };
                links: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                };
            };
            required: string[];
            $ref: string;
            unevaluatedProperties: boolean;
        };
        "response-or-reference": {
            if: {
                type: string;
                required: string[];
            };
            then: {
                $ref: string;
            };
            else: {
                $ref: string;
            };
        };
        callbacks: {
            $comment: string;
            type: string;
            $ref: string;
            additionalProperties: {
                $ref: string;
            };
        };
        "callbacks-or-reference": {
            if: {
                type: string;
                required: string[];
            };
            then: {
                $ref: string;
            };
            else: {
                $ref: string;
            };
        };
        example: {
            $comment: string;
            type: string;
            properties: {
                summary: {
                    type: string;
                };
                description: {
                    type: string;
                };
                value: boolean;
                externalValue: {
                    type: string;
                    format: string;
                };
            };
            not: {
                required: string[];
            };
            $ref: string;
            unevaluatedProperties: boolean;
        };
        "example-or-reference": {
            if: {
                type: string;
                required: string[];
            };
            then: {
                $ref: string;
            };
            else: {
                $ref: string;
            };
        };
        link: {
            $comment: string;
            type: string;
            properties: {
                operationRef: {
                    type: string;
                    format: string;
                };
                operationId: {
                    type: string;
                };
                parameters: {
                    $ref: string;
                };
                requestBody: boolean;
                description: {
                    type: string;
                };
                body: {
                    $ref: string;
                };
            };
            oneOf: {
                required: string[];
            }[];
            $ref: string;
            unevaluatedProperties: boolean;
        };
        "link-or-reference": {
            if: {
                type: string;
                required: string[];
            };
            then: {
                $ref: string;
            };
            else: {
                $ref: string;
            };
        };
        header: {
            $comment: string;
            type: string;
            properties: {
                description: {
                    type: string;
                };
                required: {
                    default: boolean;
                    type: string;
                };
                deprecated: {
                    default: boolean;
                    type: string;
                };
                schema: {
                    $ref: string;
                };
                content: {
                    $ref: string;
                    minProperties: number;
                    maxProperties: number;
                };
            };
            oneOf: {
                required: string[];
            }[];
            dependentSchemas: {
                schema: {
                    properties: {
                        style: {
                            default: string;
                            const: string;
                        };
                        explode: {
                            default: boolean;
                            type: string;
                        };
                    };
                    $ref: string;
                };
            };
            $ref: string;
            unevaluatedProperties: boolean;
        };
        "header-or-reference": {
            if: {
                type: string;
                required: string[];
            };
            then: {
                $ref: string;
            };
            else: {
                $ref: string;
            };
        };
        tag: {
            $comment: string;
            type: string;
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
            required: string[];
            $ref: string;
            unevaluatedProperties: boolean;
        };
        reference: {
            $comment: string;
            type: string;
            properties: {
                $ref: {
                    type: string;
                    format: string;
                };
                summary: {
                    type: string;
                };
                description: {
                    type: string;
                };
            };
            unevaluatedProperties: boolean;
        };
        schema: {
            $comment: string;
            $dynamicAnchor: string;
            type: string[];
        };
        "security-scheme": {
            $comment: string;
            type: string;
            properties: {
                type: {
                    enum: string[];
                };
                description: {
                    type: string;
                };
            };
            required: string[];
            allOf: {
                $ref: string;
            }[];
            unevaluatedProperties: boolean;
            $defs: {
                "type-apikey": {
                    if: {
                        properties: {
                            type: {
                                const: string;
                            };
                        };
                        required: string[];
                    };
                    then: {
                        properties: {
                            name: {
                                type: string;
                            };
                            in: {
                                enum: string[];
                            };
                        };
                        required: string[];
                    };
                };
                "type-http": {
                    if: {
                        properties: {
                            type: {
                                const: string;
                            };
                        };
                        required: string[];
                    };
                    then: {
                        properties: {
                            scheme: {
                                type: string;
                            };
                        };
                        required: string[];
                    };
                };
                "type-http-bearer": {
                    if: {
                        properties: {
                            type: {
                                const: string;
                            };
                            scheme: {
                                type: string;
                                pattern: string;
                            };
                        };
                        required: string[];
                    };
                    then: {
                        properties: {
                            bearerFormat: {
                                type: string;
                            };
                        };
                    };
                };
                "type-oauth2": {
                    if: {
                        properties: {
                            type: {
                                const: string;
                            };
                        };
                        required: string[];
                    };
                    then: {
                        properties: {
                            flows: {
                                $ref: string;
                            };
                        };
                        required: string[];
                    };
                };
                "type-oidc": {
                    if: {
                        properties: {
                            type: {
                                const: string;
                            };
                        };
                        required: string[];
                    };
                    then: {
                        properties: {
                            openIdConnectUrl: {
                                type: string;
                                format: string;
                            };
                        };
                        required: string[];
                    };
                };
            };
        };
        "security-scheme-or-reference": {
            if: {
                type: string;
                required: string[];
            };
            then: {
                $ref: string;
            };
            else: {
                $ref: string;
            };
        };
        "oauth-flows": {
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
            $ref: string;
            unevaluatedProperties: boolean;
            $defs: {
                implicit: {
                    type: string;
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
                            $ref: string;
                        };
                    };
                    required: string[];
                    $ref: string;
                    unevaluatedProperties: boolean;
                };
                password: {
                    type: string;
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
                            $ref: string;
                        };
                    };
                    required: string[];
                    $ref: string;
                    unevaluatedProperties: boolean;
                };
                "client-credentials": {
                    type: string;
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
                            $ref: string;
                        };
                    };
                    required: string[];
                    $ref: string;
                    unevaluatedProperties: boolean;
                };
                "authorization-code": {
                    type: string;
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
                            $ref: string;
                        };
                    };
                    required: string[];
                    $ref: string;
                    unevaluatedProperties: boolean;
                };
            };
        };
        "security-requirement": {
            $comment: string;
            type: string;
            additionalProperties: {
                type: string;
                items: {
                    type: string;
                };
            };
        };
        "specification-extensions": {
            $comment: string;
            patternProperties: {
                "^x-": boolean;
            };
        };
        examples: {
            properties: {
                example: boolean;
                examples: {
                    type: string;
                    additionalProperties: {
                        $ref: string;
                    };
                };
            };
        };
        "map-of-strings": {
            type: string;
            additionalProperties: {
                type: string;
            };
        };
    };
};
export default _default;
//# sourceMappingURL=schema.d.ts.map