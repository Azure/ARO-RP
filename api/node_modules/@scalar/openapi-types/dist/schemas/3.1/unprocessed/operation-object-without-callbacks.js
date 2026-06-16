import { z } from "zod";
import { OperationObjectSchemaWithoutCallbacks as OriginalOperationObjectSchemaWithoutCallbacks } from "../processed/operation-object-without-callbacks.js";
import { ExternalDocumentationObjectSchema } from "./external-documentation-object.js";
import { ParameterObjectSchema } from "./parameter-object.js";
import { ReferenceObjectSchema } from "./reference-object.js";
import { RequestBodyObjectSchema } from "./request-body-object.js";
import { SecurityRequirementObjectSchema } from "./security-requirement-object.js";
const OperationObjectSchemaWithoutCallbacks = OriginalOperationObjectSchemaWithoutCallbacks.extend({
  /**
   * External documentation object
   */
  "externalDocs": ExternalDocumentationObjectSchema.optional(),
  /**
   * A list of parameters that are applicable for this operation. If a parameter is already defined at the Path Item,
   * the new definition will override it but can never remove it. The list MUST NOT include duplicated parameters. A
   * unique parameter is defined by a combination of a name and location. The list can use the Reference Object to link
   * to parameters that are defined in the OpenAPI Object's components.parameters.
   */
  "parameters": z.union([ReferenceObjectSchema, ParameterObjectSchema]).array().optional(),
  /**
   * The request body applicable for this operation. The requestBody is fully supported in HTTP methods where the
   * HTTP 1.1 specification [RFC7231] has explicitly defined semantics for request bodies. In other cases where the
   * HTTP spec is vague (such as GET, HEAD and DELETE), requestBody is permitted but does not have well-defined
   * semantics and SHOULD be avoided if possible.
   */
  "requestBody": z.union([ReferenceObjectSchema, RequestBodyObjectSchema]).optional(),
  /**
   * A declaration of which security mechanisms can be used across the API. The list of
   * values includes alternative security requirement objects that can be used. Only
   * one of the security requirement objects need to be satisfied to authorize a request.
   * Individual operations can override this definition. To make security optional, an empty
   * security requirement ({}) can be included in the array.
   */
  "security": z.array(SecurityRequirementObjectSchema).optional()
});
export {
  OperationObjectSchemaWithoutCallbacks
};
//# sourceMappingURL=operation-object-without-callbacks.js.map
