import { JsonLoader } from "../swagger/jsonLoader";
import { LoggingFn, Parameter, refSelfSymbol, Schema } from "../swagger/swaggerTypes";
import { SchemaValidator } from "../swaggerValidator/schemaValidator";
import { GlobalTransformer, sortTransformers, SpecTransformer, Transformer } from "./transformer";

export interface TransformContext {
  jsonLoader: JsonLoader;
  schemaValidator: SchemaValidator;

  objSchemas: Schema[];
  arrSchemas: Schema[];
  primSchemas: Schema[];
  allParams: Parameter[];
  baseSchemas: Set<Schema>;

  specTransformers: SpecTransformer[];
  globalTransformers: GlobalTransformer[];

  logging?: LoggingFn;
}

export const getTransformContext = (
  jsonLoader: JsonLoader,
  schemaValidator: SchemaValidator,
  transformers: Array<Transformer | undefined>,
  logging?: LoggingFn
): TransformContext => {
  return {
    jsonLoader,
    schemaValidator,
    objSchemas: [],
    arrSchemas: [],
    primSchemas: [],
    allParams: [],
    baseSchemas: new Set(),
    ...sortTransformers(transformers.filter(Boolean) as Transformer[]),
    logging,
  };
};

export const getNameFromRef = (sch: Schema | undefined) => {
  const sp = sch?.[refSelfSymbol]?.split("/");
  return sp === undefined ? undefined : sp[sp.length - 1];
};
