// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
import { FilePosition, flatMap, fold, StringMap } from "@azure-tools/openapi-tools-common";
import { JSONPath } from "jsonpath-plus";
import _ from "lodash";
import { Severity } from "./severity";

/**
 * @class
 * Error that results from validations.
 */
interface ErrorCodeMetadata {
  readonly severity: Severity;
  readonly docUrl: string;
}

export type ValidationErrorMetadata = ErrorCodeMetadata & { code: ExtendedErrorCode };

export type ExtendedErrorCode = ErrorCode | WrapperErrorCode | RuntimeErrorCode;
export type ErrorCode = keyof typeof errorConstants;
export type WrapperErrorCode = keyof typeof wrapperErrorConstants;
export type RuntimeErrorCode = keyof typeof runtimeErrorConstants;

const errorConstants = {
  INVALID_TYPE: {
    severity: Severity.Critical,
    docUrl: "",
  },
  INVALID_FORMAT: { severity: Severity.Critical, docUrl: "" },
  ENUM_MISMATCH: { severity: Severity.Critical, docUrl: "" },
  ENUM_CASE_MISMATCH: { severity: Severity.Error, docUrl: "" },
  PII_MISMATCH: { severity: Severity.Warning, docUrl: "" },
  NOT_PASSED: { severity: Severity.Critical, docUrl: "" },
  ARRAY_LENGTH_SHORT: { severity: Severity.Critical, docUrl: "" },
  ARRAY_LENGTH_LONG: { severity: Severity.Critical, docUrl: "" },
  ARRAY_UNIQUE: { severity: Severity.Critical, docUrl: "" },
  ARRAY_ADDITIONAL_ITEMS: {
    severity: Severity.Critical,
    docUrl: "",
  },
  MULTIPLE_OF: { severity: Severity.Critical, docUrl: "" },
  MINIMUM: { severity: Severity.Critical, docUrl: "" },
  MINIMUM_EXCLUSIVE: { severity: Severity.Critical, docUrl: "" },
  MAXIMUM: { severity: Severity.Critical, docUrl: "" },
  MAXIMUM_EXCLUSIVE: { severity: Severity.Critical, docUrl: "" },
  READONLY_PROPERTY_NOT_ALLOWED_IN_REQUEST: {
    severity: Severity.Critical,
    docUrl: "",
  },
  UNRESOLVABLE_REFERENCE: { severity: Severity.Critical, docUrl: "" },
  SECRET_PROPERTY: { severity: Severity.Critical, docUrl: "" },
  WRITEONLY_PROPERTY_NOT_ALLOWED_IN_RESPONSE: { severity: Severity.Critical, docUrl: "" },
  OBJECT_PROPERTIES_MINIMUM: {
    severity: Severity.Critical,
    docUrl: "",
  },
  OBJECT_PROPERTIES_MAXIMUM: {
    severity: Severity.Critical,
    docUrl: "",
  },
  OBJECT_MISSING_REQUIRED_PROPERTY: {
    severity: Severity.Critical,
    docUrl: "",
  },
  MISSING_REQUIRED_PARAMETER: {
    severity: Severity.Critical,
    docUrl: "",
  },
  OBJECT_ADDITIONAL_PROPERTIES: {
    severity: Severity.Critical,
    docUrl: "",
  },
  OBJECT_DEPENDENCY_KEY: { severity: Severity.Warning, docUrl: "" },
  MIN_LENGTH: { severity: Severity.Critical, docUrl: "" },
  MAX_LENGTH: { severity: Severity.Critical, docUrl: "" },
  PATTERN: { severity: Severity.Critical, docUrl: "" },
  INVALID_RESPONSE_CODE: { severity: Severity.Critical, docUrl: "" },
  INVALID_CONTENT_TYPE: { severity: Severity.Error, docUrl: "" },
  DISCRIMINATOR_VALUE_NOT_FOUND: { severity: Severity.Critical, docUrl: "" },
  INVALID_RESPONSE_HEADER: { severity: Severity.Error, docUrl: "" },
  INVALID_RESPONSE_BODY: { severity: Severity.Critical, docUrl: "" },
  MISSING_RESOURCE_ID: { severity: Severity.Critical, docUrl: "" },
  LRO_RESPONSE_CODE: { severity: Severity.Critical, docUrl: "" },
  LRO_RESPONSE_HEADER: { severity: Severity.Critical, docUrl: "" },
};

