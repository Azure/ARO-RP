import type { z } from 'zod';
import { BasePathItemObjectSchema } from './base-path-item-object.js';
import { OperationObjectSchemaWithoutCallbacks } from './operation-object-without-callbacks.js';
type PathItemObjectSchemaWithoutCallbacks = z.infer<typeof BasePathItemObjectSchema> & {
    get?: z.infer<typeof OperationObjectSchemaWithoutCallbacks>;
    put?: z.infer<typeof OperationObjectSchemaWithoutCallbacks>;
    post?: z.infer<typeof OperationObjectSchemaWithoutCallbacks>;
    delete?: z.infer<typeof OperationObjectSchemaWithoutCallbacks>;
    options?: z.infer<typeof OperationObjectSchemaWithoutCallbacks>;
    head?: z.infer<typeof OperationObjectSchemaWithoutCallbacks>;
    patch?: z.infer<typeof OperationObjectSchemaWithoutCallbacks>;
    trace?: z.infer<typeof OperationObjectSchemaWithoutCallbacks>;
};
/**
 * Path Item Object (without callbacks)
 *
 * Describes the operations available on a single path. A Path Item MAY be empty, due to ACL constraints. The path
 * itself is still exposed to the documentation viewer but they will not know which operations and parameters are
 * available.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#path-item-object
 */
export declare const PathItemObjectSchemaWithoutCallbacks: z.ZodType<PathItemObjectSchemaWithoutCallbacks>;
export {};
//# sourceMappingURL=path-item-object-without-callbacks.d.ts.map