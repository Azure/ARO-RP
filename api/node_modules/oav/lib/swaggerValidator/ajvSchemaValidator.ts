import * as lodash from "lodash";
import {
  ChildObjectInfo,
  getInfo,
  getRootObjectInfo,
  RootObjectInfo,
} from "@azure-tools/openapi-tools-common";
import { Ajv, default as ajvInit, ErrorObject, ValidateFunction } from "ajv";
import { inject, injectable } from "inversify";
import { TYPES } from "../inversifyUtils";
import { $id, JsonLoader } from "../swagger/jsonLoader";
import { isSuppressed } from "../swagger/suppressionLoader";
import { refSelfSymbol, Schema, SwaggerSpec } from "../swagger/swaggerTypes";
import { getNameFromRef } from "../transform/context";
import {
  xmsAzureResource,
  xmsEnum,
  xmsMutability,
  xmsReadonlyRef,
  xmsSecret,
} from "../util/constants";
import { getOavErrorMeta, TrafficValidationErrorCode } from "../util/errorDefinitions";
import { Severity } from "../util/severity";
import { Writable } from "../util/utils";
import { SourceLocation } from "../util/validationError";
import { ajvEnableAll, ajvEnableArmRule, ajvEnableArmIdFormat } from "./ajv";
import {
  getIncludeErrorsMap,
  SchemaValidateContext,
  SchemaValidateFunction,
  SchemaValidateIssue,
  SchemaValidator,
  SchemaValidatorOption,
} from "./schemaValidator";

@injectable()
export class AjvSchemaValidator implements SchemaValidator {
  private ajv: Ajv;

  public constructor(
    loader: JsonLoader,
    @inject(TYPES.opts) schemaValidatorOption?: SchemaValidatorOption
  ) {
    this.ajv = ajvInit({
      // tslint:disable-next-line: no-submodule-imports
      meta: require("ajv/lib/refs/json-schema-draft-04.json"),
      schemaId: "auto",
      extendRefs: "fail",
      format: "full",
      missingRefs: true,
      addUsedSchema: false,
      removeAdditional: false,
      nullable: true,
      allErrors: true,
      messages: true,
      verbose: true,
      inlineRefs: false,
      passContext: true,
      loopRequired: 2,
      unknownFormats: "ignore",
      loadSchema: async (uri) => {
        const spec: SwaggerSpec = await loader.resolveFile(uri);
        return { [$id]: spec[$id], definitions: spec.definitions, parameters: spec.parameters };
      },
    });
    ajvEnableAll(this.ajv, loader);

    // always enable the armId format validation
    ajvEnableArmIdFormat(this.ajv);

    if (schemaValidatorOption?.isArmCall === true) {
      ajvEnableArmRule(this.ajv);
    }
  }

  public async compileAsync(schema: Schema): Promise<SchemaValidateFunction> {
    const validate = await this.ajv.compileAsync(schema);
    return this.getValidateFunction(validate);
  }

  public compile(schema: Schema): SchemaValidateFunction {
    const validate = this.ajv.compile(schema);
    return this.getValidateFunction(validate);
  }

  private getValidateFunction(validate: ValidateFunction) {
    const ret = function validateSchema(ctx: SchemaValidateContext, data: any) {
      const result: SchemaValidateIssue[] = [];
      const isValid = validateSchema.validate.call(ctx, data);
      if (!isValid) {
        const errors = ReValidateIfNeed(
          validateSchema.validate.errors!,
          ctx,
          data,
          validateSchema.validate
        );
        if (errors.length > 0) {
          ajvErrorListToSchemaValidateIssueList(errors, ctx, result);
        }
        validateSchema.validate.errors = null;
      }
      return result;
    };
    ret.validate = validate;
    return ret;
  }
}

