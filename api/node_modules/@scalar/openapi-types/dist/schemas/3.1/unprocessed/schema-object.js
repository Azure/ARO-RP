import { z } from "zod";
import { SchemaObjectSchema as OriginalSchemaObjectSchema } from "../processed/schema-object.js";
import { ReferenceObjectSchema } from "./reference-object.js";
const SchemaObjectSchema = z.lazy(
  () => z.intersection(
    OriginalSchemaObjectSchema,
    z.object({
      // Reference-related fields
      $ref: z.string().optional(),
      $id: z.string().optional(),
      $schema: z.string().optional(),
      $defs: z.record(
        z.string(),
        z.lazy(() => SchemaObjectSchema)
      ).optional(),
      $dynamicRef: z.string().optional(),
      $dynamicAnchor: z.string().optional(),
      // Override the recursive fields to use this schema instead of the original
      properties: z.record(z.string(), z.union([z.lazy(() => SchemaObjectSchema), ReferenceObjectSchema])).optional(),
      additionalProperties: z.union([z.boolean(), z.lazy(() => SchemaObjectSchema), ReferenceObjectSchema]).optional(),
      patternProperties: z.record(z.string(), z.union([z.lazy(() => SchemaObjectSchema), ReferenceObjectSchema])).optional(),
      items: z.union([z.lazy(() => SchemaObjectSchema), ReferenceObjectSchema]).optional(),
      prefixItems: z.array(z.union([z.lazy(() => SchemaObjectSchema), ReferenceObjectSchema])).optional(),
      allOf: z.array(z.union([z.lazy(() => SchemaObjectSchema), ReferenceObjectSchema])).optional(),
      oneOf: z.array(z.union([z.lazy(() => SchemaObjectSchema), ReferenceObjectSchema])).optional(),
      anyOf: z.array(z.union([z.lazy(() => SchemaObjectSchema), ReferenceObjectSchema])).optional(),
      not: z.union([z.lazy(() => SchemaObjectSchema), ReferenceObjectSchema]).optional()
    })
  )
);
export {
  SchemaObjectSchema
};
//# sourceMappingURL=schema-object.js.map
