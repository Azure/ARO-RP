// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

import * as fs from "fs";
import * as pathlib from "path";
import { URL } from "url";
import {
  MutableStringMap,
  StringMap,
  entries,
  mapEntries,
  keys,
  values,
} from "@azure-tools/openapi-tools-common";
import swaggerParser from "@apidevtools/swagger-parser";
import { log } from "./util/logging";
import { kvPairsToObject } from "./util/utils";

export interface Options {
  output?: string;
  shouldResolveXmsExamples?: unknown;
  matchApiVersion?: unknown;
}

const mkdirRecursiveSync = (dir: string) => {
  if (!fs.existsSync(dir)) {
    const parent = pathlib.dirname(dir);
    if (parent !== dir) {
      mkdirRecursiveSync(parent);
    }
    fs.mkdirSync(dir);
  }
};

/**
 * @class
 */
export class XMsExampleExtractor {
  private readonly specPath: string;
  private readonly recordings: string;
  private readonly options: Options;
  /**
   * @constructor
   * Initializes a new instance of the xMsExampleExtractor class.
   *
   * @param {string} specPath the swagger spec path
   *
   * @param {object} recordings the folder for recordings
   *
   * @param {object} [options] The options object
   *
   * @param {object} [options.matchApiVersion] Only generate examples if api-version matches.
   * Default: false
   *
   * @param {object} [options.output] Output folder for the generated examples.
   */
  public constructor(specPath: string, recordings: string, options: Options) {
    if (
      specPath === null ||
      specPath === undefined ||
      typeof specPath.valueOf() !== "string" ||
      !specPath.trim().length
    ) {
      throw new Error(
        "specPath is a required property of type string and it cannot be an empty string."
      );
    }

    if (
      recordings === null ||
      recordings === undefined ||
      typeof recordings.valueOf() !== "string" ||
      !recordings.trim().length
    ) {
      throw new Error(
        "recordings is a required property of type string and it cannot be an empty string."
      );
    }

    this.specPath = specPath;
    this.recordings = recordings;
    if (!options) {
      options = {};
    }
    if (options.output === null || options.output === undefined) {
      options.output = process.cwd() + "/output";
    }
    if (
      options.shouldResolveXmsExamples === null ||
      options.shouldResolveXmsExamples === undefined
    ) {
      options.shouldResolveXmsExamples = true;
    }
    if (options.matchApiVersion === null || options.matchApiVersion === undefined) {
      options.matchApiVersion = false;
    }

    this.options = options;
    log.debug(`specPath : ${this.specPath}`);
    log.debug(`recordings : ${this.recordings}`);
    log.debug(`options.output : ${this.options.output}`);
    log.debug(`options.matchApiVersion : ${this.options.matchApiVersion}`);
  }

