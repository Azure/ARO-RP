import { z } from "zod";
import { ReferenceObjectSchema } from "./reference-object.js";
import { ResponseObjectSchema } from "./response-object.js";
const ResponsesObjectSchema = z.record(
  /**
   * Response Object | Reference Object	Any HTTP status code can be used as the property name, but only one property per
   *  code, to describe the expected response for that HTTP status code. This field MUST be enclosed in quotation marks
   * (for example, "200") for compatibility between JSON and YAML. To define a range of response codes, this field MAY
   * contain the uppercase wildcard character X. For example, 2XX represents all response codes between 200 and 299.
   * Only the following range definitions are allowed: 1XX, 2XX, 3XX, 4XX, and 5XX. If a response is defined using an
   * explicit code, the explicit code definition takes precedence over the range definition for that code.
   */
  z.string(),
  z.union([ReferenceObjectSchema, ResponseObjectSchema])
);
export {
  ResponsesObjectSchema
};
//# sourceMappingURL=responses-object.js.map
