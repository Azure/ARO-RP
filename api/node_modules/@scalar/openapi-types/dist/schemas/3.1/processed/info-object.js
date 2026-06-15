import { z } from "zod";
import { ContactObjectSchema } from "./contact-object.js";
import { LicenseObjectSchema } from "./license-object.js";
const InfoObjectSchema = z.object({
  /**
   * REQUIRED. The title of the API.
   */
  title: z.string().catch("API"),
  /**
   * A short summary of the API.
   */
  summary: z.string().optional().catch(void 0),
  /**
   * A description of the API. CommonMark syntax MAY be used for rich text representation.
   */
  description: z.string().optional().catch(void 0),
  /**
   * A URL to the Terms of Service for the API. This MUST be in the form of a URL.
   */
  termsOfService: z.string().url().optional().catch(void 0),
  /**
   * The contact information for the exposed API.
   */
  contact: ContactObjectSchema.optional().catch(void 0),
  /**
   * The license information for the exposed API.
   **/
  license: LicenseObjectSchema.optional().catch(void 0),
  /**
   * REQUIRED. The version of the OpenAPI Document (which is distinct from the OpenAPI Specification version or the
   * version of the API being described or the version of the OpenAPI Description).
   */
  version: z.string().catch("1.0")
});
export {
  InfoObjectSchema
};
//# sourceMappingURL=info-object.js.map
