import { RuleValidatorFunc } from "./exampleRule";

/* tslint:disable:max-classes-per-file */
interface BaseCache {
  get(modelName: string): CacheItem | undefined;
  set(modelName: string, example: CacheItem): void;
  has(modelName: string): boolean;
}

const isBaseResource = (cacheKey: string) => {
  const pieces = cacheKey.split("/");
  if (pieces.length < 2) {
    return false;
  }
  return ["resource", "proxyresource", "trackedresource", "azureentityresource"].some(
    (r) => r === pieces[pieces.length - 1].toLowerCase()
  );
};
export class MockerCache implements BaseCache {
  private caches = new Map<string, CacheItem>();

  public get(modelName: string) {
    if (this.has(modelName)) {
      return this.caches.get(modelName);
    }
    return undefined;
  }
  public set(modelName: string, example: CacheItem) {
    if (!this.has(modelName)) {
      this.caches.set(modelName, example);
    }
  }
  public has(modelName: string) {
    return this.caches.has(modelName);
  }
  public checkAndCache(schema: any, example: CacheItem) {
    if (!schema || !example) {
      return;
    }
    const cacheKey =
      schema.$ref && schema.$ref.includes("#") ? schema.$ref.split("#")[1] : undefined;
    if (cacheKey && !isBaseResource(cacheKey) && !this.has(cacheKey)) {
      this.set(cacheKey, example);
    }
  }
}
export class PayloadCache implements BaseCache {
  private requestCaches = new Map<string, CacheItem>();
  private responseCaches = new Map<string, CacheItem>();
  private mergedCaches = new Map<string, CacheItem>();

  private hasByDirection(modelName: string, isRequest: boolean) {
    const cache = isRequest ? this.requestCaches : this.responseCaches;
    return cache.has(modelName);
  }
  private setByDirection(modelName: string, example: CacheItem, isRequest: boolean) {
    const cache = isRequest ? this.requestCaches : this.responseCaches;
    if (!cache.has(modelName)) {
      cache.set(modelName, example);
    }
  }
  private getByDirection(modelName: string, isRequest: boolean) {
    const cache = isRequest ? this.requestCaches : this.responseCaches;
    if (cache.has(modelName)) {
      return cache.get(modelName);
    }
    return undefined;
  }
  public get(modelName: string) {
    if (this.mergedCaches.has(modelName)) {
      return this.mergedCaches.get(modelName);
    }
    return undefined;
  }

  public set(key: string, value: CacheItem) {
    if (!this.mergedCaches.has(key)) {
      this.mergedCaches.set(key, value);
    }
  }

  public has(modelName: string) {
    return this.mergedCaches.has(modelName);
  }

  public checkAndCache(schema: any, example: CacheItem | undefined, isRequest: boolean) {
    if (!schema || !example) {
      return;
    }
    const cacheKey =
      schema.$ref && schema.$ref.includes("#") ? schema.$ref.split("#")[1] : undefined;
    if (cacheKey && !isBaseResource(cacheKey) && !this.hasByDirection(cacheKey, isRequest)) {
      this.setByDirection(cacheKey, example, isRequest);
    }
  }
  /**
   *  picking value priority : non-mocked value > mocked value , target value > source value
   * @param target The target item that to be merged into
   * @param source The source item that needs to merge
   */
  public mergeItem(target: CacheItem, source: CacheItem): CacheItem {
    const result = target;
    if (!source || !target) {
      return target ? target : source;
    }

    if (Array.isArray(result.child) && Array.isArray(source.child)) {
      const resultArr = result.child as CacheItem[];
      const sourceArr = source.child as CacheItem[];
      if (resultArr.length === 0 || sourceArr.length === 0) {
        return resultArr.length === 0 ? source : result;
      }
      // only when source is not mocked and target is mocked , choose source cache.
      if (resultArr[0].isMocked && !sourceArr[0].isMocked) {
        return source;
      }
      for (let i = 0; i < resultArr.length; i++) {
        if (i < sourceArr.length) {
          resultArr[i] = this.mergeItem(resultArr[i], sourceArr[i]);
        }
      }
    } else if (source.child && result.child) {
      const resultObj = result.child as CacheItemObject;
      const sourceObj = source.child as CacheItemObject;
      for (const key of Object.keys(sourceObj)) {
        if (!resultObj[key]) {
          resultObj[key] = sourceObj[key];
        } else {
          resultObj[key] = this.mergeItem(resultObj[key], sourceObj[key]);
        }
      }
    } else {
      return result.isMocked && !source.isMocked ? source : result;
    }
    return result;
  }

