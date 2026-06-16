import { z } from 'zod';
export declare const XTagGroupSchema: z.ZodObject<{
    name: z.ZodString;
    tags: z.ZodCatch<z.ZodArray<z.ZodCoercedString<unknown>>>;
}, z.core.$strip>;
/**
 * x-tagGroups
 *
 * List of tags to include in this group.
 */
export declare const XTagGroupsSchema: z.ZodCatch<z.ZodArray<z.ZodObject<{
    name: z.ZodString;
    tags: z.ZodCatch<z.ZodArray<z.ZodCoercedString<unknown>>>;
}, z.core.$strip>>>;
//# sourceMappingURL=x-tag-groups.d.ts.map