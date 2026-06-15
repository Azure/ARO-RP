import { inject, injectable } from "inversify";
import { cloneDeep } from "lodash";
import * as jp from "json-pointer";
import { TYPES } from "../inversifyUtils";
import { Schema } from "../swagger/swaggerTypes";
import { SchemaValidator } from "../swaggerValidator/schemaValidator";
import { jsonPathToPointer } from "../util/jsonUtils";
import { jsonPatchApply } from "./diffUtils";

@injectable()
export class BodyTransformer {
  public constructor(@inject(TYPES.schemaValidator) private validator: SchemaValidator) {}

  public async resourceToRequest(resource: any, responseSchema: Schema): Promise<any> {
    const validateFn = await this.validator.compileAsync(responseSchema);
    // Readonly field cannot be set in response, so we could filter readonly fields
    const errors = validateFn(
      { isResponse: false, includeErrors: ["READONLY_PROPERTY_NOT_ALLOWED_IN_REQUEST"] },
      resource
    );

    const result = cloneDeep(resource);
    for (const err of errors) {
      for (const jsonPath of err.jsonPathsInPayload) {
        const jsonPointer = jsonPathToPointer(jsonPath);
        jsonPatchApply(result, [{ remove: jsonPointer }]);
      }
    }

    // console.log(body);
    // console.log(errors);
    // console.log(result);

    return result;
  }

  public async resourceToResponse(resource: any, requestSchema: Schema): Promise<any> {
    const validateFn = await this.validator.compileAsync(requestSchema);
    // Writeonly field cannot be set in request, so we could filter writeonly fields
    const errors = validateFn(
      {
        isResponse: false,
        includeErrors: ["WRITEONLY_PROPERTY_NOT_ALLOWED_IN_RESPONSE", "SECRET_PROPERTY"],
      },
      resource
    );

    const result = cloneDeep(resource);
    for (const err of errors) {
      for (const jsonPath of err.jsonPathsInPayload) {
        const jsonPointer = jsonPathToPointer(jsonPath);
        jsonPatchApply(result, [{ remove: jsonPointer }]);
      }
    }

    // console.log(body);
    // console.log(errors);
    // console.log(result);

    return result;
  }

  public deepMerge(dst: any, src: any): { result: any; inconsistentWarningPaths: string[] } {
    const inconsistentPaths: string[] = [];
    const result = this.innerDeepMerge(src, dst, [], inconsistentPaths);
    return {
      result,
      inconsistentWarningPaths: inconsistentPaths,
    };
  }

  private innerDeepMerge(dst: any, src: any, path: string[], inconsistentPaths: string[]): any {
    if (typeof src !== typeof dst) {
      inconsistentPaths.push(jp.compile(path));
      return src ?? dst;
    }

    if (Array.isArray(dst) || Array.isArray(src)) {
      if (!Array.isArray(src) || !Array.isArray(dst)) {
        inconsistentPaths.push(jp.compile(path));
        return src;
      }
      let length = dst.length;
      if (dst.length !== src.length) {
        inconsistentPaths.push(jp.compile(path));
        if (src.length < dst.length) {
          length = src.length;
        }
      }
      const result = new Array(length);
      for (let idx = 0; idx < length; idx++) {
        result[idx] = this.innerDeepMerge(
          dst[idx],
          src[idx],
          path.concat(idx.toString()),
          inconsistentPaths
        );
      }
      return result;
    }

    if (typeof dst === "object" && typeof src === "object") {
      const result: any = { ...dst };
      if (dst !== null && src !== null) {
        for (const key of Object.keys(dst)) {
          if (key in src) {
            result[key] = this.innerDeepMerge(
              dst[key],
              src[key],
              path.concat([key]),
              inconsistentPaths
            );
          }
        }
      }
      if (src !== null && dst !== null) {
        for (const key of Object.keys(src)) {
          if (!(key in dst)) {
            result[key] = src[key];
          }
        }
      }
      return result;
    }

    if (dst !== src) {
      inconsistentPaths.push(jp.compile(path));
    }
    return src;
  }
}
