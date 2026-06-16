import { z } from 'zod';
/**
 * External Documentation Object
 *
 * Allows referencing an external resource for extended documentation.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#external-documentation-object
 */
export declare const ExternalDocumentationObjectSchema: z.ZodObject<{
    description: z.ZodOptional<z.ZodString>;
    url: z.ZodString;
}, z.core.$strip>;
//# sourceMappingURL=external-documentation-object.d.ts.map