export const ajvErrorListToSchemaValidateIssueList = (
  errors: ErrorObject[],
  ctx: SchemaValidateContext,
  result: SchemaValidateIssue[]
) => {
  const includeErrorsSet = getIncludeErrorsMap(ctx.includeErrors);

  const similarIssues: Map<string, SchemaValidateIssue> = new Map();
  for (const error of errors) {
    const issue = ajvErrorToSchemaValidateIssue(error, ctx);
    if (
      issue === undefined ||
      (includeErrorsSet !== undefined && !includeErrorsSet.has(issue.code))
    ) {
      continue;
    }

    const issueHashKey = [
      issue.code,
      issue.message,
      issue.source.url,
      issue.source.position.column.toString(),
      issue.source.position.line.toString(),
    ].join("|");
    const similarIssue = similarIssues.get(issueHashKey);
    if (similarIssue === undefined) {
      similarIssues.set(issueHashKey, issue);
      result.push(issue);
      continue;
    }

    similarIssue.jsonPathsInPayload.push(issue.jsonPathsInPayload[0]);
  }

  for (const issue of result) {
    if (issue.jsonPathsInPayload.length > 1) {
      (issue as any).jsonPathsInPayload = [...new Set(issue.jsonPathsInPayload)];
    }
  }
};

export const sourceMapInfoToSourceLocation = (
  info?: ChildObjectInfo | RootObjectInfo
): Writable<SourceLocation> => {
  return info === undefined
    ? {
        url: "",
        position: { line: -1, column: -1 },
      }
    : {
        url: getRootObjectInfo(info).url,
        position: { line: info.position.line, column: info.position.column },
      };
};

export const ajvErrorToSchemaValidateIssue = (
  err: ErrorObject,
  ctx: SchemaValidateContext
): SchemaValidateIssue | undefined => {
  const { parentSchema, params } = err;
  let { schema } = err;

  if (shouldSkipError(err, ctx)) {
    return undefined;
  }

  let dataPath = err.dataPath;
  const extraDataPath =
    (params as any).additionalProperty ??
    (params as any).missingProperty ??
    (parentSchema as Schema).discriminator;
  if (extraDataPath !== undefined) {
    dataPath = `${dataPath}.${extraDataPath}`;
    if (schema[extraDataPath] !== undefined) {
      schema = schema[extraDataPath];
    }
  }

  const errInfo = ajvErrorCodeToOavErrorCode(err, ctx);
  if (errInfo === undefined) {
    return undefined;
  }
  if (isSuppressed(err.parentSchema, errInfo.code, errInfo.message)) {
    return undefined;
  }

  let sch: Schema | undefined = parentSchema;
  let info = getInfo(sch);
  if (info === undefined) {
    sch = schema;
    info = getInfo(sch);
  }
  const source = sourceMapInfoToSourceLocation(info);
  if (sch?.[refSelfSymbol] !== undefined) {
    source.jsonRef = sch[refSelfSymbol]!.substr(sch[refSelfSymbol]!.indexOf("#"));
  }

  const result = errInfo as SchemaValidateIssue;
  result.jsonPathsInPayload = [dataPath];
  result.schemaPath = err.schemaPath;
  result.source = source;

  return result;
};

