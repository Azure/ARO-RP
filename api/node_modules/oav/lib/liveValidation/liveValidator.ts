// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

import * as http from "http";
import * as os from "os";
import * as path from "path";
import { resolve as pathResolve } from "path";
import { ParsedUrlQuery } from "querystring";
import * as util from "util";
import { URL } from "url";
import * as glob from "glob";
import * as _ from "lodash";
import { diffRequestResponse } from "../armValidator/roundTripValidator";
import * as models from "../models";
import { requestResponseDefinition } from "../models/requestResponse";
import { LowerHttpMethods, SwaggerSpec } from "../swagger/swaggerTypes";
import {
  SchemaValidateFunction,
  SchemaValidateIssue,
  SchemaValidator,
} from "../swaggerValidator/schemaValidator";
import * as C from "../util/constants";
import { log } from "../util/logging";
import * as utils from "../util/utils";
import { RuntimeException } from "../util/validationError";
import { inversifyGetContainer, inversifyGetInstance, TYPES } from "../inversifyUtils";
import { setDefaultOpts } from "../swagger/loader";
import { apiValidationErrors, ApiValidationErrorCode } from "../util/errorDefinitions";
import {
  kvPairsToObject,
  getProviderFromPathTemplate,
  getProviderFromSpecPath,
} from "../util/utils";
import { LiveValidatorLoader, LiveValidatorLoaderOption } from "./liveValidatorLoader";
import { OperationSearcher } from "./operationSearcher";
import {
  LiveRequest,
  LiveResponse,
  OperationContext,
  validateSwaggerLiveRequest,
  validateSwaggerLiveResponse,
  ValidationRequest,
} from "./operationValidator";

export interface LiveValidatorOptions extends LiveValidatorLoaderOption {
  swaggerPaths: string[];
  git: {
    shouldClone: boolean;
    url?: string;
    branch?: string;
  };
  useRelativeSourceLocationUrl?: boolean;
  directory: string;
  swaggerPathsPattern: string[];
  excludedSwaggerPathsPattern: string[];
  isPathCaseSensitive: boolean;
  loadValidatorInBackground: boolean;
  loadValidatorInInitialize: boolean;
  enableRoundTripValidator?: boolean;
  isArmCall?: boolean;
}

export interface RequestResponsePair {
  readonly liveRequest: LiveRequest;
  readonly liveResponse: LiveResponse;
}

export interface LiveValidationResult {
  readonly isSuccessful?: boolean;
  readonly operationInfo: OperationContext;
  readonly errors: LiveValidationIssue[];
  readonly runtimeException?: RuntimeException;
}

export interface RequestResponseLiveValidationResult {
  readonly requestValidationResult: LiveValidationResult;
  readonly responseValidationResult: LiveValidationResult;
  readonly runtimeException?: RuntimeException;
}

export type LiveValidationIssue = {
  code: ApiValidationErrorCode;
  pathsInPayload: string[];
  resourceIds?: string[];
  documentationUrl?: string;
} & Omit<SchemaValidateIssue, "code">;

/**
 * Additional data to log.
 */
interface Meta {
  [key: string]: any;
}

/**
 * Options for a validation operation.
 * If `includeErrors` is missing or empty, all error codes will be included.
 */
export interface ValidateOptions {
  readonly includeErrors?: ApiValidationErrorCode[];
  readonly includeOperationMatch?: boolean;
}

export enum LiveValidatorLoggingLevels {
  error = "error",
  warn = "warn",
  info = "info",
  verbose = "verbose",
  debug = "debug",
  silly = "silly",
}

export enum LiveValidatorLoggingTypes {
  trace = "trace",
  perfTrace = "perfTrace",
  error = "error",
  incomingRequest = "incomingRequest",
  specTrace = "specTrace",
}

/**
 * @class
 * Live Validator for Azure swagger APIs.
 */
export class LiveValidator {
  public options: LiveValidatorOptions;

  public operationSearcher: OperationSearcher;

  public swaggerList: string[] = [];

  private logFunction?: (message: string, level: string, meta?: Meta) => void;

  private loader?: LiveValidatorLoader;

  private loadInBackgroundComplete: boolean = false;

  private validateRequestResponsePair?: SchemaValidateFunction;

