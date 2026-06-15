import * as path from "path";
import deepdash from "deepdash";
import lodash from "lodash";
import { JsonLoader } from "../swagger/jsonLoader";
import { Operation, SwaggerSpec } from "../swagger/swaggerTypes";
import { traverseSwaggerAsync } from "../transform/traverseSwagger";
import { ModelValidationError } from "../util/modelValidationError";
import * as validate from "../validate";
import { AjvSchemaValidator } from "../swaggerValidator/ajvSchemaValidator";
import { TransformContext, getTransformContext } from "../transform/context";
import { xmsPathsTransformer } from "../transform/xmsPathsTransformer";
import { resolveNestedDefinitionTransformer } from "../transform/resolveNestedDefinitionTransformer";
import { referenceFieldsTransformer } from "../transform/referenceFieldsTransformer";
import { discriminatorTransformer } from "../transform/discriminatorTransformer";
import { allOfTransformer } from "../transform/allOfTransformer";
import { noAdditionalPropertiesTransformer } from "../transform/noAdditionalPropertiesTransformer";
import { applySpecTransformers, applyGlobalTransformers } from "../transform/transformer";
import { log } from "../util/logging";
import { inversifyGetInstance } from "../inversifyUtils";
import { ExampleRule, RuleSet } from "./exampleRule";
import * as util from "./util";
import Translator from "./translator";
import SwaggerMocker from "./swaggerMocker";
import { MockerCache, PayloadCache } from "./exampleCache";
const _ = deepdash(lodash);

export default class Generator {
  private translator: Translator;
  private spec!: SwaggerSpec;
  private specFilePath: string;
  private payloadDir?: string;
  private jsonLoader: JsonLoader;
  private swaggerMocker: SwaggerMocker;
  private shouldMock: boolean;
  private mockerCache: MockerCache;
  private payloadCache: PayloadCache;
  private generationRule?: "Max" | "Min";
  public readonly transformContext: TransformContext;

  public constructor(specFilePath: string, payloadDir?: string, generationRule?: "Max" | "Min") {
    this.generationRule = generationRule;
    this.shouldMock = payloadDir ? false : true;
    this.specFilePath = specFilePath;
    this.payloadDir = payloadDir;
    this.jsonLoader = inversifyGetInstance(JsonLoader, {
      useJsonParser: false,
      eraseXmsExamples: false,
    });
    this.mockerCache = new MockerCache();
    this.payloadCache = new PayloadCache();
    this.swaggerMocker = new SwaggerMocker(this.jsonLoader, this.mockerCache, this.payloadCache);
    this.translator = new Translator(
      this.jsonLoader,
      this.payloadCache,
      this.shouldMock ? this.swaggerMocker : undefined
    );
    const schemaValidator = new AjvSchemaValidator(this.jsonLoader);
    this.transformContext = getTransformContext(this.jsonLoader, schemaValidator, [
      xmsPathsTransformer,
      resolveNestedDefinitionTransformer,
      referenceFieldsTransformer,
      discriminatorTransformer,
      allOfTransformer,
      noAdditionalPropertiesTransformer,
    ]);
  }

  private getSpecItem(spec: any, operationId: string): any {
    const paths = spec.paths;
    for (const pathName of Object.keys(paths)) {
      for (const methodName of Object.keys(paths[pathName])) {
        if (paths[pathName][methodName].operationId === operationId) {
          return {
            path: pathName,
            methodName,
            content: paths[pathName][methodName],
          };
        }
      }
    }
    return null;
  }

  public async load() {
    this.spec = (await (this.jsonLoader.load(this.specFilePath) as unknown)) as SwaggerSpec;
    applySpecTransformers(this.spec, this.transformContext);
    applyGlobalTransformers(this.transformContext);
    await this.cacheExistingExamples();
  }

  public async generateAll(): Promise<readonly ModelValidationError[]> {
    if (!this.spec) {
      await this.load();
    }
    const errs: any[] = [];
    await traverseSwaggerAsync(this.spec, {
      onPath: async (apiPath, pathTemplate) => {
        apiPath._pathTemplate = pathTemplate;
      },
      onOperation: async (operation: Operation, pathObject, methodName) => {
        const pathName = pathObject._pathTemplate;
        const specItem = {
          path: pathName,
          methodName,
          content: operation,
        };
        const operationId: string = operation.operationId || "";
        const errors = await this.generate(operationId, specItem);
        if (errors.length > 0) {
          errs.push(...errors);
          return false;
        }
        return true;
      },
    });
    return errs;
  }

