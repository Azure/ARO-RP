import { z } from 'zod';
/**
 * Contact Object
 *
 * Contact information for the exposed API.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#contact-object
 */
export declare const ContactObjectSchema: z.ZodObject<{
    name: z.ZodOptional<z.ZodString>;
    url: z.ZodCatch<z.ZodOptional<z.ZodString>>;
    email: z.ZodCatch<z.ZodOptional<z.ZodString>>;
}, z.core.$strip>;
//# sourceMappingURL=contact-object.d.ts.map