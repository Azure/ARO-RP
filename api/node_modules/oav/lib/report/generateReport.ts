import * as fs from "fs";
import * as path from "path";
import * as Mustache from "mustache";
import {
  OperationCoverageInfo,
  OperationMeta,
  TrafficValidationIssue,
  TrafficValidationOptions,
} from "../swaggerValidator/trafficValidator";
import { LiveValidationIssue } from "../liveValidation/liveValidator";
import { FileLoader } from "../swagger/fileLoader";
import { OperationContext } from "../liveValidation/operationValidator";

export interface TrafficValidationIssueForRendering extends TrafficValidationIssue {
  payloadFileLinkLabel?: string;
  payloadFilePathWithPosition?: string;
  errorsForRendering?: LiveValidationIssueForRendering[];
  errorCodeLen: number;
}

export interface runtimeExceptionList {
  payloadFilePath?: string;
  code: string;
  message: string;
  specList: Array<{ specLabel: string; specLink: string }>;
}

export interface TrafficValidationIssueForRenderingInner {
  generalErrorsInner: TrafficValidationIssueForRendering[];
  errorCodeLen: number;
  specFilePath?: string;
  specFilePathWithPosition?: string;
  operationInfo: OperationContext;
  errorsForRendering: LiveValidationIssueForRendering[];
}

export interface LiveValidationIssueForRendering extends LiveValidationIssue {
  friendlyName?: string;
  link?: string;
  payloadFilePath?: string | undefined;
  payloadFilePathWithPosition?: string;
  schemaPathWithPosition?: string;
  payloadFileLinkLabel?: string | undefined;
}

export interface ErrorDefinitionDoc {
  ErrorDefinitions: ErrorDefinition[];
}

export interface ErrorDefinition {
  code: string;
  friendlyName?: string;
  link?: string;
}

export interface ValidationPassOperationsFormatInner extends OperationMeta {
  readonly key: string;
}

export interface ValidationPassOperationsFormat {
  readonly operationIdList: ValidationPassOperationsFormatInner[];
}

export interface OperationCoverageInfoForRendering extends OperationCoverageInfo {
  specLinkLabel?: string;
  validationPassOperations?: number;
  validationPassOperationList: ValidationPassOperationsFormat[];
  generalErrorsInnerList: TrafficValidationIssueForRenderingInner[];
}

export interface resultForRendering
  extends OperationCoverageInfoForRendering,
    TrafficValidationIssueForRendering {
  index?: number;
}

export async function loadErrorDefinitions(): Promise<Map<string, ErrorDefinition>> {
  const errorDefinitionDoc =
    require("../../../documentation/error-definitions.json") as ErrorDefinitionDoc;
  const errorsMap: Map<string, ErrorDefinition> = new Map();
  errorDefinitionDoc.ErrorDefinitions.forEach((def) => {
    errorsMap.set(def.code, def);
  });
  return errorsMap;
}

// used to pass data to the template rendering engine
export class CoverageView {
  public package: string;
  public language: string;
  public apiVersion: string = "unknown";
  public generatedDate: Date;
  public markdownPath: string;
  public markdown: string;

  public undefinedOperationCount: number = 0;
  public operationValidated: number = 0;
  public operationFailed: number = 0;
  public operationUnValidated: number = 0;
  public generalErrorResults: Map<string, TrafficValidationIssue[]>;

  public validationResultsForRendering: TrafficValidationIssueForRendering[] = [];
  public coverageResultsForRendering: OperationCoverageInfoForRendering[] = [];
  public resultsForRendering: resultForRendering[] = [];

  private validationResults: TrafficValidationIssue[];
  private sortedValidationResults: TrafficValidationIssue[];
  private coverageResults: OperationCoverageInfo[];

  private specLinkPrefix: string;
  private payloadLinkPrefix: string;
  private overrideLinkInReport: boolean;
  private outputExceptionInReport: boolean;

  public constructor(
    validationResults: TrafficValidationIssue[],
    coverageResults: OperationCoverageInfo[],
    undefinedOperationCount: number = 0,
    packageName: string = "",
    language: string = "",
    markdownPath: string = "",
    overrideLinkInReport: boolean = false,
    outputExceptionInReport: boolean = false,
    specLinkPrefix: string = "",
    payloadLinkPrefix: string = ""
  ) {
    this.package = packageName;
    this.markdownPath = markdownPath;
    this.validationResults = validationResults;
    this.coverageResults = coverageResults;
    this.undefinedOperationCount = undefinedOperationCount;
    this.generatedDate = new Date();
    this.generalErrorResults = new Map();
    this.language = language;
    this.overrideLinkInReport = overrideLinkInReport;
    this.outputExceptionInReport = outputExceptionInReport;
    this.specLinkPrefix = specLinkPrefix;
    this.payloadLinkPrefix = payloadLinkPrefix;

    if (this.overrideLinkInReport === true) {
      if (this.specLinkPrefix.endsWith("/")) {
        this.specLinkPrefix = this.specLinkPrefix.substring(0, this.specLinkPrefix.length - 1);
      }

      if (this.payloadLinkPrefix.endsWith("/")) {
        this.payloadLinkPrefix = this.payloadLinkPrefix.substring(
          0,
          this.payloadLinkPrefix.length - 1
        );
      }
    }

    this.setMetrics();
    this.sortOperationIds();
  }