  public async cacheExistingExamples() {
    if (!this.shouldMock) {
      return;
    }
    await traverseSwaggerAsync(this.spec, {
      onOperation: async (operation: Operation, pathObject, methodName) => {
        const pathName = pathObject._pathTemplate;
        const specItem = {
          path: pathName,
          methodName,
          content: operation,
        };
        const examples = operation["x-ms-examples"] || undefined;
        if (!examples) {
          return;
        }

        const operationId = operation.operationId;
        /*
        const validateErrors = await validate.validateExamples(this.specFilePath, operationId, {
        });
        if(validateErrors.length > 0) {
          console.warn(`invalid examples for operation:${operationId}.`);
          console.warn(validateErrors);
          return
        } */
        for (const key of Object.keys(examples)) {
          if (key.match(new RegExp(`^${operationId}_.*_Gen$`))) {
            continue;
          }
          const example = this.jsonLoader.resolveRefObj(examples[key]);
          if (!example) {
            continue;
          }
          this.translator.extractParameters(specItem, example.parameters);
          for (const code of Object.keys(operation.responses)) {
            if (example.responses && example.responses[code]) {
              this.translator.extractResponse(specItem, example.responses[code], code);
            }
          }
        }
        return true;
      },
    });
    // reuse the payloadCache as exampleCache.
    this.payloadCache.mergeCache();
  }

  private async generateExample(operationId: string, specItem: any, rule: ExampleRule) {
    this.translator.setRule(rule);
    this.swaggerMocker.setRule(rule);
    let example;
    console.log(`start generated example for ${operationId}, rule:${rule.ruleName}`);
    if (!this.shouldMock) {
      example = this.getExampleFromPayload(operationId, specItem, rule);
      if (!example) {
        return [];
      }
    } else {
      const xMsExamples = specItem?.content?.["x-ms-examples"] || {};
      const xMsExampleKeys = Object.getOwnPropertyNames(xMsExamples);
      const title = xMsExampleKeys.length > 0 ? xMsExampleKeys[0] : "";
      example = {
        title:
          title.length > 0
            ? title.concat(" - generated by [", rule.ruleName!, "] rule")
            : specItem.content.summary
            ? specItem.content.summary
            : specItem.content.description
            ? specItem.content.description
            : `${operationId}_${rule.exampleNamePostfix}`,
        operationId: operationId,
        parameters: {},
        responses: this.extractResponse(specItem, {}),
      };
      this.swaggerMocker.mockForExample(
        example,
        specItem,
        this.spec,
        util.getBaseName(this.specFilePath).split(".")[0]
      );
    }

    log.info(example);
    const unifiedExample = this.unifyCommonProperty(example);
    const newSpec = util.referenceExmInSpec(
      this.specFilePath,
      specItem.path,
      specItem.methodName,
      `${operationId}_${rule.exampleNamePostfix}_Gen`
    );
    util.updateExmAndSpecFile(
      unifiedExample,
      newSpec,
      this.specFilePath,
      `${operationId}_${rule.exampleNamePostfix}_Gen.json`
    );

    log.info(`start validating generated example for ${operationId}`);
    const validateErrors = await validate.validateExamples(this.specFilePath, operationId, {
      //   consoleLogLevel: "error"
    });
    if (validateErrors.length > 0) {
      log.error(`the validation raised below error:`);
      log.error(validateErrors);
      return validateErrors;
    }
    console.log(`generated example for ${operationId}, rule:${rule.ruleName} successfully!`);
    return [];
  }

  public async generate(
    operationId: string,
    specItem?: any
  ): Promise<readonly ModelValidationError[]> {
    if (!this.spec) {
      await this.load();
    }
    if (!specItem) {
      specItem = this.getSpecItem(this.spec, operationId);
      if (!specItem) {
        console.error(`no specItem for the operation id ${operationId}`);
        return [];
      }
    }
    const ruleSet: RuleSet = [];
    if (this.generationRule) {
      ruleSet.push({
        exampleNamePostfix: `${this.generationRule}imumSet`,
        ruleName: `${this.generationRule}imumSet`,
      });
    } else {
      ruleSet.push(
        {
          exampleNamePostfix: "MaximumSet",
          ruleName: "MaximumSet",
        },
        {
          exampleNamePostfix: "MinimumSet",
          ruleName: "MinimumSet",
        }
      );
    }
    for (const rule of ruleSet) {
      const error = await this.generateExample(operationId, specItem, rule);
      if (error.length) {
        return error;
      }
    }
    return [];
  }

