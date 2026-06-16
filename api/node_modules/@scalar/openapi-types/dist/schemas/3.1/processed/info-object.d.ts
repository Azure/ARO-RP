import { z } from 'zod';
/**
 * Info Object
 *
 * The object provides metadata about the API. The metadata MAY be used by the clients if needed,
 * and MAY be presented in editing or documentation generation tools for convenience.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#info-object
 */
export declare const InfoObjectSchema: z.ZodObject<{
    title: z.ZodCatch<z.ZodString>;
    summary: z.ZodCatch<z.ZodOptional<z.ZodString>>;
    description: z.ZodCatch<z.ZodOptional<z.ZodString>>;
    termsOfService: z.ZodCatch<z.ZodOptional<z.ZodString>>;
    contact: z.ZodCatch<z.ZodOptional<z.ZodObject<{
        name: z.ZodOptional<z.ZodString>;
        url: z.ZodCatch<z.ZodOptional<z.ZodString>>;
        email: z.ZodCatch<z.ZodOptional<z.ZodString>>;
    }, z.core.$strip>>>;
    license: z.ZodCatch<z.ZodOptional<z.ZodObject<{
        name: z.ZodCatch<z.ZodNullable<z.ZodOptional<z.ZodString>>>;
        identifier: z.ZodCatch<z.ZodOptional<z.ZodString>>;
        url: z.ZodCatch<z.ZodOptional<z.ZodString>>;
    }, z.core.$strip>>>;
    version: z.ZodCatch<z.ZodString>;
}, z.core.$strip>;
//# sourceMappingURL=info-object.d.ts.map