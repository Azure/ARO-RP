import { JSONPath } from "jsonpath-plus";
import { inject, injectable } from "inversify";
import * as _ from "lodash";

import { FileLoaderOption } from "../swagger/fileLoader";
import { JsonLoaderOption, JsonLoader } from "../swagger/jsonLoader";
import { SwaggerLoader, SwaggerLoaderOption } from "../swagger/swaggerLoader";
import { getTransformContext, TransformContext } from "../transform/context";
import { SchemaValidator } from "../swaggerValidator/schemaValidator";
import { xmsPathsTransformer } from "../transform/xmsPathsTransformer";
import { resolveNestedDefinitionTransformer } from "../transform/resolveNestedDefinitionTransformer";
import { referenceFieldsTransformer } from "../transform/referenceFieldsTransformer";
import { discriminatorTransformer } from "../transform/discriminatorTransformer";
import { allOfTransformer } from "../transform/allOfTransformer";
import { noAdditionalPropertiesTransformer } from "../transform/noAdditionalPropertiesTransformer";
import { nullableTransformer } from "../transform/nullableTransformer";
import { pureObjectTransformer } from "../transform/pureObjectTransformer";
import { AjvSchemaValidator } from "../swaggerValidator/ajvSchemaValidator";
import { getJsonPatchDiff } from "../apiScenario/diffUtils";
import { BodyTransformer } from "../apiScenario/bodyTransformer";
import { ErrorCodes } from "../util/constants";
import { SeverityString } from "../util/severity";
import { inversifyGetInstance, TYPES } from "./../inversifyUtils";
import { setDefaultOpts } from "./../swagger/loader";
import { traverseSwaggerAsync } from "./../transform/traverseSwagger";
import { applyGlobalTransformers, applySpecTransformers } from "./../transform/transformer";
import {
  LowerHttpMethods,
  Path,
  Operation,
  SwaggerSpec,
  SwaggerExample,
  BodyParameter,
} from "./../swagger/swaggerTypes";

export interface ExampleQualityValidatorOption
  extends FileLoaderOption,
    JsonLoaderOption,
    SwaggerLoaderOption {
  swaggerFilePaths?: string[];
  exampleMapping?: Map<string, string>;
}

interface exampleValidationContext {
  exampleName: string;
  exampleFilePath: string;
  bodyTransform?: BodyTransformer;
}

type ExampleValidationFunc = (
  example: SwaggerExample,
  operation: Operation,
  jsonLoader: JsonLoader,
  exampleValidationContext: exampleValidationContext
) => Promise<any>;

interface ExampleValidationRule {
  id: string;
  severity: SeverityString;
  code: string;
  jsonPath: string;
  message: string;
  exampleName: string;
  exampleFilePath: string;
  detail?: any;
}

// ARM RPC guide: https://github.com/Azure/azure-resource-manager-rpc/blob/master/v1.0/Addendum.md#provisioningstate-property
// https://docs.microsoft.com/en-us/javascript/api/@azure/arm-databricks/provisioningstate?view=azure-node-latest
const incorrectProvisioningState: ExampleValidationFunc = async (
  example: SwaggerExample,
  _operation: Operation,
  _jsonLoader: JsonLoader,
  exampleValidationContext: exampleValidationContext
) => {
  const ret: ExampleValidationRule[] = [];
  if (example?.responses["200"] !== undefined) {
    const provisioningStatePath = "$.responses.200..[?(@.provisioningState)]";
    const provisioningStates = JSONPath({
      path: provisioningStatePath,
      json: example,
      resultType: "all",
    });
    const terminalStatues = ["succeeded", "failed", "canceled", "ready", "created", "deleted"];
    for (const it of provisioningStates) {
      if (!terminalStatues.includes(it.value.provisioningState.toLowerCase())) {
        ret.push({
          id: ErrorCodes.IncorrectProvisioningState.id,
          severity: "Error",
          code: ErrorCodes.IncorrectProvisioningState.name,
          jsonPath: `${it.pointer}/provisioningState`,
          message:
            "The resource's provisioning state should be terminal status in http 200 response.",
          exampleName: exampleValidationContext.exampleName,
          exampleFilePath: exampleValidationContext.exampleFilePath,
        });
      }
    }
  }
  return ret;
};

