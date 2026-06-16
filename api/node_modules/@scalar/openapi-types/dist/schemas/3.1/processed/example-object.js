import { z } from "zod";
const ExampleObjectSchema = z.object({
  /**
   * Short description for the example.
   */
  summary: z.string().optional(),
  /**
   * Long description for the example. CommonMark syntax MAY be used for rich text representation.
   */
  description: z.string().optional(),
  /**
   * Embedded literal example. The value field and externalValue field are mutually exclusive. To represent examples of media types that cannot naturally represented in JSON or YAML, use a string value to contain the example, escaping where necessary.
   */
  value: z.any().optional(),
  /**
   * A URI that identifies the literal example. This provides the capability to reference examples that cannot easily be
   * included in JSON or YAML documents. The value field and externalValue field are mutually exclusive. See the rules
   * for resolving Relative References.
   */
  externalValue: z.string().optional()
});
export {
  ExampleObjectSchema
};
//# sourceMappingURL=example-object.js.map