  /**
   * Constructs LiveValidator based on provided options.
   *
   * @param {object} ops The configuration options.
   * @param {callback function} logCallback The callback logger.
   *
   * @returns CacheBuilder Returns the configured CacheBuilder object.
   */
  public constructor(
    options?: Partial<LiveValidatorOptions>,
    logCallback?: (message: string, level: string, meta?: Meta) => void
  ) {
    const ops: Partial<LiveValidatorOptions> = options || {};
    this.logFunction = logCallback;

    setDefaultOpts(ops, {
      swaggerPaths: [],
      excludedSwaggerPathsPattern: C.DefaultConfig.ExcludedSwaggerPathsPattern,
      directory: path.resolve(os.homedir(), "repo"),
      isPathCaseSensitive: false,
      loadValidatorInBackground: true,
      loadValidatorInInitialize: false,
      isArmCall: false,
    });

    if (!ops.git) {
      ops.git = {
        url: "https://github.com/Azure/azure-rest-api-specs.git",
        shouldClone: false,
      };
    }

    if (!ops.git.url) {
      ops.git.url = "https://github.com/Azure/azure-rest-api-specs.git";
    }
    if (!ops.git.shouldClone) {
      ops.git.shouldClone = false;
    }

    this.options = ops as LiveValidatorOptions;
    this.logging(`Creating livevalidator with options:${JSON.stringify(this.options)}`);
    this.operationSearcher = new OperationSearcher(this.logging);
  }

  /**
   * Initializes the Live Validator.
   */
  public async initialize(): Promise<void> {
    const startTime = Date.now();
    // Clone github repository if required
    if (this.options.git.shouldClone && this.options.git.url) {
      const cloneStartTime = Date.now();
      utils.gitClone(this.options.directory, this.options.git.url, this.options.git.branch);
      this.logging(
        `Clone spec repository ${this.options.git.url}, branch:${this.options.git.branch} in livevalidator.initialize`
      );
      this.logging(
        `Clone spec repository ${this.options.git.url}, branch:${this.options.git.branch}`,
        LiveValidatorLoggingLevels.info,
        LiveValidatorLoggingTypes.perfTrace,
        "Oav.liveValidator.initialize.gitclone",
        Date.now() - cloneStartTime
      );
    }

    // Construct array of swagger paths to be used for building a cache
    this.logging("Get swagger path.");
    const swaggerPaths = await this.getSwaggerPaths();
    const container = inversifyGetContainer();
    this.loader = inversifyGetInstance(LiveValidatorLoader, {
      container,
      fileRoot: this.options.directory,
      ...this.options,
      loadSuppression: this.options.loadSuppression ?? Object.keys(apiValidationErrors),
    });
    this.loader.logging = this.logging;

    // re-set the transform context after set the logging function
    this.loader.setTransformContext();
    const schemaValidator = container.get(TYPES.schemaValidator) as SchemaValidator;
    this.validateRequestResponsePair = await schemaValidator.compileAsync(
      requestResponseDefinition
    );

    const allSpecs: SwaggerSpec[] = [];
    while (swaggerPaths.length > 0) {
      const swaggerPath = swaggerPaths.shift()!;
      this.swaggerList.push(swaggerPath);
      const spec = await this.getSwaggerInitializer(this.loader!, swaggerPath);
      if (spec !== undefined) {
        allSpecs.push(spec);
      }
    }

    this.logging("Apply global transforms for all specs");
    try {
      this.loader.transformLoadedSpecs();
    } catch (e) {
      // keeps building validator if it fails to tranform specs coz global transformers catches the exceptions and continue other schema transformings;
      // this error will be reported in validator building or validation runtime.
      const errMsg = `Failed to transform loaded specs, detail error message:${
        (e as any)?.message
      }.\nError stack:${(e as any)?.stack}`;
      this.logging(
        errMsg,
        LiveValidatorLoggingLevels.error,
        LiveValidatorLoggingTypes.specTrace,
        "Oav.liveValidator.initialize.transformLoadedSpecs"
      );
    }

    if (this.options.loadValidatorInInitialize) {
      this.logging("Building validator in initialization time...");
      let spec;
      while (allSpecs.length > 0) {
        try {
          spec = allSpecs.shift()!;
          const loadStart = Date.now();
          await this.loader.buildAjvValidator(spec);
          const durationInMs = Date.now() - loadStart;
          this.logging(
            `Complete building validator for ${spec._filePath} in initialization time`,
            LiveValidatorLoggingLevels.info,
            LiveValidatorLoggingTypes.perfTrace,
            "Oav.liveValidator.initialize.loader.buildAjvValidator",
            durationInMs
          );
          this.logging(
            `Complete building validator for spec ${spec._filePath}`,
            LiveValidatorLoggingLevels.info,
            LiveValidatorLoggingTypes.specTrace,
            "Oav.liveValidator.initialize.loader.buildAjvValidator",
            durationInMs,
            {
              providerNamespace: spec._providerNamespace ?? "unknown",
              apiVersion: spec.info.version,
              specName: spec._filePath,
            }
          );
        } catch (e) {
          this.logging(
            `Failed to build validator for spec ${spec?._filePath}.\nErrorMessage:${
              (e as any)?.message
            }.\nErrorStack:${(e as any)?.stack}`,
            LiveValidatorLoggingLevels.error,
            LiveValidatorLoggingTypes.specTrace,
            "Oav.liveValidator.initialize.loader.buildAjvValidator",
            undefined,
            {
              providerNamespace: spec?._providerNamespace ?? "unknown",
              apiVersion: spec?.info.version ?? "unknown",
              specName: spec?._filePath,
            }
          );
        }
      }

      this.loader = undefined;
    }

    this.logging("Cache initialization complete.");
    const elapsedTime = Date.now() - startTime;
    this.logging(
      `Cache complete initialization with DurationInMs:${elapsedTime}`,
      LiveValidatorLoggingLevels.info,
      LiveValidatorLoggingTypes.trace,
      "Oav.liveValidator.initialize"
    );
    this.logging(
      `Cache complete initialization`,
      LiveValidatorLoggingLevels.info,
      LiveValidatorLoggingTypes.perfTrace,
      "Oav.liveValidator.initialize",
      elapsedTime
    );

    if (this.options.loadValidatorInBackground) {
      this.logging("Building validator in background...");
      // eslint-disable-next-line @typescript-eslint/no-floating-promises
      this.loadAllSpecValidatorInBackground(allSpecs);
    }
  }

