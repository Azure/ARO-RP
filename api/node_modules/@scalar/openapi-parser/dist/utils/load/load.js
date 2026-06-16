import { ERRORS } from "../../configuration/index.js";
import { getEntrypoint } from "../../utils/get-entrypoint.js";
import { getListOfReferences } from "../../utils/get-list-of-references.js";
import { makeFilesystem } from "../../utils/make-filesystem.js";
import { normalize } from "../../utils/normalize.js";
async function load(value, options) {
  const errors = [];
  if (options?.filesystem?.find((entry) => entry.filename === value)) {
    return {
      specification: getEntrypoint(options.filesystem)?.specification,
      filesystem: options.filesystem,
      errors
    };
  }
  const plugin = options?.plugins?.find((thisPlugin) => thisPlugin.check(value));
  let content;
  if (plugin) {
    try {
      content = normalize(await plugin.get(value));
    } catch (_error) {
      if (options?.throwOnError) {
        throw new Error(ERRORS.EXTERNAL_REFERENCE_NOT_FOUND.replace("%s", value));
      }
      errors.push({
        code: "EXTERNAL_REFERENCE_NOT_FOUND",
        message: ERRORS.EXTERNAL_REFERENCE_NOT_FOUND.replace("%s", value)
      });
      return {
        specification: null,
        filesystem: [],
        errors
      };
    }
  } else {
    content = normalize(value);
  }
  if (content === void 0) {
    if (options?.throwOnError) {
      throw new Error("No content to load");
    }
    errors.push({
      code: "NO_CONTENT",
      message: ERRORS.NO_CONTENT
    });
    return {
      specification: null,
      filesystem: [],
      errors
    };
  }
  let filesystem = makeFilesystem(content, {
    filename: options?.filename ?? null
  });
  const newEntry = options?.filename ? filesystem.find((entry) => entry.filename === options?.filename) : getEntrypoint(filesystem);
  const listOfReferences = newEntry.references ?? getListOfReferences(content);
  if (listOfReferences.length === 0) {
    return {
      specification: getEntrypoint(filesystem)?.specification,
      filesystem,
      errors
    };
  }
  for (const reference of listOfReferences) {
    const otherPlugin = options?.plugins?.find((thisPlugin) => thisPlugin.check(reference));
    if (!otherPlugin) {
      continue;
    }
    const target = otherPlugin.check(reference) && otherPlugin.resolvePath ? otherPlugin.resolvePath(value, reference) : reference;
    if (filesystem.find((entry) => entry.filename === reference)) {
      continue;
    }
    const { filesystem: referencedFiles, errors: newErrors } = await load(target, {
      ...options,
      // Make the filename the exact same value as the $ref
      // TODO: This leads to problems, if there are multiple references with the same file name but in different folders
      filename: reference
    });
    errors.push(...newErrors);
    filesystem = [
      ...filesystem,
      ...referencedFiles.map((file) => {
        return {
          ...file,
          isEntrypoint: false
        };
      })
    ];
  }
  return {
    specification: getEntrypoint(filesystem)?.specification,
    filesystem,
    errors
  };
}
export {
  load
};
//# sourceMappingURL=load.js.map
