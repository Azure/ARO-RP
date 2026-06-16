import { ParsedUrlQuery } from "querystring";
import {
  FilePosition,
  getInfo,
  MutableStringMap,
  StringMap,
} from "@azure-tools/openapi-tools-common";
import {
  LoggingFn,
  LowerHttpMethods,
  Operation,
  Response,
  TransformFn,
} from "../swagger/swaggerTypes";
import { sourceMapInfoToSourceLocation } from "../swaggerValidator/ajvSchemaValidator";
import { SchemaValidateContext, SchemaValidateIssue } from "../swaggerValidator/schemaValidator";
import { jsonPathToPointer } from "../util/jsonUtils";
import { Writable } from "../util/utils";
import { SourceLocation } from "../util/validationError";
import { extractPathParamValue } from "../transform/pathRegexTransformer";
import {
  ApiValidationErrorCode,
  getOavErrorMeta,
  TrafficValidationErrorCode,
} from "../util/errorDefinitions";
import {
  LiveValidationIssue,
  LiveValidatorLoggingLevels,
  LiveValidatorLoggingTypes,
} from "./liveValidator";
import { LiveValidatorLoader } from "./liveValidatorLoader";
import { OperationMatch } from "./operationSearcher";

export interface ValidationRequest {
  providerNamespace: string;
  resourceType?: string;
  apiVersion: string;
  requestMethod?: LowerHttpMethods;
  host?: string;
  pathStr?: string;
  query?: ParsedUrlQuery;
  correlationId?: string;
  activityId?: string;
  requestUrl?: string;
  specName?: string;
}

export interface OperationContext {
  operationId: string;
  apiVersion: string;
  operationMatch?: OperationMatch;
  validationRequest?: ValidationRequest;
  position?: FilePosition | undefined;
}

export interface LiveRequest {
  query?: ParsedUrlQuery;
  readonly url: string;
  readonly method: string;
  headers?: { [propertyName: string]: string };
  body?: StringMap<unknown>;
}

export interface LiveResponse {
  statusCode: string;
  headers?: { [propertyName: string]: string };
  body?: StringMap<unknown>;
}

export const validateSwaggerLiveRequest = async (
  request: LiveRequest,
  operationContext: OperationContext,
  loader?: LiveValidatorLoader,
  includeErrors?: ApiValidationErrorCode[],
  isArmCall?: boolean,
  logging?: LoggingFn
) => {
  const { operation } = operationContext.operationMatch!;
  const { body, query } = request;
  const result: LiveValidationIssue[] = [];

  let validate = operation._validate;
  if (validate === undefined) {
    if (loader === undefined) {
      throw new Error("Loader is undefined but request validator isn't built yet");
    }
    const startTimeToBuild = Date.now();
    validate = await loader.getRequestValidator(operation);
    const elapsedTime = Date.now() - startTimeToBuild;
    if (logging) {
      logging(
        `On-demand build request validator with DurationInMs:${elapsedTime}`,
        LiveValidatorLoggingLevels.debug,
        LiveValidatorLoggingTypes.trace,
        "Oav.OperationValidator.validateSwaggerLiveRequest.loader.getRequestValidator",
        undefined,
        operationContext.validationRequest
      );
      logging(
        `On-demand build request validator`,
        LiveValidatorLoggingLevels.info,
        LiveValidatorLoggingTypes.perfTrace,
        "Oav.OperationValidator.validateSwaggerLiveRequest.loader.getRequestValidator",
        elapsedTime,
        operationContext.validationRequest
      );
    }
  }

  const pathParam = extractPathParamValue(operationContext.operationMatch!);
  transformMapValue(pathParam, operation._pathTransform);
  transformMapValue(query, operation._queryTransform);
  const headers = transformLiveHeader(request.headers ?? {}, operation);
  validateContentType(operation.consumes!, headers, true, result);

  // for rpaas calls, temp solution to log invalid_type errors for additional properties
  // rather than returning the error to rpaas
  const ctx = { isResponse: false, includeErrors: includeErrors as any };
  const errors = validate(ctx, {
    path: pathParam,
    body: transformBodyValue(body, operation),
    headers,
    query,
  });
  schemaValidateIssueToLiveValidationIssue(
    errors,
    operation,
    ctx,
    result,
    operationContext,
    isArmCall,
    logging,
    body
  );

  return result;
};

