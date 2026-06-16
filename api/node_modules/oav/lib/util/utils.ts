/* eslint-disable no-bitwise */
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

import { execSync } from "child_process";
import * as fs from "fs";
import * as http from "http";
import * as path from "path";
import * as util from "util";
import * as jsonPointer from "json-pointer";
import * as YAML from "js-yaml";
import * as lodash from "lodash";
import {
  cloneDeep,
  Data,
  mapEntries,
  MutableStringMap,
  StringMap,
  parseMarkdown,
  readFile,
} from "@azure-tools/openapi-tools-common";
import * as amd from "@azure/openapi-markdown";
import * as commonmark from "commonmark";
import { log } from "./logging";

/*
 * Executes an array of promises sequentially. Inspiration of this method is here:
 * https://pouchdb.com/2015/05/18/we-have-a-problem-with-promises.html. An awesome blog on promises!
 *
 * @param {Array} promiseFactories An array of promise factories(A function that return a promise)
 *
 * @return A chain of resolved or rejected promises
 */
export async function executePromisesSequentially<T>(
  promiseFactories: ReadonlyArray<() => Promise<T>>
): Promise<readonly T[]> {
  const result: T[] = [];
  for (const promiseFactory of promiseFactories) {
    result.push(await promiseFactory());
  }
  return result;
}

export interface Reference {
  readonly filePath?: string;
  readonly localReference?: LocalReference;
}

export interface LocalReference {
  readonly value: string;
  readonly accessorProperty: string;
}

/*
 * Parses a [inline|relative] [model|parameter] reference in the swagger spec.
 * This method does not handle parsing paths "/subscriptions/{subscriptionId}/etc.".
 *
 * @param {string} reference Reference to be parsed.
 *
 * @return {object} result
 *         {string} [result.filePath] Filepath present in the reference. Examples are:
 *             - '../newtwork.json#/definitions/Resource' => '../network.json'
 *             - '../examples/nic_create.json' => '../examples/nic_create.json'
 *         {object} [result.localReference] Provides information about the local reference in the
 *                                          json document.
 *         {string} [result.localReference.value] The json reference value. Examples are:
 *           - '../newtwork.json#/definitions/Resource' => '#/definitions/Resource'
 *           - '#/parameters/SubscriptionId' => '#/parameters/SubscriptionId'
 *         {string} [result.localReference.accessorProperty] The json path expression that can be
 *                                                           used by
 *         eval() to access the desired object. Examples are:
 *           - '../newtwork.json#/definitions/Resource' => 'definitions.Resource'
 *           - '#/parameters/SubscriptionId' => 'parameters,SubscriptionId'
 */
export function parseReferenceInSwagger(reference: string): Reference {
  if (!reference || (reference && reference.trim().length === 0)) {
    throw new Error("reference cannot be null or undefined and it must be a non-empty string.");
  }

  if (reference.includes("#")) {
    // local reference in the doc
    if (reference.startsWith("#/")) {
      return {
        localReference: {
          value: reference,
          accessorProperty: reference.slice(2).replace("/", "."),
        },
      };
    } else {
      // filePath+localReference
      const segments = reference.split("#");
      return {
        filePath: segments[0],
        localReference: {
          value: "#" + segments[1],
          accessorProperty: segments[1].slice(1).replace("/", "."),
        },
      };
    }
  } else {
    // we are assuming that the string is a relative filePath
    return { filePath: reference };
  }
}

/*
 * Same as path.join(), however, it converts backward slashes to forward slashes.
 * This is required because path.join() joins the paths and converts all the
 * forward slashes to backward slashes if executed on a windows system. This can
 * be problematic while joining a url. For example:
 * path.join(
 *  'https://github.com/Azure/openapi-validation-tools/blob/master/lib',
 *  '../examples/foo.json')
 * returns
 * 'https:\\github.com\\Azure\\openapi-validation-tools\\blob\\master\\examples\\foo.json'
 * instead of
 * 'https://github.com/Azure/openapi-validation-tools/blob/master/examples/foo.json'
 *
 * @param variable number of arguments and all the arguments must be of type string. Similar to
 * the API provided by path.join()
 * https://nodejs.org/dist/latest-v6.x/docs/api/path.html#path_path_join_paths
 * @return {string} resolved path
 */
