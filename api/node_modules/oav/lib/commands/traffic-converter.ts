// node ./build/cli.js convert --directory ./input-example/ --out ./output-example/
import * as fs from "fs";
import * as path from "path";
import { URL } from "url";
import * as yargs from "yargs";
import { cliSuppressExceptions } from "../cliSuppressExceptions";
import { log } from "../util/logging";

let InputDirectory: string = "";
let OutputDirectory: string = "";
const CONTENT_TYPE_KEY: string = "Content-Type";

interface IndexableAny {
  [index: string]: string;
}

interface ProxyPayload {
  RequestUri: string;
  RequestMethod: string;
  RequestHeaders: IndexableAny;
  RequestBody: string;
  ResponseHeaders: IndexableAny;
  ResponseBody: {};
  StatusCode: string;
}

interface LiveRequest {
  body: any;
  method: string;
  url: string;
  headers: any;
}

interface LiveResponse {
  body: any;
  statusCode: string;
  headers: any;
}

interface ValidationPayload {
  liveRequest: LiveRequest;
  liveResponse: LiveResponse;
}

function requestBodyConversion(body: string, headers: any) {
  if (CONTENT_TYPE_KEY in headers) {
    let content: string = headers[CONTENT_TYPE_KEY];

    if (content.indexOf("application/json") > -1) {
      // if RequestBody is a unicode string, it need JSON.parse to format
      // eg. JSON.parse("{\u0022TableName\u0022: \u0022pytablesync70281ff8\u0022}") => {TableName: 'pytablesync70281ff8'}
      // else return directly includes Object or Array
      try {
        return JSON.parse(body);
      } catch (ex) {
        log.error(ex);
        return body;
      }
    }
  }

  return body;
}

function requestUriConversion(uri: string, version: string): string {
  const parsedUrl = new URL(uri);

  if (!parsedUrl.searchParams.get("api-version")) {
    parsedUrl.searchParams.set("api-version", version);
  }

  return parsedUrl.toString();
}

function processFile(file: string, inputJson: any) {
  if (inputJson.Entries !== undefined && inputJson.Entries.length > 0) {
    const filePrefix = file.substring(0, file.lastIndexOf("."));
    inputJson.Entries.forEach((entry: ProxyPayload, idx: number) => {
      const outFile = `${filePrefix}_${String(idx).padStart(4, "0")}.json`;
      const newEntry: ValidationPayload = {
        liveRequest: <LiveRequest>{},
        liveResponse: <LiveResponse>{},
      };

      // manipulate the request URI
      newEntry.liveRequest.url = requestUriConversion(
        entry.RequestUri,
        entry.RequestHeaders["x-ms-version"]
      );
      newEntry.liveRequest.headers = entry.RequestHeaders;

      // the request body is expected to be a JSON entry. Force that conversion if we can.
      newEntry.liveRequest.body = requestBodyConversion(entry.RequestBody, entry.RequestHeaders);
      newEntry.liveRequest.method = entry.RequestMethod;

      newEntry.liveResponse.body = entry.ResponseBody;
      // ensure string status code
      newEntry.liveResponse.statusCode = entry.StatusCode.toString();
      newEntry.liveResponse.headers = entry.ResponseHeaders;

      outputFile(outFile, newEntry);
    });
  }
}

function readFile(file: string) {
  const input_location = path.join(InputDirectory, file);

  fs.readFile(input_location, "utf8", (err: any, data: any) => {
    if (err) {
      throw err;
    }
    let convertedJson: any = {};

    try {
      // handle byte order mark
      convertedJson = JSON.parse(data.charCodeAt(0) === 0xfeff ? data.slice(1) : data);
    } catch (ex) {
      log.error(ex);
      throw ex;
    }
    processFile(file, convertedJson);
  });
}

function outputFile(file: string, outputJson: ValidationPayload) {
  const data = JSON.stringify(outputJson, null, 4);
  const outputLocation = path.join(OutputDirectory, file);

  fs.writeFile(outputLocation, data, (err: any) => {
    if (err) {
      throw err;
    }
  });
}

function convert(directory: string, outDirectory: string) {
  log.info(`Converting files in folder ${directory} ${outDirectory}`);
  log.info(`Input Directory: ${directory}. Output Directory: ${outDirectory}`);

  let files: string[] = fs.readdirSync(directory);
  log.info(`Operating on ${files.length} files.`);

  files.forEach((file: string) => {
    readFile(file);
  });
}

export const command = "traffic-convert <input-dir> <output-dir>";
export const describe = "Convert a folder files to traffic files";

export const builder: yargs.CommandBuilder = {
  inputDir: {
    alias: "d",
    describe: "The targeted input directory.",
    string: true,
  },
  outputDir: {
    alias: "o",
    describe: "The targeted output directory.",
    string: true,
  },
};

export async function handler(argv: yargs.Arguments): Promise<void> {
  await cliSuppressExceptions(async () => {
    log.debug(argv.toString());
    InputDirectory = argv.inputDir;
    OutputDirectory = argv.outputDir;
    convert(InputDirectory, OutputDirectory);
    return 0;
  });
}
