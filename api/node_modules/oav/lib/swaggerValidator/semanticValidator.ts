import {
  FilePosition,
  getInfo,
  getRootObjectInfo,
  ParseError,
} from "@azure-tools/openapi-tools-common";
import { inject, injectable } from "inversify";
import swaggerSchemaDoc from "@autorest/schemas/swagger-extensions.json";
import swaggerExampleSchemaDoc from "@autorest/schemas/example-schema.json";
import jsonPointer from "json-pointer";
import { inversifyGetContainer, inversifyGetInstance, TYPES } from "../inversifyUtils";
import { $id, JsonLoader, JsonLoaderRefError } from "../swagger/jsonLoader";
import { SwaggerLoaderOption } from "../swagger/swaggerLoader";
import { Parameter, refSelfSymbol, Schema, SwaggerSpec } from "../swagger/swaggerTypes";
import { BaseValidationError } from "../util/baseValidationError";
import {
  getOavErrorMeta,
  SemanticValidationErrorCode,
  semanticValidationErrors,
} from "../util/errorDefinitions";
import { ValidationResultSource } from "../util/validationResultSource";
import { LiveValidatorLoader } from "../liveValidation/liveValidatorLoader";
import { getTransformContext, TransformContext } from "../transform/context";
import { referenceFieldsTransformer } from "../transform/referenceFieldsTransformer";
import { resolveNestedDefinitionTransformer } from "../transform/resolveNestedDefinitionTransformer";
import { xmsPathsTransformer } from "../transform/xmsPathsTransformer";
import { applyGlobalTransformers, applySpecTransformers } from "../transform/transformer";
import { traverseSwagger, traverseSwaggerAsync } from "../transform/traverseSwagger";
import { FileLoader } from "../swagger/fileLoader";
import { getFilePositionFromJsonPath, jsonPathToPointer } from "../util/jsonUtils";
import { pathRegexTransformer } from "../transform/pathRegexTransformer";
import { discriminatorTransformer } from "../transform/discriminatorTransformer";
import { allOfTransformer } from "../transform/allOfTransformer";
import { noAdditionalPropertiesTransformer } from "../transform/noAdditionalPropertiesTransformer";
import { nullableTransformer } from "../transform/nullableTransformer";
import { pureObjectTransformer } from "../transform/pureObjectTransformer";
import { isSuppressedInPath, SuppressionLoader } from "../swagger/suppressionLoader";
import { xmsDiscriminatorValue } from "../util/constants";
import {
  SchemaValidateFunction,
  SchemaValidateIssue,
  SchemaValidator,
  SchemaValidatorOption,
} from "./schemaValidator";
import swagger2SchemaDoc from "./swagger-2.0.json";

export interface SemanticErrorDetail {
  inner?: any; // Compatible with old NodeError. Always undefined.
  message: string;
  code: SemanticValidationErrorCode;
  position?: FilePosition;
  url?: string;
  jsonPath?: string;
}

export interface SemanticValidationError extends BaseValidationError<SemanticErrorDetail> {
  source?: ValidationResultSource;
  path?: string;
  readonly inner?: any;
  readonly "json-path"?: string;
}

export interface SemanticValidationOption extends SwaggerLoaderOption, SchemaValidatorOption {}

const loadSuppression = [];
for (const errorCode of Object.keys(semanticValidationErrors)) {
  const meta = semanticValidationErrors[errorCode as SemanticValidationErrorCode];
  if ("id" in meta) {
    loadSuppression.push(meta.id);
  }
}

// Set isArmCall flag to true so that the ARM rules schema will be applied to swaggers too
const defaultOpts: SemanticValidationOption = {
  eraseDescription: false,
  eraseXmsExamples: false,
  useJsonParser: true,
  loadSuppression,
  isArmCall: true,
};

@injectable()
export class SwaggerSemanticValidator {
  private validateSwaggerSch!: SchemaValidateFunction;

  public constructor(
    @inject(TYPES.opts) _opts: SemanticValidationOption,
    private jsonLoader: JsonLoader,
    private fileLoader: FileLoader,
    private suppressionLoader: SuppressionLoader,
    private liveValidatorLoader: LiveValidatorLoader,
    @inject(TYPES.schemaValidator) private schemaValidator: SchemaValidator
  ) {}

