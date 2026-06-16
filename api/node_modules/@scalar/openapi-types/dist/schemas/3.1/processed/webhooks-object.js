import { z } from "zod";
import { PathItemObjectSchemaWithoutCallbacks } from "./path-item-object-without-callbacks.js";
const WebhooksObjectSchema = z.record(z.string(), PathItemObjectSchemaWithoutCallbacks);
export {
  WebhooksObjectSchema
};
//# sourceMappingURL=webhooks-object.js.map
