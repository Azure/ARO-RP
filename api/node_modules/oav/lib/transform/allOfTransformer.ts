import { LiveValidatorLoggingLevels } from "../liveValidation/liveValidator";
import { JsonLoader } from "../swagger/jsonLoader";
import { Schema } from "../swagger/swaggerTypes";
import { GlobalTransformer, TransformerType } from "./transformer";

const transformAllOfSchema = (schema: Schema, baseSchemas: Set<Schema>, jsonLoader: JsonLoader) => {
  if (schema.type !== undefined && schema.type !== "object") {
    return;
  }
  if (schema.allOf === undefined) {
    return;
  }

  if (schema.properties === undefined) {
    schema.properties = {};
  }
  for (const s of schema.allOf) {
    const sch = jsonLoader.resolveRefObj(s);
    transformAllOfSchema(sch, baseSchemas, jsonLoader);

    const { properties, required, additionalProperties: aProperties } = sch;
    if (sch["x-ms-azure-resource"] === true && schema["x-ms-azure-resource"] === undefined) {
      schema["x-ms-azure-resource"] = true;
    }
    if (properties !== undefined) {
      for (const propertyName of Object.keys(properties)) {
        if (!(propertyName in schema.properties)) {
          schema.properties[propertyName] = properties[propertyName];
        }
      }
    }
    if (required !== undefined && required.length > 0) {
      if (schema.required === undefined) {
        schema.required = [...required];
      } else {
        for (const key of required) {
          if (!schema.required.includes(key)) {
            schema.required.push(key);
          }
        }
      }
    }
    if (aProperties !== undefined && schema.additionalProperties === undefined) {
      // schema.additionalProperties = aProperties;
    }
  }
  if (!baseSchemas.has(schema) || schema.discriminator !== undefined) {
    // A -> B -> C, A has discriminator and B don't have, and C has discriminatorValue
    // If some schema references B, then we need to depends on B's allOf to validate on A
    // which will finally validate C via discriminatorMap. In this case we won't remove
    // allOf on B, which is: isBaseSchema && discriminator === undefined
    delete schema.allOf;
  }
};

// Must after transformDiscriminator
export const allOfTransformer: GlobalTransformer = {
  type: TransformerType.Global,
  transform({ objSchemas, baseSchemas, jsonLoader, logging }) {
    for (const sch of objSchemas) {
      try {
        if (sch.allOf !== undefined) {
          transformAllOfSchema(sch, baseSchemas, jsonLoader);
        }
      } catch (e) {
        if (logging) {
          logging(
            `Fail to transform ${sch}}. ErrorMessage:${e?.message};ErrorStack:${e?.stack}.`,
            LiveValidatorLoggingLevels.error
          );
        } else {
          console.log(
            `Fail to transform ${sch}. ErrorMessage:${e?.message};ErrorStack:${e?.stack}.`
          );
        }
      }
    }
  },
};
