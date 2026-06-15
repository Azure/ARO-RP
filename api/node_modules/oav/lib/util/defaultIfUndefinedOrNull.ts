// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

export const defaultIfUndefinedOrNull = <T>(value: T | null | undefined, defaultValue: T): T =>
  value !== null && value !== undefined ? value : defaultValue;
