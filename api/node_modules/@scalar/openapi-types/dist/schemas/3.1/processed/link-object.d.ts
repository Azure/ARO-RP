import { z } from 'zod';
/**
 * Link Object
 *
 * The Link Object represents a possible design-time link for a response. The presence of a link does not guarantee the
 * caller's ability to successfully invoke it, rather it provides a known relationship and traversal mechanism between
 * responses and other operations.
 *
 * Unlike dynamic links (i.e. links provided in the response payload), the OAS linking mechanism does not require link
 * information in the runtime response.
 *
 * For computing links and providing instructions to execute them, a runtime expression is used for accessing values in an
 * operation and using them as parameters while invoking the linked operation.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#link-object
 */
export declare const LinkObjectSchema: z.ZodObject<{
    operationRef: z.ZodOptional<z.ZodString>;
    operationId: z.ZodOptional<z.ZodString>;
    parameters: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodString>>;
    requestBody: z.ZodOptional<z.ZodString>;
    description: z.ZodOptional<z.ZodString>;
    server: z.ZodOptional<z.ZodObject<{
        url: z.ZodString;
        description: z.ZodOptional<z.ZodString>;
        variables: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodObject<{
            enum: z.ZodOptional<z.ZodArray<z.ZodString>>;
            default: z.ZodOptional<z.ZodString>;
            description: z.ZodOptional<z.ZodString>;
        }, z.core.$strip>>>;
    }, z.core.$strip>>;
}, z.core.$strip>;
//# sourceMappingURL=link-object.d.ts.map