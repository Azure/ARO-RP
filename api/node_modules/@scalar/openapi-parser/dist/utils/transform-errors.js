import { betterAjvErrors } from "./betterAjvErrors/index.js";
function transformErrors(specification, errors) {
  if (typeof errors === "string") {
    return [
      {
        message: errors
      }
    ];
  }
  if (!specification || typeof specification !== "object") {
    return [
      {
        message: "Invalid specification"
      }
    ];
  }
  let processedErrors;
  try {
    processedErrors = betterAjvErrors(specification, null, errors, {
      indent: 2
    }).map((error) => {
      error.message = error.message.trim();
      return error;
    });
  } catch (error) {
    console.error(error);
    if (Array.isArray(errors)) {
      return errors.map((err) => {
        let message = err.message || "Validation error";
        if (err.keyword === "additionalProperties" && err.params?.additionalProperty) {
          message = `Property ${err.params.additionalProperty} is not expected to be here`;
        }
        return {
          message,
          path: err.dataPath || err.instancePath
        };
      });
    }
    return [
      {
        message: "Validation failed"
      }
    ];
  }
  const seen = /* @__PURE__ */ new Set();
  return processedErrors.filter((error) => {
    const key = `${error.message}||${error.path}`;
    if (seen.has(key)) {
      return false;
    }
    seen.add(key);
    return true;
  });
}
export {
  transformErrors
};
//# sourceMappingURL=transform-errors.js.map
