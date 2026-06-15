// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

/* eslint-disable id-blacklist */

import * as yargs from "yargs";

import { cliSuppressExceptions } from "../cliSuppressExceptions";
import { log } from "../util/logging";
import * as validate from "../validate";

export const command = "extract-xmsexamples <spec-path> <recordings>";

export const describe =
  "Extracts the x-ms-examples for a given swagger from the .NET session recordings and saves " +
  "them in a file.";

export const builder: yargs.CommandBuilder = {
  d: {
    alias: "outDir",
    describe:
      "The output directory where the x-ms-examples files need to be stored. If not provided " +
      'then the output will be stored in a folder name "output" adjacent to the working directory.',
    string: true,
  },
  m: {
    alias: "matchApiVersion",
    describe: "Only generate examples if api-version matches.",
    boolean: true,
    default: true,
  },
};

export async function handler(argv: yargs.Arguments): Promise<void> {
  await cliSuppressExceptions(async () => {
    log.debug(argv.toString());
    const specPath = argv.specPath;
    const recordings = argv.recordings;
    const vOptions = {
      consoleLogLevel: argv.logLevel,
      logFilepath: argv.f,
      output: argv.outDir,
      matchApiVersion: argv.matchApiVersion,
    };
    await validate.extractXMsExamples(specPath, recordings, vOptions);
    return 0;
  });
}