export const validateSwaggerLiveResponse = async (
  response: LiveResponse,
  operationContext: OperationContext,
  loader?: LiveValidatorLoader,
  includeErrors?: ApiValidationErrorCode[],
  isArmCall?: boolean,
  logging?: LoggingFn
) => {
  const { operation } = operationContext.operationMatch!;
  const { statusCode, body } = response;
  const rspDef = operation.responses;
  const result: LiveValidationIssue[] = [];

  let rsp = rspDef[statusCode];
  const realCode = parseInt(statusCode, 10);
  if (rsp === undefined && 400 <= realCode && realCode <= 599) {
    rsp = rspDef.default;
  }
  if (rsp === undefined) {
    result.push(issueFromErrorCode("INVALID_RESPONSE_CODE", { statusCode }, rspDef));
    return result;
  }

  let validate = rsp._validate;
  if (validate === undefined) {
    if (loader === undefined) {
      throw new Error("Loader is undefined but request validator isn't built yet");
    }
    const startTimeToBuild = Date.now();
    validate = await loader.getResponseValidator(rsp);
    const elapsedTime = Date.now() - startTimeToBuild;
    if (logging) {
      logging(
        `On-demand build response validator with DurationInMs:${elapsedTime}`,
        LiveValidatorLoggingLevels.debug,
        LiveValidatorLoggingTypes.trace,
        "Oav.OperationValidator.validateSwaggerLiveResponse.loader.getResponseValidator",
        undefined,
        operationContext.validationRequest
      );
      logging(
        `On-demand build request validator`,
        LiveValidatorLoggingLevels.info,
        LiveValidatorLoggingTypes.perfTrace,
        "Oav.OperationValidator.validateSwaggerLiveResponse.loader.getResponseValidator",
        elapsedTime,
        operationContext.validationRequest
      );
    }
  }

  const headers = transformLiveHeader(response.headers ?? {}, rsp);
  if (rsp.schema !== undefined) {
    validateContentType(operation.produces!, headers, false, result);
    if (isArmCall && realCode >= 200 && realCode < 300) {
      validateLroOperation(operation, statusCode, headers, result);
    }
  }

  const ctx = {
    isResponse: true,
    includeErrors: includeErrors as any,
    statusCode,
    httpMethod: operation._method,
  };
  const errors = validate(ctx, {
    headers,
    body,
  });
  schemaValidateIssueToLiveValidationIssue(
    errors,
    operation,
    ctx,
    result,
    operationContext,
    isArmCall,
    logging,
    body
  );

  return result;
};

export const transformBodyValue = (body: any, operation: Operation): any => {
  return operation._bodyTransform === undefined ? body : operation._bodyTransform(body);
};

export const transformLiveHeader = (
  headers: StringMap<string>,
  it: Operation | Response
): StringMap<string> => {
  const result: MutableStringMap<string> = {};
  for (const headerName of Object.keys(headers)) {
    result[headerName.toLowerCase()] = headers[headerName];
  }
  transformMapValue(result, it._headerTransform);
  return result;
};

export const transformMapValue = (
  data?: MutableStringMap<string | number | boolean | Array<string | number | boolean>>,
  transforms?: StringMap<TransformFn>
) => {
  if (transforms === undefined || data === undefined) {
    return;
  }
  for (const key of Object.keys(transforms)) {
    const transform = transforms[key]!;
    const val = data[key];
    if (typeof val === "string") {
      data[key] = transform(val);
    } else if (Array.isArray(val)) {
      data[key] = val.map(transform as any);
    }
  }
};

