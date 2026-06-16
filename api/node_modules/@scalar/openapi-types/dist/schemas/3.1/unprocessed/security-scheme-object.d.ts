/**
 * Security Scheme Object
 *
 * Defines a security scheme that can be used by the operations.
 *
 * Supported schemes are HTTP authentication, an API key (either as a header, a cookie parameter or as a query
 * parameter), mutual TLS (use of a client certificate), OAuth2's common flows (implicit, password, client credentials
 * and authorization code) as defined in RFC6749, and [[OpenID-Connect-Core]]. Please note that as of 2020, the implicit
 * flow is about to be deprecated by OAuth 2.0 Security Best Current Practice. Recommended for most use cases is
 * Authorization Code Grant flow with PKCE.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#security-scheme-object
 */
export declare const SecuritySchemeObjectSchema: import("zod").ZodUnion<readonly [import("zod").ZodObject<{
    description: import("zod").ZodOptional<import("zod").ZodString>;
    type: import("zod").ZodLiteral<"apiKey">;
    name: import("zod").ZodDefault<import("zod").ZodOptional<import("zod").ZodString>>;
    in: import("zod").ZodCatch<import("zod").ZodDefault<import("zod").ZodOptional<import("zod").ZodEnum<{
        query: "query";
        cookie: "cookie";
        header: "header";
    }>>>>;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
    description: import("zod").ZodOptional<import("zod").ZodString>;
    type: import("zod").ZodLiteral<"http">;
    scheme: import("zod").ZodDefault<import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodString, import("zod").ZodEnum<{
        basic: "basic";
        bearer: "bearer";
    }>>>>;
    bearerFormat: import("zod").ZodOptional<import("zod").ZodUnion<readonly [import("zod").ZodLiteral<"JWT">, import("zod").ZodLiteral<"bearer">, import("zod").ZodString]>>;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
    description: import("zod").ZodOptional<import("zod").ZodString>;
    type: import("zod").ZodLiteral<"mutualTLS">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
    description: import("zod").ZodOptional<import("zod").ZodString>;
    type: import("zod").ZodLiteral<"oauth2">;
    flows: import("zod").ZodObject<{
        implicit: import("zod").ZodOptional<import("zod").ZodOptional<import("zod").ZodObject<{
            refreshUrl: import("zod").ZodOptional<import("zod").ZodString>;
            scopes: import("zod").ZodCatch<import("zod").ZodDefault<import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodOptional<import("zod").ZodString>>>>>;
            type: import("zod").ZodOptional<import("zod").ZodLiteral<"implicit">>;
            authorizationUrl: import("zod").ZodDefault<import("zod").ZodString>;
        }, import("zod/v4/core").$strip>>>;
        password: import("zod").ZodOptional<import("zod").ZodOptional<import("zod").ZodObject<{
            refreshUrl: import("zod").ZodOptional<import("zod").ZodString>;
            scopes: import("zod").ZodCatch<import("zod").ZodDefault<import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodOptional<import("zod").ZodString>>>>>;
            type: import("zod").ZodOptional<import("zod").ZodLiteral<"password">>;
            tokenUrl: import("zod").ZodDefault<import("zod").ZodString>;
        }, import("zod/v4/core").$strip>>>;
        clientCredentials: import("zod").ZodOptional<import("zod").ZodOptional<import("zod").ZodObject<{
            refreshUrl: import("zod").ZodOptional<import("zod").ZodString>;
            scopes: import("zod").ZodCatch<import("zod").ZodDefault<import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodOptional<import("zod").ZodString>>>>>;
            type: import("zod").ZodOptional<import("zod").ZodLiteral<"clientCredentials">>;
            tokenUrl: import("zod").ZodDefault<import("zod").ZodString>;
        }, import("zod/v4/core").$strip>>>;
        authorizationCode: import("zod").ZodOptional<import("zod").ZodOptional<import("zod").ZodObject<{
            refreshUrl: import("zod").ZodOptional<import("zod").ZodString>;
            scopes: import("zod").ZodCatch<import("zod").ZodDefault<import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodOptional<import("zod").ZodString>>>>>;
            type: import("zod").ZodOptional<import("zod").ZodLiteral<"authorizationCode">>;
            authorizationUrl: import("zod").ZodDefault<import("zod").ZodString>;
            tokenUrl: import("zod").ZodDefault<import("zod").ZodString>;
        }, import("zod/v4/core").$strip>>>;
    }, import("zod/v4/core").$strip>;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
    description: import("zod").ZodOptional<import("zod").ZodString>;
    type: import("zod").ZodLiteral<"openIdConnect">;
    openIdConnectUrl: import("zod").ZodDefault<import("zod").ZodOptional<import("zod").ZodString>>;
}, import("zod/v4/core").$strip>]>;
export declare const ApiKeyInValues: readonly ["query", "header", "cookie"];
export declare const ApiKeySchema: import("zod").ZodObject<{
    description: import("zod").ZodOptional<import("zod").ZodString>;
    type: import("zod").ZodLiteral<"apiKey">;
    name: import("zod").ZodDefault<import("zod").ZodOptional<import("zod").ZodString>>;
    in: import("zod").ZodCatch<import("zod").ZodDefault<import("zod").ZodOptional<import("zod").ZodEnum<{
        query: "query";
        cookie: "cookie";
        header: "header";
    }>>>>;
}, import("zod/v4/core").$strip>;
export declare const HttpSchema: import("zod").ZodObject<{
    description: import("zod").ZodOptional<import("zod").ZodString>;
    type: import("zod").ZodLiteral<"http">;
    scheme: import("zod").ZodDefault<import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodString, import("zod").ZodEnum<{
        basic: "basic";
        bearer: "bearer";
    }>>>>;
    bearerFormat: import("zod").ZodOptional<import("zod").ZodUnion<readonly [import("zod").ZodLiteral<"JWT">, import("zod").ZodLiteral<"bearer">, import("zod").ZodString]>>;
}, import("zod/v4/core").$strip>;
export declare const MutualTlsSchema: import("zod").ZodObject<{
    description: import("zod").ZodOptional<import("zod").ZodString>;
    type: import("zod").ZodLiteral<"mutualTLS">;
}, import("zod/v4/core").$strip>;
export declare const OpenIdConnectSchema: import("zod").ZodObject<{
    description: import("zod").ZodOptional<import("zod").ZodString>;
    type: import("zod").ZodLiteral<"openIdConnect">;
    openIdConnectUrl: import("zod").ZodDefault<import("zod").ZodOptional<import("zod").ZodString>>;
}, import("zod/v4/core").$strip>;
export declare const OAuthFlowsObjectSchema: import("zod").ZodObject<{
    description: import("zod").ZodOptional<import("zod").ZodString>;
    type: import("zod").ZodLiteral<"oauth2">;
    flows: import("zod").ZodObject<{
        implicit: import("zod").ZodOptional<import("zod").ZodOptional<import("zod").ZodObject<{
            refreshUrl: import("zod").ZodOptional<import("zod").ZodString>;
            scopes: import("zod").ZodCatch<import("zod").ZodDefault<import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodOptional<import("zod").ZodString>>>>>;
            type: import("zod").ZodOptional<import("zod").ZodLiteral<"implicit">>;
            authorizationUrl: import("zod").ZodDefault<import("zod").ZodString>;
        }, import("zod/v4/core").$strip>>>;
        password: import("zod").ZodOptional<import("zod").ZodOptional<import("zod").ZodObject<{
            refreshUrl: import("zod").ZodOptional<import("zod").ZodString>;
            scopes: import("zod").ZodCatch<import("zod").ZodDefault<import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodOptional<import("zod").ZodString>>>>>;
            type: import("zod").ZodOptional<import("zod").ZodLiteral<"password">>;
            tokenUrl: import("zod").ZodDefault<import("zod").ZodString>;
        }, import("zod/v4/core").$strip>>>;
        clientCredentials: import("zod").ZodOptional<import("zod").ZodOptional<import("zod").ZodObject<{
            refreshUrl: import("zod").ZodOptional<import("zod").ZodString>;
            scopes: import("zod").ZodCatch<import("zod").ZodDefault<import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodOptional<import("zod").ZodString>>>>>;
            type: import("zod").ZodOptional<import("zod").ZodLiteral<"clientCredentials">>;
            tokenUrl: import("zod").ZodDefault<import("zod").ZodString>;
        }, import("zod/v4/core").$strip>>>;
        authorizationCode: import("zod").ZodOptional<import("zod").ZodOptional<import("zod").ZodObject<{
            refreshUrl: import("zod").ZodOptional<import("zod").ZodString>;
            scopes: import("zod").ZodCatch<import("zod").ZodDefault<import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodOptional<import("zod").ZodString>>>>>;
            type: import("zod").ZodOptional<import("zod").ZodLiteral<"authorizationCode">>;
            authorizationUrl: import("zod").ZodDefault<import("zod").ZodString>;
            tokenUrl: import("zod").ZodDefault<import("zod").ZodString>;
        }, import("zod/v4/core").$strip>>>;
    }, import("zod/v4/core").$strip>;
}, import("zod/v4/core").$strip>;
export declare const OAuthFlowObjectSchema: import("zod").ZodObject<{
    refreshUrl: import("zod").ZodOptional<import("zod").ZodString>;
    scopes: import("zod").ZodCatch<import("zod").ZodDefault<import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodOptional<import("zod").ZodString>>>>>;
}, import("zod/v4/core").$strip>;
export declare const AuthorizationCodeFlowSchema: import("zod").ZodObject<{
    refreshUrl: import("zod").ZodOptional<import("zod").ZodString>;
    scopes: import("zod").ZodCatch<import("zod").ZodDefault<import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodOptional<import("zod").ZodString>>>>>;
    type: import("zod").ZodOptional<import("zod").ZodLiteral<"authorizationCode">>;
    authorizationUrl: import("zod").ZodDefault<import("zod").ZodString>;
    tokenUrl: import("zod").ZodDefault<import("zod").ZodString>;
}, import("zod/v4/core").$strip>;
export declare const ClientCredentialsFlowSchema: import("zod").ZodObject<{
    refreshUrl: import("zod").ZodOptional<import("zod").ZodString>;
    scopes: import("zod").ZodCatch<import("zod").ZodDefault<import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodOptional<import("zod").ZodString>>>>>;
    type: import("zod").ZodOptional<import("zod").ZodLiteral<"clientCredentials">>;
    tokenUrl: import("zod").ZodDefault<import("zod").ZodString>;
}, import("zod/v4/core").$strip>;
export declare const ImplicitFlowSchema: import("zod").ZodObject<{
    refreshUrl: import("zod").ZodOptional<import("zod").ZodString>;
    scopes: import("zod").ZodCatch<import("zod").ZodDefault<import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodOptional<import("zod").ZodString>>>>>;
    type: import("zod").ZodOptional<import("zod").ZodLiteral<"implicit">>;
    authorizationUrl: import("zod").ZodDefault<import("zod").ZodString>;
}, import("zod/v4/core").$strip>;
export declare const PasswordFlowSchema: import("zod").ZodObject<{
    refreshUrl: import("zod").ZodOptional<import("zod").ZodString>;
    scopes: import("zod").ZodCatch<import("zod").ZodDefault<import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodOptional<import("zod").ZodString>>>>>;
    type: import("zod").ZodOptional<import("zod").ZodLiteral<"password">>;
    tokenUrl: import("zod").ZodDefault<import("zod").ZodString>;
}, import("zod/v4/core").$strip>;
//# sourceMappingURL=security-scheme-object.d.ts.map