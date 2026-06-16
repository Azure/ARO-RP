import { ParsedUrlQuery } from "querystring";
import { inject, injectable } from "inversify";
import { FilePosition, getInfo, ParseError, StringMap } from "@azure-tools/openapi-tools-common";
import * as openapiToolsCommon from "@azure-tools/openapi-tools-common";
import jsonPointer from "json-pointer";
import { parseInt } from "lodash";
import * as C from "../util/constants";
import { inversifyGetContainer, inversifyGetInstance, TYPES } from "../inversifyUtils";
import { LiveValidatorLoader } from "../liveValidation/liveValidatorLoader";
import { JsonLoader, JsonLoaderRefError } from "../swagger/jsonLoader";
import { isSuppressedInPath, SuppressionLoader } from "../swagger/suppressionLoader";
import { SwaggerLoaderOption } from "../swagger/swaggerLoader";
import {
  BodyParameter,
  LowerHttpMethods,
  Operation,
  Parameter,
  Path,
  refSelfSymbol,
  SwaggerExample,
  SwaggerSpec,
} from "../swagger/swaggerTypes";
import {
  ErrorCodeConstants,
  getOavErrorMeta,
  ModelValidationErrorCode,
  modelValidationErrors,
} from "../util/errorDefinitions";
import { traverseSwaggerAsync } from "../transform/traverseSwagger";
import { xmsPathsTransformer } from "../transform/xmsPathsTransformer";
import { resolveNestedDefinitionTransformer } from "../transform/resolveNestedDefinitionTransformer";
import { referenceFieldsTransformer } from "../transform/referenceFieldsTransformer";
import { pathRegexTransformer } from "../transform/pathRegexTransformer";
import { discriminatorTransformer } from "../transform/discriminatorTransformer";
import { allOfTransformer } from "../transform/allOfTransformer";
import { noAdditionalPropertiesTransformer } from "../transform/noAdditionalPropertiesTransformer";
import { nullableTransformer } from "../transform/nullableTransformer";
import { pureObjectTransformer } from "../transform/pureObjectTransformer";
import { getTransformContext } from "../transform/context";
import { applyGlobalTransformers, applySpecTransformers } from "../transform/transformer";
import {
  transformBodyValue,
  transformLiveHeader,
  transformMapValue,
} from "../liveValidation/operationValidator";
import { log } from "../util/logging";
import { getFilePositionFromJsonPath } from "../util/jsonUtils";
import { checkAndResolveGithubUrl, getProviderFromSpecPath } from "../util/utils";
import { Severity } from "../util/severity";
import { ValidationResultSource } from "../util/validationResultSource";
import { SchemaValidateIssue, SchemaValidator, SchemaValidatorOption } from "./schemaValidator";

@injectable()
export class SwaggerExampleValidator {
  private specPath: string;
  private swagger: SwaggerSpec;
  public errors: SwaggerExampleErrorDetail[] = [];

  public constructor(
    @inject(TYPES.opts) _opts: ExampleValidationOption,
    private jsonLoader: JsonLoader,
    private suppressionLoader: SuppressionLoader,
    private liveValidatorLoader: LiveValidatorLoader,
    @inject(TYPES.schemaValidator) private schemaValidator: SchemaValidator
  ) {}

  public async validateOperations(operationIds?: string): Promise<void> {
    try {
      if (
        operationIds !== null &&
        operationIds !== undefined &&
        typeof operationIds.valueOf() !== "string"
      ) {
        throw new Error(
          `please pass in operationIds parameter correctly. It must be a string separated by comma if containing multiple operationIds.`
        );
      }
      let operationIdArray: string[] = [];
      if (operationIds !== undefined) {
        operationIdArray = operationIds.trim().split(",");
      }

      // Parse the swagger resolving the ref
      await this.loadSwagger(this.specPath, false);
      if (this.swagger === undefined || this.errors.length > 0) {
        return;
      }
      await this.suppressionLoader.load(this.swagger);
      await this.transformSwagger(this.swagger);
      await this.liveValidatorLoader.buildAjvValidator(this.swagger);
      await traverseSwaggerAsync(this.swagger, {
        onOperation: async (_operation: Operation, _path: Path, _method: LowerHttpMethods) => {
          if (
            operationIdArray.length === 0 ||
            (operationIdArray.length > 0 && operationIdArray.includes(_operation.operationId!))
          ) {
            if (!_operation["x-ms-examples"]) {
              const meta = getOavErrorMeta(ErrorCodeConstants.XMS_EXAMPLE_NOTFOUND_ERROR as any, {
                operationId: _operation.operationId,
              });
              this.addErrorsFromErrorCode(_operation.operationId!, undefined, meta, _operation);
              return;
            }
            await this.validateOperation(_operation);
          }
        },
      });
    } catch (e) {
      log.error(`validateOperations - ErrorMessage:${e?.message}.ErrorStack:${e?.stack}`);
      throw e;
    }
  }

