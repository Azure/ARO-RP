import { FilePosition, getInfo } from "@azure-tools/openapi-tools-common";
import { default as jsonPointer } from "json-pointer";
import { JSONPath } from "jsonpath-plus";

export const jsonPathToArray = (jsonPath: string): string[] => {
  return (JSONPath as any).toPathArray(jsonPath);
};

export const jsonPathToPointer = (jsonPath: string): string => {
  return jsonPointer.compile(jsonPathToArray(jsonPath).slice(1));
};

export const getFilePositionFromJsonPath = (
  obj: any,
  jsonPath: string
): FilePosition | undefined => {
  if (!jsonPath) {
    return undefined;
  }
  const pathArr = jsonPathToArray(jsonPath.substr(1));
  /*
   * when jsonPath='/providers/Microsoft.Provider/resource',
   * the split pathArr will be ['/providers/Microsoft','Provider/resource'].
   * Only in this case, these two elements in the array need to be composed together by '.'.
   * So restrict the condition to the path element ends with /providers/Microsoft.
   */
  const newPathArr = pathArr.slice(0);
  const index = newPathArr.findIndex((str) => str.includes("/providers/Microsoft"));
  if (
    index !== -1 &&
    newPathArr[index + 1] !== undefined &&
    newPathArr[index].slice(-20) === "/providers/Microsoft"
  ) {
    newPathArr[index] += "." + newPathArr[index + 1];
    newPathArr.splice(index + 1, 1);
  }
  try {
    const target = jsonPointer.get(obj, jsonPointer.compile(newPathArr));
    const info = getInfo(target);
    if (info !== undefined) {
      return info.position;
    }
  } catch (e) {
    // Pass
  }

  const lastProperty = newPathArr.pop();
  const target = jsonPointer.get(obj, jsonPointer.compile(newPathArr));
  const info = getInfo(target);
  if (info !== undefined && lastProperty !== undefined) {
    return info.primitiveProperties[lastProperty];
  }

  return undefined;
};
