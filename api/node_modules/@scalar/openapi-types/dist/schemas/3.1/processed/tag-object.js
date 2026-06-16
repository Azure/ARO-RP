import { z } from "zod";
import { ExternalDocumentationObjectSchema } from "./external-documentation-object.js";
const TagObjectSchema = z.object({
  /** REQUIRED. The name of the tag. */
  "name": z.string(),
  /** A description for the tag. CommonMark syntax MAY be used for rich text representation. */
  "description": z.string().optional().catch(void 0),
  /** Additional external documentation for this tag. */
  "externalDocs": ExternalDocumentationObjectSchema.optional()
});
export {
  TagObjectSchema
};
//# sourceMappingURL=tag-object.js.map
