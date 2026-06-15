import { bundle } from "../bundle/index.js";
import { fetchUrls } from "../bundle/plugins/fetch-urls/index.js";
import { createMagicProxy } from "../magic-proxy/index.js";
const dereference = (input, options) => {
  if (options?.sync) {
    return {
      success: true,
      data: createMagicProxy(input)
    };
  }
  const errors = [];
  return bundle(input, {
    plugins: [fetchUrls()],
    treeShake: false,
    urlMap: true,
    hooks: {
      onResolveError(node) {
        errors.push(`Failed to resolve ${node.$ref}`);
      }
    }
  }).then((result) => {
    if (errors.length > 0) {
      return {
        success: false,
        errors
      };
    }
    return {
      success: true,
      data: createMagicProxy(result)
    };
  });
};
export {
  dereference
};
//# sourceMappingURL=dereference.js.map
