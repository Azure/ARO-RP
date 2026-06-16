import { z } from "zod";
import { CallbackObjectSchema } from "./callback-object.js";
import { OperationObjectSchemaWithoutCallbacks } from "./operation-object-without-callbacks.js";
const OperationObjectSchema = OperationObjectSchemaWithoutCallbacks.extend({
  /**
   * A map of possible out-of-band callbacks related to the parent operation. Each value in the map is a
   * Path Item Object that describes a set of requests that may be initiated by the API provider and the
   * expected responses. The key value used to identify the callback object is an expression, evaluated
   * at runtime, that identifies a URL to be used for the callback operation.
   */
  "callbacks": z.record(z.string(), CallbackObjectSchema).optional()
});
export {
  OperationObjectSchema
};
//# sourceMappingURL=operation-object.js.map