const roundtripInconsistentProperty: ExampleValidationFunc = async (
  example: SwaggerExample,
  operation: Operation,
  jsonLoader: JsonLoader,
  exampleValidationContext: exampleValidationContext
) => {
  const ret: ExampleValidationRule[] = [];
  if (operation._method === "put") {
    const reqSchema = operation.parameters?.filter((it) => it?.in === "body")[0] as BodyParameter;
    for (const statusCode of Object.keys(example.responses)) {
      if (operation.responses[statusCode] === undefined) {
        continue;
      }
      const respSchema = operation.responses[statusCode].schema;
      if (reqSchema && reqSchema.schema?.$ref === respSchema?.$ref) {
        const schema = jsonLoader.resolveRefObj(respSchema);
        const responseObj = await exampleValidationContext.bodyTransform?.resourceToRequest(
          example.responses[statusCode].body,
          schema!
        );
        const requestObj = example.parameters.parameters;
        if (!!requestObj && !!responseObj) {
          const delta = getJsonPatchDiff(requestObj, responseObj, {
            includeOldValue: true,
            minimizeDiff: false,
          });
          // filter replace operation.
          delta
            .filter((it) => (it as any).replace !== undefined)
            .map((it) => {
              (it as any).replace = `/${statusCode}/body${(it as any).replace}`;
              return it;
            })
            .forEach((it) =>
              ret.push({
                id: ErrorCodes.RoundtripInconsistentProperty.id,
                severity: "Error",
                code: ErrorCodes.RoundtripInconsistentProperty.name,
                jsonPath: (it as any).replace,
                detail: JSON.stringify(it),
                message: `The property's value in the response is different from what was set in the request. Path: ${
                  (it as any).replace
                }. Request: ${(it as any).oldValue}. Response: ${(it as any).value}`,
                exampleName: exampleValidationContext.exampleName,
                exampleFilePath: exampleValidationContext.exampleFilePath,
              })
            );
        }
      }
    }
  }
  return ret;
};

/*
const recommendUsingBooleanType: ExampleValidationFunc = async (
  example: SwaggerExample,
  _operation: Operation,
  _jsonLoader: JsonLoader,
  exampleValidationContext: exampleValidationContext
) => {
  const ret: ExampleValidationRule[] = [];
  const allElementPath = "$..*";
  const allElements = JSONPath({
    path: allElementPath,
    json: example,
    resultType: "all",
  });
  const stringBooleans = ["false", "true"];
  for (const it of allElements) {
    if (typeof it.value === "string" && stringBooleans.includes(it.value.toLowerCase())) {
      ret.push({
        id: ErrorCodes.RecommendUsingBooleanType.id,
        severity: Severity.Warning,
        code: ErrorCodes.RecommendUsingBooleanType.name,
        path: `${it.pointer}`,
        message:
          "If the property only return two string value 'true' or 'false', recommend using boolean type.",
        exampleName: exampleValidationContext.exampleName,
        exampleFilePath: exampleValidationContext.exampleFilePath,
      });
    }
  }
  //TODO: find schema by pointer. and check whether it type is string.
  return ret;
};
*/

@injectable()
export class ExampleQualityValidator {
  private swaggerSpecs: SwaggerSpec[];
  private initialized: boolean = false;
  private transformContext: TransformContext;
  private schemaValidator: SchemaValidator;
  private validationFuncs: ExampleValidationFunc[];
  private operationMapping: { [operationId: string]: Operation };
  // eslint-disable-next-line @typescript-eslint/explicit-member-accessibility
  constructor(
    @inject(TYPES.opts) private opts: ExampleQualityValidatorOption,
    public jsonLoader: JsonLoader,
    private swaggerLoader: SwaggerLoader,
    private bodyTransformer: BodyTransformer
  ) {
    this.swaggerSpecs = [];
    this.validationFuncs = [incorrectProvisioningState, roundtripInconsistentProperty];
    this.schemaValidator = new AjvSchemaValidator(this.jsonLoader);
    this.transformContext = getTransformContext(this.jsonLoader, this.schemaValidator, [
      xmsPathsTransformer,
      resolveNestedDefinitionTransformer,
      referenceFieldsTransformer,

      discriminatorTransformer,
      allOfTransformer,
      noAdditionalPropertiesTransformer,
      nullableTransformer,
      pureObjectTransformer,
    ]);
    this.operationMapping = {};
  }

