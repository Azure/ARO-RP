import { default as stableStringify } from "fast-json-stable-stringify";
import * as jsonPointer from "json-pointer";
import { cloneDeep } from "@azure-tools/openapi-tools-common";
import { SequenceMatcher } from "difflib";
import {
  JsonPatchOp,
  JsonPatchOpAdd,
  JsonPatchOpCopy,
  JsonPatchOpMove,
  JsonPatchOpRemove,
  JsonPatchOpReplace,
  JsonPatchOpTest,
} from "./apiScenarioTypes";

interface PatchContext {
  root: any;
  obj: any;
  propertyName: string;
  arrIdx: number;
}
export interface DiffPatchOptions {
  includeOldValue?: boolean;
  minimizeDiff?: boolean;
}
const rootName = "ROOT";

const getCtx = (obj: any, path: string): PatchContext => {
  if (path === "/") {
    path = "";
  }
  const pathSegments = jsonPointer.parse(path);
  pathSegments.unshift(rootName);
  const propertyName = pathSegments.pop()!;
  const target = jsonPointer.get(obj, jsonPointer.compile(pathSegments));
  const result: PatchContext = {
    root: obj,
    obj: target,
    propertyName,
    arrIdx: -1,
  };

  if (Array.isArray(obj)) {
    result.arrIdx = parseInt(propertyName);
  }

  return result;
};

const patchAdd = ({ obj, propertyName, arrIdx }: PatchContext, op: JsonPatchOpAdd) => {
  if (Array.isArray(obj)) {
    obj.splice(arrIdx, 0, op.value);
  } else {
    obj[propertyName] = op.value;
  }
};

const patchRemove = ({ obj, propertyName, arrIdx }: PatchContext) => {
  if (Array.isArray(obj)) {
    obj.splice(arrIdx, 1);
  } else {
    delete obj[propertyName];
  }
};

const patchReplace = ({ obj, propertyName }: PatchContext, op: JsonPatchOpReplace) => {
  obj[propertyName] = op.value;
};

const patchCopy = ({ root, obj, propertyName }: PatchContext, op: JsonPatchOpCopy) => {
  const val = cloneDeep(obj[propertyName]);
  jsonPointer.set(root, `/${rootName}${op.copy}`, val);
};

const patchMove = (ctx: PatchContext, op: JsonPatchOpMove) => {
  const { propertyName, obj, root } = ctx;
  const val = obj[propertyName];
  patchRemove(ctx);
  jsonPointer.set(root, `/${rootName}${op.move}`, val);
};

const patchTest = ({ obj, propertyName }: PatchContext, op: JsonPatchOpTest) => {
  const val = obj[propertyName];
  const factStr = stableStringify(val);
  const expectStr = stableStringify(op.value);
  if (factStr !== expectStr) {
    throw new Error(
      `JsonPatch Test failed for path: ${op.test}\nExpect: ${factStr}\nActual: ${factStr}`
    );
  }
};

const jsonPatchApplyOp = (obj: any, op: JsonPatchOp) => {
  if ("add" in op) {
    return patchAdd(getCtx(obj, op.add), op);
  }
  if ("remove" in op) {
    return patchRemove(getCtx(obj, op.remove));
  }
  if ("replace" in op) {
    return patchReplace(getCtx(obj, op.replace), op);
  }
  if ("copy" in op) {
    return patchCopy(getCtx(obj, op.from), op);
  }
  if ("move" in op) {
    return patchMove(getCtx(obj, op.from), op);
  }
  if ("test" in op) {
    return patchTest(getCtx(obj, op.test), op);
  }

  throw new Error(`Unknown jsonPatchOp: ${JSON.stringify(op)}`);
};

export const jsonPatchApply = (obj: any, ops: JsonPatchOp[]): any => {
  const rootObj = {
    [rootName]: obj,
  };
  for (const op of ops) {
    jsonPatchApplyOp(rootObj, op);
  }
  return rootObj[rootName];
};

export const getJsonPatchDiff = (
  from: any,
  to: any,
  opts: DiffPatchOptions = {}
): JsonPatchOp[] => {
  const patches = calcDiff(from, to, [], opts);

  if (opts.includeOldValue) {
    for (const patch of patches) {
      const p = patch as JsonPatchOpReplace & JsonPatchOpRemove;
      const oldPath = p.remove ?? p.replace;
      if (oldPath !== undefined) {
        p.oldValue = getObjValueFromPointer(from, oldPath);
      }
    }
  }

  return patches;
};

export const getObjValueFromPointer = (obj: any, pointer: string) => {
  return jsonPointer.get(obj, pointer === "/" ? "" : pointer);
};

