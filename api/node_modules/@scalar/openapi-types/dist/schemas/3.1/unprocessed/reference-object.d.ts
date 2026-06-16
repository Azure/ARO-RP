import { z } from 'zod';
/**
 * Reference Object
 *
 * A simple object to allow referencing other components in the OpenAPI Description, internally and externally.
 *
 * The $ref string value contains a URI RFC3986, which identifies the value being referenced.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#reference-object
 */
export declare const ReferenceObjectSchema: z.ZodObject<{
    $ref: z.ZodString;
    summary: z.ZodOptional<z.ZodString>;
    description: z.ZodOptional<z.ZodString>;
}, z.core.$strip>;
//# sourceMappingURL=reference-object.d.ts.map