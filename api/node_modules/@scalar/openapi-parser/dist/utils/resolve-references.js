import { isObject } from "@scalar/helpers/object/is-object";
import { ERRORS } from "../configuration/index.js";
import { getEntrypoint } from "./get-entrypoint.js";
import { getSegmentsFromPath } from "./get-segments-from-path.js";
import { makeFilesystem } from "./make-filesystem.js";
function resolveReferences(input, options, file, errors = []) {
  const clonedInput = structuredClone(input);
  const filesystem = makeFilesystem(clonedInput);
  const entrypoint = getEntrypoint(filesystem);
  const finalInput = file?.specification ?? entrypoint.specification;
  if (!isObject(finalInput)) {
    if (options?.throwOnError) {
      throw new Error(ERRORS.NO_CONTENT);
    }
    return {
      valid: false,
      errors,
      schema: finalInput
    };
  }
  dereference(finalInput, filesystem, file ?? entrypoint, /* @__PURE__ */ new WeakSet(), errors, options);
  errors = errors.filter(
    (error, index, self) => index === self.findIndex((t) => t.message === error.message && t.code === error.code)
  );
  return {
    valid: errors.length === 0,
    errors,
    schema: finalInput
  };
}
function dereference(schema, filesystem, entrypoint, resolvedSchemas, errors, options) {
  if (schema === null || resolvedSchemas.has(schema)) {
    return;
  }
  resolvedSchemas.add(schema);
  function resolveExternal(externalFile) {
    dereference(externalFile.specification, filesystem, externalFile, resolvedSchemas, errors, options);
    return externalFile;
  }
  const processedRefs = /* @__PURE__ */ new Set();
  while (schema.$ref !== void 0) {
    const selfReferenceDetected = processedRefs.has(schema.$ref);
    if (selfReferenceDetected) {
      errors.push({
        code: "SELF_REFERENCE",
        message: ERRORS.SELF_REFERENCE.replace("%s", schema.$ref)
      });
      delete schema.$ref;
      break;
    }
    processedRefs.add(schema.$ref);
    const resolved = resolveUri(schema.$ref, options, entrypoint, filesystem, resolveExternal, errors);
    if (typeof resolved !== "object" || resolved === null) {
      break;
    }
    const dereferencedRef = schema.$ref;
    delete schema.$ref;
    for (const key of Object.keys(resolved)) {
      if (schema[key] === void 0) {
        schema[key] = resolved[key];
      }
    }
    if (dereferencedRef) {
      options?.onDereference?.({ schema, ref: dereferencedRef });
    }
  }
  for (const value of Object.values(schema)) {
    if (typeof value === "object" && value !== null) {
      dereference(value, filesystem, entrypoint, resolvedSchemas, errors, options);
    }
  }
}
function resolveUri(uri, options, file, filesystem, resolve, errors) {
  if (typeof uri !== "string") {
    if (options?.throwOnError) {
      throw new Error(ERRORS.INVALID_REFERENCE.replace("%s", uri));
    }
    errors.push({
      code: "INVALID_REFERENCE",
      message: ERRORS.INVALID_REFERENCE.replace("%s", uri)
    });
    return void 0;
  }
  const [prefix, path] = uri.split("#", 2);
  const isDifferentFile = prefix !== file.filename;
  if (prefix && isDifferentFile) {
    const externalReference = filesystem.find((entry) => {
      return entry.filename === prefix;
    });
    if (!externalReference) {
      if (options?.throwOnError) {
        throw new Error(ERRORS.EXTERNAL_REFERENCE_NOT_FOUND.replace("%s", prefix));
      }
      errors.push({
        code: "EXTERNAL_REFERENCE_NOT_FOUND",
        message: ERRORS.EXTERNAL_REFERENCE_NOT_FOUND.replace("%s", prefix)
      });
      return void 0;
    }
    if (path === void 0) {
      return externalReference.specification;
    }
    return resolveUri(`#${path}`, options, resolve(externalReference), filesystem, resolve, errors);
  }
  const segments = getSegmentsFromPath(path);
  try {
    return segments.reduce((acc, key) => {
      return acc[key];
    }, file.specification);
  } catch (_error) {
    if (options?.throwOnError) {
      throw new Error(ERRORS.INVALID_REFERENCE.replace("%s", uri));
    }
    errors.push({
      code: "INVALID_REFERENCE",
      message: ERRORS.INVALID_REFERENCE.replace("%s", uri)
    });
  }
  return void 0;
}
export {
  resolveReferences
};
//# sourceMappingURL=resolve-references.js.map