  /**
   * 1 for each request cache , if exists in response cache, merge it with response cache and put into merged cache .
   * 2 for each response cache, if not exists in merged cache, then put into merged cache.
   */
  public mergeCache() {
    for (const [key, requestCache] of this.requestCaches.entries()) {
      if (this.hasByDirection(key, false) && !requestCache.isLeaf) {
        const responseCache = this.getByDirection(key, false);
        if (responseCache) {
          if (responseCache.isLeaf) {
            console.error(`the response cache and request cache is inconsistent! key:${key}`);
          } else {
            const mergedCache = this.mergeItem(requestCache, responseCache);
            this.set(key, mergedCache);
            continue;
          }
        }
      }
      this.set(key, requestCache);
    }
    for (const [key, responseCache] of this.responseCaches.entries()) {
      if (!this.hasByDirection(key, true)) {
        this.set(key, responseCache);
      }
    }
    this.requestCaches.clear();
    this.responseCaches.clear();
  }
}

type CacheItemValue = string | number | object | boolean;
interface CacheItemObject {
  [index: string]: CacheItem;
}
type CacheItemChild = CacheItemObject | CacheItem[];
interface CacheItemOptions {
  isReadonly?: boolean;
  isXmsSecret?: boolean;
  isRequired?: boolean;
  isWriteOnly?: boolean;
}
export interface CacheItem {
  value?: CacheItemValue;
  child?: CacheItemChild;
  options?: CacheItemOptions;
  isLeaf: boolean;
  required?: string[];
  isMocked?: boolean;
}

export const buildItemOption = (schema: any) => {
  if (schema) {
    const isReadonly = !!schema.readOnly;
    const isXmsSecret = !!schema["x-ms-secret"];
    const isRequired = !!schema.required;
    const isWriteOnly = schema["x-ms-mutability"]
      ? schema["x-ms-mutability"].indexOf("read") === -1
      : false;
    if (!isReadonly && !isXmsSecret && !isRequired && !isWriteOnly) {
      return undefined;
    }
    let option: CacheItemOptions = {};
    if (isReadonly) {
      option = { isReadonly: true };
    }
    if (isXmsSecret) {
      option = { ...option, isXmsSecret: true };
    }
    if (isWriteOnly) {
      option = { ...option, isWriteOnly: true };
    }
    if (schema.required === true) {
      option = { ...option, isRequired: true };
    }
    return option;
  }
  return undefined;
};

export const createLeafItem = (
  itemValue: CacheItemValue,
  option: CacheItemOptions | undefined = undefined
): CacheItem => {
  const item = {
    isLeaf: true,
    value: itemValue,
  } as CacheItem;
  if (option) {
    item.options = option;
  }
  return item;
};

export const createTrunkItem = (
  itemValue: CacheItemChild,
  option: CacheItemOptions | undefined
): CacheItem => {
  const item = {
    isLeaf: false,
    child: itemValue,
  } as CacheItem;
  if (option) {
    item.options = option;
  }
  return item;
};

export const reBuildExample = (
  cache: CacheItem | undefined,
  isRequest: boolean,
  schema: any,
  validator: RuleValidatorFunc | undefined
): any => {
  if (!cache) {
    return undefined;
  }
  if (validator && !validator({ schemaCache: cache, isRequest })) {
    return undefined;
  }
  if (cache.isLeaf) {
    return cache.value;
  }
  if (Array.isArray(cache.child)) {
    const result = [];
    for (const item of cache.child) {
      if (validator && !validator({ schemaCache: item, isRequest, schema })) {
        continue;
      }
      result.push(reBuildExample(item, isRequest, schema, validator));
    }
    return result;
  } else if (cache.child) {
    const result: any = {};
    for (const key of Object.keys(cache.child)) {
      if (!validator || validator({ schemaCache: cache, propertyName: key, isRequest, schema })) {
        const value = reBuildExample(cache.child[key], isRequest, schema, validator);
        if (value !== undefined) {
          result[key] = value;
        }
      }
    }
    return result;
  }
  return undefined;
};
