import { z } from "zod";
const XTagGroupSchema = z.object({
  /**
   * The group name.
   */
  name: z.string(),
  /**
   * List of tags to include in this group.
   */
  tags: z.coerce.string().array().catch([])
});
const XTagGroupsSchema = XTagGroupSchema.array().catch([]);
export {
  XTagGroupSchema,
  XTagGroupsSchema
};
//# sourceMappingURL=x-tag-groups.js.map
