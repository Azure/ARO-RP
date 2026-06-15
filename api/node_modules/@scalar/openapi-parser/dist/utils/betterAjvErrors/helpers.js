import {
  concatAll,
  getChildren,
  getErrors,
  getSiblings,
  isAdditionalPropertiesError,
  isAnyOfError,
  isEnumError,
  isIfError,
  isOneOfError,
  isRequiredError,
  isUnevaluatedPropertiesError,
  notUndefined
} from "./utils.js";
import {
  AdditionalPropValidationError,
  DefaultValidationError,
  EnumValidationError,
  PatternValidationError,
  RequiredValidationError,
  UnevaluatedPropValidationError
} from "./validation-errors/index.js";
const JSON_POINTERS_REGEX = /\/[\w_-]+(\/\d+)?/g;
function makeTree(ajvErrors = []) {
  const root = { children: {} };
  ajvErrors.forEach((ajvError) => {
    const instancePath = typeof ajvError.instancePath !== "undefined" ? ajvError.instancePath : ajvError.dataPath;
    const paths = instancePath === "" ? [""] : instancePath.match(JSON_POINTERS_REGEX);
    if (paths) {
      paths.reduce((obj, path, i) => {
        obj.children[path] = obj.children[path] || { children: {}, errors: [] };
        if (i === paths.length - 1) {
          obj.children[path].errors.push(ajvError);
        }
        return obj.children[path];
      }, root);
    }
  });
  return root;
}
function filterRedundantErrors(root, parent, key) {
  const errors = getErrors(root);
  const hasOneOfError = errors.some(isOneOfError);
  const hasAnyOfError = errors.some(isAnyOfError);
  const hasRequiredError = errors.some(isRequiredError);
  const hasIfError = errors.some(isIfError);
  const hasAdditionalPropertiesError = errors.some(isAdditionalPropertiesError);
  const hasUnevaluatedPropertiesError = errors.some(isUnevaluatedPropertiesError);
  const hasChildren = Object.keys(root.children || {}).length > 0;
  if (hasIfError && (hasAdditionalPropertiesError || hasUnevaluatedPropertiesError)) {
    root.errors = errors.filter((error) => !isIfError(error));
  }
  if (hasOneOfError && hasRequiredError) {
    if (hasAdditionalPropertiesError || hasUnevaluatedPropertiesError) {
      root.errors = errors.filter((error) => !isRequiredError(error) && !isOneOfError(error));
    } else if (hasChildren) {
      delete root.errors;
    } else {
      root.errors = errors.filter((error) => isOneOfError(error));
    }
  } else if (hasOneOfError && !hasRequiredError && hasChildren) {
    delete root.errors;
  } else if (hasOneOfError && !hasRequiredError && !hasChildren) {
    const oneOfErrors = errors.filter(isOneOfError);
    if (oneOfErrors.length > 1) {
      root.errors = [oneOfErrors[0]];
    }
  } else if (hasRequiredError && !hasOneOfError) {
    root.errors = errors.filter(isRequiredError);
    root.children = {};
  }
  if (hasAnyOfError) {
    if (Object.keys(root.children).length > 0) {
      delete root.errors;
    }
  }
  if (root.errors?.length && getErrors(root).every(isEnumError)) {
    if (getSiblings(parent)(root).filter(notUndefined).some(getErrors)) {
      delete parent.children[key];
    }
  }
  Object.entries(root.children).forEach(([k, child]) => filterRedundantErrors(child, root, k));
}
function createErrorInstances(root, options) {
  const errors = getErrors(root);
  if (errors.length && errors.every(isEnumError)) {
    const uniqueValues = new Set(concatAll([])(errors.map((e) => e.params.allowedValues)));
    const allowedValues = [...uniqueValues];
    const error = errors[0];
    return [
      new EnumValidationError(
        {
          ...error,
          params: { allowedValues }
        },
        options
      )
    ];
  }
  return concatAll(
    errors.reduce((ret, error) => {
      switch (error.keyword) {
        case "additionalProperties":
          return ret.concat(new AdditionalPropValidationError(error, options));
        case "pattern":
          return ret.concat(new PatternValidationError(error, options));
        case "required":
          return ret.concat(new RequiredValidationError(error, options));
        case "unevaluatedProperties":
          return ret.concat(new UnevaluatedPropValidationError(error, options));
        default:
          return ret.concat(new DefaultValidationError(error, options));
      }
    }, [])
  )(getChildren(root).map((child) => createErrorInstances(child, options)));
}
function prettify(ajvErrors, options) {
  const tree = makeTree(ajvErrors || []);
  filterRedundantErrors(tree);
  return createErrorInstances(tree, options);
}
export {
  prettify as default
};
//# sourceMappingURL=helpers.js.map