  public async initialize(specPath: string): Promise<void> {
    if (
      specPath === null ||
      specPath === undefined ||
      typeof specPath.valueOf() !== "string" ||
      !specPath.trim().length
    ) {
      throw new Error(
        "specPath is a required parameter of type string and it cannot be an empty string."
      );
    }
    this.specPath = checkAndResolveGithubUrl(specPath);
  }

  private async validateOperation(operation: Operation): Promise<void> {
    // load example content
    let exampleContent;
    let exampleFileUrl;
    for (const scenario of Object.keys(operation["x-ms-examples"]!)) {
      const mockUrl = operation["x-ms-examples"]![scenario].$ref!;
      if (mockUrl === undefined) {
        throw new Error(
          `$ref is undefined in x-ms-examples defintion for operation ${operation.operationId}.`
        );
      }
      exampleContent = this.jsonLoader.resolveMockedFile(mockUrl) as SwaggerExample;
      exampleFileUrl = this.jsonLoader.getRealPath(mockUrl);
      this.validateRequest(operation, exampleContent, exampleFileUrl);
      this.validateResponse(operation, exampleContent, exampleFileUrl);
    } // end of scenario for loop
  }

  private validateResponse(operation: Operation, exampleContent: any, exampleFileUrl: string) {
    const exampleResponsesObj = exampleContent.responses;
    if (
      exampleResponsesObj === null ||
      exampleResponsesObj === undefined ||
      typeof exampleResponsesObj !== "object"
    ) {
      throw new Error(
        `For operation:${operation.operationId}, responses in example file ${exampleFileUrl} cannot be null or undefined and must be of type 'object'.`
      );
    }

    const definedResponsesInSwagger: string[] = [];
    for (const statusCode of openapiToolsCommon.keys(operation.responses)) {
      definedResponsesInSwagger.push(statusCode);
    }
    // loop for each response code
    for (const exampleResponseStatusCode of openapiToolsCommon.keys(exampleResponsesObj)) {
      const responseDefinition = operation.responses;

      const responseSchema = responseDefinition[exampleResponseStatusCode];
      if (!responseSchema) {
        const meta = getOavErrorMeta(ErrorCodeConstants.RESPONSE_STATUS_CODE_NOT_IN_SPEC as any, {
          exampleResponseStatusCode,
          operationId: operation.operationId,
        });
        this.addErrorsFromErrorCode(
          operation.operationId!,
          exampleFileUrl,
          meta,
          responseDefinition,
          undefined,
          exampleResponsesObj,
          `$response.${exampleResponseStatusCode}`,
          ValidationResultSource.RESPONSE
        );
        continue;
      }

      // remove response entry after validation
      const responseIndex = definedResponsesInSwagger.indexOf(exampleResponseStatusCode);
      if (responseIndex > -1) {
        definedResponsesInSwagger.splice(responseIndex, 1);
      }

      const exampleResponseHeaders = exampleResponsesObj[exampleResponseStatusCode].headers || {};
      const exampleResponseBody = exampleResponsesObj[exampleResponseStatusCode].body;

      // Fail when example provides the response body but the swagger spec doesn't define the schema for the response.
      if (exampleResponseBody !== undefined) {
        if (!responseSchema.schema) {
          // having response body but doesn't have schema defined in response swagger definition
          const meta = getOavErrorMeta(ErrorCodeConstants.RESPONSE_SCHEMA_NOT_IN_SPEC as any, {
            exampleResponseStatusCode,
            operationId: operation.operationId,
          });
          this.addErrorsFromErrorCode(
            operation.operationId!,
            exampleFileUrl,
            meta,
            responseDefinition,
            undefined,
            exampleResponsesObj,
            `$response.${exampleResponseStatusCode}/body`,
            ValidationResultSource.RESPONSE
          );
          continue;
        } else if (responseSchema.schema.type !== typeof exampleResponseBody) {
          const validBody = this.validateBodyType(
            operation.operationId!,
            responseSchema,
            exampleResponseBody,
            exampleFileUrl,
            responseDefinition,
            exampleResponsesObj,
            exampleResponseStatusCode
          );
          if (!validBody) {
            continue;
          }
        }
      } else if (exampleResponseBody === undefined && responseSchema.schema) {
        // Fail when example doesn't provide the response body but the swagger spec define the schema for the response.
        const meta = getOavErrorMeta(ErrorCodeConstants.RESPONSE_BODY_NOT_IN_EXAMPLE as any, {
          exampleResponseStatusCode,
          operationId: operation.operationId,
        });
        this.addErrorsFromErrorCode(
          operation.operationId!,
          exampleFileUrl,
          meta,
          responseDefinition,
          undefined,
          exampleResponsesObj,
          `$response.${exampleResponseStatusCode}`,
          ValidationResultSource.RESPONSE
        );
        continue;
      }
      // validate headers
      const headers = transformLiveHeader(exampleResponseHeaders, responseSchema);
      this.validateLroOperation(
        exampleFileUrl,
        operation,
        exampleResponseStatusCode,
        headers,
        exampleResponseHeaders
      );
      if (responseSchema.schema !== undefined) {
        if (headers["content-type"] !== undefined) {
          this.validateContentType(
            operation.operationId!,
            exampleFileUrl,
            operation.produces!,
            headers,
            false,
            exampleResponseStatusCode,
            operation,
            exampleResponseHeaders
          );
        }
        const validate = responseSchema._validate!;
        const ctx = {
          isResponse: true,
          statusCode: exampleResponseStatusCode,
          httpMethod: operation._method,
        };
        const ajvValidatorErrors = validate(ctx, {
          headers,
          body: exampleResponseBody,
        });
        this.schemaIssuesToModelValidationIssues(
          operation.operationId!,
          false,
          ajvValidatorErrors,
          exampleFileUrl,
          exampleContent,
          undefined,
          exampleResponseStatusCode,
          operation._method
        );
      }
    }

    // report missing response in example
    for (const statusCodeInSwagger of definedResponsesInSwagger) {
      if (statusCodeInSwagger !== "default") {
        const meta = getOavErrorMeta(
          ErrorCodeConstants.RESPONSE_STATUS_CODE_NOT_IN_EXAMPLE as any,
          { statusCodeInSwagger, operationId: operation.operationId }
        );
        this.addErrorsFromErrorCode(
          operation.operationId!,
          exampleFileUrl,
          meta,
          operation.responses[statusCodeInSwagger],
          undefined,
          exampleResponsesObj,
          `response`,
          ValidationResultSource.RESPONSE
        );
      }
    }
  }

