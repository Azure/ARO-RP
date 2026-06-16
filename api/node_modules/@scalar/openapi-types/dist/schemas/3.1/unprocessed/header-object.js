import { z } from "zod";
import { HeaderObjectSchema as OriginalHeaderObjectSchema } from "../processed/header-object.js";
import { ExampleObjectSchema } from "./example-object.js";
import { MediaTypeObjectSchemaWithoutEncoding } from "./media-type-object-without-encoding.js";
import { ReferenceObjectSchema } from "./reference-object.js";
import { SchemaObjectSchema } from "./schema-object.js";
const HeaderObjectSchema = OriginalHeaderObjectSchema.extend({
  /**
   * The schema defining the type used for the header.
   */
  schema: SchemaObjectSchema.optional(),
  /**
   * Examples of the parameter's potential value.
   */
  examples: z.record(z.string(), z.union([ReferenceObjectSchema, ExampleObjectSchema])).optional(),
  /**
   * A map containing the representations for the parameter.
   * The key is the media type and the value describes it.
   * Only one of content or schema should be specified.
   */
  content: z.record(z.string(), MediaTypeObjectSchemaWithoutEncoding).optional()
});
export {
  HeaderObjectSchema
};
//# sourceMappingURL=header-object.js.map
