import { z } from 'zod';
export declare const ApiKeyInValues: readonly ["query", "header", "cookie"];
export declare const ApiKeySchema: z.ZodObject<{
    description: z.ZodOptional<z.ZodString>;
    type: z.ZodLiteral<"apiKey">;
    name: z.ZodDefault<z.ZodOptional<z.ZodString>>;
    in: z.ZodCatch<z.ZodDefault<z.ZodOptional<z.ZodEnum<{
        query: "query";
        cookie: "cookie";
        header: "header";
    }>>>>;
}, z.core.$strip>;
export declare const HttpSchema: z.ZodObject<{
    description: z.ZodOptional<z.ZodString>;
    type: z.ZodLiteral<"http">;
    scheme: z.ZodDefault<z.ZodOptional<z.ZodPipe<z.ZodString, z.ZodEnum<{
        basic: "basic";
        bearer: "bearer";
    }>>>>;
    bearerFormat: z.ZodOptional<z.ZodUnion<readonly [z.ZodLiteral<"JWT">, z.ZodLiteral<"bearer">, z.ZodString]>>;
}, z.core.$strip>;
export declare const OpenIdConnectSchema: z.ZodObject<{
    description: z.ZodOptional<z.ZodString>;
    type: z.ZodLiteral<"openIdConnect">;
    openIdConnectUrl: z.ZodDefault<z.ZodOptional<z.ZodString>>;
}, z.core.$strip>;
/**
 * OAuth Flow Object
 *
 * Configuration details for a supported OAuth Flow
 */
export declare const OAuthFlowObjectSchema: z.ZodObject<{
    refreshUrl: z.ZodOptional<z.ZodString>;
    scopes: z.ZodCatch<z.ZodDefault<z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodOptional<z.ZodString>>>>>;
}, z.core.$strip>;
export declare const ImplicitFlowSchema: z.ZodObject<{
    refreshUrl: z.ZodOptional<z.ZodString>;
    scopes: z.ZodCatch<z.ZodDefault<z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodOptional<z.ZodString>>>>>;
    type: z.ZodOptional<z.ZodLiteral<"implicit">>;
    authorizationUrl: z.ZodDefault<z.ZodString>;
}, z.core.$strip>;
export declare const PasswordFlowSchema: z.ZodObject<{
    refreshUrl: z.ZodOptional<z.ZodString>;
    scopes: z.ZodCatch<z.ZodDefault<z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodOptional<z.ZodString>>>>>;
    type: z.ZodOptional<z.ZodLiteral<"password">>;
    tokenUrl: z.ZodDefault<z.ZodString>;
}, z.core.$strip>;
export declare const ClientCredentialsFlowSchema: z.ZodObject<{
    refreshUrl: z.ZodOptional<z.ZodString>;
    scopes: z.ZodCatch<z.ZodDefault<z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodOptional<z.ZodString>>>>>;
    type: z.ZodOptional<z.ZodLiteral<"clientCredentials">>;
    tokenUrl: z.ZodDefault<z.ZodString>;
}, z.core.$strip>;
export declare const AuthorizationCodeFlowSchema: z.ZodObject<{
    refreshUrl: z.ZodOptional<z.ZodString>;
    scopes: z.ZodCatch<z.ZodDefault<z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodOptional<z.ZodString>>>>>;
    type: z.ZodOptional<z.ZodLiteral<"authorizationCode">>;
    authorizationUrl: z.ZodDefault<z.ZodString>;
    tokenUrl: z.ZodDefault<z.ZodString>;
}, z.core.$strip>;
/**
 * OAuth Flows Object
 *
 * Allows configuration of the supported OAuth Flows.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#oauth-flows-object
 */
export declare const OAuthFlowsObjectSchema: z.ZodObject<{
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
}, z.core.$strip>;
export declare const MutualTlsSchema: z.ZodObject<{
    description: z.ZodOptional<z.ZodString>;
    type: z.ZodLiteral<"mutualTLS">;
}, z.core.$strip>;
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
export declare const SecuritySchemeObjectSchema: z.ZodUnion<readonly [z.ZodObject<{
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
}, z.core.$strip>]>;
//# sourceMappingURL=security-scheme-object.d.ts.map