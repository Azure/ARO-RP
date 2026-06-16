import { z } from "zod";
const PostResponseSchema = z.string();
const XPostResponseSchema = z.object({
  "x-post-response": PostResponseSchema.optional()
});
export {
  PostResponseSchema,
  XPostResponseSchema
};
//# sourceMappingURL=x-post-response.js.map
