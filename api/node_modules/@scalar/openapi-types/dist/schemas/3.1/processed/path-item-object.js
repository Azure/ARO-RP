import { BasePathItemObjectSchema } from "./base-path-item-object.js";
import { OperationObjectSchema } from "./operation-object.js";
const PathItemObjectSchema = BasePathItemObjectSchema.extend({
  /**
   * A definition of a GET operation on this path.
   */
  get: OperationObjectSchema.optional(),
  /**
   * A definition of a PUT operation on this path.
   */
  put: OperationObjectSchema.optional(),
  /**
   * A definition of a POST operation on this path.
   */
  post: OperationObjectSchema.optional(),
  /**
   * A definition of a DELETE operation on this path.
   */
  delete: OperationObjectSchema.optional(),
  /**
   * A definition of a OPTIONS operation on this path.
   */
  options: OperationObjectSchema.optional(),
  /**
   * A definition of a HEAD operation on this path.
   */
  head: OperationObjectSchema.optional(),
  /**
   * A definition of a PATCH operation on this path.
   */
  patch: OperationObjectSchema.optional(),
  /**
   * A definition of a TRACE operation on this path.
   */
  trace: OperationObjectSchema.optional()
});
export {
  PathItemObjectSchema
};
//# sourceMappingURL=path-item-object.js.map