const wrapperErrorConstants = {
  ANY_OF_MISSING: { severity: Severity.Critical, docUrl: "" },
  ONE_OF_MISSING: { severity: Severity.Critical, docUrl: "" },
  ONE_OF_MULTIPLE: { severity: Severity.Critical, docUrl: "" },
  MULTIPLE_OPERATIONS_FOUND: {
    severity: Severity.Critical,
    docUrl: "",
  },
  INVALID_RESPONSE_HEADER: {
    severity: Severity.Critical,
    docUrl: "",
  },
  INVALID_RESPONSE_BODY: { severity: Severity.Critical, docUrl: "" },
  INVALID_REQUEST_PARAMETER: {
    severity: Severity.Critical,
    docUrl: "",
  },
};

const runtimeErrorConstants = {
  OPERATION_NOT_FOUND_IN_CACHE: {
    severity: Severity.Critical,
    docUrl: "",
  },
  OPERATION_NOT_FOUND_IN_CACHE_WITH_VERB: {
    severity: Severity.Critical,
    docUrl: "",
  },
  OPERATION_NOT_FOUND_IN_CACHE_WITH_API: {
    severity: Severity.Critical,
    docUrl: "",
  },
  OPERATION_NOT_FOUND_IN_CACHE_WITH_PROVIDER: {
    severity: Severity.Critical,
    docUrl: "",
  },
  INTERNAL_ERROR: { severity: Severity.Critical, docUrl: "" },
};

export const allErrorConstants = {
  ...errorConstants,
  ...wrapperErrorConstants,
  ...runtimeErrorConstants,
};
/**
 * Gets the validation error metadata from an error code. If the code is unknown assume critical.
 */
export const errorCodeToErrorMetadata = (code: ExtendedErrorCode): ValidationErrorMetadata => ({
  ...(allErrorConstants[code] || {
    severity: Severity.Critical,
    docUrl: "",
  }),
  code,
});

export interface SourceLocation {
  readonly url: string;
  readonly jsonRef?: string;
  readonly jsonPath?: string;
  readonly position: {
    readonly column: number;
    readonly line: number;
  };
}

export interface RuntimeException {
  code: string;
  readonly message: string;
}

export interface NodeError<T extends NodeError<T>> {
  code?: string;
  message?: string;
  path?: string | string[];
  jsonPath?: string;
  schemaPath?: string;
  similarPaths?: string[];
  similarJsonPaths?: string[];
  errors?: T[];
  innerErrors?: T[];
  in?: string;
  name?: string;
  params?: unknown[];
  inner?: T[];
  title?: string;

  position?: FilePosition;
  url?: string;

  jsonPosition?: FilePosition;
  jsonUrl?: string;

  directives?: StringMap<unknown>;
}

export interface ValidationResult<T extends NodeError<T>> {
  readonly requestValidationResult: T;
  readonly responseValidationResult: T;
}

/**
 * Serializes error tree
 */
export function serializeErrors<T extends NodeError<T>>(node: T, path: string[]): T[] {
  if (isLeaf(node)) {
    if (isTrueError(node)) {
      setPathProperties(node, path);
      return [node];
    }
    return [];
  }

  if (node.path) {
    // in this case the path will be set to the url instead of the path to the property
    if (node.code === "INVALID_REQUEST_PARAMETER" && node.in === "body") {
      node.path = [];
    } else if (
      (node.in === "query" || node.in === "path") &&
      node.path[0] === "paths" &&
      node.name
    ) {
      // in this case we will want to normalize the path with the uri and the paramter name
      node.path = [node.path[1], node.name];
    }
    path = consolidatePath(path, node.path);
  }

  const serializedErrors = flatMap(node.errors, (validationError) =>
    serializeErrors(validationError, path)
  ).toArray();

  const serializedInner = fold(
    node.inner,
    (acc, validationError) => {
      const errs = serializeErrors(validationError, path);
      errs.forEach((err) => {
        const similarErr = acc.find((el) => areErrorsSimilar(err, el));
        if (similarErr && similarErr.path) {
          if (!similarErr.similarPaths) {
            similarErr.similarPaths = [];
          }
          similarErr.similarPaths.push(err.path as string);

          if (!similarErr.similarJsonPaths) {
            similarErr.similarJsonPaths = [];
          }
          similarErr.similarJsonPaths.push(err.jsonPath as string);
        } else {
          acc.push(err);
        }
      });
      return acc;
    },
    new Array<T>()
  );

  if (isDiscriminatorError(node)) {
    setPathProperties(node, path);
    node.inner = serializedInner;
    return [node];
  }
  return [...serializedErrors, ...serializedInner];
}

