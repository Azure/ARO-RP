// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

import * as fs from "fs";
import * as os from "os";
import * as path from "path";
import * as winston from "winston";

let logDir = path.resolve(os.homedir(), "oav_output");

let currentLogFile: string;

/*
 * Provides current time in custom format that will be used in naming log files. Example:
 * '20140820_151113'
 * @return {string} Current time in a custom string format
 */
function getTimeStamp(): string {
  // We pad each value so that sorted directory listings show the files in chronological order
  function pad(num: number): string {
    return num < 10 ? "0" + num.toString() : num.toString();
  }

  const now = new Date();
  return (
    pad(now.getFullYear()) +
    pad(now.getMonth() + 1) +
    pad(now.getDate()) +
    "_" +
    pad(now.getHours()) +
    pad(now.getMinutes()) +
    pad(now.getSeconds())
  );
}

const customLogLevels = {
  off: 0,
  json: 1,
  error: 2,
  warn: 3,
  info: 4,
  verbose: 5,
  debug: 6,
  silly: 7,
};

export type ILogger = winston.Logger & {
  consoleLogLevel: unknown;
  filepath: unknown;
  directory: unknown;
};

export const log = winston.createLogger({
  transports: [
    new winston.transports.Console({
      level: "warn",
      format: winston.format.combine(winston.format.colorize(), winston.format.prettyPrint()),
    }),
  ],
  levels: customLogLevels,
}) as ILogger;

Object.defineProperties(log, {
  consoleLogLevel: {
    enumerable: true,
    get() {
      const transport = (this as ILogger).transports.find(
        (t) => t instanceof winston.transports.Console
      );
      return transport !== undefined ? transport.level : undefined;
    },
    set(level) {
      if (!level) {
        level = "warn";
      }
      const validLevels = Object.keys(customLogLevels);
      if (!validLevels.some((item) => item === level)) {
        throw new Error(
          `The logging level provided is "${level}". Valid values are: "${validLevels}".`
        );
      }
      const transport = (this as ILogger).transports.find(
        (t) => t instanceof winston.transports.Console
      );
      if (transport !== undefined) {
        transport.level = level;
      }
    },
  },
  directory: {
    enumerable: true,
    get(): string {
      return logDir;
    },
    set(logDirectory: string): void {
      if (!logDirectory || (logDirectory && typeof logDirectory.valueOf() !== "string")) {
        throw new Error('logDirectory cannot be null or undefined and must be of type "string".');
      }

      if (!fs.existsSync(logDirectory)) {
        fs.mkdirSync(logDirectory);
      }
      logDir = logDirectory;
    },
  },
  filepath: {
    enumerable: true,
    get(): string {
      if (!currentLogFile) {
        const filename = `validate_log_${getTimeStamp()}.log`;
        currentLogFile = path.join(this.directory, filename);
      }

      return currentLogFile;
    },
    set(logFilePath: string): void {
      const self = this as ILogger;
      if (!logFilePath || (logFilePath && typeof logFilePath.valueOf() !== "string")) {
        throw new Error(
          "filepath cannot be null or undefined and must be of type string. It must be " +
            "an absolute file path."
        );
      }
      currentLogFile = logFilePath;
      self.directory = path.dirname(logFilePath);
      if (!self.transports.some((t) => t instanceof winston.transports.File)) {
        self.add(
          new winston.transports.File({
            level: "silly",
            format: winston.format.prettyPrint(),
            silent: false,
            filename: logFilePath,
          })
        );
      }
    },
  },
});