  private validateRequest(operation: Operation, exampleContent: any, exampleFileUrl: string) {
    const parameterizedHostDef = operation._path._spec["x-ms-parameterized-host"];
    const useSchemePrefix = parameterizedHostDef
      ? parameterizedHostDef.useSchemePrefix === undefined
        ? true
        : parameterizedHostDef.useSchemePrefix
      : null;
    const pathTemplate = operation._path._pathTemplate;
    const parameters = operation.parameters;
    const publicParameters = operation._path.parameters;
    const mergedParameters = [...(parameters ?? []), ...(publicParameters ?? [])];
    const pathParameters: { [key: string]: string } = {};
    let bodyParameter: any = {};
    const queryParameters: ParsedUrlQuery = {};
    const formData: { [key: string]: string } = {};
    const exampleRequestHeaders: { [propertyName: string]: string } = {};
    if (mergedParameters === undefined) {
      return;
    }
    for (const p of mergedParameters) {
      const parameter = this.jsonLoader.resolveRefObj(p);
      let parameterValue = exampleContent?.parameters[parameter.name];
      if (!parameterValue) {
        if (parameter.required) {
          const meta = getOavErrorMeta(
            ErrorCodeConstants.REQUIRED_PARAMETER_EXAMPLE_NOT_FOUND as any,
            { operationId: operation.operationId, name: parameter.name }
          );
          this.addErrorsFromErrorCode(
            operation.operationId!,
            exampleFileUrl,
            meta,
            operation,
            undefined,
            p
          );
          break;
        }
        continue;
      }
      const location = parameter.in;
      if (location === "path") {
        if (location === "path" && parameterValue && typeof parameterValue === "string") {
          // "/{scope}/scopes/resourceGroups/{resourceGroupName}" In the aforementioned path
          // template, we will search for the path parameter based on it's name
          // for example: "scope". Find it's index in the string and move backwards by 2 positions.
          // If the character at that position is a forward slash "/" and
          // the value for the parameter starts with a forward slash "/" then we have found the case
          // where there will be duplicate forward slashes in the url.
          if (
            pathTemplate.charAt(pathTemplate.indexOf(`${parameter.name}`) - 2) === "/" &&
            parameterValue.startsWith("/")
          ) {
            const meta = getOavErrorMeta(ErrorCodeConstants.DOUBLE_FORWARD_SLASHES_IN_URL as any, {
              operationId: operation.operationId,
              parameterName: parameter.name,
              parameterValue,
              pathTemplate,
            });
            this.addErrorsFromErrorCode(
              operation.operationId!,
              exampleFileUrl,
              meta,
              operation,
              undefined,
              p
            );
            break;
          }
          // replacing characters that may cause validator failed  with empty string because this messes up Sways regex
          // validation of path segment.
          parameterValue = parameterValue.replace(/\//gi, "");

          // replacing scheme that may cause validator failed when x-ms-parameterized-host enbaled & useSchemePrefix enabled
          // because if useSchemePrefix enabled ,the parameter value in x-ms-parameterized-host should not has the scheme (http://|https://)
          if (useSchemePrefix) {
            parameterValue = (parameterValue as string).replace(/^https{0,1}:/gi, "");
          }
        }
        // todo skip url encoding
        pathParameters[parameter.name] = parameterValue;
      } else if (location === "query") {
        // validate the api version value
        if (parameter.name === "api-version" && parameterValue !== this.swagger.info.version) {
          const meta = getOavErrorMeta("INVALID_REQUEST_PARAMETER", {
            parameterName: "api-version",
            apiVersion: parameterValue,
          });
          this.addErrorsFromErrorCode(
            operation.operationId!,
            exampleFileUrl,
            meta,
            operation,
            undefined,
            exampleContent?.parameters,
            `$parameters["api-version"]`
          );
          continue;
        }
        queryParameters[parameter.name] = parameterValue;
      } else if (location === "body") {
        if ((parameter as BodyParameter).schema?.format === "file") {
          continue;
        }
        bodyParameter = parameterValue;
      } else if (location === "header") {
        exampleRequestHeaders[parameter.name] = parameterValue;
      } else if (location === "formData") {
        formData[parameter.name] = parameterValue;
      }
    } // end of mergedParameters for loop
    transformMapValue(queryParameters, operation._queryTransform);
    transformMapValue(pathParameters, operation._pathTransform);
    const validate = operation._validate!;
    const ctx = { isResponse: false };
    const headers = transformLiveHeader(exampleRequestHeaders, operation);
    const ajvValidatorErrors = validate(ctx, {
      path: pathParameters,
      body: transformBodyValue(bodyParameter, operation),
      query: queryParameters,
      headers,
      formData,
    });
    this.schemaIssuesToModelValidationIssues(
      operation.operationId!,
      true,
      ajvValidatorErrors,
      exampleFileUrl,
      exampleContent,
      mergedParameters
    );
  }
  private async loadSwagger(swaggerFilePath: string, skipResolveRef: boolean) {
    try {
      this.swagger = (await this.jsonLoader.load(
        swaggerFilePath,
        skipResolveRef
      )) as unknown as SwaggerSpec;
      this.swagger._filePath = swaggerFilePath;
    } catch (e) {
      if (typeof e.kind === "string") {
        const ex = e as ParseError;
        const errInfo = getOavErrorMeta("JSON_PARSING_ERROR", { details: ex.code });
        this.errors.push({
          code: errInfo.code as any,
          message: errInfo.message,
          schemaPosition: ex.position,
          schemaUrl: ex.url,
        });
      } else if (e instanceof JsonLoaderRefError) {
        const errInfo = getOavErrorMeta("UNRESOLVABLE_REFERENCE", { ref: e.ref });
        this.errors.push({
          code: errInfo.code as any,
          message: errInfo.message,
          schemaPosition: e.position,
          schemaUrl: e.url,
        });
      } else {
        throw e;
      }
      return;
    }
  }

  private async transformSwagger(spec: SwaggerSpec) {
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
  }

  private addErrorsFromErrorCode(
    operationId: string,
    exampleUrl: string | undefined,
    meta: ReturnType<typeof getOavErrorMeta>,
    schemaObj?: any,
    schemaJsonPath?: string,
    exampleObj?: any,
    exampleJsonPath?: string,
    source?: ValidationResultSource
  ) {
    if (
      isSuppressedInPath(schemaObj, meta.id!, meta.message) ||
      isSuppressedInPath(schemaObj, meta.code, meta.message)
    ) {
      return;
    }

    const schemaInfo = getInfo(schemaObj);
    const exampleInfo = getInfo(exampleObj);
    this.errors.push({
      code: meta.code as ModelValidationErrorCode,
      message: meta.message,
      schemaUrl: this.specPath,
      exampleUrl,
      schemaPosition: schemaInfo?.position,
      schemaJsonPath: schemaJsonPath ?? schemaObj?.[refSelfSymbol],
      examplePosition: exampleInfo?.position,
      exampleJsonPath: exampleJsonPath,
      severity: meta.severity,
      source: source ?? ValidationResultSource.GLOBAL,
      operationId,
    });
  }

  private schemaIssuesToModelValidationIssues(
    operationId: string,
    isRequest: boolean,
    errors: SchemaValidateIssue[],
    exampleUrl: string,
    exampleContent: any,
    parameters?: Parameter[],
    statusCode?: string,
    httpMethod?: string
  ) {
    for (const err of errors) {
      // ignore below schema errors
      if (
        (err.code === "NOT_PASSED" &&
          (err.message.includes('should match "else" schema') ||
            err.message.includes('should match "then" schema'))) ||
        ((err.code as any) === "SECRET_PROPERTY" && httpMethod === "post")
      ) {
        continue;
      }
      let isSuppressed = false;
      let schemaPosition;
      let examplePosition;
      let externalSwagger;
      let schemaUrl = this.specPath;
      const exampleJsonPaths: string[] = [];
      if (isRequest) {
        for (const p of parameters!) {
          const parameter = this.jsonLoader.resolveRefObj(p);
          if (
            err.schemaPath.indexOf(parameter.name) > 0 ||
            (err.jsonPathsInPayload.length > 0 &&
              (err.jsonPathsInPayload[0].includes(parameter.name) ||
                (err.jsonPathsInPayload[0].includes(".body") && parameter.in === "body")))
          ) {
            schemaPosition = err.source.position;
            if (parameter.in !== "body") {
              const index = err.schemaPath.indexOf(parameter.name);
              err.schemaPath = index > 0 ? err.schemaPath.substr(index) : err.schemaPath;
            }
            if (isSuppressedInPath(parameter, err.code, err.message)) {
              isSuppressed = true;
            }
            exampleJsonPaths.push(`$parameters.${parameter.name}`);
            break;
          }
        }
      } else {
        // response error case
        let sourceSwagger = this.swagger;
        if (err.source.url !== this.swagger._filePath) {
          // use external swagger when the error points to external file
          sourceSwagger = this.jsonLoader.getFileContentFromCache(err.source.url) as any;
          schemaUrl = err.source.url;
        }

        // check x-nullable value when body is null
        if (err.jsonPathsInPayload.length === 1 && err.jsonPathsInPayload[0] === ".body") {
          const idx = err.source.jsonRef?.indexOf("#");
          if (idx !== undefined && idx !== -1) {
            const jsonRef = err.source.jsonRef?.substr(idx + 1);
            const bodySchema = jsonPointer.get(sourceSwagger, jsonRef!);
            if (bodySchema?.[C.xNullable] === true) {
              continue;
            }
          }
        }

        if (
          (err.code as any) === "MISSING_RESOURCE_ID" &&
          exampleContent.responses[statusCode!].body &&
          Object.keys(exampleContent.responses[statusCode!].body).length === 0
        ) {
          // ignore this error when whole body of response is empty
          continue;
        }

        const node = this.getNotSuppressedErrorPath(err);
        if (node === undefined) {
          continue;
        }
        if (externalSwagger === undefined) {
          schemaPosition = getInfo(node)?.position;
        } else {
          schemaPosition = err.source.position;
        }

        for (let path of err.jsonPathsInPayload) {
          // If parameter name includes ".", path should use "[]" for better understand.
          for (const parameter of err.params) {
            if (
              typeof parameter === "string" &&
              path.includes(`.${parameter}`) &&
              parameter.includes(".")
            ) {
              path = path.substring(0, path.indexOf(`.${parameter}`)) + `['${parameter}']`;
            }
          }
          exampleJsonPaths.push(`$responses.${statusCode}${path}`);
        }
      }

      if (!isSuppressed) {
        for (const jsonPath of exampleJsonPaths) {
          examplePosition = getFilePositionFromJsonPath(exampleContent, jsonPath);
          this.errors.push({
            code: err.code,
            message: err.message,
            schemaUrl,
            exampleUrl,
            schemaPosition: schemaPosition,
            schemaJsonPath: err.schemaPath,
            examplePosition,
            exampleJsonPath: jsonPath,
            severity: err.severity,
            source: isRequest ? ValidationResultSource.REQUEST : ValidationResultSource.RESPONSE,
            operationId,
          });
        }
      }
    }
  }

  private getNotSuppressedErrorPath = (err: SchemaValidateIssue): any => {
    let node;
    let jsonRef;
    if (err.source.jsonRef === undefined) {
      return undefined;
    }
    const idx = err.source.jsonRef.indexOf("#");
    jsonRef = err.source.jsonRef.substr(idx + 1);
    /*eslint no-constant-condition: ["error", { "checkLoops": false }]*/
    while (true) {
      try {
        if (err.source.url === this.swagger._filePath) {
          node = jsonPointer.get(this.swagger, jsonRef);
        } else {
          const externalSwagger = this.jsonLoader.getFileContentFromCache(err.source.url);
          node = jsonPointer.get(externalSwagger as openapiToolsCommon.JsonObject, jsonRef);
        }
        const isSuppressed = isSuppressedInPath(node, err.code, err.message);
        if (isSuppressed) {
          return undefined;
        } else {
          break;
        }
      } catch (e) {
        let isContinue = false;
        // the jsonRef will include non-existed path, so it needs to walk back to
        // exclude the unexisted path
        if (e.message.includes("Invalid reference token:")) {
          const token = e.message.substring("Invalid reference token:".length + 1);
          const index = jsonRef.lastIndexOf(token);
          if (index > 0) {
            jsonRef = jsonRef.substring(0, index);
            if (jsonRef.endsWith(".") || jsonRef.endsWith("/")) {
              jsonRef = jsonRef.substring(0, jsonRef.length - 1);
            }
            isContinue = true;
          }
        }
        // if it's not the case of containing unexisted path, then throw this error
        if (isContinue === false) {
          log.error(`Exception in filtering not suppressed error path:${jsonRef}.`);
          throw e;
        }
      }
    }
    return node;
  };
  private issueFromErrorCode = (
    operationId: string,
    examplePath: string,
    code: ModelValidationErrorCode,
    param: any,
    schemaObj?: any,
    schemaJsonPath?: string,
    exampleObj?: any,
    exampleJsonPath?: string,
    source?: ValidationResultSource
  ): SwaggerExampleErrorDetail => {
    const meta = getOavErrorMeta(code, param);
    const schemaInfo = schemaObj ? getInfo(schemaObj) : schemaObj;
    const exampleInfo = exampleObj ? getInfo(exampleObj) : exampleObj;
    return {
      code,
      message: meta.message,
      schemaUrl: this.specPath,
      exampleUrl: examplePath,
      schemaPosition: schemaInfo?.position,
      schemaJsonPath: schemaJsonPath,
      examplePosition: exampleInfo?.position,
      exampleJsonPath: exampleJsonPath,
      severity: meta.severity,
      source: source ?? ValidationResultSource.GLOBAL,
      operationId,
    };
  };

  private validateBodyType = (
    operationId: string,
    responseSchema: any,
    body: any,
    exampleFileUrl: string,
    responseDefinition: any,
    exampleResponsesObj: any,
    exampleResponseStatusCode: string
  ): boolean => {
    let expectedType = responseSchema.schema.type;
    const actualType = typeof body;
    let bodySchema: any = {};

    if (responseSchema.schema.format === "file" || expectedType === "file") {
      // ignore validation in case of file type
      return true;
    } else if (expectedType === undefined) {
      // in case of body providing primitive type
      bodySchema = this.jsonLoader.resolveRefObj(responseSchema.schema);
      // set excpectedType as 'object' if the schema has properties key or additionalProperties key
      if (
        bodySchema.type === undefined &&
        (Object.keys(bodySchema).includes("properties") ||
          Object.keys(bodySchema).includes("additionalProperties"))
      ) {
        expectedType = "object";
      } else {
        expectedType = bodySchema.type;
      }
    }

    let invalidTypeError: boolean = false;
    if (
      (actualType === "number" && expectedType === "integer") ||
      (actualType === "object" && expectedType === "array") ||
      (Object.keys(bodySchema)?.length === 0 && Object.keys(body)?.length === 0)
    ) {
      invalidTypeError = false;
    } else if (expectedType !== actualType) {
      invalidTypeError = true;
    }

    if (invalidTypeError) {
      const meta = getOavErrorMeta(ErrorCodeConstants.INVALID_TYPE as any, {
        type: expectedType,
        data: actualType,
      });
      this.addErrorsFromErrorCode(
        operationId,
        exampleFileUrl,
        meta,
        responseDefinition,
        undefined,
        exampleResponsesObj,
        `$response.${exampleResponseStatusCode}/body`,
        ValidationResultSource.RESPONSE
      );
      return false;
    }
    return true;
  };

  private validateContentType = (
    operationId: string,
    examplePath: string,
    allowedContentTypes: string[],
    headers: StringMap<string>,
    isRequest: boolean,
    statusCode: string,
    schemaObj?: any,
    exampleObj?: any
  ) => {
    const contentType =
      headers["content-type"]?.split(";")[0] ||
      (isRequest ? undefined : "application/octet-stream");
    if (contentType !== undefined && !allowedContentTypes.includes(contentType)) {
      // in some cases, produces value could have colon in type like 'application/json;odata=minimalmetadata'
      for (const allowedContentType of allowedContentTypes) {
        if (allowedContentType.includes(";")) {
          const subAllowedContentType = allowedContentType.split(";")[0];
          if (subAllowedContentType.includes(contentType)) {
            return;
          }
        }
      }
      this.errors.push(
        this.issueFromErrorCode(
          operationId,
          examplePath,
          "INVALID_CONTENT_TYPE",
          {
            contentType,
            supported: allowedContentTypes.join(", "),
          },
          schemaObj,
          undefined,
          exampleObj,
          `responses/${statusCode}/headers`,
          ValidationResultSource.RESPONSE
        )
      );
    }
  };

  private validateLroOperation = (
    examplePath: string,
    operation: Operation,
    statusCode: string,
    headers: StringMap<string>,
    exampleObj?: any
  ) => {
    if (operation["x-ms-long-running-operation"] === true && parseInt(statusCode, 10) < 300) {
      if (operation._method === "post") {
        if (statusCode === "202" || statusCode === "201") {
          this.validateLroHeader(examplePath, operation, statusCode, headers);
        } else if (statusCode !== "200" && statusCode !== "204") {
          this.errors.push(
            this.issueFromErrorCode(
              operation.operationId!,
              examplePath,
              "LRO_RESPONSE_CODE",
              { statusCode },
              operation.responses,
              undefined,
              exampleObj,
              `responses/${statusCode}`,
              ValidationResultSource.RESPONSE
            )
          );
        }
      } else if (operation._method === "patch") {
        if (statusCode === "202" || statusCode === "201") {
          this.validateLroHeader(examplePath, operation, statusCode, headers);
        } else if (statusCode !== "200") {
          this.errors.push(
            this.issueFromErrorCode(
              operation.operationId!,
              examplePath,
              "LRO_RESPONSE_CODE",
              { statusCode },
              operation.responses,
              undefined,
              exampleObj,
              `responses/${statusCode}`,
              ValidationResultSource.RESPONSE
            )
          );
        }
      } else if (operation._method === "delete") {
        if (statusCode === "202") {
          this.validateLroHeader(examplePath, operation, statusCode, headers);
        } else if (statusCode !== "200" && statusCode !== "204") {
          this.errors.push(
            this.issueFromErrorCode(
              operation.operationId!,
              examplePath,
              "LRO_RESPONSE_CODE",
              { statusCode },
              operation.responses,
              undefined,
              exampleObj,
              `responses/${statusCode}`,
              ValidationResultSource.RESPONSE
            )
          );
        }
      } else if (operation._method === "put") {
        if (statusCode !== "200" && statusCode !== "201" && statusCode !== "202") {
          this.errors.push(
            this.issueFromErrorCode(
              operation.operationId!,
              examplePath,
              "LRO_RESPONSE_CODE",
              { statusCode },
              operation.responses,
              undefined,
              exampleObj,
              `responses/${statusCode}`,
              ValidationResultSource.RESPONSE
            )
          );
        }
      }
    }
  };

  private validateLroHeader = (
    examplePath: string,
    operation: Operation,
    statusCode: string,
    headers: StringMap<string>,
    exampleObj?: any
  ) => {
    if (statusCode === "201") {
      // Ignore LRO header check cause RPC says azure-AsyncOperation is optional if using 201/200+ provisioningState
      return;
    }

    const provider = getProviderFromSpecPath(this.specPath);
    const providerType = provider?.type;
    if (providerType !== undefined && this.isMissingHeader(providerType, headers)) {
      this.errors.push(
        this.issueFromErrorCode(
          operation.operationId!,
          examplePath,
          "LRO_RESPONSE_HEADER",
          {
            header:
              providerType === "resource-manager"
                ? "location or azure-AsyncOperation"
                : "Operation-Id or Operation-Location or azure-AsyncOperation", // providerType === "data-plane"
          },
          operation.responses,
          undefined,
          exampleObj,
          `responses/${statusCode}/headers`,
          ValidationResultSource.RESPONSE
        )
      );
    }
  };

  private isMissingHeader = (
    providerType: "resource-manager" | "data-plane",
    header: StringMap<string>
  ) => {
    const headerConfig = {
      "resource-manager": {
        checkFirst: ["Location", "location", "azure-AsyncOperation", "azure-asyncoperation"],
        checkSecond: [],
      },
      "data-plane": {
        checkFirst: ["Operation-Id", "operation-id", "Operation-Location", "operation-location"],
        checkSecond: ["azure-AsyncOperation", "azure-asyncoperation"],
      },
    };

    const config = headerConfig[providerType];
    for (const checkFirst of config.checkFirst) {
      const value = header[checkFirst];
      if (typeof value === "string" && value !== "") {
        return false;
      }
    }
    for (const checkSecond of config.checkSecond) {
      const value = header[checkSecond];
      if (typeof value === "string" && value !== "") {
        return false;
      }
    }

    return true;
  };
}

// load all error codes of model validation errors to as suppression option
const loadSuppression = [];
for (const errorCode of Object.keys(modelValidationErrors)) {
  const meta = modelValidationErrors[errorCode as ModelValidationErrorCode];
  if ("id" in meta) {
    loadSuppression.push(meta.id);
  }
  loadSuppression.push(errorCode);
}

// Set 'isArmCall flag to true so that the special ARM rules can be applied to examples validation too'
const defaultOpts: ExampleValidationOption = {
  eraseDescription: false,
  eraseXmsExamples: false,
  useJsonParser: true,
  loadSuppression,
  isArmCall: true,
};

// Compatible wrapper for old ModelValidator
export class NewModelValidator {
  public validator: SwaggerExampleValidator;
  public result: SwaggerExampleErrorDetail[] = [];
  public constructor(public specPath: string) {
    const container = inversifyGetContainer();
    this.validator = inversifyGetInstance(SwaggerExampleValidator, {
      ...defaultOpts,
      container,
    });
    this.result = this.validator.errors;
  }

  public async initialize() {
    await this.validator.initialize(this.specPath);
    // API compatible
    return null as any;
  }

  public async validateOperations(operationIds?: string) {
    await this.validator.validateOperations(operationIds);
  }
}

export interface SwaggerExampleErrorDetail {
  inner?: any; // Compatible with old NodeError. Always undefined.
  message: string;
  code: ModelValidationErrorCode;
  schemaPosition?: FilePosition;
  schemaUrl?: string;
  schemaJsonPath?: string;
  examplePosition?: FilePosition;
  exampleUrl?: string;
  exampleJsonPath?: string;
  severity?: Severity;
  source?: ValidationResultSource;
  operationId?: string;
}

export interface ExampleValidationOption extends SwaggerLoaderOption, SchemaValidatorOption {}
