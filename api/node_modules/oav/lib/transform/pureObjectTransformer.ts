import { GlobalTransformer, TransformerType } from "./transformer";

export const pureObjectTransformer: GlobalTransformer = {
  type: TransformerType.Global,
  transform({ objSchemas }) {
    for (const sch of objSchemas) {
      if (
        sch.type === "object" &&
        (sch.properties === undefined || Object.keys(sch.properties).length === 0) &&
        sch.additionalProperties === undefined &&
        sch.additionalPropertiesWithObjectType !== true
      ) {
        delete sch.type;
      }
    }
  },
};