  public isLoadInBackgroundCompleted() {
    return this.loadInBackgroundComplete;
  }

  private async loadAllSpecValidatorInBackground(allSpecs: SwaggerSpec[]) {
    const backgroundStartTime = Date.now();
    utils.shuffleArray(allSpecs);
    while (allSpecs.length > 0) {
      let spec;
      try {
        spec = allSpecs.shift()!;
        const startTime = Date.now();
        this.logging(
          `Start building validator for ${spec._filePath} in background`,
          LiveValidatorLoggingLevels.debug,
          LiveValidatorLoggingTypes.trace,
          "Oav.liveValidator.loadAllSpecValidatorInBackground"
        );
        await this.loader!.buildAjvValidator(spec, { inBackground: true });
        const elapsedTime = Date.now() - startTime;
        this.logging(
          `Complete building validator for ${spec._filePath} in background with DurationInMs:${elapsedTime}.`,
          LiveValidatorLoggingLevels.debug,
          LiveValidatorLoggingTypes.trace,
          "Oav.liveValidator.loadAllSpecValidatorInBackground"
        );
        this.logging(
          `Complete building validator for ${spec._filePath} in background`,
          LiveValidatorLoggingLevels.info,
          LiveValidatorLoggingTypes.perfTrace,
          "Oav.liveValidator.loadAllSpecValidatorInBackground",
          elapsedTime
        );
        this.logging(
          `Complete building validator for spec ${spec._filePath}`,
          LiveValidatorLoggingLevels.info,
          LiveValidatorLoggingTypes.specTrace,
          "Oav.liveValidator.loadAllSpecValidatorInBackground",
          elapsedTime,
          {
            providerNamespace: spec._providerNamespace ?? "unknown",
            apiVersion: spec.info.version,
            specName: spec._filePath,
          }
        );
      } catch (e) {
        this.logging(
          `Failed to build validator for spec ${spec?._filePath}.\nErrorMessage:${
            (e as any)?.message
          }.\nErrorStack:${(e as any)?.stack}`,
          LiveValidatorLoggingLevels.error,
          LiveValidatorLoggingTypes.specTrace,
          "Oav.liveValidator.loadAllSpecValidatorInBackground",
          undefined,
          {
            providerNamespace: spec?._providerNamespace ?? "unknown",
            apiVersion: spec?.info.version ?? "unknown",
            specName: spec?._filePath,
          }
        );
      }
    }

    this.loader = undefined;
    this.loadInBackgroundComplete = true;
    const elapsedTimeForBuild = Date.now() - backgroundStartTime;
    this.logging(
      `Completed building validator for all specs in background with DurationInMs:${elapsedTimeForBuild}.`,
      LiveValidatorLoggingLevels.info,
      LiveValidatorLoggingTypes.trace,
      "Oav.liveValidator.loadAllSpecValidatorInBackground"
    );
    this.logging(
      `Completed building validator for all specs in background`,
      LiveValidatorLoggingLevels.info,
      LiveValidatorLoggingTypes.perfTrace,
      "Oav.liveValidator.loadAllSpecValidatorInBackground",
      elapsedTimeForBuild
    );
  }

