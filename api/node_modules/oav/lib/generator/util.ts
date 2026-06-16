import * as fs from "fs";
import * as path from "path";
import { URL } from "url";

const ATTR_MAP: any = {
  location: "location",
  "azure-async-operation": "Azure-AsyncOperation",
  "original-uri": "original-uri",
};

export function getBaseName(filePath: string) {
  return path.basename(filePath);
}
export function isObject(definitionSpec: any) {
  return (
    definitionSpec.type === "object" || "properties" in definitionSpec || "allOf" in definitionSpec
  );
}
export function randomKey() {
  return `key${Math.ceil(Math.random() * 10000)}`;
}

export function getRp(specFilePath: string): string {
  return "microsoft." + specFilePath.split("/").slice(-1)[0].split(".")[0].toLowerCase();
}

export function fileExist(...pathName: string[]): boolean {
  return fs.existsSync(path.resolve(...pathName));
}

export function isLongRunning(specItem: any) {
  return specItem.content["x-ms-long-running-operation"];
}

export function readPayloadFile(payloadDir: string, operationId: string): any {
  if (!fileExist(payloadDir, operationId)) {
    return null;
  }
  const inputPath = path.resolve(payloadDir, operationId);
  const filenames: string[] = fs.readdirSync(inputPath).filter((filename: string) => {
    return (
      filename.endsWith(".json") &&
      ["200", "201", "202", "204"].includes(filename.split(".json")[0])
    );
  });
  if (filenames.length === 0) {
    return null;
  }
  const payload: any = {};
  filenames.forEach((filename: string) => {
    const statusCode = filename.split(".json")[0];
    const data = fs.readFileSync(path.resolve(inputPath, filename));
    payload[statusCode] = JSON.parse(data.toString());
  });
  return payload;
}

export function getPollingAttr(specItem: any) {
  if (!specItem.content["x-ms-long-running-operation"]) {
    return;
  }
  const lrOption = specItem.content["x-ms-long-running-operation-options"];
  if (lrOption && lrOption["final-state-via"]) {
    return ATTR_MAP[lrOption["final-state-via"]];
  }

  return "location";
}

export function getPollingUrl(payload: any, specItem: any) {
  let attr = "location";
  const asyncOperation: string = payload.liveResponse.headers["azure-AsyncOperation"];

  const lrOption = specItem.content["x-ms-long-running-operation-options"];
  let location = payload.liveResponse.headers.location;

  if (lrOption && lrOption["final-state-via"]) {
    attr = ATTR_MAP[lrOption["final-state-via"]];
    location = payload.liveResponse.headers[attr];
  } else if (asyncOperation) {
    location = asyncOperation;
  }
  const url = new URL(location);
  return url.pathname + url.search;
}

export function updateExmAndSpecFile(
  example: any,
  newSpec: any,
  specFilePath: string,
  exampleName: string
): any {
  const outputDir = path.resolve(specFilePath, "../", "examples");
  if (!fs.existsSync(outputDir)) {
    fs.mkdirSync(outputDir);
  }
  const outputPath = path.resolve(outputDir, exampleName);
  console.log("example file path: " + outputPath);
  fs.writeFileSync(outputPath, JSON.stringify(example, null, 2), "utf8");
  if (newSpec) {
    //log.info("updated swagger file path: " + specFilePath);
    fs.writeFileSync(specFilePath, JSON.stringify(newSpec, null, 2), "utf8");
  }
}

export function referenceExmInSpec(
  specFilePath: string,
  apiPath: string,
  methodName: string,
  exampleName: string
): any {
  const data = fs.readFileSync(specFilePath);
  const spec = JSON.parse(data.toString());
  if (!spec.paths[apiPath][methodName]["x-ms-examples"]) {
    spec.paths[apiPath][methodName]["x-ms-examples"] = {};
  }
  if (!(exampleName in spec.paths[apiPath][methodName]["x-ms-examples"])) {
    spec.paths[apiPath][methodName]["x-ms-examples"][exampleName] = {
      $ref: `./examples/${exampleName}.json`,
    };
    return spec;
  }
}