const validateContentType = (
  allowedContentTypes: string[],
  headers: StringMap<string>,
  isRequest: boolean,
  result: LiveValidationIssue[]
) => {
  const contentType =
    headers["content-type"]?.split(";")[0] || (isRequest ? undefined : "application/octet-stream");
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
    result.push(
      issueFromErrorCode("INVALID_CONTENT_TYPE", {
        contentType,
        supported: allowedContentTypes.join(", "),
      })
    );
  }
};

/**
 * Finds the resource ID for a given JSON path. Example inputs:
 *   - $.properties.lastname
 *   - $.properties.groups[0].variable
 * @param bodyPayload The full payload object.
 * @param jsonPath The JSON path referring to the problematic property.
 * @returns The resource ID, or undefined if not found.
 */
const findResourceId = (bodyPayload: any, jsonPath: string): string | undefined => {
  // schemaValidateIssueToLiveValidationIssue will provide a valid jsonPath to this function
  const keys = jsonPath
    .replace(/^\$\./, "") // Remove the leading "$."
    .split(/\.|\[(\d+)\]/) // Split by dots or array brackets
    .filter((key) => key !== undefined && key !== "");

  let current: any = bodyPayload;
  const stack: any[] = [];

  for (const key of keys) {
    if (current && typeof current === "object") {
      stack.push(current);

      if (Array.isArray(current)) {
        // Handle array indices
        const index = parseInt(key, 10);
        if (!isNaN(index) && index < current.length) {
          current = current[index];
        } else {
          return undefined;
        }
      } else if (key in current) {
        current = current[key];
      } else {
        return undefined;
      }
    } else {
      return undefined;
    }
  }

  // notice we only ever check for parent id. This means that we will never accidentally grab the value of an id
  // that has been added erroneously to the payload. (for instance if payload is to an id field that SHOULD NOT be set.)
  for (let i = stack.length - 1; i >= 0; i--) {
    const parent = stack[i];
    if (parent && typeof parent === "object" && "id" in parent) {
      return parent.id;
    }
  }

  return undefined; // No `id` field found
};

export const schemaValidateIssueToLiveValidationIssue = (
  input: SchemaValidateIssue[],
  operation: Operation,
  ctx: SchemaValidateContext,
  output: LiveValidationIssue[],
  _operationContext: OperationContext,
  _isArmCall?: boolean,
  _logging?: LoggingFn,
  _bodyPayload?: any
) => {
  for (const i of input) {
    const issue = i as Writable<LiveValidationIssue>;
    issue.resourceIds = [];
    issue.documentationUrl = "";

    const source = issue.source as Writable<SourceLocation>;
    if (!source.url) {
      source.url = operation._path._spec._filePath;
    }

    let skipIssue = false;
    issue.pathsInPayload = issue.jsonPathsInPayload.map((path, idx) => {
      if (issue.code === "MISSING_RESOURCE_ID") {
        // ignore this error for sub level resources
        if (path.includes("properties")) {
          skipIssue = true;
          return "";
        }
      }

      const isMissingRequiredProperty = issue.code === "OBJECT_MISSING_REQUIRED_PROPERTY";
      const isBodyIssue = path.startsWith(".body");

      if (isBodyIssue && (path.length > 5 || !isMissingRequiredProperty)) {
        path = "$" + path.substr(5);
        issue.jsonPathsInPayload[idx] = path;
        const resolvedJsonPath = jsonPathToPointer(path);

        if (ctx.isResponse && isBodyIssue) {
          const resourceId = findResourceId(_bodyPayload, path);
          if (resourceId) {
            issue.resourceIds!.push(resourceId);
          }
        }

        return resolvedJsonPath;
      }

      if (isMissingRequiredProperty) {
        if (ctx.isResponse) {
          if (isBodyIssue) {
            issue.code = "INVALID_RESPONSE_BODY";
            // If a long running operation with code 201 or 202 then it could has empty body
            if (
              operation["x-ms-long-running-operation"] &&
              (ctx.statusCode === "201" || ctx.statusCode === "202")
            ) {
              skipIssue = true;
            }
          } else if (path.startsWith(".headers")) {
            issue.code = "INVALID_RESPONSE_HEADER";
          }
        } else {
          // In request
          issue.code = "MISSING_REQUIRED_PARAMETER";
        }

        const meta = getOavErrorMeta(issue.code, { missingProperty: issue.params[0] });
        issue.severity = meta.severity;
        issue.message = meta.message;
      }

      const resolvedJsonPath = jsonPathToPointer(path);

      if (ctx.isResponse && isBodyIssue) {
        const resourceId = findResourceId(_bodyPayload, path);
        if (resourceId) {
          issue.resourceIds!.push(resourceId);
        }
      }
      return resolvedJsonPath;
    });

    if (!skipIssue) {
      output.push(issue);
    }
  }
};

