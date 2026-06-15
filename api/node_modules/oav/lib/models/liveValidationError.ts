/*
 * Copyright (c) Microsoft Corporation. All rights reserved.
 * Licensed under the MIT License. See License.txt in the project root for
 * license information.
 */

/**
 * @class
 * Initializes a new instance of the LiveValidationError class.
 * @constructor
 * Describes the error occurred while performing validation on live
 * request/response.
 *
 * @member {string} [code] The unique error code describing an error.
 *
 * @member {string} [message] The error message providing meaningful
 * information.
 *
 */
export class LiveValidationError {
  public constructor(public code?: string, public message?: string) {}

  /**
   * Defines the metadata of LiveValidationError
   *
   * @returns {object} metadata of LiveValidationError
   *
   */
  // eslint-disable-next-line @typescript-eslint/explicit-module-boundary-types
  public mapper() {
    return {
      required: false,
      serializedName: "LiveValidationError",
      type: {
        name: "Composite",
        className: "LiveValidationError",
        modelProperties: {
          code: {
            required: true,
            serializedName: "code",
            type: {
              name: "String",
            },
          },
          message: {
            required: true,
            serializedName: "message",
            type: {
              name: "String",
            },
          },
        },
      },
    };
  }
}
