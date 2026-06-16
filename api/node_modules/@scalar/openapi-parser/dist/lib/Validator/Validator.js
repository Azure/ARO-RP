import Ajv, {} from "ajv";
import Ajv2020 from "ajv/dist/2020.js";
import Ajv04 from "ajv-draft-04";
import addFormats from "ajv-formats";
import { ERRORS, OpenApiSpecifications, OpenApiVersions } from "../../configuration/index.js";
import { details as getOpenApiVersion } from "../../utils/details.js";
import { resolveReferences } from "../../utils/resolve-references.js";
import { transformErrors } from "../../utils/transform-errors.js";
const jsonSchemaVersions = {
  "http://json-schema.org/draft-04/schema#": Ajv04,
  "http://json-schema.org/draft-07/schema#": Ajv,
  "https://json-schema.org/draft/2020-12/schema": Ajv2020
};
class Validator {
  version;
  static supportedVersions = OpenApiVersions;
  // Object with function *or* object { errors: string }
  ajvValidators = {};
  errors;
  specificationVersion;
  specificationType;
  specification;
  /**
   * Checks whether a specification is valid and all references can be resolved.
   */
  validate(filesystem, options) {
    const entrypoint = filesystem.find((file) => file.isEntrypoint);
    const specification = entrypoint?.specification;
    this.specification = specification;
    if (this.specification?.info && !this.specification.info.version) {
      this.specification.info.version = "0.0.1";
    }
    try {
      if (specification === void 0 || specification === null) {
        if (options?.throwOnError) {
          throw new Error(ERRORS.EMPTY_OR_INVALID);
        }
        return {
          valid: false,
          errors: transformErrors(specification, ERRORS.EMPTY_OR_INVALID)
        };
      }
      const { version, specificationType, specificationVersion } = getOpenApiVersion(specification);
      this.version = version;
      this.specificationVersion = specificationVersion;
      this.specificationType = specificationType;
      if (!version) {
        if (options?.throwOnError) {
          throw new Error(ERRORS.OPENAPI_VERSION_NOT_SUPPORTED);
        }
        return {
          valid: false,
          errors: transformErrors(specification, ERRORS.OPENAPI_VERSION_NOT_SUPPORTED)
        };
      }
      const validateSchema = this.getAjvValidator(version);
      const schemaResult = validateSchema(specification);
      if (validateSchema.errors) {
        if (validateSchema.errors.length > 0) {
          if (options?.throwOnError) {
            throw new Error(validateSchema.errors[0].message);
          }
          return {
            valid: false,
            errors: transformErrors(specification, validateSchema.errors)
          };
        }
      }
      const resolvedReferences = resolveReferences(filesystem, options);
      return {
        valid: schemaResult && resolvedReferences.valid,
        errors: [...resolvedReferences.errors],
        schema: resolvedReferences.schema
      };
    } catch (error) {
      if (options?.throwOnError) {
        throw error;
      }
      return {
        valid: false,
        errors: transformErrors(specification, error.message ?? error)
      };
    }
  }
  /**
   * Ajv JSON schema validator
   */
  getAjvValidator(version) {
    if (this.ajvValidators[version]) {
      return this.ajvValidators[version];
    }
    const schema = OpenApiSpecifications[version];
    const AjvClass = jsonSchemaVersions[schema.$schema];
    const ajv = new AjvClass({
      // Ajv is a bit too strict in its strict validation of OpenAPI schemas.
      // Switch strict mode off.
      strict: false,
      // Enable discriminator support for better oneOf error messages
      discriminator: true,
      // Show all errors, not just the first one
      allErrors: true
    });
    addFormats(ajv);
    if (version === "3.1" || version === "3.2") {
      ajv.addFormat("media-range", true);
    }
    return this.ajvValidators[version] = ajv.compile(schema);
  }
}
export {
  Validator
};
//# sourceMappingURL=Validator.js.map