  public async initialize() {
    this.fileLoader.preloadExtraFile(
      "https://raw.githubusercontent.com/Azure/autorest/master/schema/example-schema.json",
      JSON.stringify(swaggerExampleSchemaDoc)
    );
    this.fileLoader.preloadExtraFile(
      "http://json.schemastore.org/swagger-2.0", //DevSkim: ignore DS137138
      JSON.stringify(swagger2SchemaDoc)
    );
    const properties = swaggerSchemaDoc.properties as any;
    properties[$id] = {};
    properties._filePath = {};

    this.validateSwaggerSch = await this.schemaValidator.compileAsync(swaggerSchemaDoc as Schema);
  }

  public async validateSwaggerSpec(swaggerFilePath: string) {
    const errors = await this.validateSwaggerSpecPipeline(swaggerFilePath);

    return errors;
  }

  private async validateSwaggerSpecPipeline(swaggerFilePath: string) {
    const errors: SemanticErrorDetail[] = [];

    try {
      const swagger = await this.loadSwagger(swaggerFilePath, errors);
      if (swagger === undefined || errors.length > 0) {
        return errors;
      }

      await this.suppressionLoader.load(swagger);

      // validate x-ms-* extensions
      await this.validateSwaggerSchema(swagger, errors);
      if (errors.length > 0) {
        return errors;
      }

      // compile swagger schema
      const transformCtx = await this.validateCompile(swagger, errors);

      await this.validateDiscriminator(transformCtx, errors);

      await this.validateDefaultValue(transformCtx, swagger._filePath, errors);

      await this.validateSchemaRequiredProperties(transformCtx, swagger._filePath, errors);

      await this.validateOperation(swagger, errors);
    } catch (e) {
      const errInfo = getOavErrorMeta("INTERNAL_ERROR", { message: `${e.message}\n${e.stack}` });
      errors.unshift({
        code: errInfo.code,
        message: errInfo.message,
        url: swaggerFilePath,
      });
    }

    return errors;
  }

  private async loadSwagger(swaggerFilePath: string, errors: SemanticErrorDetail[]) {
    try {
      const swagger = (await this.jsonLoader.load(swaggerFilePath)) as unknown as SwaggerSpec;
      swagger._filePath = swaggerFilePath;
      return swagger;
    } catch (e) {
      if (typeof e.kind === "string") {
        const ex = e as ParseError;
        const errInfo = getOavErrorMeta("JSON_PARSING_ERROR", { details: ex.code });
        errors.push({
          code: errInfo.code,
          message: errInfo.message,
          position: ex.position,
          url: ex.url,
        });
      } else if (e instanceof JsonLoaderRefError) {
        const errInfo = getOavErrorMeta("UNRESOLVABLE_REFERENCE", { ref: e.ref });
        errors.push({
          code: errInfo.code,
          message: errInfo.message,
          position: e.position,
          url: e.url,
        });
      } else {
        throw e;
      }
      return;
    }
  }

  private async validateSwaggerSchema(swagger: SwaggerSpec, errors: SemanticErrorDetail[]) {
    const result = this.validateSwaggerSch({}, swagger);
    const rootInfo = getRootObjectInfo(getInfo(swagger)!);

    this.addErrorsFromSchemaValidation(result, rootInfo.url, swagger, errors);
  }

