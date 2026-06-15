import { copyInfo } from "@azure-tools/openapi-tools-common";
import { LiveValidatorLoggingLevels } from "../liveValidation/liveValidator";
import { isRefLike, JsonLoader } from "../swagger/jsonLoader";
import { refSelfSymbol, Schema } from "../swagger/swaggerTypes";
import { xmsDiscriminatorValue } from "../util/constants";
import { allOfTransformer } from "./allOfTransformer";
import { getNameFromRef } from "./context";
import { GlobalTransformer, TransformerType } from "./transformer";

const getDiscriminatorRoot = (
  sch: Schema,
  visited: Map<Schema, string | null>,
  baseSchemas: Set<Schema>,
  jsonLoader: JsonLoader
): string | null => {
  if (sch.discriminator !== undefined) {
    return sch[refSelfSymbol] ?? null;
  }
  if (sch.allOf === undefined) {
    return null;
  }
  let root = visited.get(sch);
  if (root !== undefined) {
    return root;
  }
  visited.set(sch, null);
  for (let subSch of sch.allOf) {
    if (!isRefLike(subSch)) {
      continue;
    }
    subSch = jsonLoader.resolveRefObj(subSch);
    root = getDiscriminatorRoot(subSch, visited, baseSchemas, jsonLoader);
    if (root !== null) {
      baseSchemas.add(subSch);
      visited.set(sch, root);
      return root;
    }
  }
  return null;
};

const getDiscriminatorValue = (sch: Schema) => {
  const discriminatorValue = sch[xmsDiscriminatorValue] ?? getNameFromRef(sch);
  if (discriminatorValue === undefined) {
    throw new Error("undefined discriminatorValue!");
  }
  return discriminatorValue;
};

export const discriminatorTransformer: GlobalTransformer = {
  type: TransformerType.Global,
  before: [allOfTransformer],
  transform({ objSchemas, baseSchemas, jsonLoader, logging }) {
    const visited = new Map<Schema, string | null>();
    for (const sch of objSchemas) {
      try {
        const rootRef = getDiscriminatorRoot(sch, visited, baseSchemas, jsonLoader);
        if (rootRef === null) {
          if (sch[xmsDiscriminatorValue] !== undefined) {
            sch._missingDiscriminator = true;
          }
          continue;
        }

        const baseSch = jsonLoader.resolveRefObj({ $ref: rootRef } as Schema);
        const $ref = sch[refSelfSymbol];
        const discriminatorValue = getDiscriminatorValue(sch);

        if (baseSch.discriminatorMap === undefined) {
          baseSch.discriminatorMap = {
            [getDiscriminatorValue(baseSch)]: null,
          };
          copyInfo(baseSch, baseSch.discriminatorMap);
        }
        baseSch.discriminatorMap[discriminatorValue] = { $ref } as unknown as Schema;
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
  },
};