  public static create(opts: ExampleQualityValidatorOption) {
    setDefaultOpts(opts, {
      eraseXmsExamples: false,
      eraseDescription: false,
    });
    return inversifyGetInstance(ExampleQualityValidator, opts);
  }

  public async initialize() {
    if (this.initialized) {
      throw new Error("Already initialized");
    }
    for (const swaggerFilePath of this.opts.swaggerFilePaths ?? []) {
      const swaggerSpec = await this.swaggerLoader.load(swaggerFilePath);
      this.swaggerSpecs.push(swaggerSpec);
      applySpecTransformers(swaggerSpec, this.transformContext);
      await traverseSwaggerAsync(swaggerSpec, {
        onOperation: async (operation) => {
          this.operationMapping[operation.operationId!] = operation;
        },
      });
    }
    applyGlobalTransformers(this.transformContext);
  }

  public async validateExternalExamples(
    examples: Array<{
      exampleFilePath: string;
      example: SwaggerExample | undefined;
      operationId: string;
    }>
  ): Promise<ExampleValidationRule[]> {
    if (!this.initialized) {
      await this.initialize();
    }
    let res: any[] = [];
    for (const example of examples) {
      const operationId = example.operationId;
      const operation = this.operationMapping[operationId];
      if (operation === undefined) {
        throw new Error(`Cannot find operation for ${example.exampleFilePath}`);
      }
      const ctx: exampleValidationContext = {
        exampleName: example.exampleFilePath,
        exampleFilePath: example.exampleFilePath,
        bodyTransform: this.bodyTransformer,
      };
      const exampleObj =
        example.example ??
        ((await this.jsonLoader.load(example.exampleFilePath)) as SwaggerExample);
      for (const func of this.validationFuncs) {
        const ruleRes = await func(exampleObj, operation, this.jsonLoader, ctx);
        res = res.concat(ruleRes);
      }
    }
    return res;
  }

  /**
   * Validate swagger example quality.
   * @param filteredOperationIds If filtered is undefined, validate the whole x-ms-examples in swagger.
   * @returns
   */
  public async validateSwaggerExamples(
    filteredOperationIds: string[] | undefined = undefined
  ): Promise<ExampleValidationRule[]> {
    if (!this.initialized) {
      await this.initialize();
    }
    let res: ExampleValidationRule[] = [];
    const onOperation = async (
      operation: Operation,
      _httpPath: Path,
      _method: LowerHttpMethods
    ) => {
      if (
        filteredOperationIds === undefined ||
        filteredOperationIds.includes(operation.operationId!)
      ) {
        const xMsExamples = operation["x-ms-examples"] ?? {};
        for (const exampleName of Object.keys(xMsExamples)) {
          const example = xMsExamples[exampleName];
          if (typeof example.$ref !== "string") {
            throw new Error(`Example doesn't use $ref: ${exampleName}`);
          }
          const exampleObj: SwaggerExample = (await this.jsonLoader.resolveFile(
            example.$ref
          )) as SwaggerExample;
          const ctx: exampleValidationContext = {
            exampleName: exampleName,
            exampleFilePath: this.jsonLoader.getRealPath(example.$ref),
            bodyTransform: this.bodyTransformer,
          };
          for (const func of this.validationFuncs) {
            const ruleRes = await func(exampleObj, operation, this.jsonLoader, ctx);
            res = res.concat(ruleRes);
          }
        }
      }
    };
    for (const spec of this.swaggerSpecs) {
      await traverseSwaggerAsync(spec, { onOperation: onOperation });
    }
    return res;
  }
}