const ReValidateIfNeed = (
  originalErrors: ErrorObject[],
  ctx: SchemaValidateContext,
  data: any,
  validate: ValidateFunction
): ErrorObject[] => {
  const result: ErrorObject[] = [];
  const newData = lodash.cloneDeep(data);

  for (const originalError of originalErrors) {
    validate.errors = null;
    const { schema, parentSchema: parentSch, keyword, data: errorData, dataPath } = originalError;
    const parentSchema = parentSch as Schema;

    // If the value of query parameter is in string format, we can revalidate this error
    if (
      !ctx.isResponse &&
      keyword === "type" &&
      schema === "array" &&
      typeof errorData === "string" &&
      (parentSchema as any)?.["in"] === "query"
    ) {
      const arrayData = errorData.split(",").map((item) => {
        // when item is number
        const numberRegex = /^[+-]?\d+(\.\d+)?([Ee]\+?\d+)?$/g;
        if (numberRegex.test(item)) {
          return parseFloat(item);
        }
        // when item is boolean
        if (item === "true" || item === "false") {
          return item === "true";
        }
        return item;
      });
      const position = dataPath.substr(1);
      lodash.set(newData, position, arrayData);
      const isValid = validate.call(ctx, newData);
      if (!isValid) {
        // if validate.errors have new errors, add them to result
        for (const newError of validate.errors!) {
          let [includedInResult, includedInOriginalErrors] = [false, false];
          for (const resultError of result) {
            if (lodash.isEqual(newError, resultError)) {
              // error is included in result
              includedInResult = true;
              break;
            }
          }
          if (!includedInResult) {
            for (const eachOriginalError of originalErrors) {
              if (lodash.isEqual(newError, eachOriginalError)) {
                // error is included in originalErrors
                includedInOriginalErrors = true;
                break;
              }
            }
            if (!includedInOriginalErrors) {
              result.push(newError);
            }
          }
        }
      }
      continue;
    }
    result.push(originalError);
  }

  return result;
};

const shouldSkipError = (error: ErrorObject, cxt: SchemaValidateContext) => {
  const { parentSchema: parentSch, params, keyword } = error;
  const parentSchema = parentSch as Schema;
  // If schema has allof property, we can get schema in "_realschema", so is data
  const schema = Object.keys(error).includes("_realSchema")
    ? (error as any)._realSchema
    : error.schema;
  const data = Object.keys(error).includes("_realData") ? (error as any)._realData : error.data;

  if (schema?._skipError || parentSchema._skipError) {
    return true;
  }

  // If we're erroring on the added property refWithReadOnly simply ignore the error
  if (
    error.keyword === "additionalProperties" &&
    (params as any).additionalProperty === "refWithReadOnly"
  ) {
    return true;
  }

  // If a response has x-ms-mutability property and its missing the read we can skip this error
  if (
    cxt.isResponse &&
    ((keyword === "required" &&
      (parentSchema.properties?.[(params as any).missingProperty]?.[xmsMutability]?.indexOf(
        "read"
      ) === -1 ||
        // required check is ignored when x-ms-secret is true
        (parentSchema.properties?.[(params as any).missingProperty] as any)?.[xmsSecret] ===
          true)) ||
      (keyword === "type" && data === null && parentSchema[xmsMutability]?.indexOf("read") === -1))
  ) {
    return true;
  }

  // If a request is missing a required property that is readOnly we can skip this error
  if (
    !cxt.isResponse &&
    keyword === "required" &&
    (parentSchema.properties?.[(params as any).missingProperty]?.[xmsReadonlyRef] ||
      parentSchema.properties?.[(params as any).missingProperty]?.readOnly)
  ) {
    return true;
  }

  // If a response has property which x-ms-secret value is "true" in post we can skip this error
  if (
    cxt.isResponse &&
    (cxt as any)?.httpMethod === "post" &&
    // should skip error when x-ms-secret is "true"
    ((keyword === "x-ms-secret" && (parentSchema as any)?.[xmsSecret] === true) ||
      // should skip error when x-ms-secret is "true" and x-ms-mutability is "create" and "update"
      (keyword === "x-ms-mutability" &&
        (parentSchema as any)?.[xmsSecret] === true &&
        parentSchema[xmsMutability]?.indexOf("read") === -1))
  ) {
    return true;
  }

  // If payload has property with date-time parameter and its value is valid except missing "Z" in the end we can skip this error
  if (keyword === "format" && schema === "date-time" && typeof data === "string") {
    const reg = /^\d+-(0\d|1[0-2])-([0-2]\d|3[01])T([01]\d|2[0-3]):[0-5][0-9]:[0-5][0-9]/;
    // intercept time, example: 2008-09-22T14:01:54
    const time = data.slice(0, 19);
    if (reg.test(time)) {
      const dateZ = new Date(data + "Z").toUTCString();
      // validate hour
      const ifHoursAreSame = time.slice(11, 13) === dateZ.slice(17, 19);
      // validate day for leap year, example: 2008-02-29
      const ifDaysAreSame = time.slice(8, 10) === dateZ.slice(5, 7);
      return ifHoursAreSame && ifDaysAreSame;
    }
  }

  // If a response data has multipleOf property, and it divided by multipleOf value is an integer, we can skip this error
  if (keyword === "multipleOf" && typeof schema === "number" && typeof data === "number") {
    let [newSchema, newData] = [schema, data];
    while (newSchema < 1) {
      newSchema *= 10;
      newData *= 10;
    }
    const result = newData / newSchema;
    // should skip error when response data divided by multipleOf value is an integer
    return result === parseInt(String(result));
  }

  return false;
};

