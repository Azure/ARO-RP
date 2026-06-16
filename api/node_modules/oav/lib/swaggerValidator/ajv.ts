import { Ajv, CompilationContext } from "ajv";
import { JsonLoader } from "../swagger/jsonLoader";
import { Schema } from "../swagger/swaggerTypes";
import { xmsAzureResource, xmsMutability, xmsSecret } from "../util/constants";
import { ajvEnableDiscriminatorMap } from "./ajvDiscriminatorMap";

export const ajvEnableReadOnlyAndXmsMutability = (ajv: Ajv) => {
  ajv.removeKeyword("readOnly");
  ajv.addKeyword("readOnly", {
    metaSchema: { type: "boolean" },
    inline: (
      it: CompilationContext,
      _keyword: string,
      isReadOnly: boolean,
      parentSchema: Schema
    ) => {
      if (parentSchema?.[xmsMutability] !== undefined) {
        return "1";
      }
      const data = `data${it.dataLevel || ""}`;
      return isReadOnly ? `this.isResponse || ${data} === null || ${data} === undefined` : "1";
    },
  });

  ajv.addKeyword(xmsMutability, {
    metaSchema: { type: "array", items: { enum: ["create", "update", "read"] } } as Schema,
    inline: (
      it: CompilationContext,
      _keyword: string,
      mutability: Exclude<Schema[typeof xmsMutability], undefined>
    ) => {
      const validInRequest = mutability.includes("create") || mutability.includes("update");
      const validInResponse = mutability.includes("read");
      if (validInRequest && validInResponse) {
        return "1";
      }
      if (!validInRequest && !validInResponse) {
        throw new Error(`Invalid ${xmsMutability} value: ${JSON.stringify(mutability)}`);
      }
      const data = `data${it.dataLevel || ""}`;
      return `${
        validInRequest ? "!" : ""
      }this.isResponse || ${data} === null || ${data} === undefined`;
    },
  });
};

export const ajvEnableXmsSecret = (ajv: Ajv) => {
  ajv.addKeyword(xmsSecret, {
    metaSchema: { type: "boolean" } as Schema,
    inline: (it: CompilationContext, _keyword: string, isSecret: boolean) => {
      const data = `data${it.dataLevel || ""}`;
      return isSecret ? `!this.isResponse || ${data} === null || ${data} === undefined` : "1";
    },
  });
};

export const ajvEnableXmsAzureResource = (ajv: Ajv) => {
  ajv.addKeyword(xmsAzureResource, {
    metaSchema: { type: "boolean" } as Schema,
    inline: (it: CompilationContext, _keyword: string, isResource: boolean) => {
      const data = `data${it.dataLevel || ""}`;
      return isResource
        ? `!(this.isResponse && (this.httpMethod === 'get' || this.httpMethod === 'put')) || (${data}.id !== null && ${data}.id !== undefined)`
        : "1";
    },
  });
};

export const ajvEnableInt32AndInt64Format = (ajv: Ajv) => {
  ajv.addFormat("int32", {
    type: "number",
    validate: (x) => x % 1 === 0 && x >= -2_147_483_648 && x <= 2_147_483_647,
  });

  // TODO int64 range exceed Number.MAX_SAFE_INTEGER so we will lost precision when JSON.parse
  const int64Max = BigInt(2) ** BigInt(63) - BigInt(1);
  const int64Min = BigInt(2) ** BigInt(63) * BigInt(-1);
  ajv.addFormat("int64", {
    type: "number",
    validate: (x) => x % 1 === 0 && x >= int64Min && x <= int64Max,
  });
};

export const ajvEnableUnixTimeFormat = (ajv: Ajv) => {
  ajv.addFormat("unixtime", {
    type: "number",
    validate: (x) => x % 1 === 0,
  });
};

export const ajvAddFormatsDefaultValidation = (
  ajv: Ajv,
  type: "string" | "number",
  formats: string[]
) => {
  for (const format of formats) {
    ajv.addFormat(format, {
      type,
      validate: () => true,
    });
  }
};

