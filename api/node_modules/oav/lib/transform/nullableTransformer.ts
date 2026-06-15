import { LiveValidatorLoggingLevels } from "../liveValidation/liveValidator";
import { JsonLoader } from "../swagger/jsonLoader";
import { Schema } from "../swagger/swaggerTypes";
import { xNullable } from "../util/constants";
import { allOfTransformer } from "./allOfTransformer";
import { GlobalTransformer, TransformerType } from "./transformer";

export const nullableTransformer: GlobalTransformer = {
  type: TransformerType.Global,
  after: [allOfTransformer],
  transform({ objSchemas, allParams, arrSchemas, jsonLoader, logging }) {
    for (const sch of objSchemas) {
      try {
        if (sch.properties !== undefined) {
          for (const key of Object.keys(sch.properties)) {
            sch.properties[key] = transformNullable(
              sch.properties[key],
              jsonLoader,
              !sch.required?.includes(key)
            );
          }
        }

        const aProperty = sch.additionalProperties;
        if (typeof aProperty === "object" && aProperty !== null) {
          sch.additionalProperties = transformNullable(
            aProperty,
            jsonLoader,
            undefined,
            aProperty.type === "object"
          );
        }
      } catch (e) {
        if (logging) {
          logging(
            `Fail to transform ${sch}. ErrorMessage:${e?.message};ErrorStack:${e?.stack}.`,
            LiveValidatorLoggingLevels.error
          );
        } else {
          console.log(
            `Fail to transform ${sch}. ErrorMessage:${e?.message};ErrorStack:${e?.stack}.`
          );
        }
      }
    }

    for (const sch of arrSchemas) {
      try {
        if (sch.items) {
          if (Array.isArray(sch.items)) {
            sch.items = sch.items.map((item) => transformNullable(item, jsonLoader));
          } else {
            sch.items = transformNullable(sch.items, jsonLoader);
          }
        }
      } catch (e) {
        if (logging) {
          logging(
            `Fail to transform ${sch}. ErrorMessage:${e?.message};ErrorStack:${e?.stack}.`,
            LiveValidatorLoggingLevels.error
          );
        } else {
          console.log(
            `Fail to transform ${sch}. ErrorMessage:${e?.message};ErrorStack:${e?.stack}.`
          );
        }
      }
    }

    for (const param of allParams) {
      if (param.in === "query" && param.allowEmptyValue) {
        param.nullable = true;
      }
    }
  },
};

const transformNullable = (
  s: Schema,
  jsonLoader: JsonLoader,
  defaultNullable?: boolean,
  additionalPropertiesWithObjectType?: boolean
) => {
  const sch = jsonLoader.resolveRefObj(s);
  const nullable = sch[xNullable] ?? sch.nullable;

  // Originally it's not nullable
  if (nullable === false) {
    return s;
  }

  // By default it's not nullable
  if (nullable === undefined && defaultNullable === false) {
    return s;
  }

  // Set nullable to true
  if (s !== sch) {
    // s isRefLike
    return {
      anyOf: [s, { type: "null", _skipError: true }],
      _skipError: true,
    } as Schema;
  } else {
    if (typeof sch === "object") {
      sch.nullable = true;
    }
    if (additionalPropertiesWithObjectType) {
      sch.additionalPropertiesWithObjectType = true;
    }
    return sch;
  }
};