  private addErrorsFromSchemaValidation(
    result: SchemaValidateIssue[],
    url: string,
    rootObj: any,
    errors: SemanticErrorDetail[]
  ) {
    const existedJsonPaths: string[] = [];
    for (const err of result) {
      // ignore below schema errors
      if (
        err.code === "NOT_PASSED" &&
        (err.message.includes('should match "else" schema') ||
          err.message.includes('should match "then" schema'))
      ) {
        continue;
      } else if (err.code === "NOT_PASSED" && err.schemaPath === "#/additionalProperties/not") {
        err.message = "path DOES NOT start with /";
      }
      err.jsonPathsInPayload = err.jsonPathsInPayload.filter((jsonPath) => {
        let node;
        /*eslint no-constant-condition: ["error", { "checkLoops": false }]*/
        while (true) {
          try {
            node = jsonPointer.get(rootObj, jsonPathToPointer(jsonPath));
            const isSuppressed = isSuppressedInPath(node, err.code, err.message);
            if (!isSuppressed) {
              existedJsonPaths.push(jsonPath);
            }
            return !isSuppressed;
          } catch (e) {
            let isContinue = false;
            // the jsonPathsInPayload will include non-existed path, so it needs to walk back to
            // exclude the unexisted path
            if (e.message.includes("Invalid reference token:")) {
              const token = e.message.substring("Invalid reference token:".length + 1);
              const index = jsonPath.lastIndexOf(token);
              if (index > 0) {
                jsonPath = jsonPath.substring(0, index);
                if (jsonPath.endsWith(".") || jsonPath.endsWith("/")) {
                  jsonPath = jsonPath.substring(0, jsonPath.length - 1);
                }
                isContinue = true;
              }
            }
            // if it's not the case of containing unexisted path, then throw this error
            if (isContinue === false) {
              throw e;
            }
          }
        }
      });

      if (err.jsonPathsInPayload.length === 0) {
        continue;
      }

      const jsonPath = existedJsonPaths[0];
      const position = getFilePositionFromJsonPath(rootObj, jsonPath);

      errors.push({
        code: err.code,
        message: err.message,
        url,
        position,
        jsonPath,
      });
    }
  }

  private addErrorsFromErrorCode(
    errors: SemanticErrorDetail[],
    url: string,
    meta: ReturnType<typeof getOavErrorMeta>,
    obj: any,
    jsonPath?: string
  ) {
    if (
      isSuppressedInPath(obj, meta.id!, meta.message) ||
      isSuppressedInPath(obj, meta.code, meta.message)
    ) {
      return;
    }

    const info = getInfo(obj);
    errors.push({
      code: meta.code as SemanticValidationErrorCode,
      message: meta.message,
      url,
      position: info?.position,
      jsonPath: jsonPath ?? obj?.[refSelfSymbol],
    });
  }

  private async validateCompile(spec: SwaggerSpec, errors: SemanticErrorDetail[]) {
    const transformCtx = getTransformContext(this.jsonLoader, this.schemaValidator, [
      xmsPathsTransformer,
      resolveNestedDefinitionTransformer,
      referenceFieldsTransformer,
      pathRegexTransformer,

      discriminatorTransformer,
      allOfTransformer,
      noAdditionalPropertiesTransformer,
      nullableTransformer,
      pureObjectTransformer,
    ]);
    applySpecTransformers(spec, transformCtx);
    applyGlobalTransformers(transformCtx);

    await traverseSwaggerAsync(spec, {
      onOperation: async (operation) => {
        try {
          await this.liveValidatorLoader.getRequestValidator(operation);
        } catch (e) {
          const info = getInfo(operation);
          errors.push({
            code: "INTERNAL_ERROR",
            message: `Failed to compile validator on operation\n${operation.operationId} ${operation._method}\n${e.message}\n${e.stack}`,
            url: spec._filePath,
            position: info?.position,
          });
        }
      },
      onResponse: async (response, operation, _, statusCode) => {
        try {
          await this.liveValidatorLoader.getResponseValidator(response);
        } catch (e) {
          const info = getInfo(operation);
          errors.push({
            code: "INTERNAL_ERROR",
            message: `Failed to compile validator on operation response\n${statusCode} ${operation.operationId} ${operation._method}\n${e.message}\n${e.stack}`,
            url: spec._filePath,
            position: info?.position,
          });
        }
      },
    });

    return transformCtx;
  }

