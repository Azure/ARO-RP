import { z } from "zod";
import { ExampleObjectSchema } from "./example-object.js";
import { MediaTypeObjectSchemaWithoutEncoding } from "./media-type-object-without-encoding.js";
import { SchemaObjectSchema } from "./schema-object.js";
const HeaderObjectSchema = z.object({
  /**
   * A brief description of the header. This could contain examples of use. CommonMark syntax MAY be used for rich text
   * representation.
   */
  description: z.string().optional(),
  /**
   * Determines whether this header is mandatory. The default value is false.
   */
  required: z.boolean().optional(),
  /**
   * Specifies that the header is deprecated and SHOULD be transitioned out of usage. Default value is false.
   */
  deprecated: z.boolean().optional(),
  /**
   * Describes how the parameter value will be serialized. Only "simple" is allowed for headers.
   */
  style: z.enum(["matrix", "label", "simple", "form", "spaceDelimited", "pipeDelimited", "deepObject"]).optional(),
  /**
   * When this is true, parameter values of type array or object generate separate parameters
   * for each value of the array or key-value pair of the map.
   */
  explode: z.boolean().optional(),
  /**
   * The schema defining the type used for the header.
   */
  schema: SchemaObjectSchema.optional(),
  /**
   * Example of the parameter's potential value.
   */
  example: z.any().optional(),
  /**
   * Examples of the parameter's potential value.
   */
  examples: z.record(z.string(), ExampleObjectSchema).optional(),
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