  public async prepareDataForRendering() {
    try {
      this.markdown = await this.readMarkdown();
      const errorDefinitions = await loadErrorDefinitions();
      let errorsForRendering: LiveValidationIssueForRendering[];
      this.sortedValidationResults.forEach((element) => {
        const payloadFile = element.payloadFilePath?.substring(
          element.payloadFilePath.lastIndexOf("/") + 1
        );
        errorsForRendering = [];
        element.errors?.forEach((error) => {
          const errorDef = errorDefinitions.get(error.code);
          errorsForRendering.push({
            friendlyName: errorDef?.friendlyName ?? error.code,
            link: errorDef?.link,
            code: error.code,
            message: error.message,
            schemaPath: error.schemaPath,
            schemaPathWithPosition: this.overrideLinkInReport
              ? `${this.specLinkPrefix}/${element.specFilePath?.substring(
                  element.specFilePath?.indexOf("specification")
                )}#L${error.source.position.line}`
              : `${element.specFilePath}#L${error.source.position.line}`,
            pathsInPayload: error.pathsInPayload,
            jsonPathsInPayload: error.jsonPathsInPayload,
            severity: error.severity,
            source: error.source,
            params: error.params,
            payloadFilePath: this.overrideLinkInReport
              ? `${this.payloadLinkPrefix}/${payloadFile}`
              : element.payloadFilePath,
            payloadFilePathWithPosition: this.overrideLinkInReport
              ? `${this.payloadLinkPrefix}/${payloadFile}#L${element.payloadFilePathPosition?.line}`
              : `${element.payloadFilePath}#L${element.payloadFilePathPosition?.line}`,
            payloadFileLinkLabel: payloadFile,
          });
        });
        this.validationResultsForRendering.push({
          payloadFilePath: this.overrideLinkInReport
            ? `${this.payloadLinkPrefix}/${payloadFile}`
            : element.payloadFilePath,
          payloadFileLinkLabel: payloadFile,
          payloadFilePathWithPosition: this.overrideLinkInReport
            ? `${this.payloadLinkPrefix}/${payloadFile}#L${element.payloadFilePathPosition?.line}`
            : `${element.payloadFilePath}#L${element.payloadFilePathPosition?.line}`,
          errors: element.errors,
          specFilePath: this.overrideLinkInReport
            ? `${this.specLinkPrefix}/${element.specFilePath?.substring(
                element.specFilePath?.indexOf("specification")
              )}`
            : element.specFilePath,
          errorsForRendering: errorsForRendering,
          errorCodeLen: errorsForRendering.length,
          operationInfo: element.operationInfo,
          runtimeExceptions: element.runtimeExceptions,
        });
      });

      const generalErrorsInnerOrigin = this.validationResultsForRendering.filter((x) => {
        return x.errors && x.errors.length > 0;
      });

      this.coverageResults.forEach((element) => {
        const specLink = this.overrideLinkInReport
          ? `${this.specLinkPrefix}/${element.spec?.substring(
              element.spec?.indexOf("specification")
            )}`
          : `${element.spec}`;

        let errorOperationIds = generalErrorsInnerOrigin.map(
          (item) => item.operationInfo?.operationId
        );
        let passOperations: ValidationPassOperationsFormatInner[] = element.coveredOperationsList
          .filter((item) => errorOperationIds.indexOf(item.operationId) === -1)
          .map((item) => {
            return {
              key: item.operationId.split("_")[0],
              operationId: item.operationId,
            };
          });

        const passOperationsInnerList: ValidationPassOperationsFormatInner[][] = Object.values(
          passOperations.reduce(
            (res: { [key: string]: ValidationPassOperationsFormatInner[] }, item) => {
              /* eslint-disable no-unused-expressions */
              res[item.key] ? res[item.key].push(item) : (res[item.key] = [item]);
              /* eslint-enable no-unused-expressions */
              return res;
            },
            {}
          )
        );

        const passOperationsListFormat: ValidationPassOperationsFormat[] = [];
        passOperationsInnerList.forEach((element) => {
          passOperationsListFormat.push({
            operationIdList: element,
          });
        });

        /**
         * Sort untested operationId by bubble sort
         * Controlling the results of localeCompare can set the sorting method
         * X.localeCompare(Y) > 0 descending sort
         * X.localeCompare(Y) < 0 ascending sort
         */
        for (let i = 0; i < passOperationsListFormat.length - 1; i++) {
          for (let j = 0; j < passOperationsListFormat.length - 1 - i; j++) {
            if (
              passOperationsListFormat[j].operationIdList[0].key.localeCompare(
                passOperationsListFormat[j + 1].operationIdList[0].key
              ) > 0
            ) {
              var temp = passOperationsListFormat[j];
              passOperationsListFormat[j] = passOperationsListFormat[j + 1];
              passOperationsListFormat[j + 1] = temp;
            }
          }
        }

        this.coverageResultsForRendering.push({
          spec: specLink,
          specLinkLabel: element.spec?.substring(element.spec?.lastIndexOf("/") + 1),
          apiVersion: element.apiVersion,
          coveredOperations: element.coveredOperations,
          coveredOperationsList: element.coveredOperationsList,
          validationPassOperations: element.coveredOperations - element.validationFailOperations,
          validationPassOperationList: passOperationsListFormat,
          validationFailOperations: element.validationFailOperations,
          unCoveredOperations: element.unCoveredOperations,
          unCoveredOperationsList: element.unCoveredOperationsList,
          unCoveredOperationsListGen: element.unCoveredOperationsListGen,
          totalOperations: element.totalOperations,
          coverageRate: element.coverageRate,
          generalErrorsInnerList: [],
        });
      });

      this.resultsForRendering = this.coverageResultsForRendering.map((item) => {
        const data = this.validationResultsForRendering.find(
          (i) =>
            i.specFilePath &&
            item.spec.split(path.win32.sep).join(path.posix.sep).includes(i.specFilePath)
        );
        return {
          ...item,
          ...data,
        } as any;
      });

      const generalErrorsInnerFormat: TrafficValidationIssueForRendering[][] = Object.values(
        generalErrorsInnerOrigin.reduce(
          (res: { [key: string]: TrafficValidationIssueForRendering[] }, item) => {
            /* eslint-disable no-unused-expressions */
            res[item!.operationInfo!.operationId + item!.specFilePath]
              ? res[item!.operationInfo!.operationId + item!.specFilePath].push(item)
              : (res[item!.operationInfo!.operationId + item!.specFilePath] = [item]);
            /* eslint-enable no-unused-expressions */
            return res;
          },
          {}
        )
      );
      const generalErrorsInnerList: TrafficValidationIssueForRenderingInner[] = [];
      generalErrorsInnerFormat.forEach((element) => {
        let errorCodeLen: number = 0;
        element.forEach((item) => {
          errorCodeLen = errorCodeLen + item.errorCodeLen;
        });
        let errorsForRendering: LiveValidationIssueForRendering[] = [];
        element.forEach((item) => {
          errorsForRendering = errorsForRendering.concat(item.errorsForRendering!);
        });
        generalErrorsInnerList.push({
          generalErrorsInner: element,
          errorCodeLen: errorCodeLen,
          errorsForRendering: errorsForRendering,
          operationInfo: element[0]!.operationInfo!,
          specFilePath: this.overrideLinkInReport
            ? `${this.specLinkPrefix}/${element[0].specFilePath?.substring(
                element[0].specFilePath?.indexOf("specification")
              )}`
            : element[0].specFilePath,
          specFilePathWithPosition: this.overrideLinkInReport
            ? `${this.specLinkPrefix}/${element[0].specFilePath?.substring(
                element[0].specFilePath?.indexOf("specification")
              )}#L${element[0]!.operationInfo!.position!.line}`
            : `${element[0].specFilePath}#L${element[0]!.operationInfo!.position!.line}`,
        });
      });

      for (const [index, e] of this.resultsForRendering.entries()) {
        e.index = index;
        for (const i of generalErrorsInnerList) {
          if (e.specFilePath === i.specFilePath && i) {
            e.generalErrorsInnerList.push(i);
          }
        }
      }
    } catch (e) {
      console.error(`Failed in prepareDataForRendering with err:${e?.stack};message:${e?.message}`);
    }
  }

