import { z } from 'zod';
import { CallbackObjectSchema } from './callback-object.js';
import { OperationObjectSchemaWithoutCallbacks } from './operation-object-without-callbacks.js';
type OperationObject = z.infer<typeof OperationObjectSchemaWithoutCallbacks> & {
    callbacks?: Record<string, z.infer<typeof CallbackObjectSchema>>;
};
/**
 * Operation Object
 *
 * Describes a single API operation on a path.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#operation-object
 */
export declare const OperationObjectSchema: z.ZodType<OperationObject>;
export {};
//# sourceMappingURL=operation-object.d.ts.map