import { traverse } from "../helpers/traverse.js";
const DEFAULT_MEDIA_TYPE = "application/json";
function extractXExampleExtensions(obj) {
  const xExample = obj["x-example"];
  const xExamples = obj["x-examples"];
  delete obj["x-example"];
  delete obj["x-examples"];
  return { xExample, xExamples };
}
function isNonEmptyObject(value) {
  return typeof value === "object" && value !== null && !Array.isArray(value) && Object.keys(value).length > 0;
}
function isNamedExamplesCollection(value) {
  return isNonEmptyObject(value) && Object.values(value).every((v) => typeof v === "object" && v !== null && !Array.isArray(v));
}
const EXAMPLE_OBJECT_PROPERTIES = /* @__PURE__ */ new Set(["summary", "description", "value", "externalValue"]);
function isExampleObject(value) {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const obj = value;
  const hasValueOrExternalValue = "value" in obj || "externalValue" in obj;
  const onlyHasAllowedProperties = Object.keys(obj).every((key) => EXAMPLE_OBJECT_PROPERTIES.has(key));
  return hasValueOrExternalValue && onlyHasAllowedProperties;
}
function wrapAsExampleObject(value) {
  if (isExampleObject(value)) {
    return value;
  }
  return { value };
}
const MEDIA_TYPE_KEY_PATTERN = /^[a-zA-Z0-9*+.-]+\/[a-zA-Z0-9*+.+-]+$/;
function isMediaTypeKey(key) {
  return MEDIA_TYPE_KEY_PATTERN.test(key);
}
function transformXExampleToExamples(xExample) {
  return Object.entries(xExample).reduce(
    (acc, [key, value]) => {
      acc[key] = { value };
      return acc;
    },
    {}
  );
}
const upgradeFlow = (flow) => {
  switch (flow) {
    case "application":
      return "clientCredentials";
    case "accessCode":
      return "authorizationCode";
    case "implicit":
      return "implicit";
    case "password":
      return "password";
    default:
      return flow;
  }
};
function upgradeFromTwoToThree(originalSpecification) {
  let document = originalSpecification;
  if (document !== null && typeof document === "object" && typeof document.swagger === "string" && document.swagger?.startsWith("2.0")) {
    document.openapi = "3.0.4";
    delete document.swagger;
  } else {
    return document;
  }
  if (document.host) {
    const schemes = Array.isArray(document.schemes) && document.schemes?.length ? document.schemes : ["http"];
    document.servers = schemes.map((scheme) => ({
      url: `${scheme}://${document.host}${document.basePath ?? ""}`
    }));
    delete document.basePath;
    delete document.schemes;
    delete document.host;
  } else if (document.basePath) {
    document.servers = [{ url: document.basePath }];
    delete document.basePath;
  }
  if (document.definitions) {
    document.components = Object.assign({}, document.components, {
      schemas: document.definitions
    });
    delete document.definitions;
    document = traverse(document, (schema) => {
      if (typeof schema.$ref === "string" && schema.$ref.startsWith("#/definitions/")) {
        schema.$ref = schema.$ref.replace(/^#\/definitions\//, "#/components/schemas/");
      }
      return schema;
    });
  }
  document = traverse(document, (schema) => {
    if (schema.type === "file") {
      schema.type = "string";
      schema.format = "binary";
    }
    return schema;
  });
  if (Object.hasOwn(document, "parameters")) {
    document = traverse(document, (schema) => {
      if (typeof schema.$ref === "string" && schema.$ref.startsWith("#/parameters/")) {
        const schemaName = schema.$ref.split("/")[2];
        if (!schemaName) {
          return schema;
        }
        const param = document.parameters && typeof document.parameters === "object" && schemaName in document.parameters ? document.parameters[schemaName] : void 0;
        if (param && typeof param === "object" && "in" in param && (param.in === "body" || param.in === "formData")) {
          schema.$ref = schema.$ref.replace(/^#\/parameters\//, "#/components/requestBodies/");
        } else {
          schema.$ref = schema.$ref.replace(/^#\/parameters\//, "#/components/parameters/");
        }
      }
      return schema;
    });
    document.components ??= {};
    const params = {};
    const bodyParams = {};
    const parameters = document.parameters && typeof document.parameters === "object" ? document.parameters : {};
    for (const [name, param] of Object.entries(parameters)) {
      if (param && typeof param === "object") {
        if ("$ref" in param) {
          const convertedParam = transformParameterObject(param);
          params[name] = convertedParam;
        } else if ("in" in param) {
          if (param.in === "body") {
            bodyParams[name] = migrateBodyParameter(
              param,
              document.consumes ?? [DEFAULT_MEDIA_TYPE]
            );
          } else if (param.in === "formData") {
            bodyParams[name] = migrateFormDataParameter(
              [param],
              document.consumes
            );
          } else {
            const convertedParam = transformParameterObject(param);
            params[name] = convertedParam;
          }
        }
      }
    }
    if (Object.keys(params).length > 0) {
      ;
      document.components.parameters = params;
    }
    if (Object.keys(bodyParams).length > 0) {
      ;
      document.components.requestBodies = bodyParams;
    }
    delete document.parameters;
  }
  if (Object.hasOwn(document, "responses") && typeof document.responses === "object" && document.responses !== null) {
    document = traverse(document, (schema) => {
      if (typeof schema.$ref === "string" && schema.$ref.startsWith("#/responses/")) {
        schema.$ref = schema.$ref.replace(/^#\/responses\//, "#/components/responses/");
      }
      return schema;
    });
    document.components ??= {};
    const migratedResponses = {};
    const responses = document.responses;
    for (const [name, response] of Object.entries(responses)) {
      if (response && typeof response === "object") {
        if ("$ref" in response) {
          migratedResponses[name] = response;
        } else {
          const responseObj = response;
          const produces = document.produces ?? [DEFAULT_MEDIA_TYPE];
          if (responseObj.schema) {
            if (typeof responseObj.content !== "object") {
              responseObj.content = {};
            }
            for (const type of produces) {
              ;
              responseObj.content[type] = {
                schema: responseObj.schema
              };
            }
            delete responseObj.schema;
          }
          if (responseObj.examples && typeof responseObj.examples === "object") {
            if (typeof responseObj.content !== "object") {
              responseObj.content = {};
            }
            const defaultMediaType = produces[0] ?? DEFAULT_MEDIA_TYPE;
            for (const [key, exampleValue] of Object.entries(responseObj.examples)) {
              if (isMediaTypeKey(key)) {
                if (typeof responseObj.content[key] !== "object") {
                  ;
                  responseObj.content[key] = {};
                }
                ;
                responseObj.content[key].example = exampleValue;
              } else {
                if (typeof responseObj.content[defaultMediaType] !== "object") {
                  ;
                  responseObj.content[defaultMediaType] = {};
                }
                const mediaEntry = responseObj.content[defaultMediaType];
                if (typeof mediaEntry.examples !== "object") {
                  mediaEntry.examples = {};
                }
                ;
                mediaEntry.examples[key] = wrapAsExampleObject(exampleValue);
              }
            }
            delete responseObj.examples;
          }
          if (responseObj.headers && typeof responseObj.headers === "object") {
            responseObj.headers = Object.entries(responseObj.headers).reduce(
              (acc, [headerName, header]) => {
                if (header && typeof header === "object") {
                  return {
                    [headerName]: transformResponseHeader(header),
                    ...acc
                  };
                }
                return acc;
              },
              {}
            );
          }
          migratedResponses[name] = responseObj;
        }
      }
    }
    if (Object.keys(migratedResponses).length > 0) {
      ;
      document.components.responses = migratedResponses;
    }
    delete document.responses;
  }
  if (typeof document.paths === "object") {
    for (const path in document.paths) {
      if (Object.hasOwn(document.paths, path)) {
        const pathItem = document.paths && typeof document.paths === "object" && path in document.paths ? document.paths[path] : void 0;
        if (!pathItem || typeof pathItem !== "object") {
          continue;
        }
        let requestBodyObject;
        for (const methodOrParameters in pathItem) {
          if (methodOrParameters === "parameters" && Object.hasOwn(pathItem, methodOrParameters)) {
            const pathItemParameters = migrateParameters(
              pathItem.parameters,
              document.consumes ?? [DEFAULT_MEDIA_TYPE]
            );
            pathItem.parameters = pathItemParameters.parameters;
            requestBodyObject = pathItemParameters.requestBody;
          } else if (Object.hasOwn(pathItem, methodOrParameters)) {
            const operationItem = pathItem[methodOrParameters];
            if (requestBodyObject) {
              operationItem.requestBody = requestBodyObject;
            }
            if (operationItem.parameters) {
              const migrationResult = migrateParameters(
                operationItem.parameters,
                operationItem.consumes ?? document.consumes ?? [DEFAULT_MEDIA_TYPE]
              );
              operationItem.parameters = migrationResult.parameters;
              if (migrationResult.requestBody) {
                operationItem.requestBody = migrationResult.requestBody;
              }
            }
            delete operationItem.consumes;
            if (operationItem.responses) {
              for (const response in operationItem.responses) {
                if (Object.hasOwn(operationItem.responses, response)) {
                  const responseItem = operationItem.responses[response];
                  if (responseItem.headers && typeof responseItem.headers === "object") {
                    responseItem.headers = Object.entries(responseItem.headers).reduce(
                      (acc, [name, header]) => {
                        if (header && typeof header === "object") {
                          return {
                            [name]: transformResponseHeader(header),
                            ...acc
                          };
                        }
                        return acc;
                      },
                      {}
                    );
                  }
                  if (responseItem.schema) {
                    const produces = document.produces ?? operationItem.produces ?? [DEFAULT_MEDIA_TYPE];
                    if (typeof responseItem.content !== "object") {
                      responseItem.content = {};
                    }
                    for (const type of produces) {
                      responseItem.content[type] = {
                        schema: responseItem.schema
                      };
                    }
                    delete responseItem.schema;
                  }
                  if (responseItem.examples && typeof responseItem.examples === "object") {
                    if (typeof responseItem.content !== "object") {
                      responseItem.content = {};
                    }
                    const produces = document.produces ?? operationItem.produces ?? [DEFAULT_MEDIA_TYPE];
                    const defaultMediaType = produces[0] ?? DEFAULT_MEDIA_TYPE;
                    for (const [key, exampleValue] of Object.entries(responseItem.examples)) {
                      if (isMediaTypeKey(key)) {
                        if (typeof responseItem.content[key] !== "object") {
                          responseItem.content[key] = {};
                        }
                        responseItem.content[key].example = exampleValue;
                      } else {
                        if (typeof responseItem.content[defaultMediaType] !== "object") {
                          responseItem.content[defaultMediaType] = {};
                        }
                        const mediaEntry = responseItem.content[defaultMediaType];
                        if (typeof mediaEntry.examples !== "object") {
                          mediaEntry.examples = {};
                        }
                        mediaEntry.examples[key] = wrapAsExampleObject(exampleValue);
                      }
                    }
                    delete responseItem.examples;
                  }
                }
              }
            }
            delete operationItem.produces;
            if (operationItem.parameters?.length === 0) {
              delete operationItem.parameters;
            }
          }
        }
      }
    }
  }
  if (document.securityDefinitions) {
    if (typeof document.components !== "object" || document.components === null) {
      document.components = {};
    }
    if (document.components && typeof document.components === "object") {
      Object.assign(document.components, { securitySchemes: {} });
    }
    for (const [key, securityScheme] of Object.entries(document.securityDefinitions)) {
      if (typeof securityScheme === "object") {
        if ("type" in securityScheme && securityScheme.type === "oauth2") {
          const { flow, authorizationUrl, tokenUrl, scopes } = securityScheme;
          if (document.components && typeof document.components === "object" && "securitySchemes" in document.components && document.components.securitySchemes) {
            Object.assign(document.components.securitySchemes, {
              [key]: {
                type: "oauth2",
                flows: {
                  [upgradeFlow(flow || "implicit")]: Object.assign(
                    {},
                    authorizationUrl && { authorizationUrl },
                    tokenUrl && { tokenUrl },
                    scopes && { scopes }
                  )
                }
              }
            });
          }
        } else if ("type" in securityScheme && securityScheme.type === "basic") {
          if (document.components && typeof document.components === "object" && "securitySchemes" in document.components && document.components.securitySchemes) {
            Object.assign(document.components.securitySchemes, {
              [key]: {
                type: "http",
                scheme: "basic"
              }
            });
          }
        } else {
          if (document.components && typeof document.components === "object" && "securitySchemes" in document.components && document.components.securitySchemes) {
            Object.assign(document.components.securitySchemes, {
              [key]: securityScheme
            });
          }
        }
      }
    }
    delete document.securityDefinitions;
  }
  delete document.consumes;
  delete document.produces;
  return document;
}
function transformItemsObject(obj) {
  const schemaProperties = [
    "type",
    "format",
    "default",
    "items",
    "maximum",
    "exclusiveMaximum",
    "minimum",
    "exclusiveMinimum",
    "maxLength",
    "minLength",
    "pattern",
    "maxItems",
    "minItems",
    "uniqueItems",
    "enum",
    "multipleOf"
  ];
  return schemaProperties.reduce((acc, property) => {
    if (Object.hasOwn(obj, property)) {
      acc[property] = obj[property];
      delete obj[property];
    }
    return acc;
  }, {});
}
function getParameterLocation(location) {
  if (location === "formData") {
    throw new Error("Encountered a formData parameter which should have been filtered out by the caller");
  }
  if (location === "body") {
    throw new Error("Encountered a body parameter which should have been filtered out by the caller");
  }
  return location;
}
function transformParameterObject(parameter) {
  if (Object.hasOwn(parameter, "$ref") && typeof parameter.$ref === "string") {
    return {
      $ref: parameter.$ref
    };
  }
  const serializationStyle = getParameterSerializationStyle(parameter);
  const schema = transformItemsObject(parameter);
  const { xExample, xExamples } = extractXExampleExtensions(parameter);
  if (isNonEmptyObject(xExample)) {
    parameter.examples = transformXExampleToExamples(xExample);
  } else if (isNonEmptyObject(xExamples)) {
    parameter.examples = Object.entries(xExamples).reduce(
      (acc, [key, exampleValue]) => {
        acc[key] = wrapAsExampleObject(exampleValue);
        return acc;
      },
      {}
    );
  }
  delete parameter.collectionFormat;
  delete parameter.default;
  if (!parameter.in) {
    throw new Error('Parameter object must have an "in" property');
  }
  return {
    schema,
    ...serializationStyle,
    ...parameter,
    in: getParameterLocation(parameter.in)
  };
}
function transformResponseHeader(header) {
  if (Object.hasOwn(header, "$ref") && typeof header.$ref === "string") {
    return {
      $ref: header.$ref
    };
  }
  const schema = transformItemsObject(header);
  return {
    ...header,
    schema
  };
}
const querySerialization = {
  ssv: {
    style: "spaceDelimited",
    explode: false
  },
  pipes: {
    style: "pipeDelimited",
    explode: false
  },
  multi: {
    style: "form",
    explode: true
  },
  csv: {
    style: "form",
    explode: false
  },
  tsv: {}
};
const pathAndHeaderSerialization = {
  ssv: {},
  pipes: {},
  multi: {},
  csv: {
    style: "simple",
    explode: false
  },
  tsv: {}
};
const serializationStyles = {
  header: pathAndHeaderSerialization,
  query: querySerialization,
  path: pathAndHeaderSerialization
};
function getParameterSerializationStyle(parameter) {
  if (parameter.type !== "array" || !(parameter.in === "query" || parameter.in === "path" || parameter.in === "header")) {
    return {};
  }
  const collectionFormat = parameter.collectionFormat ?? "csv";
  if (parameter.in in serializationStyles && collectionFormat in serializationStyles[parameter.in]) {
    return serializationStyles[parameter.in][collectionFormat];
  }
  return {};
}
function migrateBodyParameter(bodyParameter, consumes) {
  const { xExample, xExamples } = extractXExampleExtensions(bodyParameter);
  delete bodyParameter.name;
  delete bodyParameter.in;
  const { schema, ...requestBody } = bodyParameter;
  const requestBodyObject = {
    content: {},
    ...requestBody
  };
  if (requestBodyObject.content) {
    for (const type of consumes) {
      requestBodyObject.content[type] = {
        schema
      };
      if (isNonEmptyObject(xExample) && type in xExample) {
        requestBodyObject.content[type].example = xExample[type];
      }
      if (isNonEmptyObject(xExamples) && type in xExamples) {
        const examples = xExamples[type];
        const isExamplesCollection = isNonEmptyObject(examples) && Object.values(examples).every((example) => isExampleObject(example));
        if (isExamplesCollection) {
          requestBodyObject.content[type].examples = examples;
        } else if (isNamedExamplesCollection(examples)) {
          requestBodyObject.content[type].examples = Object.entries(examples).reduce(
            (acc, [key, exampleValue]) => {
              acc[key] = wrapAsExampleObject(exampleValue);
              return acc;
            },
            {}
          );
        } else {
          requestBodyObject.content[type].examples = {
            default: wrapAsExampleObject(examples)
          };
        }
      } else if (isNonEmptyObject(xExamples) && !Object.keys(xExamples).some(isMediaTypeKey)) {
        if (isNamedExamplesCollection(xExamples)) {
          requestBodyObject.content[type].examples = Object.entries(xExamples).reduce(
            (acc, [key, exampleValue]) => {
              acc[key] = wrapAsExampleObject(exampleValue);
              return acc;
            },
            {}
          );
        } else {
          requestBodyObject.content[type].examples = {
            default: wrapAsExampleObject(xExamples)
          };
        }
      }
    }
  }
  return requestBodyObject;
}
function migrateFormDataParameter(parameters, consumes = ["multipart/form-data"]) {
  const requestBodyObject = {
    content: {}
  };
  const filtered = consumes.filter(
    (type) => type === "multipart/form-data" || type === "application/x-www-form-urlencoded"
  );
  const contentTypes = filtered.length > 0 ? filtered : ["multipart/form-data"];
  if (requestBodyObject.content) {
    for (const contentType of contentTypes) {
      requestBodyObject.content[contentType] = {
        schema: {
          type: "object",
          properties: {},
          required: []
          // Initialize required array
        }
      };
      const formContent = requestBodyObject.content?.[contentType];
      if (formContent?.schema && typeof formContent.schema === "object" && "properties" in formContent.schema) {
        for (const param of parameters) {
          if (param.name && formContent.schema.properties) {
            formContent.schema.properties[param.name] = {
              type: param.type,
              description: param.description,
              ...param.format ? { format: param.format } : {}
            };
            if (param.required && Array.isArray(formContent.schema.required)) {
              formContent.schema.required.push(param.name);
            }
          }
        }
      }
    }
  }
  return requestBodyObject;
}
function migrateParameters(parameters, consumes) {
  const result = {
    parameters: parameters.filter((parameter) => !(parameter.in === "body" || parameter.in === "formData")).map((parameter) => transformParameterObject(parameter))
  };
  const bodyParameter = structuredClone(
    parameters.find((parameter) => parameter.in === "body") ?? {}
  );
  if (bodyParameter && Object.keys(bodyParameter).length) {
    result.requestBody = migrateBodyParameter(bodyParameter, consumes);
  }
  const formDataParameters = parameters.filter((parameter) => parameter.in === "formData");
  if (formDataParameters.length > 0) {
    const requestBodyObject = migrateFormDataParameter(formDataParameters, consumes);
    if (typeof result.requestBody !== "object") {
      result.requestBody = requestBodyObject;
    } else {
      result.requestBody = {
        ...result.requestBody,
        content: {
          ...result.requestBody.content,
          ...requestBodyObject.content
        }
      };
    }
    if (typeof result.requestBody !== "object") {
      result.requestBody = {
        content: {}
      };
    }
  }
  return result;
}
export {
  upgradeFromTwoToThree
};
//# sourceMappingURL=upgrade-from-two-to-three.js.map
