import { parse as urlParse } from "url";
import { Key, pathToRegexp } from "path-to-regexp";
import { lowerHttpMethods, Parameter, PathParameter, Schema } from "../swagger/swaggerTypes";
import { xmsParameterizedHost } from "../util/constants";
import { OperationMatch } from "../liveValidation/operationSearcher";
import { resolveNestedDefinitionTransformer } from "./resolveNestedDefinitionTransformer";
import { SpecTransformer, TransformerType } from "./transformer";
import { traverseSwagger } from "./traverseSwagger";

export type RegExpWithKeys = RegExp & {
  _keys: string[];
  _hostTemplate?: boolean;
  _hasMultiPathParam?: boolean;
};

/**
 * Escapes literal colons in a path string for path-to-regexp v6 compatibility.
 * Only colons not followed by a valid parameter name are escaped.
 */
function escapeLiteralColons(path: string): string {
  // Escape all colons not followed by a digit (which are parameter indices)
  return path.replace(/:(?![0-9])/g, "\\:");
}

function replaceParam(path: string, name: string, index: number) {
  const paramWithBraces = "{" + name + "}";

  const start = path.indexOf(paramWithBraces);
  const end = start + paramWithBraces.length - 1;
  const next = end + 1;
  if (next < path.length) {
    const nextChar = path[next];

    // In path-to-regexp@6.3.0, if the char following a param is a word char, it must be added
    // to a custom regex for the param.
    //
    // This only applies to a small number of paths with segments like "{width}x{height}".
    // Without this fix, the paths would fail with: "TypeError: Must have text between two parameters".
    //
    // If a param has a custom regex (e.g. "{foo}([^a]+)"), nextChar will be "(", so this is a no-op.
    if (/^\w$/.test(nextChar)) {
      // Chars excluded by default, if a param has no custom regex
      const defaultExclusions = "\\/#\\?";
      path = path.replace(
        paramWithBraces,
        paramWithBraces + `([^${defaultExclusions}${nextChar}]+)`
      );
    }
  }

  return path.replace(paramWithBraces, ":" + index);
}

const buildPathRegex = (
  hostTemplate: string,
  basePathPrefix: string,
  path: string,
  pathParams: Map<string, PathParameter>
): RegExpWithKeys => {
  hostTemplate = hostTemplate.replace("https://", "");
  hostTemplate = hostTemplate.replace("http://", "");

  if (path.endsWith("/")) {
    path = path.substr(0, path.length - 1);
  }

  const params: string[] = [];

  function collectParamName(regResult: string) {
    if (regResult) {
      const paramName = regResult.replace("{", "").replace("}", "");

      if (!params.includes(paramName)) {
        params.push(paramName);
      }
    }
  }

  // collect all parameter name
  const regHostParams = hostTemplate.match(/({[\w-]+})/gi);

  if (regHostParams) {
    regHostParams.forEach(collectParamName);
  }
  const regPathParams = path.match(/({[\w-]+})/gi);

  if (regPathParams) {
    regPathParams.forEach(collectParamName);
  }

  hostTemplate = hostTemplate.replace(/\(/g, "\\(").replace(/\)/g, "\\)");
  path = path.replace(/\(/g, "\\(").replace(/\)/g, "\\)");

  /**
   * To support parameter name with dash(-), replace the parameter names to array index,
   * and will restore the name in the result object regexp later.
   * It caused by a design issue in pathToRegexp library that the parameter names must use "word characters" ([A-Za-z0-9_]).
   * for more details,see https://github.com/pillarjs/path-to-regexp
   **/

  let hasMultiPathParam = false;
  params.forEach((v, i) => {
    if (path.startsWith(`/{${v}}`) && pathParams.get(v)) {
      // We allow first param in path as multi path param
      path = path.replace("{" + v + "}", "(.*)");
      hasMultiPathParam = true;
    } else {
      hostTemplate = replaceParam(hostTemplate, v, i);
      path = replaceParam(path, v, i);
    }
  });

  const processedPath = hostTemplate + basePathPrefix + path;

  const keys: Key[] = [];

  // in security patched versions of path-to-regexp (read > 6.2.1), colons must refer to valid parameter names
  // so now we have to escape literal colons that are in the path
  const escapedPath = escapeLiteralColons(processedPath);
  const regexp = pathToRegexp(escapedPath, keys, { sensitive: false });

  // restore parameter name
  const _keys: string[] = [];
  keys.forEach((v, i) => {
    if (params[v.name as any]) {
      _keys[i + 1] = params[v.name as any];
    }
  });

  const regexpWithKeys: RegExpWithKeys = regexp as RegExpWithKeys;
  regexpWithKeys._keys = _keys;
  if (hostTemplate !== "") {
    regexpWithKeys._hostTemplate = true;
  }
  if (hasMultiPathParam) {
    regexpWithKeys._hasMultiPathParam = true;
  }

  return regexpWithKeys;
};

export const pathRegexTransformer: SpecTransformer = {
  type: TransformerType.Spec,
  after: [resolveNestedDefinitionTransformer],
  transform(spec, { schemaValidator, jsonLoader }) {
    let basePathPrefix = spec.basePath ?? "";
    if (basePathPrefix.endsWith("/")) {
      basePathPrefix = basePathPrefix.substr(0, basePathPrefix.length - 1);
    }
    const msParameterizedHost = spec[xmsParameterizedHost];
    const hostTemplate = msParameterizedHost?.hostTemplate ?? "";
    const hostParams = msParameterizedHost?.parameters;

    traverseSwagger(spec, {
      onPath: (path, pathTemplate) => {
        let pathStr = pathTemplate;
        const queryIdx = pathTemplate.indexOf("?");
        if (queryIdx !== -1) {
          // path in x-ms-paths has query part we need to match
          const queryMatch = urlParse(pathStr, true).query;
          const querySchema: Schema = { type: "object", properties: {}, required: [] };
          for (const queryKey of Object.keys(queryMatch)) {
            const queryVal = queryMatch[queryKey];
            querySchema.required!.push(queryKey);
            querySchema.properties![queryKey] = {
              enum: typeof queryVal === "string" ? [queryVal] : queryVal,
            };
          }
          path._validateQuery = schemaValidator.compile(querySchema);
          pathStr = pathTemplate.substr(0, queryIdx);
        }

        const pathParams = new Map<string, PathParameter>();
        const collectPathParams = (params?: Parameter[]) => {
          if (params !== undefined) {
            for (const p of params) {
              const param = jsonLoader.resolveRefObj(p);
              if (param.in === "path") {
                pathParams.set(param.name, param);
              }
            }
          }
        };
        collectPathParams(hostParams);
        collectPathParams(path.parameters);
        for (const httpMethod of lowerHttpMethods) {
          collectPathParams(path[httpMethod]?.parameters);
        }

        path._pathRegex = buildPathRegex(hostTemplate, basePathPrefix, pathStr, pathParams);
      },
      onOperation: (operation) => {
        if (operation.externalDocs !== undefined) {
          operation.externalDocs = undefined;
        }
      },
      onResponse: (response) => {
        if (response.examples !== undefined) {
          response.examples = undefined;
        }
      },
    });
  },
};

export const extractPathParamValue = ({ pathRegex, pathMatch }: OperationMatch) => {
  const pathParam: { [key: string]: string } = {};
  const _keys = pathRegex._keys;
  for (let idx = 1; idx < pathMatch.length; ++idx) {
    if (_keys[idx] !== undefined) {
      pathParam[_keys[idx]] = decodeURIComponent(pathMatch[idx]);
    }
  }
  return pathParam;
};