export function joinPath(...args: string[]): string {
  let finalPath = path.join(...args);
  finalPath = finalPath.replace(/\\/gi, "/");
  finalPath = finalPath.replace(/^(http|https):\/(.*)/gi, "$1://$2");
  return finalPath;
}

// If the spec path is a url starting with https://github then let us auto convert it to an
// https://raw.githubusercontent url.
export function checkAndResolveGithubUrl(inputPath: string): string {
  if (inputPath.startsWith("https://github")) {
    return inputPath.replace(
      /^https:\/\/github\.com\/(.*)\/blob\/(.*)/gi,
      "https://raw.githubusercontent.com/$1/$2"
    );
  }
  return inputPath;
}

/**
 * Finds the git root directory for the given directory.
 */
export function findGitRootDirectory(dir: string): string | undefined {
  while (true) {
    const gitDir = path.join(dir, ".git");
    if (fs.existsSync(gitDir)) {
      return dir;
    }
    const newDIr = path.dirname(dir);
    if (newDIr === dir) {
      return undefined;
    }
    dir = newDIr;
  }
}

/*
 * Merges source object into the target object
 * @param {object} source The object that needs to be merged
 *
 * @param {object} target The object to be merged into
 *
 * @returns {object} target - Returns the merged target object.
 */
export function mergeObjects<T extends MutableStringMap<Data>>(source: T, target: T): T {
  const result: MutableStringMap<Data> = target;
  for (const [key, sourceProperty] of mapEntries(source)) {
    if (Array.isArray(sourceProperty)) {
      const targetProperty = target[key];
      if (!targetProperty) {
        result[key] = sourceProperty;
      } else if (!Array.isArray(targetProperty)) {
        throw new Error(
          `Cannot merge ${key} from source object into target object because the same property ` +
            `in target object is not (of the same type) an Array.`
        );
      } else {
        result[key] = mergeArrays(sourceProperty, targetProperty);
      }
    } else {
      result[key] = cloneDeep(sourceProperty);
    }
  }
  return result as T;
}

/*
 * Merges source array into the target array
 * @param {array} source The array that needs to be merged
 *
 * @param {array} target The array to be merged into
 *
 * @returns {array} target - Returns the merged target array.
 */
export function mergeArrays<T extends Data>(source: readonly T[], target: T[]): T[] {
  if (!Array.isArray(target) || !Array.isArray(source)) {
    return target;
  }
  source.forEach((item) => {
    target.push(cloneDeep(item));
  });
  return target;
}

/*
 * Gets the object from the given doc based on the provided json reference pointer.
 * It returns undefined if the location is not found in the doc.
 * @param {object} doc The source object.
 *
 * @param {string} ptr The json reference pointer
 *
 * @returns {unknown} result - Returns the value that the ptr points to, in the doc.
 */
export function getObject(doc: StringMap<unknown>, ptr: string): unknown {
  let result: unknown;
  try {
    result = jsonPointer.get(doc, ptr);
  } catch (err) {
    log.error(`cannot get object from jsonPointer ${ptr}`);
    log.error(err);
    throw err;
  }
  return result;
}

/*
 * Sets the given value at the location provided by the ptr in the given doc.
 * @param {object} doc The source object.
 *
 * @param {string} ptr The json reference pointer.
 *
 * @param {unknown} value The value that needs to be set at the
 * location provided by the ptr in the doc.
 * @param {overwrite} Optional parameter to decide if a pointer value should be overwritten.
 */
export function setObject(
  doc: StringMap<unknown>,
  ptr: string,
  value: unknown,
  overwrite = true
): any {
  let result;
  try {
    if (overwrite || !jsonPointer.has(doc, ptr)) {
      result = jsonPointer.set(doc, ptr, value);
    }
  } catch (err) {
    log.error(err);
  }
  return result;
}

/**
 * Gets provider namespace from the given path. In case of multiple, last one will be returned.
 * @param {string} pathStr The path of the operation.
 *                 Example "/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/
 *                  providers/{resourceProviderNamespace}/{parentResourcePath}/{resourceType}/
 *                  {resourceName}/providers/Microsoft.Authorization/roleAssignments"
 *                 will return "Microsoft.Authorization".
 *
 * @returns {string} result - provider namespace from the given path.
 */
