import { GlobalTransformer, TransformerType } from "./transformer";

export const schemaV4ToV7Transformer: GlobalTransformer = {
  type: TransformerType.Global,
  transform({ primSchemas }) {
    // Transform from json schema draft 04 to draft 07
    for (const sch of primSchemas) {
      if (typeof sch.exclusiveMinimum === "boolean") {
        sch.exclusiveMinimum = sch.exclusiveMinimum ? sch.minimum : sch.minimum! - 1;
      }
      if (typeof sch.exclusiveMaximum === "boolean") {
        sch.exclusiveMaximum = sch.exclusiveMaximum ? sch.maximum : sch.maximum! + 1;
      }
    }
  },
};
