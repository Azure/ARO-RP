import * as yargs from "yargs";

import { cliSuppressExceptions } from "../cliSuppressExceptions";
import { TrafficValidationOptions } from "../swaggerValidator/trafficValidator";
import { log } from "../util/logging";
import * as validate from "../validate";

export const command = "validate-traffic <traffic-path> <spec-path>";
export const describe = "Validate traffic payload against the spec.";

export const builder: yargs.CommandBuilder = {
  trafficPath: {
    alias: "t",
    describe: "The recording payload path.",
    string: true,
  },
  specPath: {
    alias: "s",
    describe: "The targeted swagger spec path.",
    string: true,
  },
  package: {
    alias: "pkg",
    describe: "The target SDK package name",
    string: true,
    default: "azure-data-tables",
  },
  language: {
    alias: "lang",
    describe: "The target language of SDK",
    string: true,
    default: "Dotnet",
  },
  report: {
    alias: "r",
    describe: "path and file name for the report",
    string: true,
    default: "./SwaggerAccuracyReport.html",
  },
  overrideLinkInReport: {
    alias: "overridelink",
    describe: "override spec link and payload link in report with github url",
    boolean: true,
    default: false,
  },
  outputExceptionInReport: {
    alias: "oe",
    describe: "Whether rendering the runtime exceptions in the validation report",
    boolean: true,
    default: false,
  },
  specLinkPrefix: {
    alias: "slp",
    describe: "github specification link prefix",
    string: true,
    default: "https://github.com/Azure/azure-rest-api-specs/blob/main/",
  },
  payloadLinkPrefix: {
    alias: "plp",
    describe: "traffic payload link prefix",
    string: true,
    default: "https://github.com/scbedd/oav-traffic-converter/blob/main/sample-tables-input/",
  },
  jsonReport: {
    describe: "path and file name for json report",
    string: true,
  },
};

export async function handler(argv: yargs.Arguments): Promise<void> {
  await cliSuppressExceptions(async () => {
    log.debug(argv.toString());
    const specPath = argv.specPath;
    const trafficPath = argv.trafficPath;
    const vOptions: TrafficValidationOptions = {
      consoleLogLevel: argv.logLevel,
      logFilepath: argv.f,
      pretty: argv.p,
      sdkPackage: argv.package,
      sdkLanguage: argv.language,
      reportPath: argv.report,
      overrideLinkInReport: argv.overrideLinkInReport,
      outputExceptionInReport: argv.outputExceptionInReport,
      specLinkPrefix: argv.specLinkPrefix,
      payloadLinkPrefix: argv.payloadLinkPrefix,
      jsonReportPath: argv.jsonReport,
    };
    const errors = await validate.validateTraffic(specPath, trafficPath, vOptions);
    return errors.length > 0 ? 1 : 0;
  });
}
