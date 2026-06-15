import { z } from 'zod';
/**
 * An OpenAPI extension to set any additional body parameters for the OAuth token request
 *
 * @example
 * ```yaml
 * x-scalar-security-body: {
 *   audience: 'https://api.example.com',
 *   resource: 'user-profile'
 * }
 * ```
 */
export declare const XScalarSecurityBody: z.ZodObject<{
    'x-scalar-security-body': z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodString>>;
}, z.core.$strip>;
//# sourceMappingURL=x-scalar-security-body.d.ts.map