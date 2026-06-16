import { z } from "zod";
const ExternalDocumentationObjectSchema = z.object({
  /** A description of the target documentation. CommonMark syntax MAY be used for rich text representation. */
  description: z.string().optional(),
  /** REQUIRED. The URL for the target documentation. This MUST be in the form of a URL. */
  url: z.string()
});
export {
  ExternalDocumentationObjectSchema
};
//# sourceMappingURL=external-documentation-object.js.map
