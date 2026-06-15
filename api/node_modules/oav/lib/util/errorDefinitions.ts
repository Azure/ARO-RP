import { Severity } from "./severity";
import { strTemplate, TemplateFunc } from "./strTemplate";

export type SchemaValidationErrorCode = keyof typeof schemaValidationErrors;

// common error code including payload validation and example validation
export type TrafficValidationErrorCode = keyof typeof trafficValidationErrors;

export type SemanticValidationErrorCode = keyof typeof semanticValidationErrors;

// error code for live validation
export type ApiValidationErrorCode = keyof typeof apiValidationErrors;

// error code for model validation
export type ModelValidationErrorCode = keyof typeof modelValidationErrors;

export type OavAllErrorCode =
  | SchemaValidationErrorCode
  | TrafficValidationErrorCode
  | SemanticValidationErrorCode
  | ApiValidationErrorCode
  | ModelValidationErrorCode;

let allErrors: {
  [code: string]: { severity: Severity; message: TemplateFunc<string>; id?: string };
};
export const getOavErrorMeta = <T extends OavAllErrorCode>(code: T, param: Record<string, any>) => {
  if (allErrors === undefined) {
    allErrors = {
      ...schemaValidationErrors,
      ...trafficValidationErrors,
      ...semanticValidationErrors,
      ...apiValidationErrors,
      ...modelValidationErrors,
    };
  }
  const errorInfo = allErrors[code];
  if (errorInfo === undefined) {
    throw new Error(`Error code "${code}" is not defined!`);
  }

  // change INVALID_REQUEST_PARAMETER message for error about api-version
  const message =
    param.parameterName === "api-version"
      ? `api-version ${param.apiVersion} is not equal to swagger version`
      : errorInfo.message(param);

  const result: {
    code: T;
    severity: Severity;
    message: string;
    id?: string;
  } = {
    code,
    severity: errorInfo.severity,
    message,
  };

  if ("id" in errorInfo) {
    result.id = errorInfo.id;
  }

  return result;
};

export const internalErrors = {
  NOT_PASSED: {
    severity: Severity.Verbose,
    message: strTemplate`Placeholder for unclassified error`,
  },
  INTERNAL_ERROR: {
    severity: Severity.Critical,
    message: strTemplate`Unexpected internal error: ${"message"}`,
  },
};