  private async readMarkdown() {
    try {
      const loader = new FileLoader({});
      const res = await loader.load(this.markdownPath);
      return res;
    } catch (e) {
      console.error(`Failed in read report.md file`);
      return "";
    }
  }

  private sortOperationIds() {
    this.sortedValidationResults = this.validationResults.sort(function (op1, op2) {
      const opId1 = op1.operationInfo!.operationId;
      const opId2 = op2.operationInfo!.operationId;
      if (opId1 < opId2) {
        return -1;
      }
      if (opId1 > opId2) {
        return 1;
      }
      return 0;
    });
  }

  private setMetrics() {
    if (this.coverageResults?.length > 0) {
      this.apiVersion = this.coverageResults[0].apiVersion;
    }
  }

  public formatGeneratedDate(): string {
    const day = this.generatedDate.getDate();
    const month = this.generatedDate.getMonth() + 1;
    const year = this.generatedDate.getFullYear();
    const hours = this.generatedDate.getHours();
    const minutes = this.generatedDate.getMinutes();

    return (
      year +
      "-" +
      (month < 10 ? "0" + month : month) +
      "-" +
      (day < 10 ? "0" + day : day) +
      " at " +
      hours +
      ":" +
      (minutes < 10 ? "0" + minutes : minutes) +
      (hours < 13 ? "AM" : "PM")
    );
  }