export function getProvider(pathStr?: string | null): string | undefined {
  if (
    pathStr === null ||
    pathStr === undefined ||
    typeof pathStr.valueOf() !== "string" ||
    !pathStr.trim().length
  ) {
    throw new Error(
      "pathStr is a required parameter of type string and it cannot be an empty string."
    );
  }

  let result;

  // Loop over the paths to find the last matched provider namespace
  // eslint-disable-next-line no-constant-condition
  while (true) {
    const pathMatch = providerRegEx.exec(pathStr);
    if (pathMatch === null) {
      break;
    }
    result = pathMatch[1];
  }

  return result;
}

export interface PathProvider {
  provider: string;
  type: "resource-manager" | "data-plane";
}

export function getProviderFromSpecPath(specPath: string): PathProvider | undefined {
  const managementPlaneProviderInSpecPathRegEx: RegExp = /\/resource-manager\/(.*?)\//gi;
  const dataPlaneProviderInSpecPathRegEx: RegExp = /\/data-plane\/(.*?)\//gi;
  const manageManagementMatch = managementPlaneProviderInSpecPathRegEx.exec(specPath);
  const dataPlaneMatch = dataPlaneProviderInSpecPathRegEx.exec(specPath);
  return manageManagementMatch === null
    ? dataPlaneMatch === null
      ? undefined
      : { provider: dataPlaneMatch[1], type: "data-plane" }
    : { provider: manageManagementMatch[1], type: "resource-manager" };
}

export const getValueByJsonPointer = (obj: any, pointer: string | string[]) => {
  const refTokens = Array.isArray(pointer) ? pointer : parse(pointer);

  for (let i = 0; i < refTokens.length; ++i) {
    const tok = refTokens[i];
    if (!(typeof obj === "object" && tok in obj)) {
      throw new Error("Invalid reference token: " + tok);
    }
    obj = obj[tok];
  }
  return obj;
};

const jsonPointerUnescape = (str: string) => {
  return str.replace(/~1/g, "/").replace(/~0/g, "~");
};

