// Copyright (c) 2021 Microsoft Corporation
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

import { Schema } from "../swagger/swaggerTypes";
import { JsonLoader, isRefLike } from "../swagger/jsonLoader";
import { logger } from "./logger";
import { getObjValueFromPointer } from "./diffUtils";

export class SchemaSearcher {
  public static findSchemaByJsonPointer(
    jsonPointer: string,
    schema: Schema,
    jsonLoader: JsonLoader,
    body?: any
  ) {
    const steps = jsonPointer.split("/");
    let curSchema: any = schema;
    let rootSchema: any = schema;
    if (isRefLike(schema)) {
      rootSchema = this.getProperties(schema, jsonLoader);
    }
    const curPaths: string[] = [];
    try {
      for (const step of steps) {
        curSchema = SchemaSearcher.getProperties(curSchema, jsonLoader);
        let found: boolean = false;
        if (step !== "") {
          // if current step is array.
          if (!isNaN(+step)) {
            if (curSchema.type === "array") {
              curSchema = curSchema.items;
            }
          }
          if (curSchema?.properties !== undefined && curSchema?.properties[step] !== undefined) {
            curSchema = curSchema.properties[step];
            found = true;
          }
        }
        curPaths.push(step);
        // If not found, find in discriminator.
        if (!found && rootSchema.discriminatorMap !== undefined) {
          let discriminatorValue = "";
          if (body !== undefined && rootSchema.discriminator !== undefined) {
            const discriminatorJsonPointer = [...curPaths, rootSchema.discriminator].join("/");
            discriminatorValue = getObjValueFromPointer(body, discriminatorJsonPointer);
          }
          if (discriminatorValue !== "") {
            const discriminatorSchemaRef = rootSchema.discriminatorMap[discriminatorValue];
            if (discriminatorSchemaRef !== undefined) {
              const discriminatorSchema = SchemaSearcher.getProperties(
                discriminatorSchemaRef as Schema,
                jsonLoader
              );
              curSchema = discriminatorSchema;
              found = true;
            }
          }
        }
      }
      return this.getProperties(curSchema, jsonLoader);
    } catch (err) {
      logger.error(err);
    }
  }

  public static getProperties(schema: Schema, jsonLoader: JsonLoader): any {
    let ret: any = {};
    if (isRefLike(schema)) {
      ret = jsonLoader.resolveRefObj(schema);
    }
    schema.allOf?.map((item: any) => {
      ret = {
        ...ret,
        ...this.getProperties(jsonLoader.resolveRefObj(item), jsonLoader),
      };
    });

    schema.anyOf?.map((item: any) => {
      ret = {
        ...ret,
        ...this.getProperties(jsonLoader.resolveRefObj(item), jsonLoader),
      };
    });
    return {
      ...ret,
      ...schema,
    };
  }
}
