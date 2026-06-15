// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

import { StringMap } from "@azure-tools/openapi-tools-common";

export const xmsParameterizedHost = "x-ms-parameterized-host";

export const xmsPaths = "x-ms-paths";

export const xmsExamples = "x-ms-examples";

export const xmsSkipUrlEncoding = "x-ms-skip-url-encoding";

export const xmsLongRunningOperation = "x-ms-long-running-operation";

export const xmsLongRunningOperationOptions = "x-ms-long-running-operation-options";

export const xmsLongRunningOperationOptionsField = "final-state-via";

export const xmsDiscriminatorValue = "x-ms-discriminator-value";

export const xmsEnum = "x-ms-enum";

export const xmsMutability = "x-ms-mutability";

export const xmsAzureResource = "x-ms-azure-resource";

export const xmsSecret = "x-ms-secret";

export const xNullable = "x-nullable";

export const exampleInSpec = "example-in-spec";

export const xmsReadonlyRef = "x-ms-readonly-ref";

export const Errors = "Errors";

export const Warnings = "Warnings";

export const ErrorCodes = {
  InternalError: { name: "INTERNAL_ERROR", id: "OAV100" },
  InitializationError: { name: "INITIALIZATION_ERROR", id: "OAV101" },
  ResolveSpecError: { name: "RESOLVE_SPEC_ERROR", id: "OAV102" },
  RefNotFoundError: { name: "REF_NOTFOUND_ERROR", id: "OAV103" },
  JsonParsingError: { name: "JSON_PARSING_ERROR", id: "OAV104" },
  RequiredParameterExampleNotFound: {
    name: "REQUIRED_PARAMETER_EXAMPLE_NOT_FOUND",
    id: "OAV105",
  },
  ErrorInPreparingRequest: { name: "ERROR_IN_PREPARING_REQUEST", id: "OAV106" },
  XmsExampleNotFoundError: {
    name: "X-MS-EXAMPLE_NOTFOUND_ERROR",
    id: "OAV107",
  },
  ResponseValidationError: { name: "RESPONSE_VALIDATION_ERROR", id: "OAV108" },
  RequestValidationError: { name: "REQUEST_VALIDATION_ERROR", id: "OAV109" },
  RoundtripValidationError: { name: "ROUNDTRIP_VALIDATION_ERROR", id: "OAV135" },
  ResponseBodyValidationError: {
    name: "RESPONSE_BODY_VALIDATION_ERROR",
    id: "OAV110",
  },
  ResponseStatusCodeNotInExample: {
    name: "RESPONSE_STATUS_CODE_NOT_IN_EXAMPLE",
    id: "OAV111",
  },
  ResponseStatusCodeNotInSpec: {
    name: "RESPONSE_STATUS_CODE_NOT_IN_SPEC",
    id: "OAV112",
  },
  ResponseSchemaNotInSpec: {
    name: "RESPONSE_SCHEMA_NOT_IN_SPEC",
    id: "OAV113",
  },
  RequiredParameterNotInExampleError: {
    name: "REQUIRED_PARAMETER_NOT_IN_EXAMPLE_ERROR",
    id: "OAV114",
  },
  BodyParameterValidationError: {
    name: "BODY_PARAMETER_VALIDATION_ERROR",
    id: "OAV115",
  },
  TypeValidationError: { name: "TYPE_VALIDATION_ERROR", id: "OAV116" },
  ConstraintValidationError: {
    name: "CONSTRAINT_VALIDATION_ERROR",
    id: "OAV117",
  },
  StatuscodeNotInExampleError: {
    name: "STATUS_CODE_NOT_IN_EXAMPLE_ERROR",
    id: "OAV118",
  },
  SemanticValidationError: { name: "SEMANTIC_VALIDATION_ERROR", id: "OAV119" },
  MultipleOperationsFound: { name: "MULTIPLE_OPERATIONS_FOUND", id: "OAV120" },
  NoOperationFound: { name: "NO_OPERATION_FOUND", id: "OAV121" },
  IncorrectInput: { name: "INCORRECT_INPUT", id: "OAV122" },
  PotentialOperationSearchError: {
    name: "POTENTIAL_OPERATION_SEARCH_ERROR",
    id: "OAV123",
  },
  PathNotFoundInRequestUrl: {
    name: "PATH_NOT_FOUND_IN_REQUEST_URL",
    id: "OAV124",
  },
  OperationNotFoundInCache: {
    name: "OPERATION_NOT_FOUND_IN_CACHE",
    id: "OAV125",
  },
  OperationNotFoundInCacheWithVerb: {
    name: "OPERATION_NOT_FOUND_IN_CACHE_WITH_VERB",
    id: "OAV126",
  }, // Implies we found correct api-version + provider in cache
  OperationNotFoundInCacheWithApi: {
    name: "OPERATION_NOT_FOUND_IN_CACHE_WITH_API",
    id: "OAV127",
  }, // Implies we found correct provider in cache
  OperationNotFoundInCacheWithProvider: {
    name: "OPERATION_NOT_FOUND_IN_CACHE_WITH_PROVIDER",
    id: "OAV128",
  }, // Implies we never found correct provider in cache
  DoubleForwardSlashesInUrl: {
    name: "DOUBLE_FORWARD_SLASHES_IN_URL",
    id: "OAV129",
  },
  ResponseBodyNotInExample: {
    name: "RESPONSE_BODY_NOT_IN_EXAMPLE",
    id: "OAV130",
  },
  DiscriminatorNotRequired: {
    name: "DISCRIMINATOR_NOT_REQUIRED",
    id: "OAV131",
  },
  IncorrectProvisioningState: {
    name: "INCORRECT_PROVISIONING_STATE",
    id: "OAV132",
  },
  RoundtripInconsistentProperty: {
    name: "ROUNDTRIP_INCONSISTENT_PROPERTY",
    id: "OAV133",
  },
  RecommendUsingBooleanType: {
    name: "RECOMMENDED_BOOLEAN_TYPE",
    id: "OAV134",
  },
};