const parse = (pointer: string) => {
  if (pointer === "") {
    return [];
  }
  if (pointer.charAt(0) !== "/") {
    throw new Error("Invalid JSON pointer: " + pointer);
  }
  return pointer.substring(1).split(/\//).map(jsonPointerUnescape);
};

export function getProviderFromFilePath(pathStr: string): string | undefined {
  const resourceProviderPattern: RegExp = /[A-Z][a-z0-9]+(\.([A-Z]{1,5}[a-z0-9]+)+[A-Z]{0,5})+/g;
  const words = pathStr.split(/\\|\//gi);
  for (const it of words) {
    if (resourceProviderPattern.test(it)) {
      return it;
    }
  }
  return undefined;
}

const safeLoad = (content: string) => {
  try {
    return YAML.load(content) as any;
  } catch (err) {
    return undefined;
  }
};

/**
 * @return return undefined indicates not found, otherwise return non-empty string.
 */
export const getDefaultReadmeTag = (markDown: commonmark.Node): string | undefined => {
  const startNode = markDown;
  const codeBlockMap = amd.getCodeBlocksAndHeadings(startNode);
  const latestHeader = "Basic Information";
  const headerBlock = codeBlockMap[latestHeader];
  if (headerBlock && headerBlock.literal) {
    const latestDefinition = safeLoad(headerBlock.literal);
    if (latestDefinition && latestDefinition.tag) {
      return latestDefinition.tag;
    }
  }
  for (const idx of Object.keys(codeBlockMap)) {
    const block = codeBlockMap[idx];
    if (
      !block ||
      !block.info ||
      !block.literal ||
      !/^(yaml|json)$/.test(block.info.trim().toLowerCase())
    ) {
      continue;
    }
    const latestDefinition = safeLoad(block.literal);
    if (latestDefinition && latestDefinition.tag) {
      return latestDefinition.tag;
    }
  }
  return undefined;
};

export async function getInputFiles(readMe: string, tag?: string): Promise<string[]> {
  const result: string[] = [];
  const readMeStr = await readFile(checkAndResolveGithubUrl(readMe));
  const cmd = parseMarkdown(readMeStr);
  tag = tag ?? getDefaultReadmeTag(cmd.markDown);
  if (tag) {
    amd.getInputFilesForTag(cmd.markDown, tag)?.forEach((file) => result.push(file));
  }
  return result;
}

export async function getDefaultTag(readMe: string): Promise<string | undefined> {
  const readMeStr = await readFile(checkAndResolveGithubUrl(readMe));
  const cmd = parseMarkdown(readMeStr);
  return getDefaultReadmeTag(cmd.markDown);
}

export async function getApiScenarioFiles(
  readMe: string,
  tag: string,
  flag?: string
): Promise<string[]> {
  const readMeStr = await readFile(checkAndResolveGithubUrl(readMe));
  const cmd = parseMarkdown(readMeStr);
  const codeBlockMap = amd.getCodeBlocksAndHeadings(cmd.markDown);
  const pattern = flag ? `yaml $(tag) == '${tag}' && $(${flag})` : `yaml $(tag) == '${tag}'`;
  for (const idx of Object.keys(codeBlockMap)) {
    const block = codeBlockMap[idx];
    if (!block || !block.info || !block.literal || !(block.info.trim() === pattern)) {
      continue;
    }
    const latestDefinition = safeLoad(block.literal);
    if (latestDefinition && latestDefinition["test-resources"]) {
      return latestDefinition["test-resources"];
    }
  }
  return [];
}

export function getApiVersionFromFilePath(filePath: string): string {
  const apiVersionPattern: RegExp =
    /^.*\/(stable|preview)+\/([0-9]{4}-[0-9]{2}-[0-9]{2}(-preview)?)\/.*\.(json|yaml)$/i;
  const apiVersionMatch = apiVersionPattern.exec(filePath);
  return apiVersionMatch === null ? "" : apiVersionMatch[2];
}

const providerRegEx = new RegExp("/providers/(:?[^{/]+)", "gi");
/**
 * Gets provider namespace from the given path. In case of multiple, last one will be returned.
 * @param {string} pathStr The path of the operation.
 *                 Example "/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/
 *                  providers/{resourceProviderNamespace}/{parentResourcePath}/{resourceType}/
 *                  {resourceName}/providers/Microsoft.Authorization/roleAssignments"
 *                 will return "Microsoft.Authorization".
 *
 * @returns {string} result - provider namespace from the given path.
 */
export function getProviderFromPathTemplate(pathStr?: string | null): string | undefined {
  if (
    pathStr === null ||
    pathStr === undefined ||
    typeof pathStr.valueOf() !== "string" ||
    !pathStr.trim().length
  ) {
    throw new Error(
      "pathStr is a required parameter of type string and it cannot be an empty string."
    );
  }

  let result;

  // Loop over the paths to find the last matched provider namespace
  // eslint-disable-next-line no-constant-condition
  while (true) {
    const pathMatch = providerRegEx.exec(pathStr);
    if (pathMatch === null) {
      break;
    }
    result = pathMatch[1];
  }

  return result;
}

/**
 * Gets provider resource type from the given path.
 * @param {string} pathStr The path of the operation.
 *                 Example "/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/
 *                  providers/{resourceProviderNamespace}/{parentResourcePath}/{resourceType}/
 *                  {resourceName}/providers/Microsoft.Authorization/roleAssignments"
 *                 will return "roleAssignments".
 *
 * @returns {string} result - provider resource type from the given path.
 */
export function getResourceType(pathStr: string, provider?: string): string {
  if (provider !== undefined && provider !== null) {
    const index = pathStr.indexOf(provider);
    if (index > 0) {
      pathStr = pathStr.substring(index + provider.length + 1);
    }
  }

  let resourceType = pathStr;
  const slashIndex = pathStr.indexOf("/");
  if (slashIndex > 0) {
    resourceType = pathStr.substring(0, slashIndex);
  }

  return resourceType;
}

/**
 * Gets last child resouce url to match.
 * @param {string} requestUrl The request url.
 *                 Example "/subscriptions/randomSub/resourceGroups/randomRG/providers/providers/Microsoft.Storage/
 *                 storageAccounts/storageoy6qv/blobServices/default/containers/
 *                 privatecontainer/providers/Microsoft.Authorization/roleAssignments/3fa73e4b-d60d-43b2-a248-fb776fd0bf60"
 *                 will return "roleAssignments".
 *
 * @returns {string} last child resource url.
 */
export function getLastResourceUrlToMatch(requestUrl: string): string {
  let index = requestUrl.lastIndexOf("/providers");
  if (index > 0) {
    const originUrlWithoutLastChildResource = requestUrl.substring(0, index);
    index = originUrlWithoutLastChildResource.lastIndexOf("/");
    if (index > 0) {
      requestUrl = requestUrl.substring(index);
    }
  }
  return requestUrl;
}

/**
/*
 * Clones a github repository in the given directory.
 * @param {string} directory to where to clone the repository.
 *
 * @param {string} url of the repository to be cloned.
 *                 Example "https://github.com/Azure/azure-rest-api-specs.git" or
 *                         "git@github.com:Azure/azure-rest-api-specs.git".
 *
 * @param {string} [branch] to be cloned instead of the default branch.
 */
export function gitClone(directory: string, url: string, branch: string | undefined): void {
  if (
    url === null ||
    url === undefined ||
    typeof url.valueOf() !== "string" ||
    !url.trim().length
  ) {
    throw new Error("url is a required parameter of type string and it cannot be an empty string.");
  }

  if (
    directory === null ||
    directory === undefined ||
    typeof directory.valueOf() !== "string" ||
    !directory.trim().length
  ) {
    throw new Error(
      "directory is a required parameter of type string and it cannot be an empty string."
    );
  }

  // If the directory exists then we assume that the repo to be cloned is already present.
  if (fs.existsSync(directory)) {
    if (fs.lstatSync(directory).isDirectory()) {
      try {
        removeDirSync(directory);
      } catch (err) {
        const text = util.inspect(err, { depth: null });
        throw new Error(`An error occurred while deleting directory ${directory}: ${text}.`);
      }
    } else {
      try {
        fs.unlinkSync(directory);
      } catch (err) {
        const text = util.inspect(err, { depth: null });
        throw new Error(`An error occurred while deleting file ${directory}: ${text}.`);
      }
    }
  }

  try {
    fs.mkdirSync(directory);
  } catch (err) {
    const text = util.inspect(err, { depth: null });
    throw new Error(`An error occurred while creating directory ${directory}: ${text}.`);
  }

  try {
    const isBranchDefined =
      branch !== null && branch !== undefined && typeof branch.valueOf() === "string";
    const cmd = isBranchDefined
      ? `git clone --depth=1 --branch ${branch} ${url} ${directory}`
      : `git clone --depth=1 ${url} ${directory}`;
    execSync(cmd, { encoding: "utf8" });
  } catch (err) {
    throw new Error(
      `An error occurred while cloning git repository: ${util.inspect(err, {
        depth: null,
      })}.`
    );
  }
}

/*
 * Removes given directory recursively.
 * @param {string} dir directory to be deleted.
 */
export function removeDirSync(dir: string): void {
  if (fs.existsSync(dir)) {
    fs.readdirSync(dir).forEach((file) => {
      const current = dir + "/" + file;
      if (fs.statSync(current).isDirectory()) {
        removeDirSync(current);
      } else {
        fs.unlinkSync(current);
      }
    });
    fs.rmdirSync(dir);
  }
}

/*
 * Finds the first content-type that contains "/json". Only supported Content-Types are
 * "text/json" & "application/json" so we perform first best match that contains '/json'
 *
 * @param {array} consumesOrProduces Array of content-types.
 * @returns {string} firstMatchedJson content-type that contains "/json".
 */
export function getJsonContentType(consumesOrProduces: string[]): string | undefined {
  return consumesOrProduces
    ? consumesOrProduces.find((contentType) => contentType.match(/.*\/json.*/gi) !== null)
    : undefined;
}

/**
 * Determines whether the given string is url encoded
 * @param {string} str - The input string to be verified.
 * @returns {boolean} result - true if str is url encoded; false otherwise.
 */
export function isUrlEncoded(str: string): boolean {
  str = str || "";
  try {
    return str !== decodeURIComponent(str);
  } catch (e) {
    return false;
  }
}

export function kvPairsToObject(entries: any) {
  const result: any = {};
  for (const [key, value] of entries) {
    // each 'entry' is a [key, value] tupple
    result[key] = value;
  }
  return result;
}

/**
 * Sanitizes the file name by replacing special characters with
 * empty string and by replacing space(s) with _.
 * @param {string} str - The string to be sanitized.
 * @returns {string} result - The sanitized string.
 */
export const sanitizeFileName = (str: string): string =>
  // eslint-disable-next-line no-useless-escape
  str ? str.replace(/[{}[\]'";(\)#@~`!%&\^\$\+=,\/\\?<>\|\*:]/gi, "").replace(/(\s+)/gi, "_") : str;

/**
 * Contains the reverse mapping of http.STATUS_CODES
 */
export const statusCodeStringToStatusCode = lodash.invert(
  lodash.mapValues(http.STATUS_CODES, (value: string) => value.replace(/ |-/g, "").toLowerCase())
);

export type Writable<T> = { -readonly [P in keyof T]: T[P] };

export const waitUntilLowLoad = async () => {
  let lastTime = Date.now();
  let waterMark = 0;
  const startTime = lastTime;
  // eslint-disable-next-line no-constant-condition
  while (true) {
    await new Promise((resolve) => setTimeout(resolve, 0));
    const now = Date.now();
    // If event loop lag is less than 2ms then assume we are under low load
    if (now - startTime > 60000) {
      return;
    }
    if (now - lastTime <= 5) {
      ++waterMark;
      if (waterMark > 1) {
        return;
      }
    } else {
      waterMark = 0;
    }
    lastTime = now;
  }
};

export const shuffleArray = (a: any[]) => {
  for (let i = a.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [a[i], a[j]] = [a[j], a[i]];
  }
  return a;
};

export const usePseudoRandom = {
  seed: Math.floor(Math.random() * 10000000000),
};

/**
 * Generates a psudorandom number with seed
 */
function* mulberry32(seed: number) {
  let t = (seed += 0x6d2b79f5);
  while (true) {
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    yield (t ^ ((t >>> 14) >>> 0)) / 4294967296;
  }
}

let generator: any = undefined;

export const resetPseudoRandomSeed = (seed?: number) => {
  usePseudoRandom.seed = seed ?? Math.floor(Math.random() * 10000000000);
  generator = undefined;
};

export const getRandomString = (length?: number) => {
  if (generator === undefined) {
    generator = mulberry32(usePseudoRandom.seed);
  }
  return generator
    .next()
    .value.toString(36)
    .slice(0 - (length ?? 6));
};

export const findPathsToKey = (options: {
  key: string;
  obj: any;
  pathToKey?: string;
}): string[] => {
  const results = [];
  (function findKey({ key, obj, pathToKey }) {
    const oldPath = `${pathToKey ? pathToKey : ""}`;
    if (obj && obj.hasOwnProperty(key)) {
      results.push(`${oldPath}.${key}`);
      return;
    }
    if (obj !== null && typeof obj === "object" && !Array.isArray(obj)) {
      for (const k in obj) {
        if (obj.hasOwnProperty(k)) {
          if (Array.isArray(obj[k])) {
            for (let j = 0; j < obj[k].length; j++) {
              findKey({
                obj: obj[k][j],
                key,
                pathToKey: `${oldPath}${k}['${j}']`,
              });
            }
          }
          if (obj[k] !== null && typeof obj[k] === "object") {
            findKey({
              obj: obj[k],
              key,
              pathToKey: /[\*|\{|\[|\}|\}|\,|\.]/.test(k)
                ? `${oldPath}['${k}']`
                : `${oldPath}.${k}`,
            });
          }
        }
      }
    }
  })(options);

  return results;
};

export const findPathToValue = (arr: string[], obj: any, value: string) => {
  return arr.reduce((pre: string[], cur: string) => {
    lodash.get(obj, cur.substr(1)) === value && pre.push(cur);
    return pre;
  }, []);
};
