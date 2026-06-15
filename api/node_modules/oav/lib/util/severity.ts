// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

export const enum Severity {
  Critical = 0,
  Error = 1,
  Warning = 2,
  Information = 3,
  Verbose = 4,
}

export type SeverityString = "Error" | "Critical" | "Warning" | "Info" | "Verbose";

const severityRevMap: { [key: number]: SeverityString } = {
  [Severity.Critical]: "Critical",
  [Severity.Error]: "Error",
  [Severity.Warning]: "Warning",
  [Severity.Information]: "Info",
  [Severity.Verbose]: "Verbose",
};

export const severityStringFromValue = (value: number): SeverityString => {
  return severityRevMap[value];
};