const errorKeywordsMapping: { [key: string]: TrafficValidationErrorCode } = {
  additionalProperties: "OBJECT_ADDITIONAL_PROPERTIES",
  required: "OBJECT_MISSING_REQUIRED_PROPERTY",
  format: "INVALID_FORMAT",
  type: "INVALID_TYPE",
  pattern: "PATTERN",
  minimum: "MINIMUM",
  maximum: "MAXIMUM",
  exclusiveMinimum: "MINIMUM_EXCLUSIVE",
  exclusiveMaximum: "MAXIMUM_EXCLUSIVE",
  minLength: "MIN_LENGTH",
  maxLength: "MAX_LENGTH",
  maxItems: "ARRAY_LENGTH_LONG",
  minItems: "ARRAY_LENGTH_SHORT",
  maxProperties: "OBJECT_PROPERTIES_MAXIMUM",
  minProperties: "OBJECT_PROPERTIES_MINIMUM",
  uniqueItems: "ARRAY_UNIQUE",
  additionalItems: "ARRAY_ADDITIONAL_ITEMS",
  anyOf: "ANY_OF_MISSING",
  dependencies: "OBJECT_DEPENDENCY_KEY",
  multiple: "MULTIPLE_OF",
  discriminatorMap: "DISCRIMINATOR_VALUE_NOT_FOUND",
  [xmsAzureResource]: "MISSING_RESOURCE_ID",
};
// Should be type "never" to ensure we've covered all the errors
// export type MissingErrorCode = Exclude<
//   ExtendedErrorCode,
//   | keyof typeof validateErrorMessages
//   | "PII_MISMATCH" // Used in openapi-validate
//   | "INTERNAL_ERROR" // Used in liveValidator
//   | "UNRESOLVABLE_REFERENCE"
//   | "NOT_PASSED" // If keyword mapping not found then we use this error
//   | "OPERATION_NOT_FOUND_IN_CACHE_WITH_PROVIDER" // Covered by liveValidator
//   | "OPERATION_NOT_FOUND_IN_CACHE_WITH_API"
//   | "OPERATION_NOT_FOUND_IN_CACHE_WITH_VERB"
//   | "OPERATION_NOT_FOUND_IN_CACHE"
//   | "MULTIPLE_OPERATIONS_FOUND"
//   | "INVALID_RESPONSE_HEADER"
//   | "INVALID_REQUEST_PARAMETER"
// >;

const transformParamsKeyword = new Set([
  "pattern",
  "additionalProperties",
  "type",
  "format",
  "multipleOf",
  "required",
  xmsAzureResource,
]);

const transformReverseParamsKeyword = new Set([
  "minLength",
  "maxLength",
  "maxItems",
  "minItems",
  "maxProperties",
  "minProperties",
  "minimum",
  "maximum",
  "exclusiveMinimum",
  "exclusiveMaximum",
  "discriminatorMap",
]);

