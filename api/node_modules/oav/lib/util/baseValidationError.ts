// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

import { Severity } from "./severity";
import { NodeError } from "./validationError";
import { ValidationResultSource } from "./validationResultSource";

export interface BaseValidationError<T extends NodeError<T>> {
  severity?: Severity;
  code?: string;
  details?: T;
  source?: ValidationResultSource;
  count?: number;
}
