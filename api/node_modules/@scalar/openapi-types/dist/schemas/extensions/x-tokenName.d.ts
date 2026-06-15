import { z } from 'zod';
/**
 * An OpenAPI extension to specify a custom token name for OAuth2 flows
 *
 * @example
 * ```yaml
 * x-tokenName: 'custom_access_token'
 * ```
 */
export declare const XTokenName: z.ZodObject<{
    'x-tokenName': z.ZodOptional<z.ZodString>;
}, z.core.$strip>;
//# sourceMappingURL=x-tokenName.d.ts.map