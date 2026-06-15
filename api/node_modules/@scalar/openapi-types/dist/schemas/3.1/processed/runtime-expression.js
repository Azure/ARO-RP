import { z } from "zod";
const isValidRuntimeExpression = (value) => {
  if (value.startsWith("$")) {
    return validatePureExpression(value);
  }
  if (value.includes("{")) {
    const expressions = value.match(/\{([^}]+)\}/g);
    if (!expressions) {
      return false;
    }
    return expressions.every((expr) => {
      const innerExpr = expr.slice(1, -1);
      return validatePureExpression(innerExpr);
    });
  }
  return false;
};
const validatePureExpression = (value) => {
  const expression = value.startsWith("$") ? value.slice(1) : value;
  if (["method", "url", "statusCode"].includes(expression)) {
    return true;
  }
  const [mainPart, jsonPointer] = expression.split("#");
  const [source, type, ...rest] = mainPart?.split(".") ?? [];
  if (!["request", "response"].includes(source ?? "")) {
    return false;
  }
  if (!["header", "query", "path", "body"].includes(type ?? "")) {
    return false;
  }
  if (type === "body") {
    if (jsonPointer === void 0) {
      return false;
    }
    if (jsonPointer === "" || jsonPointer === "/") {
      return true;
    }
    if (!jsonPointer.startsWith("/")) {
      return false;
    }
    const segments = jsonPointer.slice(1).split("/");
    return segments.every((segment) => {
      const decoded = segment.replace(/~1/g, "/").replace(/~0/g, "~");
      return decoded.length > 0;
    });
  }
  if (type === "header") {
    const headerName = rest.join(".");
    return !headerName.includes(" ");
  }
  return rest.length === 1;
};
const RuntimeExpressionSchema = z.string().refine(isValidRuntimeExpression, {
  message: `Invalid runtime expression. Runtime expressions must:
  - Start with $ or contain expressions in curly braces {}
  - Use one of: $method, $url, $statusCode
  - Or follow pattern: $request|response.(header|query|path|body)
  - For body refs, include valid JSON pointer (e.g. #/user/id)
  - For headers, use valid header names without spaces
  Example valid expressions:
  - Pure: $method, $request.path.id, $response.body#/status
  - Embedded: "Hello {$request.body#/name}!", "Status: {$statusCode}"`
});
export {
  RuntimeExpressionSchema
};
//# sourceMappingURL=runtime-expression.js.map
