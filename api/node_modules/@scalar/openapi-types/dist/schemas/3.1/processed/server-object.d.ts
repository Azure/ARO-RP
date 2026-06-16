import { z } from 'zod';
/**
 * Server Object
 *
 * An object representing a Server.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#server-object
 */
export declare const ServerObjectSchema: z.ZodObject<{
    url: z.ZodString;
    description: z.ZodOptional<z.ZodString>;
    variables: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
        enum: z.ZodOptional<z.ZodArray<z.ZodString>>;
        default: z.ZodOptional<z.ZodString>;
        description: z.ZodOptional<z.ZodString>;
    }, z.core.$strip>>>;
}, z.core.$strip>;
//# sourceMappingURL=server-object.d.ts.map