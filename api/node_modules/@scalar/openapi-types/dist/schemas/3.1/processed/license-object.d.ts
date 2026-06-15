import { z } from 'zod';
/**
 * License Object
 *
 * License information for the exposed API.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#license-object
 */
export declare const LicenseObjectSchema: z.ZodObject<{
    name: z.ZodCatch<z.ZodNullable<z.ZodOptional<z.ZodString>>>;
    identifier: z.ZodCatch<z.ZodOptional<z.ZodString>>;
    url: z.ZodCatch<z.ZodOptional<z.ZodString>>;
}, z.core.$strip>;
//# sourceMappingURL=license-object.d.ts.map