interface MetaErr {
  code: string;
  message: string;
  severity: Severity;
  params?: any;
}
export const ajvErrorCodeToOavErrorCode = (
  error: ErrorObject,
  ctx: SchemaValidateContext
): MetaErr | undefined => {
  const { keyword, parentSchema: parentSch } = error;
  const parentSchema = parentSch as Schema | undefined;
  let { params, data } = error;
  let result: MetaErr | undefined = {
    code: "NOT_PASSED",
    message: error.message!,
    severity: Severity.Verbose,
  };

  // Workaround for incorrect ajv behavior.
  // See https://github.com/ajv-validator/ajv/blob/v6/lib/dot/custom.jst#L74
  if ((error as any)._realData !== undefined) {
    data = (error as any)._realData;
  }

  switch (keyword) {
    case "enum":
      const { allowedValues } = params as any;
      result =
        data === null && parentSchema?.nullable
          ? undefined
          : isEnumCaseMismatch(data, allowedValues)
          ? getOavErrorMeta("ENUM_CASE_MISMATCH", { data })
          : parentSchema?.[xmsEnum]?.modelAsString
          ? undefined
          : getOavErrorMeta("ENUM_MISMATCH", { data });
      params = [data, allowedValues];
      break;

    case "readOnly":
    case xmsMutability:
    case xmsSecret:
      const param = {
        key: getNameFromRef(parentSchema),
        value: Array.isArray(data) ? data.join(",") : JSON.stringify(data),
      };
      params = [param.key, null];
      result =
        keyword === xmsSecret
          ? getOavErrorMeta("SECRET_PROPERTY", param)
          : ctx.isResponse
          ? getOavErrorMeta("WRITEONLY_PROPERTY_NOT_ALLOWED_IN_RESPONSE", param)
          : getOavErrorMeta("READONLY_PROPERTY_NOT_ALLOWED_IN_REQUEST", param);
      break;

    case "oneOf":
      result =
        (params as any).passingSchemas === null
          ? getOavErrorMeta("ONE_OF_MISSING", {})
          : getOavErrorMeta("ONE_OF_MULTIPLE", {});
      params = [];
      break;

    case "type":
      (params as any).type = (params as any).type.replace(",null", "");
      data = schemaType(data);
      break;

    case "discriminatorMap":
      data = (params as any).discriminatorValue;
      break;

    case "maxLength":
    case "minLength":
    case "maxItems":
    case "minItems":
      data = data.length;
      break;

    case "maxProperties":
    case "minProperties":
      data = Object.keys(data).length;
      break;

    case "minimum":
    case "maximum":
    case "exclusiveMinimum":
    case "exclusiveMaximum":
      params = { limit: (params as any).limit };
      break;
  }

  const code = errorKeywordsMapping[keyword];
  if (code !== undefined) {
    result = getOavErrorMeta(code, { ...params, data });
  }

  if (transformParamsKeyword.has(keyword)) {
    params = Object.values(params);
    if (typeof data !== "object") {
      (params as any).push(data);
    }
  } else if (transformReverseParamsKeyword.has(keyword)) {
    params = Object.values(params);
    (params as any[]).unshift(data);
  }

  if (result !== undefined) {
    result.params = params;
  }
  return result;
};

const isEnumCaseMismatch = (data: string, enumList: Array<string | number>) => {
  if (typeof data !== "string") {
    return false;
  }
  data = data.toLowerCase();
  for (const val of enumList) {
    if (typeof val === "string" && val.toLowerCase() === data) {
      return true;
    }
  }
  return false;
};

const schemaType = (what: any): string => {
  const to = typeof what;

  if (to === "object") {
    if (what === null) {
      return "null";
    }
    if (Array.isArray(what)) {
      return "array";
    }
    return "object"; // typeof what === 'object' && what === Object(what) && !Array.isArray(what);
  }

  if (to === "number") {
    if (Number.isFinite(what)) {
      if (what % 1 === 0) {
        return "integer";
      } else {
        return "number";
      }
    }
    if (Number.isNaN(what)) {
      return "not-a-number";
    }
    return "unknown-number";
  }
  return to; // undefined, boolean, string, function
};