export const schemaValidationErrors = {
  ...internalErrors,

  UNRESOLVABLE_REFERENCE: {
    severity: Severity.Critical,
    message: strTemplate`Reference could not be resolved: ${"ref"}`,
  },
  DISCRIMINATOR_VALUE_NOT_FOUND: {
    severity: Severity.Critical,
    message: strTemplate`Discriminator value "${"data"}" not found`,
  },
  ANY_OF_MISSING: {
    severity: Severity.Critical,
    message: strTemplate`Data does not match any schemas from 'anyOf'`,
  },
  ONE_OF_MULTIPLE: {
    severity: Severity.Critical,
    message: strTemplate`Data is valid against more than one schema from 'oneOf'`,
  },
  OBJECT_DEPENDENCY_KEY: {
    severity: Severity.Warning,
    message: strTemplate`Dependency failed - key must exist: ${"missingProperty"} (due to key: ${"property"})`,
  },
  ONE_OF_MISSING: {
    severity: Severity.Critical,
    message: strTemplate`Data does not match any schemas from 'oneOf'`,
  },
  OBJECT_ADDITIONAL_PROPERTIES: {
    severity: Severity.Critical,
    message: strTemplate`Additional properties not allowed: ${"additionalProperty"}`,
  },
  OBJECT_PROPERTIES_MAXIMUM: {
    severity: Severity.Critical,
    message: strTemplate`Too many properties defined (${"data"}), maximum ${"limit"}`,
  },
  OBJECT_MISSING_REQUIRED_PROPERTY: {
    severity: Severity.Critical,
    message: strTemplate`Missing required property: ${"missingProperty"}`,
  },
  OBJECT_PROPERTIES_MINIMUM: {
    severity: Severity.Critical,
    message: strTemplate`Too few properties defined (${"data"}), minimum ${"limit"}`,
  },
  ARRAY_LENGTH_SHORT: {
    severity: Severity.Critical,
    message: strTemplate`Array is too short (${"data"}), minimum ${"limit"}`,
  },
  ARRAY_UNIQUE: {
    severity: Severity.Critical,
    message: strTemplate`Array items are not unique (indexes ${"i"} and ${"j"})`,
  },
  ARRAY_ADDITIONAL_ITEMS: {
    severity: Severity.Critical,
    message: strTemplate`Additional items not allowed`,
  },
  ARRAY_LENGTH_LONG: {
    severity: Severity.Critical,
    message: strTemplate`Array is too long (${"data"}), maximum ${"limit"}`,
  },
  INVALID_TYPE: {
    severity: Severity.Critical,
    message: strTemplate`Expected type ${"type"} but found type ${"data"}`,
  },
  INVALID_FORMAT: {
    severity: Severity.Critical,
    message: strTemplate`Object didn't pass validation for format ${"format"}: ${"data"}`,
  },
  PATTERN: {
    severity: Severity.Critical,
    message: strTemplate`String does not match pattern ${"pattern"}: ${"data"}`,
  },
  MULTIPLE_OF: {
    severity: Severity.Critical,
    message: strTemplate`Value ${"data"} is not a multiple of ${"multipleOf"}`,
  },
  ENUM_CASE_MISMATCH: {
    severity: Severity.Critical,
    message: strTemplate`Enum does not match case for: ${"data"}`,
  },
  ENUM_MISMATCH: {
    severity: Severity.Critical,
    message: strTemplate`No enum match for: ${"data"}`,
  },
  MAX_LENGTH: {
    severity: Severity.Critical,
    message: strTemplate`String is too long (${"data"} chars), maximum ${"limit"}`,
  },
  MIN_LENGTH: {
    severity: Severity.Critical,
    message: strTemplate`String is too short (${"data"} chars), minimum ${"limit"}`,
  },
  MINIMUM: {
    severity: Severity.Critical,
    message: strTemplate`Value ${"data"} is less than minimum ${"limit"}`,
  },
  MAXIMUM: {
    severity: Severity.Critical,
    message: strTemplate`Value ${"data"} is greater than maximum ${"limit"}`,
  },
  MINIMUM_EXCLUSIVE: {
    severity: Severity.Critical,
    message: strTemplate`Value ${"data"} is equal or less than exclusive minimum ${"limit"}`,
  },
  MAXIMUM_EXCLUSIVE: {
    severity: Severity.Critical,
    message: strTemplate`Value ${"data"} is equal or greater than exclusive maximum ${"limit"}`,
  },
} as const;

// used in both example validation and api validation
export const trafficValidationErrors = {
  ...schemaValidationErrors,

  READONLY_PROPERTY_NOT_ALLOWED_IN_REQUEST: {
    severity: Severity.Critical,
    message: strTemplate`ReadOnly property "${"key"}" cannot be sent in the request`,
  },
  WRITEONLY_PROPERTY_NOT_ALLOWED_IN_RESPONSE: {
    severity: Severity.Critical,
    message: strTemplate`Write-only property "${"key"}" is not allowed in the response`,
  },
  SECRET_PROPERTY: {
    severity: Severity.Critical,
    message: strTemplate`Secret property "${"key"}" cannot be sent in the response`,
  },
  INVALID_RESPONSE_CODE: {
    severity: Severity.Critical,
    message: strTemplate`The swagger file does not define '${"statusCode"}' response code`,
  },
  INVALID_CONTENT_TYPE: {
    severity: Severity.Error,
    message: strTemplate`Invalid Content-Type (${"contentType"}).  These are supported: ${"supported"}`,
  },
  MISSING_REQUIRED_PARAMETER: {
    severity: Severity.Critical,
    message: strTemplate`Value is required but was not provided`,
  },
  INVALID_RESPONSE_BODY: {
    severity: Severity.Critical,
    message: strTemplate`Body is required in response but not provided`,
  },
  INVALID_RESPONSE_HEADER: {
    severity: Severity.Error,
    message: strTemplate`Header ${"missingProperty"} is required in response but not provided`,
  },
  MISSING_RESOURCE_ID: {
    severity: Severity.Critical,
    message: strTemplate`id is required to return in response of GET/PUT resource calls but not being provided`,
  },
  LRO_RESPONSE_CODE: {
    severity: Severity.Critical,
    message: strTemplate`Respond to the initial request of a long running operation, Patch/Post call must return 201 or 202, Delete call must return 202 or 204, Put call must return 202 or 201 or 200, but ${"statusCode"} being returned`,
  },
  LRO_RESPONSE_HEADER: {
    severity: Severity.Critical,
    message: strTemplate`Long running operation should return ${"header"} in header but not provided`,
  },
  INVALID_REQUEST_PARAMETER: {
    severity: Severity.Critical,
    message: strTemplate`The type of request parameter ${"param"} is invalid`,
  },
} as const;