export const ajvEnableDateTimeRfc1123Format = (ajv: Ajv) => {
  // https://tools.ietf.org/html/rfc822#section-5
  ajv.addFormat("date-time-rfc1123", {
    type: "string",
    validate:
      /^(?:(?:Mon|Tue|Wed|Thu|Fri|Sat|Sun), )?[0-3]\d (?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) \d\d(?:\d\d)? (?:[0-2]\d:[0-5]\d(?::[0-5]\d)|23:59:60) (?:[A-Z]{1,3})?(?:[+-]\d\d\d\d)?$/,
  });
};

export const ajvEnableDurationFormat = (ajv: Ajv) => {
  // https://en.wikipedia.org/wiki/ISO_8601#Durations
  ajv.addFormat("duration", {
    type: "string",
    validate:
      /^P([0-9]+(?:[,\.][0-9]+)?Y)?([0-9]+(?:[,\.][0-9]+)?M)?([0-9]+(?:[,\.][0-9]+)?D)?(?:T([0-9]+(?:[,\.][0-9]+)?H)?([0-9]+(?:[,\.][0-9]+)?M)?([0-9]+(?:[,\.][0-9]+)?S)?)?$/,
  });
};

export const ajvEnableByteFormat = (ajv: Ajv) => {
  // https://datatracker.ietf.org/doc/html/rfc4648#section-4
  ajv.addFormat("byte", {
    type: "string",
    validate: (x) => {
      const decodedValue = Buffer.from(x, "base64");
      const reencodedValue = Buffer.from(decodedValue).toString("base64");
      return reencodedValue === x;
    },
  });
};

// TODO: This could be more advanced, looking at the allowedResources field (see https://github.com/Azure/autorest/tree/main/docs/extensions#schema) and ensuring that only the allowed resources are referenced, but
// TODO: for now a generic ARM ID check is better than nothing.
export const ajvEnableArmIdFormat = (ajv: Ajv) => {
  ajv.addFormat("arm-id", {
    type: "string",
    // Note that this regex isn't perfect but it does an OK job. See https://regex101.com/r/96g3K5/1
    validate: new RegExp(
      "(^(/subscriptions/([^/]+)(/resourcegroups/([^/]+))?)?/providers/([^/]+)/([^/]+/[^/]+)(/([^/]+/[^/]+))*$|^/subscriptions/([^/]+)(/resourcegroups/([^/]+))?$)",
      "i"
    ),
  });
};

// for (const keyword of [
//   "name",
//   "in",
//   "example",
//   "parameters",
//   "externalDocs",
//   "x-nullable",
//   "x-ms-enum",
//   "x-ms-azure-resource",
//   "x-ms-parameter-location",
//   "x-ms-client-name",
//   "x-ms-external",
//   "x-ms-skip-url-encoding",
//   "x-ms-client-flatten",
//   "x-ms-api-version",
//   "x-ms-parameter-grouping",
//   "x-ms-discriminator-value",
//   "x-ms-client-request-id",
//   "x-apim-code-nillable",
//   "x-new-pattern",
//   "x-previous-pattern",
//   "x-comment",
//   "x-abstract",
//   "allowEmptyValue",
//   "collectionFormat",
// ]) {
//   ajv.addKeyword(keyword, {});
// }

export const ajvEnableAll = (ajv: Ajv, jsonLoader: JsonLoader) => {
  ajvEnableDiscriminatorMap(ajv, jsonLoader);
  ajvEnableXmsSecret(ajv);
  ajvEnableReadOnlyAndXmsMutability(ajv);
  ajvEnableUnixTimeFormat(ajv);
  ajvEnableInt32AndInt64Format(ajv);
  ajvEnableDurationFormat(ajv);
  ajvEnableDateTimeRfc1123Format(ajv);
  ajvEnableByteFormat(ajv);
  ajvAddFormatsDefaultValidation(ajv, "string", [
    "password",
    "file",
    "base64url",
    "",
    "binary",
    "non-iso-duration",
    "char",
  ]);
  ajvAddFormatsDefaultValidation(ajv, "number", ["double", "float", "decimal"]);
};

export const ajvEnableArmRule = (ajv: Ajv) => {
  ajvEnableXmsAzureResource(ajv);
};
