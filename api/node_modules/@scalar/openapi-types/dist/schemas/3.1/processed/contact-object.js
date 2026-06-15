import { z } from "zod";
const ContactObjectSchema = z.object({
  /** The identifying name of the contact person/organization. */
  name: z.string().optional(),
  /** The URL pointing to the contact information. This MUST be in the form of a URL. */
  url: z.string().url().optional().catch(void 0),
  /** The email address of the contact person/organization. This MUST be in the form of an email address. */
  email: z.string().optional().catch(void 0)
});
export {
  ContactObjectSchema
};
//# sourceMappingURL=contact-object.js.map
