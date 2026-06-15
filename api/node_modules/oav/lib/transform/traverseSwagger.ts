import {
  LowerHttpMethods,
  lowerHttpMethods,
  Operation,
  Path,
  Response,
  SwaggerSpec,
} from "../swagger/swaggerTypes";

export const traverseSwagger = (
  spec: SwaggerSpec,
  visitors: {
    // return false to skip following level
    onPath?: (path: Path, pathTemplate: string) => boolean | void;
    onOperation?: (operation: Operation, path: Path, method: LowerHttpMethods) => boolean | void;
    onResponse?: (response: Response, operation: Operation, path: Path, statusCode: string) => void;
  }
) => {
  const { onPath, onOperation, onResponse } = visitors;
  const skipOperation = onOperation === undefined && onResponse === undefined;
  const skipResponse = onResponse === undefined;

  if (!spec.paths) {
    console.error("error");
  }

  for (const pathTemplate of Object.keys(spec.paths)) {
    const path = spec.paths[pathTemplate];
    if ((onPath !== undefined && onPath(path, pathTemplate) === false) || skipOperation) {
      continue;
    }

    for (const m of Object.keys(path)) {
      const method = m as LowerHttpMethods;
      if (!lowerHttpMethods.includes(method as LowerHttpMethods)) {
        continue;
      }

      const operation = path[method]!;
      if (
        (onOperation !== undefined && onOperation(operation, path, method) === false) ||
        skipResponse
      ) {
        continue;
      }

      for (const statusCode of Object.keys(operation.responses)) {
        const response = operation.responses[statusCode];
        onResponse!(response, operation, path, statusCode);
      }
    }
  }
};

export const traverseSwaggers = (
  specs: SwaggerSpec[],
  visitors: {
    // return false to skip following level
    onPath?: (path: Path, pathTemplate: string) => boolean | void;
    onOperation?: (operation: Operation, path: Path, method: LowerHttpMethods) => boolean | void;
    onResponse?: (response: Response, operation: Operation, path: Path, statusCode: string) => void;
  }
) => {
  specs.forEach((spec) => traverseSwagger(spec, visitors));
};

export const traverseSwaggerAsync = async (
  spec: SwaggerSpec,
  visitors: {
    // return false to skip following level
    onPath?: (path: Path, pathTemplate: string) => Promise<boolean | void>;
    onOperation?: (
      operation: Operation,
      path: Path,
      method: LowerHttpMethods
    ) => Promise<boolean | void>;
    onResponse?: (
      response: Response,
      operation: Operation,
      path: Path,
      statusCode: string
    ) => Promise<void>;
  }
) => {
  const { onPath, onOperation, onResponse } = visitors;
  const skipOperation = onOperation === undefined && onResponse === undefined;
  const skipResponse = onResponse === undefined;

  if (!spec.paths) {
    console.error("error");
  }

  for (const pathTemplate of Object.keys(spec.paths)) {
    const path = spec.paths[pathTemplate];
    if ((onPath !== undefined && (await onPath(path, pathTemplate)) === false) || skipOperation) {
      continue;
    }

    for (const m of Object.keys(path)) {
      const method = m as LowerHttpMethods;
      if (!lowerHttpMethods.includes(method as LowerHttpMethods)) {
        continue;
      }

      const operation = path[method]!;
      if (
        (onOperation !== undefined && (await onOperation(operation, path, method)) === false) ||
        skipResponse
      ) {
        continue;
      }

      for (const statusCode of Object.keys(operation.responses)) {
        const response = operation.responses[statusCode];
        await onResponse!(response, operation, path, statusCode);
      }
    }
  }
};
