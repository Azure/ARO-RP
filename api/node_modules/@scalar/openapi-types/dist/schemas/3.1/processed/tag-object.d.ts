import { z } from 'zod';
/**
 * Tag Object
 *
 * Adds metadata to a single tag that is used by the Operation Object. It is not mandatory to have a Tag Object per tag
 * defined in the Operation Object instances.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#tag-object
 */
export declare const TagObjectSchema: z.ZodObject<{
    name: z.ZodString;
    description: z.ZodCatch<z.ZodOptional<z.ZodString>>;
    externalDocs: z.ZodOptional<z.ZodObject<{
        description: z.ZodOptional<z.ZodString>;
        url: z.ZodString;
    }, z.core.$strip>>;
}, z.core.$strip>;
//# sourceMappingURL=tag-object.d.ts.map