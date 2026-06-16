import { z } from 'zod';
/**
 * An OpenAPI extension to specify where OAuth2 credentials should be sent
 *
 * @example
 * ```yaml
 * x-scalar-credentials-location: header
 * ```
 *
 * @example
 * ```yaml
 * x-scalar-credentials-location: body
 * ```
 */
export declare const XScalarCredentialsLocationSchema: z.ZodObject<{
    'x-scalar-credentials-location': z.ZodOptional<z.ZodEnum<{
        header: "header";
        body: "body";
    }>>;
}, z.core.$strip>;
//# sourceMappingURL=x-scalar-credentials-location.d.ts.map