const validateLroOperation = (
  operation: Operation,
  statusCode: string,
  headers: StringMap<string>,
  result: LiveValidationIssue[]
) => {
  if (operation["x-ms-long-running-operation"] === true) {
    if (operation._method === "post") {
      if (statusCode === "202" || statusCode === "201") {
        validateLroHeader(operation, statusCode, headers, result);
      } else if (statusCode !== "200" && statusCode !== "204") {
        result.push(issueFromErrorCode("LRO_RESPONSE_CODE", { statusCode }, operation.responses));
      }
    } else if (operation._method === "patch") {
      if (statusCode === "202" || statusCode === "201") {
        validateLroHeader(operation, statusCode, headers, result);
      } else if (statusCode !== "200") {
        result.push(issueFromErrorCode("LRO_RESPONSE_CODE", { statusCode }, operation.responses));
      }
    } else if (operation._method === "delete") {
      if (statusCode === "202") {
        validateLroHeader(operation, statusCode, headers, result);
      } else if (statusCode !== "200" && statusCode !== "204") {
        result.push(issueFromErrorCode("LRO_RESPONSE_CODE", { statusCode }, operation.responses));
      }
    } else if (operation._method === "put") {
      if (statusCode === "202" || statusCode === "201") {
        validateLroHeader(operation, statusCode, headers, result);
      } else if (statusCode !== "200") {
        result.push(issueFromErrorCode("LRO_RESPONSE_CODE", { statusCode }, operation.responses));
      }
    }
  }
};

const validateLroHeader = (
  operation: Operation,
  statusCode: string,
  headers: StringMap<string>,
  result: LiveValidationIssue[]
) => {
  if (statusCode === "201") {
    // Ignore LRO header check cause RPC says azure-AsyncOperation is optional if using 201/200+ provisioningState
    return;
  }

  if (
    (headers.location === undefined || headers.location === "") &&
    (headers["azure-AsyncOperation"] === undefined || headers["azure-AsyncOperation"] === "") &&
    (headers["azure-asyncoperation"] === undefined || headers["azure-asyncoperation"] === "")
  ) {
    result.push(
      issueFromErrorCode(
        "LRO_RESPONSE_HEADER",
        {
          header: "location or azure-AsyncOperation",
        },
        operation.responses
      )
    );
  }
};

export const issueFromErrorCode = (
  code: TrafficValidationErrorCode,
  param: any,
  relatedSchema?: {}
): LiveValidationIssue => {
  const meta = getOavErrorMeta(code, param);
  return {
    code,
    severity: meta.severity,
    message: meta.message,
    jsonPathsInPayload: [],
    pathsInPayload: [],
    schemaPath: "",
    source: sourceMapInfoToSourceLocation(getInfo(relatedSchema)),
    documentationUrl: "",
  };
};
