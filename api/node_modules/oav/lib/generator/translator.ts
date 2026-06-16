import { JsonLoader } from "../swagger/jsonLoader";
import { log } from "../util/logging";
import {
  buildItemOption,
  CacheItem,
  createLeafItem,
  createTrunkItem,
  PayloadCache,
  reBuildExample,
} from "./exampleCache";
import * as utils from "./util";
import { ExampleRule, getRuleValidator } from "./exampleRule";
import SwaggerMocker from "./swaggerMocker";

export default class Translator {
  private jsonLoader: JsonLoader;
  private payloadCache: PayloadCache;
  private exampleRule?: ExampleRule;
  private mocker: SwaggerMocker | undefined;
  public constructor(jsonLoader: JsonLoader, payloadCache: PayloadCache, mocker?: SwaggerMocker) {
    this.jsonLoader = jsonLoader;
    this.payloadCache = payloadCache;
    this.mocker = mocker;
  }

  public setRule(exampleRule?: ExampleRule) {
    this.exampleRule = exampleRule;
  }

  public extractRequest(specItem: any, request: any) {
    const path = request.url.split("?")[0];
    const pathValues = this.getPathParameters(specItem.path, path);
    const queryValues = this.getQueryParameters(specItem, request.query);
    const parameters = this.getBodyParameters(specItem, request.body);
    const result = {
      ...parameters,
      ...pathValues,
      ...queryValues,
    };
    this.maskSpecialValue(result);
    return result;
  }

  // for example cache.
  public extractParameters(specItem: any, request: any) {
    const parameters = this.getMatchedParameters(specItem.content.parameters, request);
    const result = {
      ...parameters,
    };
    this.maskSpecialValue(result);
    return result;
  }

  private maskSpecialValue(requestExample: any) {
    if (requestExample.subscriptionId) {
      requestExample.subscriptionId = requestExample.subscriptionId.replace(/[a-z0-9]/g, "0");
    }
  }

  private getMatchedParameters(paramterSchema: any, parameters: any) {
    const bodyRes: any = {};
    const validator = getRuleValidator(this.exampleRule).onParameter;
    paramterSchema.forEach((item: any) => {
      const itemSchema = this.getDefSpec(item);
      if (parameters[itemSchema.name]) {
        if (!validator || validator({ schema: itemSchema })) {
          bodyRes[itemSchema.name] = this.filterBodyContent(
            parameters[itemSchema.name],
            itemSchema.in === "body" ? itemSchema.schema : item
          );
        }
      }
    });
    return bodyRes;
  }

  private getQueryParameters(specItem: any, body: any): any {
    if (!specItem || !body) {
      return;
    }
    const parametersSpec = specItem.content.parameters
      .map((item: any) => this.getDefSpec(item))
      .filter((item: any) => "in" in item && item.in === "query");

    if (parametersSpec.length === 0) {
      log.info("no query parameter definition in spec file");
      return;
    }
    return this.getMatchedParameters(parametersSpec, body);
  }

  // for payload cache
  private getBodyParameters(specItem: any, body: any): any {
    if (!specItem || !body) {
      return;
    }
    const parametersSpec = specItem.content.parameters
      .map((item: any) => this.getDefSpec(item))
      .filter((item: any) => "in" in item && item.in === "body");

    if (parametersSpec.length === 0) {
      log.info("no body parameter definition in spec file");
      return;
    }
    const bodyParameterName = parametersSpec[0].name;
    const exampleBody: any = {};
    exampleBody[bodyParameterName] = body;
    return this.getMatchedParameters(parametersSpec, exampleBody);
  }

  /**
   * filter body's fields which is not defined in spec file
   * @param body
   * @param schema
   */