  /**
   *  Validates live request.
   */
  public async validateLiveRequest(
    liveRequest: LiveRequest,
    options: ValidateOptions = {},
    operationInfo?: OperationContext
  ): Promise<LiveValidationResult> {
    const startTime = Date.now();
    const { info, error } = this.getOperationInfo(liveRequest, operationInfo);
    if (error !== undefined) {
      this.logging(
        `ErrorMessage:${error.message}.ErrorStack:${error.stack}`,
        LiveValidatorLoggingLevels.error,
        LiveValidatorLoggingTypes.error,
        "Oav.liveValidator.validateLiveRequest",
        undefined,
        info.validationRequest
      );
      return {
        isSuccessful: undefined,
        errors: [],
        runtimeException: error,
        operationInfo: info,
      };
    }
    if (!liveRequest.query) {
      liveRequest.query = kvPairsToObject(
        new URL(liveRequest.url, "https://management.azure.com").searchParams
      );
    }
    let errors: LiveValidationIssue[] = [];
    let runtimeException;
    try {
      errors = await validateSwaggerLiveRequest(
        liveRequest,
        info,
        this.loader,
        options.includeErrors,
        this.options.isArmCall,
        this.logging
      );
    } catch (reqValidationError) {
      const msg =
        `An error occurred while validating the live request for operation ` +
        `"${info.operationId}". The error is:\n ` +
        `${util.inspect(reqValidationError, { depth: null })}`;
      runtimeException = { code: C.ErrorCodes.RequestValidationError.name, message: msg };
      this.logging(
        msg,
        LiveValidatorLoggingLevels.error,
        LiveValidatorLoggingTypes.error,
        "Oav.liveValidator.validateLiveRequest",
        undefined,
        info.validationRequest
      );
    }
    const elapsedTime = Date.now() - startTime;
    this.logging(
      `Complete request validation`,
      LiveValidatorLoggingLevels.info,
      LiveValidatorLoggingTypes.perfTrace,
      "Oav.liveValidator.validateLiveRequest",
      elapsedTime,
      info.validationRequest
    );
    if (!options.includeOperationMatch) {
      delete info.operationMatch;
      delete info.validationRequest;
    }
    return {
      isSuccessful: runtimeException ? undefined : errors.length === 0,
      operationInfo: info,
      errors,
      runtimeException,
    };
  }

  /**
   * Validates live response.
   */
  public async validateLiveResponse(
    liveResponse: LiveResponse,
    specOperation: { url: string; method: string },
    options: ValidateOptions = {},
    operationInfo?: OperationContext
  ): Promise<LiveValidationResult> {
    const startTime = Date.now();
    const { info, error } = this.getOperationInfo(specOperation, operationInfo);
    if (error !== undefined) {
      this.logging(
        `ErrorMessage:${error.message}.ErrorStack:${error.stack}`,
        LiveValidatorLoggingLevels.error,
        LiveValidatorLoggingTypes.error,
        "Oav.liveValidator.validateLiveResponse",
        undefined,
        info.validationRequest
      );
      return {
        isSuccessful: undefined,
        errors: [],
        runtimeException: error,
        operationInfo: { apiVersion: C.unknownApiVersion, operationId: C.unknownOperationId },
      };
    }
    let errors: LiveValidationIssue[] = [];
    let runtimeException;
    this.transformResponseStatusCode(liveResponse);
    try {
      errors = await validateSwaggerLiveResponse(
        liveResponse,
        info,
        this.loader,
        options.includeErrors,
        this.options.isArmCall,
        this.logging
      );
    } catch (resValidationError) {
      const msg =
        `An error occurred while validating the live response for operation ` +
        `"${info.operationId}". The error is:\n ` +
        `${util.inspect(resValidationError, { depth: null })}`;
      runtimeException = { code: C.ErrorCodes.RequestValidationError.name, message: msg };
      this.logging(
        msg,
        LiveValidatorLoggingLevels.error,
        LiveValidatorLoggingTypes.error,
        "Oav.liveValidator.validateLiveResponse",
        undefined,
        info.validationRequest
      );
    }
    const elapsedTime = Date.now() - startTime;
    this.logging(
      `Complete response validation`,
      LiveValidatorLoggingLevels.info,
      LiveValidatorLoggingTypes.perfTrace,
      "Oav.liveValidator.validateLiveResponse",
      elapsedTime,
      info.validationRequest
    );
    if (!options.includeOperationMatch) {
      delete info.operationMatch;
      delete info.validationRequest;
    }
    return {
      isSuccessful: runtimeException ? undefined : errors.length === 0,
      operationInfo: info,
      errors,
      runtimeException,
    };
  }

