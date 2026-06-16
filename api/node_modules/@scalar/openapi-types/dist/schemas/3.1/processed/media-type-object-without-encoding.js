import { z } from "zod";
import { ExampleObjectSchema } from "./example-object.js";
import { SchemaObjectSchema } from "./schema-object.js";
const MediaTypeObjectSchemaWithoutEncoding = z.object({
  /**
   * The schema defining the content of the request, response, or parameter.
   */
  schema: SchemaObjectSchema.optional(),
  /**
   * Example of the media type. The example object SHOULD be in the correct format as specified by the media type.
   * The example field is mutually exclusive of the examples field. Furthermore, if referencing a schema which contains
   * an example, the example value SHALL override the example provided by the schema.
   */
  example: z.any().optional(),
  /**
   * Examples of the media type. Each example object SHOULD match the media type and specified schema if present.
   * The examples field is mutually exclusive of the example field. Furthermore, if referencing a schema which contains
   * an example, the examples value SHALL override the example provided by the schema.
   */
  examples: z.record(z.string(), ExampleObjectSchema).optional()
  // Note: Don't add `encoding` here.
  // The MediaTypeObjectSchema is used in multiple places. And when it's used in headers, we don't want the encoding.
  // That's what the OpenAPI specification says.
});
export {
  MediaTypeObjectSchemaWithoutEncoding
};
//# sourceMappingURL=media-type-object-without-encoding.js.map