  private async validateDiscriminator(
    transformCtx: TransformContext,
    errors: SemanticErrorDetail[]
  ) {
    const { objSchemas, jsonLoader } = transformCtx;
    for (const sch of objSchemas) {
      const d = sch.discriminator;
      if (d === undefined) {
        continue;
      }

      const info = getInfo(sch);
      const rootInfo = getRootObjectInfo(info!);

      if (sch.required?.find((x) => x === d) === undefined) {
        const meta = getOavErrorMeta("DISCRIMINATOR_NOT_REQUIRED", { property: d });
        this.addErrorsFromErrorCode(errors, rootInfo.url, meta, sch);
      }

      if (sch.properties?.[d] === undefined) {
        const meta = getOavErrorMeta("OBJECT_MISSING_REQUIRED_PROPERTY_DEFINITION", {
          property: d,
        });
        this.addErrorsFromErrorCode(errors, rootInfo.url, meta, sch);
      } else {
        const discriminatorProp = jsonLoader.resolveRefObj(sch.properties[d]);
        if (discriminatorProp.type !== "string") {
          const meta = getOavErrorMeta("INVALID_DISCRIMINATOR_TYPE", {
            property: d,
          });
          this.addErrorsFromErrorCode(errors, rootInfo.url, meta, sch);
          continue;
        }

        if (discriminatorProp.enum !== undefined && sch.discriminatorMap !== undefined) {
          for (const childSchRef of Object.values(sch.discriminatorMap)) {
            if (childSchRef === null) {
              continue;
            }

            const childSch = jsonLoader.resolveRefObj(childSchRef);
            const discriminatorValue = childSch[xmsDiscriminatorValue];
            if (
              discriminatorValue !== undefined &&
              !discriminatorProp.enum.includes(discriminatorValue)
            ) {
              const meta = getOavErrorMeta("INVALID_XMS_DISCRIMINATOR_VALUE", {
                value: discriminatorValue,
              });
              const url = getRootObjectInfo(getInfo(childSch)!).url;
              this.addErrorsFromErrorCode(errors, url, meta, childSch);
            }
          }
        }
      }
    }

    for (const sch of objSchemas) {
      if (!sch._missingDiscriminator) {
        continue;
      }

      const info = getInfo(sch);
      const rootInfo = getRootObjectInfo(info!);
      const meta = getOavErrorMeta("DISCRIMINATOR_PROPERTY_NOT_FOUND", {
        value: sch[xmsDiscriminatorValue],
      });
      this.addErrorsFromErrorCode(errors, rootInfo.url, meta, sch);
    }
  }

  private async validateDefaultValue(
    transformCtx: TransformContext,
    url: string,
    errors: SemanticErrorDetail[]
  ) {
    for (const sch of transformCtx.objSchemas) {
      if (sch.default === undefined) {
        continue;
      }
      const validate = await this.schemaValidator.compileAsync({
        properties: {
          default: sch,
        },
      });
      const result = validate({}, sch);
      this.addErrorsFromSchemaValidation(result, url, sch, errors);
    }
  }

  private async validateSchemaRequiredProperties(
    transformCtx: TransformContext,
    url: string,
    errors: SemanticErrorDetail[]
  ) {
    for (const sch of transformCtx.objSchemas) {
      if (sch.required === undefined) {
        continue;
      }
      for (const name of sch.required) {
        if (sch.properties?.[name] !== undefined) {
          continue;
        }
        const meta = getOavErrorMeta("OBJECT_MISSING_REQUIRED_PROPERTY_DEFINITION", {
          property: name,
        });
        this.addErrorsFromErrorCode(errors, url, meta, sch);
      }
    }

    for (const sch of transformCtx.arrSchemas) {
      if (sch.items !== undefined) {
        continue;
      }
      const meta = getOavErrorMeta("OBJECT_MISSING_REQUIRED_PROPERTY_SCHEMA", {
        property: "items",
      });
      this.addErrorsFromErrorCode(errors, url, meta, sch);
    }
  }