  /**
   * Validates live request and response.
   */
  public async validateLiveRequestResponse(
    requestResponseObj: RequestResponsePair,
    options?: ValidateOptions
  ): Promise<RequestResponseLiveValidationResult> {
    const validationResult = {
      requestValidationResult: {
        errors: [],
        operationInfo: { apiVersion: C.unknownApiVersion, operationId: C.unknownOperationId },
      },
      responseValidationResult: {
        errors: [],
        operationInfo: { apiVersion: C.unknownApiVersion, operationId: C.unknownOperationId },
      },
    };
    if (!requestResponseObj) {
      const message =
        'requestResponseObj cannot be null or undefined and must be of type "object".';
      return {
        ...validationResult,
        runtimeException: {
          code: C.ErrorCodes.IncorrectInput.name,
          message,
        },
      };
    }

    const errors = this.validateRequestResponsePair!({}, requestResponseObj);
    if (errors.length > 0) {
      const error = errors[0];
      const message =
        `Found errors "${error.message}" in the provided input in path ${error.jsonPathsInPayload[0]}:\n` +
        `${util.inspect(requestResponseObj, { depth: null })}.`;
      return {
        ...validationResult,
        runtimeException: {
          code: C.ErrorCodes.IncorrectInput.name,
          message,
        },
      };
    }

    const request = requestResponseObj.liveRequest;
    const response = requestResponseObj.liveResponse;
    this.transformResponseStatusCode(response);

    const requestValidationResult = await this.validateLiveRequest(request, {
      ...options,
      includeOperationMatch: true,
    });
    const info = requestValidationResult.operationInfo;
    const responseValidationResult =
      requestValidationResult.isSuccessful === undefined &&
      requestValidationResult.runtimeException === undefined
        ? requestValidationResult
        : await this.validateLiveResponse(
            response,
            request,
            {
              ...options,
              includeOperationMatch: false,
            },
            info
          );

    delete info.validationRequest;
    delete info.operationMatch;

    return {
      requestValidationResult,
      responseValidationResult,
    };
  }

  private transformResponseStatusCode(liveResponse: LiveResponse) {
    // If status code is passed as a status code string (e.g. "OK") transform it to the status code
    // number (e.g. '200').
    if (
      !http.STATUS_CODES[liveResponse.statusCode] &&
      utils.statusCodeStringToStatusCode[liveResponse.statusCode.toLowerCase()]
    ) {
      liveResponse.statusCode =
        utils.statusCodeStringToStatusCode[liveResponse.statusCode.toLowerCase()];
    }
  }

  public getOperationInfo(
    request: { url: string; method: string; headers?: { [propertyName: string]: string } },
    operationInfo?: OperationContext
  ): {
    info: OperationContext;
    error?: any;
  } {
    const info = operationInfo ?? {
      apiVersion: C.unknownApiVersion,
      operationId: C.unknownOperationId,
    };
    try {
      if (info.validationRequest === undefined) {
        info.validationRequest = parseValidationRequest(
          request.url,
          request.method,
          request.headers
        );
      }
      if (info.operationMatch === undefined) {
        const result = this.operationSearcher.search(info.validationRequest);
        info.apiVersion = result.apiVersion;
        info.operationMatch = result.operationMatch;
      }
      info.operationId = info.operationMatch.operation.operationId!;
      return { info };
    } catch (error) {
      return { info, error };
    }
  }

  public getResolvedOperationInfo(
    request: { url: string; method: string; headers?: { [propertyName: string]: string } },
    operationInfo?: OperationContext
  ): {
    info: OperationContext;
    error?: any;
  } {
    const info = operationInfo ?? {
      apiVersion: C.unknownApiVersion,
      operationId: C.unknownOperationId,
    };
    try {
      if (info.validationRequest === undefined) {
        info.validationRequest = parseValidationRequest(
          request.url,
          request.method,
          request.headers
        );
      }
      if (info.operationMatch === undefined) {
        const result = this.operationSearcher.search(info.validationRequest);
        info.apiVersion = result.apiVersion;
        info.operationMatch = result.operationMatch;
      }
      info.operationId = info.operationMatch.operation.operationId!;
      return { info };
    } catch (error) {
      return { info, error };
    }
  }

