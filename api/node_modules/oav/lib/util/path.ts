// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

import { flatMap } from "@azure-tools/openapi-tools-common";

export const splitPathAndReverse = (p: string | undefined): string[] | undefined =>
  p === undefined ? undefined : Array.from(flatMap(p.split("/"), (s) => s.split("\\"))).reverse();

export const isSubPath = (
  mainPath: readonly string[] | undefined,
  subPath: readonly string[] | undefined
): boolean =>
  // return `true` if there are no subPath.
  subPath === undefined ||
  subPath.length === 0 ||
  // return `true` if `subPath` is a sub-path of `mainPath`.
  (mainPath !== undefined &&
    mainPath.length > subPath.length &&
    subPath.every((s, i) => mainPath[i] === s));