const calcDiff = (from: any, to: any, path: string[], opts: DiffPatchOptions): JsonPatchOp[] => {
  if (from === to) {
    return [];
  }
  const replaceOp: JsonPatchOp = {
    replace: getJsonPointer(path),
    value: to,
  };

  const fromType = typeof from;
  const toType = typeof to;

  if (fromType !== toType || fromType !== "object" || from === null || to === null) {
    return [replaceOp];
  }
  const isFromArray = Array.isArray(from);
  const isToArray = Array.isArray(to);
  if (isFromArray !== isToArray) {
    return [replaceOp];
  }

  const diffOps: JsonPatchOp[] = isFromArray
    ? calcArrayDiff(from, to, path, opts)
    : calcObjDiff(from, to, path, opts);
  if (diffOps.length === 0 || !opts.minimizeDiff) {
    return diffOps;
  }

  const diffLen = JSON.stringify(diffOps).length;
  const replaceLen = JSON.stringify(replaceOp).length + 4;

  return diffLen < replaceLen ? diffOps : [replaceOp];
};

const calcObjDiff = (from: any, to: any, path: string[], opts: DiffPatchOptions) => {
  const result: JsonPatchOp[] = [];

  for (const key of Object.keys(from)) {
    if (from[key] === undefined) {
      continue;
    }
    if (to[key] === undefined) {
      result.push({
        remove: getJsonPointer(path, key),
      });
    } else {
      const diff = calcDiff(from[key], to[key], path.concat([key]), opts);
      if (diff.length > 0) {
        result.push(...diff);
      }
    }
  }
  for (const key of Object.keys(to)) {
    if (to[key] === undefined || from[key] !== undefined) {
      continue;
    }
    result.push({
      add: getJsonPointer(path, key),
      value: to[key],
    });
  }

  return result;
};

const calcArrayDiff = (
  from: any[],
  to: any[],
  path: string[],
  opts: DiffPatchOptions
): JsonPatchOp[] => {
  let matchSeq = calcArrayDiffWithIndex(from, to);
  let isKeyMatch = true;
  if (matchSeq === undefined) {
    const matcher = new SequenceMatcher(null, from, to);
    matchSeq = matcher.getOpcodes();
    isKeyMatch = false;
  }

  const addReplaceOps: JsonPatchOp[] = [];
  const removeOps: JsonPatchOp[] = [];

  for (const [op, i0, i1, j0, j1] of matchSeq) {
    switch (op) {
      case "equal":
        if (isKeyMatch) {
          for (let ix = i0, jx = j0; ix < i1 && jx < j1; ++ix, ++jx) {
            const diff = calcDiff(from[ix], to[jx], path.concat([jx.toString()]), opts);
            addReplaceOps.push(...diff);
          }
        }
        break;

      case "insert":
        for (let idx = j0; idx < j1; ++idx) {
          addReplaceOps.push({
            add: getJsonPointer(path, idx.toString()),
            value: to[idx],
          });
        }
        break;

      case "delete":
        for (let idx = i0; idx < i1; ++idx) {
          removeOps.push({
            remove: getJsonPointer(path.concat([idx.toString()])),
          });
        }
        break;

      case "replace": {
        let ix = i0;
        let jx = j0;
        for (; ix < i1 && jx < j1; ++ix, ++jx) {
          if (isKeyMatch) {
            addReplaceOps.push({
              replace: getJsonPointer(path, jx.toString()),
              value: to[jx],
            });
          } else {
            const diff = calcDiff(from[ix], to[jx], path.concat([jx.toString()]), opts);
            addReplaceOps.push(...diff);
          }
        }
        for (; ix < i1; ++ix) {
          removeOps.push({
            remove: getJsonPointer(path, ix.toString()),
          });
        }
        for (; jx < j1; ++jx) {
          addReplaceOps.push({
            add: getJsonPointer(path, jx.toString()),
            value: to[jx],
          });
        }
      }
    }
  }

  return removeOps.reverse().concat(addReplaceOps);
};

const calcArrayDiffWithIndex = (
  from: any[],
  to: any[]
): ReturnType<SequenceMatcher<any>["getOpcodes"]> | undefined => {
  const fromKeys = new Array(from.length);
  const toKeys = new Array(to.length);
  let hasKey = false;
  for (let idx = 0; idx < from.length; ++idx) {
    const key = getObjKey(from[idx]);
    if (key !== undefined) {
      hasKey = true;
    }
    fromKeys[idx] = key;
  }
  for (let idx = 0; idx < to.length; ++idx) {
    const key = getObjKey(to[idx]);
    if (key !== undefined) {
      hasKey = true;
    }
    toKeys[idx] = key;
  }
  if (!hasKey) {
    return undefined;
  }

  const matcher = new SequenceMatcher(null, fromKeys, toKeys);
  const matchSeq = matcher.getOpcodes();

  return matchSeq;
};

const getObjKey = (item: any) => {
  if (item === undefined || item === null || typeof item !== "object") {
    return undefined;
  }
  return item.id ?? item.name ?? probeObjectKey(item);
};

const probeObjectKey = (item: any) => {
  for (const key of Object.keys(item)) {
    if (typeof item[key] !== "string") {
      continue;
    }
    const lowerKey = key.toLowerCase();
    if (lowerKey.endsWith("id") || lowerKey.endsWith("name")) {
      return item[key];
    }
  }
  return undefined;
};

export const getJsonPointer = (input: string[], additional?: string) => {
  let result = jsonPointer.compile(input);
  if (additional !== undefined) {
    result = `${result}/${jsonPointer.escape(additional)}`;
  }
  if (result === "") {
    result = "/";
  }
  return result;
};