  private async getMatchedPaths(jsonsPattern: string | string[]): Promise<string[]> {
    const startTime = Date.now();
    let matchedPaths: string[] = [];
    if (typeof jsonsPattern === "string") {
      matchedPaths = glob
        .sync(jsonsPattern, {
          ignore: this.options.excludedSwaggerPathsPattern,
          nodir: true,
        })
        // Sort alphabetically for compat with glob <= 8 (https://github.com/isaacs/node-glob/blob/v7.2.3/common.js#L20).
        // Tests pass without sorting, but safer to keep sorting for consistent behavior.
        .sort((a, b) => a.localeCompare(b, "en"));
    } else {
      for (const pattern of jsonsPattern) {
        const res: string[] = glob
          .sync(pattern, {
            ignore: this.options.excludedSwaggerPathsPattern,
            nodir: true,
          })
          // Sort alphabetically for compat with glob <= 8 (https://github.com/isaacs/node-glob/blob/v7.2.3/common.js#L20).
          // Tests pass without sorting, but safer to keep sorting for consistent behavior.
          .sort((a, b) => a.localeCompare(b, "en"));
        for (const path of res) {
          if (!matchedPaths.includes(path)) {
            matchedPaths.push(path);
          }
        }
      }
    }
    this.logging(
      `Using swaggers found from directory: "${
        this.options.directory
      }" and pattern: "${jsonsPattern.toString()}".
      Total paths count: ${matchedPaths.length}`,
      LiveValidatorLoggingLevels.info
    );
    this.logging(
      `Using swaggers found from directory: "${
        this.options.directory
      }" and pattern: "${jsonsPattern.toString()}".
      Total paths count: ${matchedPaths.length}`,
      LiveValidatorLoggingLevels.info,
      LiveValidatorLoggingTypes.perfTrace,
      "Oav.livevalidator.getMatchedPaths",
      Date.now() - startTime
    );
    return matchedPaths;
  }

  public async validateRoundTrip(
    requestResponseObj: RequestResponsePair
  ): Promise<LiveValidationResult> {
    const startTime = Date.now();
    if (this.operationSearcher === undefined) {
      const msg = "operationSearcher should be initialized before this call.";
      const runtimeException = { code: C.ErrorCodes.RoundtripValidationError.name, message: msg };
      return {
        isSuccessful: undefined,
        errors: [],
        operationInfo: {
          operationId: "",
          apiVersion: "",
        },
        runtimeException: runtimeException,
      };
    }
    const { info, error } = this.getResolvedOperationInfo(requestResponseObj.liveRequest);
    console.log(info.operationId);
    if (error !== undefined) {
      this.logging(
        `ErrorMessage:${error.message}.ErrorStack:${error.stack}`,
        LiveValidatorLoggingLevels.error,
        LiveValidatorLoggingTypes.error,
        "Oav.liveValidator.validateRoundTrip",
        undefined,
        info.validationRequest
      );
      return {
        isSuccessful: undefined,
        errors: [],
        runtimeException: error,
        operationInfo: info,
      };
    }

    let errors: LiveValidationIssue[] = [];
    let runtimeException;
    try {
      const res = diffRequestResponse(
        requestResponseObj,
        info,
        this.loader?.getResolvedJsonLoader()!
      );
      for (const re of res) {
        if (re !== undefined) {
          errors.push(re);
        }
      }
    } catch (validationErr) {
      const msg =
        `An error occurred while validating the live request for operation ` +
        `"${info.operationId}". The error is:\n ` +
        `${util.inspect(validationErr, { depth: null })}`;
      runtimeException = { code: C.ErrorCodes.RoundtripValidationError.name, message: msg };
      this.logging(
        msg,
        LiveValidatorLoggingLevels.error,
        LiveValidatorLoggingTypes.error,
        "Oav.liveValidator.validateRoundTrip",
        undefined,
        info.validationRequest
      );
      return {
        isSuccessful: undefined,
        operationInfo: info,
        errors,
        runtimeException,
      };
    }
    const elapsedTime = Date.now() - startTime;
    this.logging(
      `Complete roundtrip validation`,
      LiveValidatorLoggingLevels.info,
      LiveValidatorLoggingTypes.perfTrace,
      "Oav.liveValidator.validateRoundTrip",
      elapsedTime,
      info.validationRequest
    );
    delete info.validationRequest;
    delete info.operationMatch;
    return {
      isSuccessful: runtimeException ? undefined : errors.length === 0,
      operationInfo: info,
      errors,
      runtimeException,
    };
  }