  public extractOne(
    relativeExamplesPath: string,
    outputExamples: string,
    // eslint-disable-next-line @typescript-eslint/explicit-module-boundary-types
    api: any,
    recordingFileName: string
  ): void {
    const recording = JSON.parse(fs.readFileSync(recordingFileName).toString());
    const paths = api.paths;
    let pathIndex = 0;
    let pathParams: MutableStringMap<number> = {};
    for (const path of keys(paths)) {
      pathIndex++;
      const searchResult = path.match(/\/{\w*\}/g);
      const pathParts = path.split("/");
      let pathToMatch = path;
      pathParams = {};
      if (searchResult !== null) {
        for (const match of searchResult) {
          const splitRegEx = /[{}]/;
          const pathParam = match.split(splitRegEx)[1];

          for (const [part, value] of entries(pathParts)) {
            const pathPart = "/" + value;
            if (pathPart.localeCompare(match) === 0) {
              pathParams[pathParam] = part;
            }
          }
          pathToMatch = pathToMatch.replace(match, "/[^/]+");
        }
      }
      let newPathToMatch = pathToMatch.replace(/\//g, "\\/");
      newPathToMatch = newPathToMatch + "$";

      // for this API path (and method), try to find it in the recording file, and get
      // the data
      const recordingEntries: StringMap<any> = recording.Entries;
      let entryIndex = 0;
      let queryParams: any = {};
      for (const recordingEntry of values(recordingEntries)) {
        entryIndex++;
        const parsedUrl = new URL(recordingEntry.RequestUri, "https://management.azure.com");
        let recordingPath = parsedUrl.href || "";

        queryParams = kvPairsToObject(parsedUrl.searchParams) || {};
        const hostUrl = parsedUrl ? parsedUrl.protocol! + "//" + parsedUrl.hostname! : undefined;

        const headerParams = recordingEntry.RequestHeaders;

        // if command-line included check for API version, validate api-version from URI in
        // recordings matches the api-version of the spec
        if (
          !this.options.matchApiVersion ||
          ("api-version" in queryParams && queryParams["api-version"] === api.info.version)
        ) {
          recordingPath = recordingPath.replace(/\?.*/, "");
          const recordingPathParts = recordingPath.split("/");
          // eslint-disable-next-line @typescript-eslint/prefer-regexp-exec
          const match = recordingPath.match(newPathToMatch);
          if (match !== null) {
            log.silly("path: " + path);
            log.silly("recording path: " + recordingPath);

            const pathParamsValues: MutableStringMap<unknown> = {};
            for (const [p, v] of mapEntries(pathParams)) {
              const index = v;
              pathParamsValues[p] = recordingPathParts[index];
            }
            if (hostUrl !== undefined) {
              pathParamsValues.url = hostUrl;
            }

            // found a match in the recording
            const requestMethodFromRecording = recordingEntry.RequestMethod;
            const infoFromOperation = paths[path][requestMethodFromRecording.toLowerCase()];
            if (typeof infoFromOperation !== "undefined") {
              // need to consider each method in operation
              const fileNameArray = recordingFileName.split("/");
              let fileName = fileNameArray[fileNameArray.length - 1];
              fileName = fileName.split(".json")[0];
              fileName = fileName.replace(/\//g, "-");
              const exampleFileName = `${fileName}-${requestMethodFromRecording}-example-${pathIndex}${entryIndex}.json`;
              const ref = {
                $ref: relativeExamplesPath + exampleFileName,
              };
              const exampleFriendlyName = `${fileName}${requestMethodFromRecording}${pathIndex}${entryIndex}`;
              log.debug(`exampleFriendlyName: ${exampleFriendlyName}`);

              if (!("x-ms-examples" in infoFromOperation)) {
                infoFromOperation["x-ms-examples"] = {};
              }
              infoFromOperation["x-ms-examples"][exampleFriendlyName] = ref;
              const exampleL: {
                parameters: MutableStringMap<unknown>;
                responses: MutableStringMap<{
                  body?: unknown;
                }>;
              } = {
                parameters: {},
                responses: {},
              };
              const paramsToProcess = [
                ...mapEntries(pathParamsValues),
                ...mapEntries(queryParams),
                ...mapEntries(headerParams),
              ];
              for (const paramEntry of paramsToProcess) {
                const param = paramEntry[0];
                const v = paramEntry[1];
                exampleL.parameters[param] = v;
              }

              const params = infoFromOperation.parameters;

              for (const param of keys(infoFromOperation.parameters)) {
                if (params[param].in === "body") {
                  const bodyParamName = params[param].name;
                  const bodyParamValue = recordingEntry.RequestBody;
                  const bodyParamExample: MutableStringMap<unknown> = {};
                  bodyParamExample[bodyParamName] = bodyParamValue;

                  exampleL.parameters[bodyParamName] =
                    bodyParamValue !== "" ? JSON.parse(bodyParamValue) : "";
                }
              }

              const parseResponseBody = (body: any) => {
                try {
                  return JSON.parse(body);
                } catch (err) {
                  return body;
                }
              };

              for (const _v of keys(infoFromOperation.responses)) {
                const statusCodeFromRecording = recordingEntry.StatusCode;
                let responseBody = recordingEntry.ResponseBody;
                if (typeof responseBody === "string" && responseBody !== "") {
                  responseBody = parseResponseBody(responseBody);
                }
                exampleL.responses[statusCodeFromRecording] = {
                  body: responseBody,
                };
              }
              log.info(
                `Writing x-ms-examples at ${pathlib.resolve(outputExamples, exampleFileName)}`
              );
              const examplePath = pathlib.join(outputExamples, exampleFileName);
              const dir = pathlib.dirname(examplePath);
              mkdirRecursiveSync(dir);
              fs.writeFileSync(examplePath, JSON.stringify(exampleL, null, 2));
            }
          }
        }
      }
    }
  }

  /**
   * Extracts x-ms-examples from the recordings
   */
  public async extract(): Promise<StringMap<unknown>> {
    if (this.options.output === undefined) {
      throw new Error("this.options.output === undefined");
    }
    this.mkdirSync(this.options.output);
    this.mkdirSync(this.options.output + "/examples");
    this.mkdirSync(this.options.output + "/swagger");

    const outputExamples = pathlib.join(this.options.output, "examples");
    const relativeExamplesPath = "../examples/";
    const specName = this.specPath.split("/");
    const outputSwagger = pathlib.join(
      this.options.output,
      "swagger",
      specName[specName.length - 1].split(".")[0] + ".json"
    );

    const accErrors: MutableStringMap<unknown> = {};
    const filesArray: string[] = [];
    this.getFileList(this.recordings, filesArray);

    const recordingFiles = filesArray;

    try {
      const api = await swaggerParser.parse(this.specPath);
      for (const recordingFileName of recordingFiles) {
        log.debug(`Processing recording file: ${recordingFileName}`);

        try {
          this.extractOne(relativeExamplesPath, outputExamples, api, recordingFileName);
          log.info(`Writing updated swagger with x-ms-examples at ${outputSwagger}`);
          fs.writeFileSync(outputSwagger, JSON.stringify(api, null, 2));
        } catch (err) {
          accErrors[recordingFileName] = err.toString();
          log.warn(`Error processing recording file: "${recordingFileName}"`);
          log.warn(`Error: "${err.toString()} "`);
        }
      }

      if (JSON.stringify(accErrors) !== "{}") {
        log.error(`Errors loading/parsing recording files.`);
        log.error(`${JSON.stringify(accErrors)}`);
      }
    } catch (err) {
      process.exitCode = 1;
      log.error(err);
    }
    return accErrors;
  }

  private mkdirSync(dir: string): void {
    try {
      fs.mkdirSync(dir);
    } catch (e) {
      if (e.code !== "EEXIST") {
        throw e;
      }
    }
  }

  private getFileList(dir: string, fileList: string[]): string[] {
    const files = fs.readdirSync(dir);
    fileList = fileList || [];
    files.forEach((file) => {
      if (fs.statSync(pathlib.join(dir, file)).isDirectory()) {
        fileList = this.getFileList(pathlib.join(dir, file), fileList);
      } else {
        fileList.push(pathlib.join(dir, file));
      }
    });
    return fileList;
  }
}
