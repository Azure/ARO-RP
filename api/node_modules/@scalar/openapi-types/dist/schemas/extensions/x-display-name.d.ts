import { z } from 'zod';
/**
 * An OpenAPI extension to overwrite tag names with a display-friendly version
 *
 * @example
 * ```yaml
 * x-displayName: planets
 * ```
 */
export declare const XDisplayNameSchema: z.ZodObject<{
    'x-displayName': z.ZodCatch<z.ZodOptional<z.ZodString>>;
}, z.core.$strip>;
//# sourceMappingURL=x-display-name.d.ts.map