export const knownTitleToResourceProviders: StringMap<string> = {
  ResourceManagementClient: "Microsoft.Resources",
};

export const EnvironmentVariables = {
  ClientId: "CLIENT_ID",
  Domain: "DOMAIN",
  ApplicationSecret: "APPLICATION_SECRET",
  AzureSubscriptionId: "AZURE_SUBSCRIPTION_ID",
  AzureLocation: "AZURE_LOCATION",
  AzureResourcegroup: "AZURE_RESOURCE_GROUP",
};

export const unknownResourceProvider = "microsoft.unknown";
export const unknownApiVersion = "unknown-api-version";
export const unknownOperationId = "unknownOperationId";
export const unknownResourceType = "unknownResourceType";

// Data-plane and Azure Stack swaggers can be skipped for performance boost as ARM don't use them
export const DefaultConfig = {
  ExcludedSwaggerPathsPattern: [
    "**/examples/**/*",
    "**/scenarios/**/*",
    "**/restler/**/*",
    "**/quickstart-templates/**/*",
    "**/schema/**/*",
    "**/live/**/*",
    "**/wire-format/**/*",
    "**/azurestack/**/*",
    "**/applicationinsights/data-plane/**/*",
    "**/batch/data-plane/**/*",
    "**/cognitiveservices/data-plane/**/*",
    "**/containerregistry/data-plane/**/*",
    "**/datalake-analytics/data-plane/**/*",
    "**/datalake-store/data-plane/**/*",
    "**/eventgrid/data-plane/**/*",
    "**/graphrbac/data-plane/**/*",
    "**/hdinsight/data-plane/**/*",
    "**/imds/data-plane/**/*",
    "**/iotcentral/data-plane/**/*",
    "**/keyvault/data-plane/**/*",
    "**/machinelearningservices/data-plane/**/*",
    "**/monitor/data-plane/**/*",
    "**/operationalinsights/data-plane/**/*",
    "**/search/data-plane/**/*",
    "**/servicefabric/data-plane/**/*",
    "**/storage/data-plane/**/*",
    "**/timeseriesinsights/data-plane/**/*",
  ],
  ExcludedExamplesAndCommonFiles: [
    "**/examples/**/*",
    "**/scenarios/**/*",
    "**/restler/**/*",
    "**/quickstart-templates/**/*",
    "**/schema/**/*",
    "**/live/**/*",
    "**/wire-format/**/*",
    "**/azurestack/**/*",
  ],
};
