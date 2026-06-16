// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

import { LiveValidationError } from "./liveValidationError";

/**
 * @class
 * Initializes a new instance of the ErrorResponse class.
 * @constructor
 * Describes the error response for negative scenarios.
 *
 * @member {string} [description] Description of the model property or the
 * parameter in the swagger spec that causes validation failure.
 *
 * @member {array} [params] The parameters used when validation failed
 * (z-schema construct).
 *
 * @member {array} [path] The path to the location in the document or the model
 * where the error/warning occurred.
 *
 */
export class ErrorResponse extends LiveValidationError {
  public constructor() {
    super();
  }

  /**
   * Defines the metadata of ErrorResponse
   *
   * @returns {object} metadata of ErrorResponse
   *
   */
  // eslint-disable-next-line @typescript-eslint/explicit-module-boundary-types
  public mapper() {
    return {
      required: false,
      serializedName: "ErrorResponse",
      type: {
        name: "Composite",
        className: "ErrorResponse",
        modelProperties: {
          code: {
            required: false,
            serializedName: "code",
            type: {
              name: "String",
            },
          },
          message: {
            required: false,
            serializedName: "message",
            type: {
              name: "String",
            },
          },
          description: {
            required: false,
            serializedName: "description",
            type: {
              name: "String",
            },
          },
          params: {
            required: false,
            serializedName: "params",
            type: {
              name: "Sequence",
              element: {
                required: false,
                serializedName: "StringElementType",
                type: {
                  name: "String",
                },
              },
            },
          },
          path: {
            required: false,
            serializedName: "path",
            type: {
              name: "Sequence",
              element: {
                required: false,
                serializedName: "StringElementType",
                type: {
                  name: "String",
                },
              },
            },
          },
        },
      },
    };
  }
}
