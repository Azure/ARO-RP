// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

import * as yargs from "yargs";

import { cliSuppressExceptions } from "../cliSuppressExceptions";
import { log } from "../util/logging";
import * as validate from "../validate";

export const command = "validate-spec <spec-path>";

export const describe = "Performs semantic validation of the spec.";

export async function handler(argv: yargs.Arguments): Promise<void> {
  await cliSuppressExceptions(async () => {
    log.debug(argv.toString());
    const specPath = argv.specPath;
    const vOptions: validate.Options = {
      consoleLogLevel: argv.logLevel,
      logFilepath: argv.f,
      pretty: argv.p ?? true,
    };
    const result = await validate.validateSpec(specPath, vOptions);
    return result.validityStatus ? 0 : 1;
  });
}