  private async getSwaggerPaths(): Promise<string[]> {
    if (this.options.swaggerPaths.length !== 0) {
      this.logging(
        `Using user provided swagger paths by options.swaggerPaths. Total paths count: ${this.options.swaggerPaths.length}`
      );
      return this.options.swaggerPaths;
    } else {
      const allJsonsPattern = path.join(this.options.directory, "/specification/**/*.json");
      const swaggerPathPatterns: string[] = [];
      if (
        this.options.swaggerPathsPattern === undefined ||
        this.options.swaggerPathsPattern.length === 0
      ) {
        return this.getMatchedPaths(allJsonsPattern);
      } else {
        this.options.swaggerPathsPattern.map((item) => {
          swaggerPathPatterns.push(path.join(this.options.directory, item));
        });
        return this.getMatchedPaths(swaggerPathPatterns);
      }
    }
  }

  private async getSwaggerInitializer(
    loader: LiveValidatorLoader,
    swaggerPath: string
  ): Promise<SwaggerSpec | undefined> {
    const startTime = Date.now();
    this.logging(`Building cache from:${swaggerPath}`, LiveValidatorLoggingLevels.debug);
    let spec;
    let resolvedSpec;
    try {
      spec = await loader.load(pathResolve(swaggerPath));
      const elapsedTimeLoadSpec = Date.now() - startTime;
      this.logging(
        `Load spec ${swaggerPath}`,
        LiveValidatorLoggingLevels.info,
        LiveValidatorLoggingTypes.perfTrace,
        "Oav.liveValidator.getSwaggerInitializer.loader.load",
        elapsedTimeLoadSpec
      );

      const startTimeAddSpecToCache = Date.now();
      if (this.options.enableRoundTripValidator) {
        resolvedSpec = await loader.load(pathResolve(swaggerPath), true);
        this.operationSearcher.addSpecToCache(resolvedSpec);
      } else {
        this.operationSearcher.addSpecToCache(spec);
      }
      // TODO: add data-plane RP to cache.
      this.logging(
        `Add spec to cache ${swaggerPath}`,
        LiveValidatorLoggingLevels.info,
        LiveValidatorLoggingTypes.perfTrace,
        "Oav.liveValidator.getSwaggerInitializer.operationSearcher.addSpecToCache",
        Date.now() - startTimeAddSpecToCache
      );

      const elapsedTime = Date.now() - startTime;
      this.logging(
        `Complete loading spec ${spec._filePath}`,
        LiveValidatorLoggingLevels.info,
        LiveValidatorLoggingTypes.specTrace,
        "Oav.liveValidator.getSwaggerInitializer",
        elapsedTime,
        {
          providerNamespace: spec._providerNamespace ?? "unknown",
          apiVersion: spec.info.version,
          specName: spec._filePath,
        }
      );

      return spec;
    } catch (err) {
      this.logging(
        `Unable to initialize "${swaggerPath}" file from SpecValidator. We are ` +
          `ignoring this swagger file and continuing to build cache for other valid specs.\nErrorMessage: ${
            (err as any)?.message
          };\nErrorStack: ${(err as any)?.stack}`,
        LiveValidatorLoggingLevels.warn,
        LiveValidatorLoggingTypes.error
      );
      const pathProvider = getProviderFromSpecPath(swaggerPath);
      this.logging(
        `Failed to load spec ${swaggerPath}`,
        LiveValidatorLoggingLevels.error,
        LiveValidatorLoggingTypes.specTrace,
        "Oav.liveValidator.getSwaggerInitializer",
        undefined,
        {
          providerNamespace: pathProvider ? pathProvider.provider : "unknown",
          apiVersion: spec ? spec.info.version : "unknown",
          specName: swaggerPath,
        }
      );

      return undefined;
    }
  }

  private logging = (
    message: string,
    level?: LiveValidatorLoggingLevels,
    loggingType?: LiveValidatorLoggingTypes,
    operationName?: string,
    durationInMilliseconds?: number,
    validationRequest?: ValidationRequest
  ) => {
    level = level || LiveValidatorLoggingLevels.info;
    loggingType = loggingType || LiveValidatorLoggingTypes.trace;
    operationName = operationName || "";
    durationInMilliseconds = durationInMilliseconds || 0;
    if (this.logFunction !== undefined) {
      if (validationRequest !== undefined && validationRequest !== null) {
        this.logFunction(message, level, {
          CorrelationId: validationRequest.correlationId,
          ActivityId: validationRequest.activityId,
          ProviderNamespace: validationRequest.providerNamespace,
          ResourceType: validationRequest.resourceType,
          ApiVersion: validationRequest.apiVersion,
          OperationName: operationName,
          LoggingType: loggingType,
          DurationInMilliseconds: durationInMilliseconds,
          SpecName: validationRequest.specName,
        });
      } else {
        this.logFunction(message, level, {
          OperationName: operationName,
          LoggingType: loggingType,
          DurationInMilliseconds: durationInMilliseconds,
        });
      }
    } else {
      log.log(level, message);
    }
  };
}

