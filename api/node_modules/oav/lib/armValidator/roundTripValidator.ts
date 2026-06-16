import { getJsonPatchDiff } from "../apiScenario/diffUtils";
import { RequestResponsePair, LiveValidationIssue } from "../liveValidation/liveValidator";
import { OperationContext } from "../liveValidation/operationValidator";
import { roundTripValidationErrors } from "../util/errorDefinitions";
import * as utils from "../util/utils";
import { Parameter, Operation } from "../swagger/swaggerTypes";
import { JsonLoader } from "../swagger/jsonLoader";
import { SchemaSearcher } from "../apiScenario/schemaSearcher";

const allowed = true;
const notAllowed = false;

function checkReplacedSchemaInParameter(
  jsonPath: string,
  parameter: Parameter,
  jsonLoader: JsonLoader
) {
  const subSet = (currentValue: string) => ["create", "read"].includes(currentValue);
  if (parameter.in === "body") {
    const schema = jsonLoader.resolveRefObj(parameter.schema!);
    const foundSchema = SchemaSearcher.findSchemaByJsonPointer(jsonPath, schema, jsonLoader);
    if (
      foundSchema.readOnly ||
      foundSchema.default ||
      (foundSchema["x-ms-mutability"] && foundSchema["x-ms-mutability"].every(subSet))
    ) {
      return allowed;
    }
    return notAllowed;
  }
  return notAllowed;
}

function checkRemovedSchemaInParameter(
  jsonPath: string,
  parameter: Parameter,
  jsonLoader: JsonLoader
) {
  const subSet = (currentValue: string) => ["create", "update"].includes(currentValue);
  if (parameter.in === "body") {
    const schema = jsonLoader.resolveRefObj(parameter.schema!);
    const foundSchema = SchemaSearcher.findSchemaByJsonPointer(jsonPath, schema, jsonLoader);
    if (
      foundSchema["x-ms-secret"] ||
      (foundSchema["x-ms-mutability"] && foundSchema["x-ms-mutability"].every(subSet))
    ) {
      return allowed;
    }
    return notAllowed;
  }
  return notAllowed;
}

function checkSchemaInResponse(
  jsonPath: string,
  op: Operation,
  jsonLoader: JsonLoader,
  responseStatusCode: string
) {
  let statusCode;
  if (isNaN(+responseStatusCode)) {
    statusCode = utils.statusCodeStringToStatusCode[responseStatusCode.toLowerCase()];
    if (statusCode === undefined) {
      statusCode = "default";
    }
  } else {
    statusCode = responseStatusCode;
  }

  let responseSchema: any = op.responses[statusCode];
  if (responseSchema === undefined) {
    responseSchema = op.responses["default"];
  }
  if (responseSchema.schema) {
    responseSchema = responseSchema.schema;
  }
  const schema = jsonLoader.resolveRefObj(responseSchema);
  const foundSchema = SchemaSearcher.findSchemaByJsonPointer(jsonPath, schema, jsonLoader);
  if (foundSchema.readOnly || foundSchema.default) {
    return allowed;
  }
  return notAllowed;
}

export function diffRequestResponse(
  payload: RequestResponsePair,
  info: OperationContext,
  jsonLoader: JsonLoader
) {
  const diffs = getJsonPatchDiff(payload.liveRequest.body ?? {}, payload.liveResponse.body ?? {}, {
    includeOldValue: true,
    minimizeDiff: false,
  });

  const rest = diffs
    .map((it: any) => {
      const jsonPath: string = it.remove || it.add || it.replace;
      if (it.replace !== undefined) {
        let isAllowed = false;
        for (let parameter of info.operationMatch?.operation.parameters ?? []) {
          if (isAllowed) {
            break;
          }
          isAllowed = checkReplacedSchemaInParameter(it.replace, parameter, jsonLoader);
        }
        for (let parameter of info.operationMatch?.operation._path.parameters ?? []) {
          if (isAllowed) {
            break;
          }
          isAllowed = checkReplacedSchemaInParameter(it.replace, parameter, jsonLoader);
        }
        if (!isAllowed) {
          return buildLiveValidationIssue("ROUNDTRIP_INCONSISTENT_PROPERTY", jsonPath, it);
        }
      } else if (it.add !== undefined && it.value !== null) {
        // IF a property is not in request but returned in response as null, ignore.
        let isAllowed = checkSchemaInResponse(
          it.add,
          info.operationMatch?.operation!,
          jsonLoader,
          payload.liveResponse.statusCode
        );
        if (!isAllowed) {
          return buildLiveValidationIssue("ROUNDTRIP_ADDITIONAL_PROPERTY", jsonPath, it);
        }
      } else if (it.remove !== undefined) {
        let isAllowed = false;
        for (let parameter of info.operationMatch?.operation.parameters ?? []) {
          if (isAllowed) {
            break;
          }
          isAllowed = checkRemovedSchemaInParameter(it.remove, parameter, jsonLoader);
        }
        for (let parameter of info.operationMatch?.operation._path.parameters ?? []) {
          if (isAllowed) {
            break;
          }
          isAllowed = checkRemovedSchemaInParameter(it.remove, parameter, jsonLoader);
        }
        if (!isAllowed) {
          return buildLiveValidationIssue("ROUNDTRIP_MISSING_PROPERTY", jsonPath, it);
        }
      }
      return undefined;
    })
    .filter((a) => a !== undefined);
  return rest;
}

export function buildLiveValidationIssue(
  errorCode: string,
  path: string,
  it: any
): LiveValidationIssue {
  let severity, message;
  const properties = path.split("/");
  let property = properties.pop();
  if (!isNaN(Number(property)) && properties.length > 0) {
    property = `${properties.pop()}/${property}`;
  }
  switch (errorCode) {
    case "ROUNDTRIP_INCONSISTENT_PROPERTY": {
      severity = roundTripValidationErrors.ROUNDTRIP_INCONSISTENT_PROPERTY.severity;
      message = roundTripValidationErrors.ROUNDTRIP_INCONSISTENT_PROPERTY.message({
        getValue: it.value,
        putValue: it.oldValue,
      });
      break;
    }
    case "ROUNDTRIP_ADDITIONAL_PROPERTY": {
      severity = roundTripValidationErrors.ROUNDTRIP_ADDITIONAL_PROPERTY.severity;
      message = roundTripValidationErrors.ROUNDTRIP_ADDITIONAL_PROPERTY.message({
        property: property,
      });
      break;
    }
    case "ROUNDTRIP_MISSING_PROPERTY": {
      severity = roundTripValidationErrors.ROUNDTRIP_MISSING_PROPERTY.severity;
      message = roundTripValidationErrors.ROUNDTRIP_MISSING_PROPERTY.message({
        property: property,
      });
      break;
    }
  }
  const ret = {
    code: errorCode,
    pathsInPayload: [path],
    severity: severity,
    message: message,
    jsonPathsInPayload: [],
    schemaPath: "",
    source: {
      url: "",
      position: {
        column: 0,
        line: 0,
      },
    },
  };

  return ret as LiveValidationIssue;
}
