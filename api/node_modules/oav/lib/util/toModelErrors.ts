import { IterableEx, map } from "@azure-tools/openapi-tools-common";

import { ModelValidationError } from "./modelValidationError";
import { errorCodeToErrorMetadata, ExtendedErrorCode } from "./validationError";
import { ValidationResultSource } from "./validationResultSource";

/**
 * Transforms serialized errors to ModelValidationError
 */
export function toModelErrors(
  processedErrors: Iterable<ModelValidationError>,
  operationId: string,
  scenario: string,
  source: ValidationResultSource,
  responseCode: string
): IterableEx<ModelValidationError> {
  return map(processedErrors, (value) => {
    if (value.code === undefined) {
      value.code = "INTERNAL_ERROR";
    }
    const severity = errorCodeToErrorMetadata(value.code as ExtendedErrorCode).severity;
    const modelError: ModelValidationError = {
      operationId,
      scenario,
      source,
      responseCode,
      severity,
      code: value.code,
      details: value,
    };
    return modelError;
  });
}
