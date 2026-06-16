// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

import * as path from "path";
import { isArray, parseMarkdown, readFile, some } from "@azure-tools/openapi-tools-common";
import * as amd from "@azure/openapi-markdown";
import { log } from "../util/logging";
import { isSubPath, splitPathAndReverse } from "../util/path";

export const getSuppressions = async (specPath: string): Promise<undefined | amd.Suppression> => {
  // find readme.md
  try {
    const readMe = await amd.findReadMe(path.dirname(specPath));
    if (readMe === undefined) {
      return undefined;
    }
    const readMeStr = await readFile(readMe);
    const cmd = parseMarkdown(readMeStr);
    const suppressionCodeBlock = amd.getCodeBlocksAndHeadings(cmd.markDown).Suppression;
    if (suppressionCodeBlock === undefined) {
      return undefined;
    }
    const suppression = amd.getYamlFromNode(suppressionCodeBlock) as amd.Suppression;
    if (!isArray(suppression.directive)) {
      return undefined;
    }
    return suppression;
  } catch (err) {
    log.warn(`Unable to load and parse suppression file. Error: ${err}`);
    return undefined;
  }
};

export const existSuppression = (
  specPath: string,
  suppression: amd.Suppression,
  id: string
): boolean => {
  if (suppression.directive !== undefined) {
    const suppressionArray = getSuppressionArray(specPath, suppression.directive);
    return some(suppressionArray, (s) => s.suppress === id);
  }
  return false;
};

const getSuppressionArray = (
  specPath: string,
  suppressionItems: readonly amd.SuppressionItem[]
): readonly amd.SuppressionItem[] => {
  const urlReversed = splitPathAndReverse(specPath);
  return suppressionItems.filter((s) =>
    some(isArray(s.from) ? s.from : [s.from], (from) =>
      isSubPath(urlReversed, splitPathAndReverse(from))
    )
  );
};