  private unifyCommonProperty(example: any) {
    if (!example || !example.parameters || !example.responses) {
      return;
    }
    type pathNode = string | number;
    type pathNodes = pathNode[];

    const requestPaths = _.paths(example.parameters, { pathFormat: "array" }).map((v) =>
      (v as pathNode[]).reverse()
    );

    /**
     * construct a inverted index , the key is leaf property key, value is reverse of the path from the root to the leaf property.
     */
    const invertedIndex = new Map<string | number, pathNodes[]>();
    requestPaths.forEach((v) => {
      if (v.length && typeof v[0] === "string") {
        const parentPaths = invertedIndex.get(v[0]);
        if (!parentPaths) {
          invertedIndex.set(v[0], [v.slice(1)]);
        } else {
          parentPaths.push(v.slice(1));
        }
      }
    });

    /**
     * get two paths' common properties' count
     */
    const getMatchedNodeCnt = (baseNode: pathNodes, destNode: pathNodes) => {
      let count = 0;
      baseNode.some((v, k) => {
        if (k < destNode.length && destNode[k] === v) {
          count++;
          return false;
        } else {
          return true;
        }
      });
      return count;
    };

    /**
     * update the property value of response using the same value which is found in the request
     */
    const res = _.mapValuesDeep(
      example.responses,
      (value, key, parentValue, context) => {
        if (!parentValue) {
          log.warn(`parent is null`);
        }
        if (
          ["integer", "number", "string"].some((type) => typeof value === type) &&
          typeof key === "string"
        ) {
          const possiblePaths = invertedIndex.get(key);
          if (context.path && possiblePaths) {
            const basePath = (context.path as pathNodes).slice().reverse().slice(1);

            /**
             * to find out the most matchable path in the parameters
             */
            const candidates = possiblePaths.filter(
              (apiPath) => getMatchedNodeCnt(basePath, apiPath) > 1
            );
            if (candidates.length === 0) {
              return value;
            }
            /**
             * if only one matched one path, just use it.
             */
            if (candidates.length === 1) {
              const pathOfParameter = _.pathToString([key, ...candidates[0]].reverse());
              const parameterValue = _.get(example.parameters, pathOfParameter);
              // console.debug(`use path ${pathOfParameter} ,value :${parameterValue}
              // -- original path:${_.pathToString(context.path as pathNodes)},value:${value}`);
              return parameterValue;
            }
            const mostMatched = candidates.reduce((previous, current) => {
              const countPrevious = getMatchedNodeCnt(basePath, previous);
              const countCurrent = getMatchedNodeCnt(basePath, current);
              return countPrevious < countCurrent ? current : previous;
            });
            return _.get(example.parameters, _.pathToString([key, ...mostMatched].reverse()));
          }
        }
        return value;
      },
      {
        leavesOnly: true,
        pathFormat: "array",
      }
    );
    // console.debug(`unify common properties end!`);
    example.responses = res;
    return example;
  }

