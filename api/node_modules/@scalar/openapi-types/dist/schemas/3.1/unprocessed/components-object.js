import { z } from "zod";
import { ComponentsObjectSchema as OriginalComponentsObjectSchema } from "../processed/components-object.js";
import { CallbackObjectSchema } from "./callback-object.js";
import { ExampleObjectSchema } from "./example-object.js";
import { HeaderObjectSchema } from "./header-object.js";
import { LinkObjectSchema } from "./link-object.js";
import { ParameterObjectSchema } from "./parameter-object.js";
import { PathItemObjectSchema } from "./path-item-object.js";
import { ReferenceObjectSchema } from "./reference-object.js";
import { RequestBodyObjectSchema } from "./request-body-object.js";
import { ResponseObjectSchema } from "./response-object.js";
import { SchemaObjectSchema } from "./schema-object.js";
import { SecuritySchemeObjectSchema } from "./security-scheme-object.js";
const ComponentsObjectSchema = OriginalComponentsObjectSchema.extend({
  /**
   * An object to hold reusable Schema Objects.
   */
  schemas: z.record(z.string(), SchemaObjectSchema).optional(),
  /**
   * An object to hold reusable Response Objects.
   */
  responses: z.record(z.string(), z.union([ReferenceObjectSchema, ResponseObjectSchema])).optional(),
  /**
   * An object to hold reusable Parameter Objects.
   */
  parameters: z.record(z.string(), z.union([ReferenceObjectSchema, ParameterObjectSchema])).optional(),
  /**
   * An object to hold reusable Example Objects.
   */
  examples: z.record(z.string(), z.union([ReferenceObjectSchema, ExampleObjectSchema])).optional(),
  /**
   * An object to hold reusable Request Body Objects.
   */
  requestBodies: z.record(z.string(), z.union([ReferenceObjectSchema, RequestBodyObjectSchema])).optional(),
  /**
   * An object to hold reusable Header Objects.
   */
  headers: z.record(z.string(), z.union([ReferenceObjectSchema, HeaderObjectSchema])).optional(),
  /**
   * An object to hold reusable Security Scheme Objects.
   */
  securitySchemes: z.record(z.string(), z.union([ReferenceObjectSchema, SecuritySchemeObjectSchema])).optional(),
  /**
   * An object to hold reusable Link Objects.
   */
  links: z.record(z.string(), z.union([ReferenceObjectSchema, LinkObjectSchema])).optional(),
  /**
   * An object to hold reusable Callback Objects.
   */
  callbacks: z.record(z.string(), z.union([ReferenceObjectSchema, CallbackObjectSchema])).optional(),
  /**
   * An object to hold reusable Path Item Objects.
   */
  pathItems: z.record(z.string(), PathItemObjectSchema).optional()
});
export {
  ComponentsObjectSchema
};
//# sourceMappingURL=components-object.js.map