// used in semantic validation only
export const semanticValidationErrors = {
  ...schemaValidationErrors,

  JSON_PARSING_ERROR: {
    severity: Severity.Critical,
    message: strTemplate`Json parsing error: ${"details"}`,
  },
  OBJECT_MISSING_REQUIRED_PROPERTY_DEFINITION: {
    severity: Severity.Critical,
    message: strTemplate`Missing required property definition: ${"property"}`,
  },
  OBJECT_MISSING_REQUIRED_PROPERTY_SCHEMA: {
    severity: Severity.Critical,
    message: strTemplate`Missing required property: ${"property"}`,
  },
  DISCRIMINATOR_NOT_REQUIRED: {
    severity: Severity.Critical,
    message: strTemplate`Discriminator must be a required property.`,
    id: "OAV131",
  },
  INVALID_DISCRIMINATOR_TYPE: {
    severity: Severity.Critical,
    message: strTemplate`The property type of discriminator must be string: ${"property"}`,
    id: "OAV132",
  },
  INVALID_XMS_DISCRIMINATOR_VALUE: {
    severity: Severity.Critical,
    message: strTemplate`The value of x-ms-dicriminator-value is not in the discriminator enum list: ${"value"}`,
    id: "OAV133",
  },
  DISCRIMINATOR_PROPERTY_NOT_FOUND: {
    severity: Severity.Critical,
    message: strTemplate`Missing discriminator in base model. This derived model has x-ms-dicriminator-value: ${"value"}`,
    id: "OAV134",
  },
  MULTIPLE_BODY_PARAMETERS: {
    severity: Severity.Critical,
    message: strTemplate`Operation cannot have multiple body parameters`,
  },
  INVALID_PARAMETER_COMBINATION: {
    severity: Severity.Critical,
    message: strTemplate`Operation cannot have a body parameter and a formData parameter`,
  },
  DUPLICATE_OPERATIONID: {
    severity: Severity.Critical,
    message: strTemplate`Cannot have multiple operations with the same operationId: ${"operationId"}`,
  },
  DUPLICATE_PARAMETER: {
    severity: Severity.Critical,
    message: strTemplate`Operation cannot have duplicate parameters: ${"name"}`,
  },
  EMPTY_PATH_PARAMETER_DECLARATION: {
    severity: Severity.Critical,
    message: strTemplate`Path parameter declaration cannot be empty: ${"pathTemplate"}`,
  },
  EQUIVALENT_PATH: {
    severity: Severity.Critical,
    message: strTemplate`Equivalent path already exists: ${"pathTemplate"}`,
  },
  MISSING_PATH_PARAMETER_DECLARATION: {
    severity: Severity.Critical,
    message: strTemplate`Path parameter is defined but is not declared: ${"name"}`,
  },
  MISSING_PATH_PARAMETER_DEFINITION: {
    severity: Severity.Critical,
    message: strTemplate`Path parameter is declared but is not defined: ${"name"}`,
  },
} as const;

// used in api validation only
export const apiValidationRuntimeErrors = {
  OPERATION_NOT_FOUND_IN_CACHE: {
    severity: Severity.Critical,
    message: strTemplate`Could not find best match operation for verb "${"requestMethod"}" for api-version "${"apiVersion"}" and provider "${"providerNamespace"}" in the cache.`,
  },
  OPERATION_NOT_FOUND_IN_CACHE_WITH_VERB: {
    severity: Severity.Critical,
    message: strTemplate`Could not find any methods with verb "${"requestMethod"}" for api-version "${"apiVersion"}" and provider "${"providerNamespace"}" in the cache.`,
  },
  OPERATION_NOT_FOUND_IN_CACHE_WITH_API: {
    severity: Severity.Critical,
    message: strTemplate`Could not find exact api-version "${"apiVersion"}" for provider "${"providerNamespace"}" in the cache.`,
  },
  OPERATION_NOT_FOUND_IN_CACHE_WITH_PROVIDER: {
    severity: Severity.Critical,
    message: strTemplate`Could not find provider "${"providerNamespace"}" in the cache.`,
  },
} as const;

