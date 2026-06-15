import { z } from "zod";
import { PathItemObjectSchemaWithoutCallbacks } from "./path-item-object-without-callbacks.js";
import { RuntimeExpressionSchema } from "./runtime-expression.js";
const CallbackObjectSchema = z.record(RuntimeExpressionSchema, PathItemObjectSchemaWithoutCallbacks);
export {
  CallbackObjectSchema
};
//# sourceMappingURL=callback-object.js.map