  public filterBodyContent(body: any, schema: any, isRequest: boolean = true) {
    const cache = this.cacheBodyContent(body, schema, isRequest);
    const validator = getRuleValidator(this.exampleRule).onSchema;
    return reBuildExample(cache, isRequest, schema, validator);
  }
  public cacheBodyContent(body: any, schema: any, isRequest: boolean) {
    if (!schema) {
      return undefined;
    }
    if (schema.$ref && this.payloadCache.get(schema.$ref.split("#")[1])) {
      return this.payloadCache.get(schema.$ref.split("#")[1]);
    }
    const definitionSpec = this.getDefSpec(schema);
    const bodyContent: any = {};
    let cacheItem: CacheItem | undefined;
    if (utils.isObject(definitionSpec)) {
      const properties = this.getProperties(definitionSpec);
      if (body) {
        Object.keys(body)
          .filter((key) => key in properties)
          .forEach((key: string) => {
            bodyContent[key] = this.cacheBodyContent(body[key], properties[key], isRequest);
          });
      }
      // to mock the properties that not exists in the body.
      // it's not needed when generating from payload.
      if (this.mocker !== undefined) {
        Object.keys(properties)
          .filter((key) => !body?.[key])
          .forEach((key: string) => {
            bodyContent[key] = this.mocker?.getMockCachedObj(key, properties[key], isRequest);
          });
      }
      cacheItem = createTrunkItem(bodyContent, buildItemOption(definitionSpec));
    } else if (definitionSpec.type === "array") {
      if (body) {
        const result = body.map((i: any) => {
          return this.cacheBodyContent(i, schema.items, isRequest);
        });
        cacheItem = createTrunkItem(result, buildItemOption(definitionSpec));
      } else if (this.mocker !== undefined) {
        return this.mocker?.getMockCachedObj("mock array", schema, isRequest);
      }
    } else {
      if (body !== undefined) {
        cacheItem = createLeafItem(body, buildItemOption(definitionSpec));
      } else if (this.mocker) {
        return this.mocker?.getMockCachedObj("", schema, isRequest);
      }
    }
    const requiredProperties = this.getRequiredProperties(definitionSpec);
    if (requiredProperties && requiredProperties.length > 0 && cacheItem !== undefined) {
      cacheItem.required = requiredProperties;
    }
    this.payloadCache.checkAndCache(schema, cacheItem, isRequest);
    return cacheItem;
  }

  /**
   * return all properties of the object, including parent's properties defined by 'allOf'
   * It will not spread properties' properties.
   * @param definitionSpec
   */
  private getProperties(definitionSpec: any) {
    let properties: any = {};
    definitionSpec.allOf?.map((item: any) => {
      properties = {
        ...properties,
        ...this.getProperties(this.getDefSpec(item)),
      };
    });
    return {
      ...properties,
      ...definitionSpec.properties,
    };
  }

  /**
   * return all required properties of the object, including parent's properties defined by 'allOf'
   * It will not spread properties' properties.
   * @param definitionSpec
   */
  private getRequiredProperties(definitionSpec: any) {
    let requiredProperties: string[] = Array.isArray(definitionSpec.required)
      ? definitionSpec.required
      : [];
    definitionSpec.allOf?.map((item: any) => {
      requiredProperties = [
        ...requiredProperties,
        ...this.getRequiredProperties(this.getDefSpec(item)),
      ];
    });
    return requiredProperties;
  }

  private getDefSpec(item: any) {
    if (item) {
      return this.jsonLoader.resolveRefObj(item);
    }
  }

  public extractResponse(specItem: any, response: any, statusCode: string) {
    const specItemContent = specItem.content;
    const resp: any = {};
    if (statusCode === "201" || statusCode === "202") {
      resp.headers = {
        Location:
          response.headers && "location" in response.headers
            ? response.headers.location
            : undefined,
        "Azure-AsyncOperation":
          response.headers && "azure-AsyncOperation" in response.headers
            ? response.headers["azure-AsyncOperation"]
            : undefined,
      };
    }
    if ("schema" in specItemContent.responses[statusCode] && response.body) {
      resp.body =
        this.filterBodyContent(
          response.body,
          specItemContent.responses[statusCode].schema,
          false
        ) || {};
    }
    return resp;
  }

  private getPathParameters(pathName: string, path: string): object {
    const pathTemplateItems = pathName.split("/");
    const urlItems = path.split("/") || [];
    let res = {};
    pathTemplateItems.forEach((item, idx) => {
      if (item !== urlItems[idx]) {
        const matchParam = /^{(.*)}$/.exec(item);
        if (matchParam && matchParam.length >= 2) {
          const paramName: string = matchParam[1];
          res = {
            ...res,
            [paramName]: urlItems[idx],
          };
        }
      }
    });
    return res;
  }
}