  private async validateOperation(spec: SwaggerSpec, errors: SemanticErrorDetail[]) {
    const visitedOperationId = new Set<string>();
    const visitedPathTemplate = new Set<string>();
    const url = spec._filePath;
    const pathArgs = new Set<string>();
    let pathParams: Parameter[] | undefined;

    traverseSwagger(spec, {
      onPath: (path) => {
        pathArgs.clear();
        pathParams = path.parameters;
        const pathTemplate = path._pathTemplate;
        let normalizedPath = pathTemplate;

        const argMatches = normalizedPath.match(/\{.*?\}/g);
        let idx = 0;
        for (const arg of argMatches ?? []) {
          if (arg === "{}") {
            const meta = getOavErrorMeta("EMPTY_PATH_PARAMETER_DECLARATION", { pathTemplate });
            this.addErrorsFromErrorCode(errors, url, meta, path);
          } else {
            normalizedPath = normalizedPath.replace(arg, `arg${idx}`);
            ++idx;
            pathArgs.add(arg.substr(1, arg.length - 2));
          }
        }

        if (visitedPathTemplate.has(normalizedPath)) {
          const meta = getOavErrorMeta("EQUIVALENT_PATH", { pathTemplate });
          this.addErrorsFromErrorCode(errors, url, meta, path);
        }
        visitedPathTemplate.add(normalizedPath);
      },
      onOperation: (operation) => {
        let bodyParam: Parameter | undefined;
        const requiredPathArgs = new Set(pathArgs);
        const visitedParamName = new Set<string>();
        const { operationId, parameters, consumes } = operation;
        const mergedParameters = [...(parameters ?? []), ...(pathParams ?? [])];

        if (operationId !== undefined) {
          if (visitedOperationId.has(operationId)) {
            const meta = getOavErrorMeta("DUPLICATE_OPERATIONID", { operationId });
            this.addErrorsFromErrorCode(errors, url, meta, operation);
          } else {
            visitedOperationId.add(operationId);
          }
        }

        for (const p of mergedParameters) {
          const param = this.jsonLoader.resolveRefObj(p);
          const { name } = param;

          if (visitedParamName.has(name)) {
            const meta = getOavErrorMeta("DUPLICATE_PARAMETER", { name });
            this.addErrorsFromErrorCode(errors, url, meta, operation);
          }
          visitedParamName.add(name);

          if (param.in === "body" || param.in === "formData") {
            if (bodyParam !== undefined) {
              const meta = getOavErrorMeta(
                param.in === bodyParam.in
                  ? "MULTIPLE_BODY_PARAMETERS"
                  : "INVALID_PARAMETER_COMBINATION",
                {}
              );
              if (
                !(
                  meta.code === "MULTIPLE_BODY_PARAMETERS" &&
                  param.in === "formData" &&
                  consumes !== undefined &&
                  // Must support both "multipart/form-data" and "application/x-www-form-urlencoded" per spec:
                  // https://swagger.io/docs/specification/v2_0/describing-parameters/#form-parameters
                  //
                  // Must support "multipart/mixed" for storage/data-plane/Microsoft.BlobStorage
                  (consumes.includes("multipart/form-data") ||
                    consumes.includes("multipart/mixed") ||
                    consumes.includes("application/x-www-form-urlencoded"))
                )
              ) {
                this.addErrorsFromErrorCode(errors, url, meta, operation);
              }
            }
            bodyParam = param;
          }

          if (param.in === "path") {
            if (!requiredPathArgs.has(name)) {
              const meta = getOavErrorMeta("MISSING_PATH_PARAMETER_DECLARATION", { name });
              this.addErrorsFromErrorCode(errors, url, meta, operation);
            }
            requiredPathArgs.delete(name);
          }
        }

        for (const name of requiredPathArgs) {
          const meta = getOavErrorMeta("MISSING_PATH_PARAMETER_DEFINITION", { name });
          this.addErrorsFromErrorCode(errors, url, meta, operation);
        }
      },
    });
  }
}

// Compatible wrapper for old SemanticValidator
export class SemanticValidator {
  public validator: SwaggerSemanticValidator;
  public specValidationResult: {
    validateSpec?: {
      isValid?: boolean;
      error?: unknown;
      warning?: unknown;
      result?: unknown;
      errors?: SemanticErrorDetail[];
      warnings?: unknown;
    };
    resolveSpec?: undefined;
    validityStatus: boolean;
    operations: any;
    initialize?: unknown;
  } = { validityStatus: true, operations: {} };

  public constructor(public specPath: string, specInJson?: any) {
    const container = inversifyGetContainer();
    this.validator = inversifyGetInstance(SwaggerSemanticValidator, {
      ...defaultOpts,
      container,
    });
    if (specInJson) {
      const fileLoader = container.get(FileLoader);
      fileLoader.preloadExtraFile(specPath, JSON.stringify(specInJson));
    }
  }

  public async initialize() {
    await this.validator.initialize();
    // API compatible
    return null as any;
  }

  public async validateSpec() {
    const errors = await this.validator.validateSwaggerSpec(this.specPath);
    this.specValidationResult.validateSpec = {
      isValid: errors.length === 0,
      errors,
    };
    this.specValidationResult.validityStatus = errors.length === 0;
    return { errors, warnings: [] };
  }
}
