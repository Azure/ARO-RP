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
export declare const LinkObjectSchema: import("zod").ZodObject<{
    operationRef: import("zod").ZodOptional<import("zod").ZodString>;
    operationId: import("zod").ZodOptional<import("zod").ZodString>;
    parameters: import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodString>>;
    requestBody: import("zod").ZodOptional<import("zod").ZodString>;
    description: import("zod").ZodOptional<import("zod").ZodString>;
    server: import("zod").ZodOptional<import("zod").ZodObject<{
        url: import("zod").ZodString;
        description: import("zod").ZodOptional<import("zod").ZodString>;
        variables: import("zod").ZodOptional<import("zod").ZodRecord<import("zod").ZodString, import("zod").ZodObject<{
            enum: import("zod").ZodOptional<import("zod").ZodArray<import("zod").ZodString>>;
            default: import("zod").ZodOptional<import("zod").ZodString>;
            description: import("zod").ZodOptional<import("zod").ZodString>;
        }, import("zod/v4/core").$strip>>>;
    }, import("zod/v4/core").$strip>>;
}, import("zod/v4/core").$strip>;
//# sourceMappingURL=link-object.d.ts.map