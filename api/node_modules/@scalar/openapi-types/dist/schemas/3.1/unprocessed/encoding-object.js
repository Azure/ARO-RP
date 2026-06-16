import { z } from "zod";
import { EncodingObjectSchema as OriginalEncodingObjectSchema } from "../processed/encoding-object.js";
import { HeaderObjectSchema } from "./header-object.js";
import { ReferenceObjectSchema } from "./reference-object.js";
const EncodingObjectSchema = OriginalEncodingObjectSchema.extend({
  /**
   * A map allowing additional information to be provided as headers. Content-Type is described separately and SHALL be
   * ignored in this section. This field SHALL be ignored if the request body media type is not a multipart.
   */
  headers: z.record(z.string(), z.union([ReferenceObjectSchema, HeaderObjectSchema])).optional()
});
export {
  EncodingObjectSchema
};
//# sourceMappingURL=encoding-object.js.map
