// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

/* eslint-disable id-blacklist */

import * as yargs from "yargs";

import { cliSuppressExceptions } from "../cliSuppressExceptions";
import { log } from "../util/logging";

import * as validate from "../validate";
export const command = "generate-examples [spec-path]";

export const describe = "Generate swagger examples from real payload records.";

export const builder: yargs.CommandBuilder = {
  o: {
    alias: "operationIds",
    describe: "string of operation ids split by comma.",
    string: true,
  },
  payload: {
    alias: "payloadDir",
    describe: "the directory path contains payload.",
    string: true,
  },
  c: {
    alias: "config",
    describe: "the readme config path.",
    string: true,
  },
  tag: {
    alias: "tagName",
    describe: "the readme tag name.",
    string: true,
  },
  max: {
    alias: "maximumSet",
    describe: "generate examples by rule of MaximumSet.",
    boolean: true,
    default: false,
  },
  min: {
    alias: "minimumSet",
    describe: "generate examples by rule of MinimumSet.",
    boolean: true,
    default: false,
  },
};

export async function handler(argv: yargs.Arguments): Promise<void> {
  await cliSuppressExceptions(async () => {
    log.debug(argv.toString());
    const specPath = argv.specPath;
    const vOptions = {
      consoleLogLevel: argv.logLevel,
      logFilepath: argv.f,
    };
    let generationRule: "Max" | "Min" | undefined;
    if (argv.max && argv.min) {
      generationRule = undefined;
    } else {
      generationRule = argv.max ? "Max" : argv.min ? "Min" : undefined;
    }
    await validate.generateExamples(
      specPath,
      argv.payload,
      argv.o,
      argv.config,
      argv.tag,
      generationRule,
      vOptions
    );
    return 0;
  });
}
