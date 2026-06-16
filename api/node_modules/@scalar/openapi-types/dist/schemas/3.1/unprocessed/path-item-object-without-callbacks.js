import { BasePathItemObjectSchema } from "./base-path-item-object.js";
import { OperationObjectSchemaWithoutCallbacks } from "./operation-object-without-callbacks.js";
const PathItemObjectSchemaWithoutCallbacks = BasePathItemObjectSchema.extend({
  /**
   * A definition of a GET operation on this path.
   */
  get: OperationObjectSchemaWithoutCallbacks.optional(),
  /**
   * A definition of a PUT operation on this path.
   */
  put: OperationObjectSchemaWithoutCallbacks.optional(),
  /**
   * A definition of a POST operation on this path.
   */
  post: OperationObjectSchemaWithoutCallbacks.optional(),
  /**
   * A definition of a DELETE operation on this path.
   */
  delete: OperationObjectSchemaWithoutCallbacks.optional(),
  /**
   * A definition of a OPTIONS operation on this path.
   */
  options: OperationObjectSchemaWithoutCallbacks.optional(),
  /**
   * A definition of a HEAD operation on this path.
   */
  head: OperationObjectSchemaWithoutCallbacks.optional(),
  /**
   * A definition of a PATCH operation on this path.
   */
  patch: OperationObjectSchemaWithoutCallbacks.optional(),
  /**
   * A definition of a TRACE operation on this path.
   */
  trace: OperationObjectSchemaWithoutCallbacks.optional()
});
export {
  PathItemObjectSchemaWithoutCallbacks
};
//# sourceMappingURL=path-item-object-without-callbacks.js.map
