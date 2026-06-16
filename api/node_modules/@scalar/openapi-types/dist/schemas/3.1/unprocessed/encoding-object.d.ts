import { z } from 'zod';
/**
 * Encoding Object
 *
 * A single encoding definition applied to a single schema property. See Appendix B for a discussion of converting
 * values of various types to string representations.
 *
 * Properties are correlated with multipart parts using the name parameter of Content-Disposition: form-data, and with
 * application/x-www-form-urlencoded using the query string parameter names. In both cases, their order is
 * implementation-defined.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#encoding-object
 */
export declare const EncodingObjectSchema: z.ZodObject<{
    contentType: z.ZodString;
    headers: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodUnion<readonly [z.ZodObject<{
        $ref: z.ZodString;
        summary: z.ZodOptional<z.ZodString>;
        description: z.ZodOptional<z.ZodString>;
    }, z.core.$strip>, z.ZodObject<{
        description: z.ZodOptional<z.ZodString>;
        required: z.ZodOptional<z.ZodBoolean>;
        deprecated: z.ZodOptional<z.ZodBoolean>;
        style: z.ZodOptional<z.ZodEnum<{
            matrix: "matrix";
            label: "label";
            form: "form";
            simple: "simple";
            spaceDelimited: "spaceDelimited";
            pipeDelimited: "pipeDelimited";
            deepObject: "deepObject";
        }>>;
        explode: z.ZodOptional<z.ZodBoolean>;
        example: z.ZodOptional<z.ZodAny>;
        schema: z.ZodOptional<z.ZodType<any, unknown, z.core.$ZodTypeInternals<any, unknown>>>;
        examples: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodUnion<readonly [z.ZodObject<{
            $ref: z.ZodString;
            summary: z.ZodOptional<z.ZodString>;
            description: z.ZodOptional<z.ZodString>;
        }, z.core.$strip>, z.ZodObject<{
            summary: z.ZodOptional<z.ZodString>;
            description: z.ZodOptional<z.ZodString>;
            value: z.ZodOptional<z.ZodAny>;
            externalValue: z.ZodOptional<z.ZodString>;
        }, z.core.$strip>]>>>;
        content: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
            schema: z.ZodOptional<z.ZodType<Record<string, any>, unknown, z.core.$ZodTypeInternals<Record<string, any>, unknown>>>;
            example: z.ZodOptional<z.ZodAny>;
            examples: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
                summary: z.ZodOptional<z.ZodString>;
                description: z.ZodOptional<z.ZodString>;
                value: z.ZodOptional<z.ZodAny>;
                externalValue: z.ZodOptional<z.ZodString>;
            }, z.core.$strip>>>;
        }, z.core.$strip>>>;
    }, z.core.$strip>]>>>;
}, z.core.$strip>;
//# sourceMappingURL=encoding-object.d.ts.map