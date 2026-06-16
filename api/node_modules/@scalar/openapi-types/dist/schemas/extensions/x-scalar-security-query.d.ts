import { z } from 'zod';
/**
 * An OpenAPI extension set any query parameters for the OAuth authorize request
 *
 * @example
 * ```yaml
 * x-scalar-security-query: {
 *   prompt: 'consent',
 *   audience: 'scalar'
 * }
 * ```
 */
export declare const XScalarSecurityQuery: z.ZodObject<{
    'x-scalar-security-query': z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodString>>;
}, z.core.$strip>;
//# sourceMappingURL=x-scalar-security-query.d.ts.map