  private getExampleFromPayload(operationId: string, specItem: any, rule: ExampleRule) {
    if (this.payloadDir) {
      const subPaths = path.dirname(this.specFilePath).split(/\\|\//).slice(-3).join("/");
      const payloadDir = path.join(this.payloadDir, subPaths);
      const payload: any = util.readPayloadFile(payloadDir, operationId);
      if (!payload) {
        log.info(
          `no payload file for operationId ${operationId} under directory ${path.resolve(
            payloadDir,
            operationId
          )} named with <statusCode>.json`
        );
        return;
      }
      this.validatePayload(specItem, payload, operationId);
      this.cachePayload(specItem, payload);
      const example = {
        title: specItem.content.summary
          ? specItem.content.summary
          : specItem.content.description
          ? specItem.content.description
          : `${operationId}_${rule.exampleNamePostfix}`,
        operationId: operationId,
        parameters: this.extractRequest(specItem, payload),
        responses: this.extractResponse(specItem, payload),
      };
      return example;
    }
    return undefined;
  }

  private cachePayload(specItem: any, payload: any) {
    /**
     *  1 cache parameter model
     *
     *  2 cache response model
     *
     *  3 merged cache
     */
    this.extractRequest(specItem, payload);
    this.extractResponse(specItem, payload);
    this.payloadCache.mergeCache();
  }

  private validatePayload(specItem: any, payload: any, operationId: string) {
    const specApiVersion = this.spec.info.version;
    for (const statusCode of Object.keys(payload)) {
      // remove payload with undefined status code
      if (!(statusCode in specItem.content.responses)) {
        delete payload[statusCode];
        continue;
      }
      // remove payload with inconsistent api-version
      if (!("query" in payload[statusCode].liveRequest)) {
        continue;
      }
      const realApiVersion = payload[statusCode].liveRequest.query["api-version"];
      if (realApiVersion && realApiVersion !== specApiVersion) {
        delete payload[statusCode];
        log.error(
          `${operationId} payload ${statusCode}.json's api-version is ${realApiVersion}, inconsistent with swagger spec's api-version ${specApiVersion}`
        );
      }
    }
  }

  private extractRequest(specItem: any, payload: any) {
    log.info("extractRequest");

    const liveRequest: any = this.getRequestPayload(specItem, payload);
    if (!liveRequest) {
      log.warn(`no live request in payload`);
      return {};
    }
    const request = this.translator.extractRequest(specItem, liveRequest) || {};
    return request;
  }

  private getRequestPayload(specItem: any, payload: any) {
    const longRunning = util.isLongRunning(specItem);
    for (const statusCode in payload) {
      if (longRunning && statusCode === "200") {
        continue;
      }
      if ("liveRequest" in payload[statusCode]) {
        return payload[statusCode].liveRequest;
      }
    }
  }

  private extractResponse(specItem: any, payload: any) {
    log.info("extractResponse");

    const specResp = specItem.content.responses;
    const longRunning: boolean = specItem.content["x-ms-long-running-operation"];

    // below handled status code also should add in swaggerMocker.ts mockForExample() preHandledStatusCode array

    if (longRunning && !("202" in specResp) && !("201" in specResp)) {
      // console.warn('x-ms-long-running-operation is true, but no 202 or 201 response');
      return {};
    }

    if (longRunning && !("200" in specResp || "204" in specResp)) {
      // console.warn('x-ms-long-running-operation is true, but no 200 or 204 response');
    }

    if (!longRunning && ("202" in specResp || "201" in specResp)) {
      // console.warn('x-ms-long-running-operation is not set true, but 202 or 201 response is provided');
      return {};
    }
    const resp: any = {};

    if (!longRunning && "200" in specResp) {
      this.getResponseExample(specItem, payload, resp, "200", false);
    }

    if ("201" in specResp) {
      this.getResponseExample(specItem, payload, resp, "201", "200" in specResp);
    }

    if ("202" in specResp) {
      this.getResponseExample(specItem, payload, resp, "202", "200" in specResp);
    }

    if ("204" in specResp) {
      resp["204"] = {};
    }
    return resp;
  }

  private getLongrunResp(specItem: any, payload: any) {
    const payload200: any = payload[200];
    if (!payload200 || !("liveResponse" in payload200)) {
      // console.warn(`Payload doesn't have the response result for long running case`);
      return {};
    }
    return {
      body:
        "schema" in specItem.content.responses["200"]
          ? this.translator.filterBodyContent(
              payload200.liveResponse.body,
              specItem.content.responses["200"].schema,
              false
            )
          : undefined,
    };
  }

  private getResponseExample(
    specItem: any,
    payloadGeneral: any,
    resp: any,
    statusCode: string,
    getAsyncResp: boolean
  ) {
    const payload: any = payloadGeneral[statusCode];
    if (!payload || !("liveResponse" in payload)) {
      // console.warn(`no payload recording for status code = ${statusCode}`);
      resp[statusCode] = {};
    } else {
      resp[statusCode] = this.translator.extractResponse(
        specItem,
        payload.liveResponse,
        statusCode
      );
    }

    if (getAsyncResp) {
      resp["200"] = this.getLongrunResp(specItem, payloadGeneral);
    }
  }
}