export const roundTripValidationErrors = {
  ROUNDTRIP_INCONSISTENT_PROPERTY: {
    severity: Severity.Critical,
    message: strTemplate`The property's value '${"getValue"}' in the GET response is different from what was set '${"putValue"}' in the preceding PUT request.`,
  },
  ROUNDTRIP_MISSING_PROPERTY: {
    severity: Severity.Critical,
    message: strTemplate`The property '${"property"}' is present in the PUT request but is either never returned in the GET response or is returned with a null value.`,
  },
  ROUNDTRIP_ADDITIONAL_PROPERTY: {
    severity: Severity.Critical,
    message: strTemplate`The property '${"property"}' is returned in the GET response, but it is not declared in the PUT request.`,
  },
};

export const apiValidationErrors = {
  ...trafficValidationErrors,
  ...apiValidationRuntimeErrors,
  ...roundTripValidationErrors,

  MULTIPLE_OPERATIONS_FOUND: {
    severity: Severity.Critical,
    message: strTemplate`multiple operations matched from the operations cache`,
  },
  PII_MISMATCH: {
    severity: Severity.Warning,
    message: strTemplate`The value contains PII data`,
  },
};

export const modelValidationErrors = {
  ...trafficValidationErrors,

  XMS_EXAMPLE_NOTFOUND_ERROR: {
    severity: Severity.Critical,
    message: strTemplate`x-ms-example not found in ${"operationId"}.`,
    id: "OAV107",
  },
  REQUIRED_PARAMETER_EXAMPLE_NOT_FOUND: {
    severity: Severity.Critical,
    message: strTemplate`In operation ${"operationId"}, parameter ${"name"} is required in the swagger spec but is not present in the provided example parameter values.`,
    id: "OAV105",
  },
  DOUBLE_FORWARD_SLASHES_IN_URL: {
    severity: Severity.Critical,
    message: strTemplate`In operation ${"operationId"}, example for parameter ${"parameterName"}: ${"parameterValue"} starts with a forward slash and the path template: ${"pathTemplate"} contains a forward slash before the parameter starts. This will cause double forward slashes in the request url. Thus making it incorrect.`,
    id: "OAV129",
  },
  RESPONSE_STATUS_CODE_NOT_IN_SPEC: {
    severity: Severity.Critical,
    message: strTemplate`Response statusCode ${"exampleResponseStatusCode"} for operation ${"operationId"} is provided in exampleResponseValue, however it is not present in the swagger spec.`,
    id: "OAV112",
  },
  RESPONSE_SCHEMA_NOT_IN_SPEC: {
    severity: Severity.Critical,
    message: strTemplate`Response statusCode ${"exampleResponseStatusCode"} for operation ${"operationId"} has response body provided in the example, however the response does not have a "schema" defined in the swagger spec.`,
    id: "OAV112",
  },
  RESPONSE_BODY_NOT_IN_EXAMPLE: {
    severity: Severity.Critical,
    message: strTemplate`Response statusCode ${"exampleResponseStatusCode"} for operation ${"operationId"} has no response body provided in the example, however the response does have a "schema" defined in the swagger spec.`,
    id: "OAV130",
  },
  RESPONSE_STATUS_CODE_NOT_IN_EXAMPLE: {
    severity: Severity.Critical,
    message: strTemplate`Following response status codes ${"statusCodeInSwagger"} for operation ${"operationId"} were present in the swagger spec, however they were not present in x-ms-examples. Please provide them.`,
    id: "OAV111",
  },
};

export const ErrorCodeConstants = {
  XMS_EXAMPLE_NOTFOUND_ERROR: "XMS_EXAMPLE_NOTFOUND_ERROR",
  REQUIRED_PARAMETER_EXAMPLE_NOT_FOUND: "REQUIRED_PARAMETER_EXAMPLE_NOT_FOUND",
  DOUBLE_FORWARD_SLASHES_IN_URL: "DOUBLE_FORWARD_SLASHES_IN_URL",
  RESPONSE_STATUS_CODE_NOT_IN_SPEC: "RESPONSE_STATUS_CODE_NOT_IN_SPEC",
  RESPONSE_SCHEMA_NOT_IN_SPEC: "RESPONSE_SCHEMA_NOT_IN_SPEC",
  RESPONSE_BODY_NOT_IN_EXAMPLE: "RESPONSE_BODY_NOT_IN_EXAMPLE",
  RESPONSE_STATUS_CODE_NOT_IN_EXAMPLE: "RESPONSE_STATUS_CODE_NOT_IN_EXAMPLE",
  INVALID_TYPE: "INVALID_TYPE",
  INTERNAL_ERROR: "INTERNAL_ERROR",
  RUNTIME_ERROR: "RUNTIME_ERROR",
};