/**
 * Sets the path and jsonPath properties on an error node.
 */
function setPathProperties<T extends NodeError<T>>(node: T, path: string[]) {
  if (!node.path) {
    return;
  }
  let nodePath = typeof node.path === "string" ? [node.path] : node.path;

  if (
    node.code === "OBJECT_MISSING_REQUIRED_PROPERTY" ||
    node.code === "OBJECT_ADDITIONAL_PROPERTIES"
  ) {
    // For multiple missing/additional properties , each node would only contain one param.
    if (node.params && node.params.length > 0) {
      nodePath = nodePath.concat(node.params[0] as string);
    }
  }

  const pathSegments = consolidatePath(path, nodePath);
  pathSegments.unshift("$");
  node.path = pathSegments.join("/");
  node.jsonPath = (pathSegments.length && (JSONPath as any).toPathString(pathSegments)) || "";
}

/**
 * Checks if two errors are the same except their path.
 */
function areErrorsSimilar<T extends NodeError<T>>(node1: T, node2: T) {
  if (
    node1.code !== node2.code ||
    node1.title !== node2.title ||
    node1.message !== node2.message ||
    !arePathsSimilar(node1.path, node2.path)
  ) {
    return false;
  }

  if (!node1.inner && !node2.inner) {
    return true;
  }

  if (!node1.inner || !node2.inner || node1.inner.length !== node2.inner.length) {
    return false;
  }

  for (let i = 0; i < node1.inner.length; i++) {
    if (!areErrorsSimilar(node1.inner[i], node2.inner[i])) {
      return false;
    }
  }
  return true;
}

/**
 * Checks if paths differ only in indexes
 */
const arePathsSimilar = (
  path1: string | string[] | undefined,
  path2: string | string[] | undefined
) => {
  if (path1 === path2) {
    return true;
  }
  if (path1 === undefined || path2 === undefined) {
    return false;
  }

  const p1 = Array.isArray(path1) ? path1 : path1.split("/");
  const p2 = Array.isArray(path2) ? path2 : path2.split("/");

  return _.xor(p1, p2).every((v) => Number.isInteger(+v));
};

const isDiscriminatorError = <T extends NodeError<T>>(node: T) =>
  node.code === "ONE_OF_MISSING" && node.inner && node.inner.length > 0;

const isTrueError = <T extends NodeError<T>>(node: T): boolean =>
  // this is necessary to filter out extra errors coming from doing the ONE_OF transformation on
  // the models to allow "null"
  !(node.code === "INVALID_TYPE" && node.params && node.params[0] === "null");

const isLeaf = <T extends NodeError<T>>(node: T): boolean => !node.errors && !node.inner;

/**
 * Unifies a suffix path with a root path.
 */
function consolidatePath(path: string[], suffixPath: string | string[]): string[] {
  let newSuffixIndex = 0;
  let overlapIndex = path.lastIndexOf(suffixPath[newSuffixIndex]);
  let previousIndex = overlapIndex;

  if (overlapIndex === -1) {
    return path.concat(suffixPath);
  }

  for (newSuffixIndex = 1; newSuffixIndex < suffixPath.length; ++newSuffixIndex) {
    previousIndex = overlapIndex;
    overlapIndex = path.lastIndexOf(suffixPath[newSuffixIndex]);
    if (overlapIndex === -1 || overlapIndex !== previousIndex + 1) {
      break;
    }
  }
  let newPath: string[] = [];
  if (newSuffixIndex === suffixPath.length) {
    // if all elements are contained in the existing path, nothing to do.
    newPath = path.slice(0);
  } else if (overlapIndex === -1 && previousIndex === path.length - 1) {
    // if we didn't find element at x in the previous path and element at x -1 is the last one in
    // the path, append everything from x
    newPath = path.concat(suffixPath.slice(newSuffixIndex));
  } else {
    // otherwise it is not contained at all, so concat everything.
    newPath = path.concat(suffixPath);
  }

  return newPath;
}
