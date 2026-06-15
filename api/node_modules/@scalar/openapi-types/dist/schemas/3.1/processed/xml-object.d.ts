import { z } from 'zod';
/**
 *
 * XML Object
 *
 * A metadata object that allows for more fine-tuned XML model definitions.
 *
 * When using arrays, XML element names are not inferred (for singular/plural forms) and the name field SHOULD be used
 * to add that information. See examples for expected behavior.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#xml-object
 */
export declare const XmlObjectSchema: z.ZodObject<{
    name: z.ZodOptional<z.ZodString>;
    namespace: z.ZodOptional<z.ZodString>;
    prefix: z.ZodOptional<z.ZodString>;
    attribute: z.ZodOptional<z.ZodBoolean>;
    wrapped: z.ZodOptional<z.ZodBoolean>;
}, z.core.$strip>;
//# sourceMappingURL=xml-object.d.ts.map