  public getTotalErrors(): number {
    return this.validationResults.length;
  }

  public getGeneralErrors(): TrafficValidationIssueForRendering[] {
    return this.validationResultsForRendering.filter((x) => {
      return x.errors && x.errors.length > 0;
    });
  }

  public getTotalGeneralErrors(): number {
    return this.getGeneralErrors().length;
  }

  public getRunTimeErrors(): runtimeExceptionList[] {
    if (this.outputExceptionInReport) {
      const res = this.validationResults.filter((x) => {
        return x.runtimeExceptions && x.runtimeExceptions.length > 0;
      });
      const resFormat: runtimeExceptionList[] = [];
      res.forEach((element) => {
        element.runtimeExceptions &&
          element.runtimeExceptions.forEach((i) => {
            const specList = i.spec;
            const specListFormat: Array<{ specLabel: string; specLink: string }> = [];
            specList &&
              specList.forEach((k: string) => {
                const specLink = this.overrideLinkInReport
                  ? `${this.specLinkPrefix}/${k?.substring(k?.indexOf("specification"))}`
                  : k;
                const specLabel = k?.substring(k?.lastIndexOf("/") + 1);
                specListFormat.push({ specLabel, specLink });
              });

            resFormat.push({
              code: i.code,
              message: i.message,
              payloadFilePath: element.payloadFilePath,
              specList: specListFormat,
            });
          });
      });
      return resFormat;
    } else {
      return [];
    }
  }

  public getTotalRunTimeErrors(): number {
    return this.getRunTimeErrors().length;
  }
}

export class ReportGenerator {
  private sdkPackage: string;
  private sdkLanguage: string;
  private validationResults: TrafficValidationIssue[];
  private coverageResults: OperationCoverageInfo[];
  private undefinedOperationsCount: number;
  private reportPath: string;
  private overrideLinkInReport: boolean;
  private outputExceptionInReport: boolean;
  private specLinkPrefix: string;
  private payloadLinkPrefix: string;
  private markdownPath: string;

  public constructor(
    validationResults: TrafficValidationIssue[],
    coverageResults: OperationCoverageInfo[],
    undefinedOperationResults: number,
    options: TrafficValidationOptions
  ) {
    this.validationResults = validationResults;
    this.coverageResults = coverageResults;
    this.undefinedOperationsCount = undefinedOperationResults;
    this.reportPath = path.resolve(process.cwd(), options.reportPath!);
    this.sdkLanguage = options.sdkLanguage!;
    this.sdkPackage = options.sdkPackage!;
    this.markdownPath = options.markdownPath!;
    this.overrideLinkInReport = options.overrideLinkInReport!;
    this.outputExceptionInReport = options.outputExceptionInReport!;
    this.specLinkPrefix = options.specLinkPrefix!;
    this.payloadLinkPrefix = options.payloadLinkPrefix!;
  }

  public async generateHtmlReport() {
    const templatePath = path.join(__dirname, "../templates/baseLayout.mustache");
    const template = fs.readFileSync(templatePath, "utf-8");
    const view = new CoverageView(
      this.validationResults,
      this.coverageResults,
      this.undefinedOperationsCount,
      this.sdkPackage,
      this.sdkLanguage,
      this.markdownPath,
      this.overrideLinkInReport,
      this.outputExceptionInReport,
      this.specLinkPrefix,
      this.payloadLinkPrefix
    );
    await view.prepareDataForRendering();

    const general_errors = view.getGeneralErrors();
    const runtime_errors = view.getRunTimeErrors();

    console.log(JSON.stringify(general_errors, null, 2));
    console.log(JSON.stringify(runtime_errors, null, 2));

    const text = Mustache.render(template, view);
    fs.writeFileSync(this.reportPath, text, "utf-8");
  }
}
