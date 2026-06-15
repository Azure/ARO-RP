import { z } from 'zod';
/**
 * Example Object
 *
 * An object grouping an internal or external example value with basic summary and description metadata. This object is
 * typically used in fields named examples (plural), and is a referenceable alternative to older example (singular)
 * fields that do not support referencing or metadata.
 *
 * Examples allow demonstration of the usage of properties, parameters and objects within OpenAPI.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#example-object
 */
export declare const ExampleObjectSchema: z.ZodObject<{
    summary: z.ZodOptional<z.ZodString>;
    description: z.ZodOptional<z.ZodString>;
    value: z.ZodOptional<z.ZodAny>;
    externalValue: z.ZodOptional<z.ZodString>;
}, z.core.$strip>;
//# sourceMappingURL=example-object.d.ts.map