/**
 * OAV expects the url that is sent to match exactly with the swagger path. For this we need to keep only the part after
 * where the swagger path starts. Currently those are '/subscriptions' and '/providers'.
 */
export function formatUrlToExpectedFormat(requestUrl: string): string {
  return requestUrl.substring(requestUrl.search("/?(subscriptions|providers)/i"));
}

/**
 * Parse the validation request information.
 *
 * @param  requestUrl The url of service api call.
 *
 * @param requestMethod The http verb for the method to be used for lookup.
 *
 * @param correlationId The id to correlate the api calls.
 *
 * @param activityId The id maps to request id, used by RPaaS.
 *
 * @returns parsed ValidationRequest info.
 *
 * @deprecated use parseValidationRequest instead.
 */
export const legacyParseValidationRequest = (
  requestUrl: string,
  requestMethod: string | undefined | null,
  correlationId: string,
  activityId: string
): ValidationRequest => {
  if (
    requestUrl === undefined ||
    requestUrl === null ||
    typeof requestUrl.valueOf() !== "string" ||
    !requestUrl.trim().length
  ) {
    const msg =
      "An error occurred while trying to parse validation payload." +
      'requestUrl is a required parameter of type "string" and it cannot be an empty string.';
    const e = new models.LiveValidationError(C.ErrorCodes.PotentialOperationSearchError.name, msg);
    throw e;
  }

  if (
    requestMethod === undefined ||
    requestMethod === null ||
    typeof requestMethod.valueOf() !== "string" ||
    !requestMethod.trim().length
  ) {
    const msg =
      "An error occurred while trying to parse validation payload." +
      'requestMethod is a required parameter of type "string" and it cannot be an empty string.';
    const e = new models.LiveValidationError(C.ErrorCodes.PotentialOperationSearchError.name, msg);
    throw e;
  }
  return parseValidationRequest(requestUrl, requestMethod, {
    "x-ms-correlation-request-id": correlationId,
    "x-ms-request-id": activityId,
  });
};

/**
 * Parse the validation request information.
 *
 * @param requestUrl The url of service api call.
 *
 * @param requestMethod The http verb for the method to be used for lookup.
 *
 * @param headers Optional headers
 *
 * @returns parsed ValidationRequest info.
 */
export const parseValidationRequest = (
  requestUrl: string,
  requestMethod: string,
  headers?: { [propertyName: string]: string }
): ValidationRequest => {
  let queryStr;
  let apiVersion = "";
  let resourceType = "";
  let providerNamespace = "";

  const parsedUrl = new URL(requestUrl, "https://management.azure.com");
  const pathStr = parsedUrl.pathname || "";
  if (pathStr !== "") {
    // Lower all the keys and values of query parameters before searching for `api-version`
    const queryObject: ParsedUrlQuery = {};
    parsedUrl.searchParams.forEach((value, key) => {
      queryObject[key.toLowerCase()] = value.toLowerCase();
    });

    apiVersion = (queryObject["api-version"] || C.unknownApiVersion) as string;
    providerNamespace = getProviderFromPathTemplate(pathStr) || C.unknownResourceProvider;
    resourceType = utils.getResourceType(pathStr, providerNamespace);

    // Provider would be provider found from the path or Microsoft.Unknown
    providerNamespace = providerNamespace || C.unknownResourceProvider;
    if (providerNamespace === C.unknownResourceProvider) {
      //apiVersion = C.unknownApiVersion;
    }
    providerNamespace = providerNamespace.toLowerCase();
    apiVersion = apiVersion.toLowerCase();
    queryStr = queryObject;
    requestMethod = requestMethod.toLowerCase();
  }
  return {
    providerNamespace,
    resourceType,
    apiVersion,
    requestMethod: requestMethod as LowerHttpMethods,
    host: parsedUrl.host!,
    pathStr,
    query: queryStr,
    requestUrl: requestUrl,
    correlationId: headers?.["x-ms-correlation-request-id"] || "",
    activityId: headers?.["x-ms-request-id"] || "",
  };
};
