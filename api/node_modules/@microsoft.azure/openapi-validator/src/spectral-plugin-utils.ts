import { createFileOrFolderUri } from "@azure-tools/uri"
import { LintResultMessage, OpenApiTypes } from "@microsoft.azure/openapi-validator-core"
import { SpectralRulesetPayload, disableRulesInRuleset, spectralRulesets } from "@microsoft.azure/openapi-validator-rulesets"
import { ISpectralDiagnostic, Ruleset, Spectral } from "@stoplight/spectral-core"
import { DiagnosticSeverity } from "@stoplight/types"
import { IAutoRestPluginInitiator } from "./jsonrpc/plugin-host"
import { JsonPath, Message } from "./jsonrpc/types"
import { convertLintMsgToAutoRestMsg } from "./plugin-common"

export async function getRulesetPayload(initiator: IAutoRestPluginInitiator, openapiType: OpenApiTypes): Promise<SpectralRulesetPayload> {
  let rulesetPayload: SpectralRulesetPayload

  switch (openapiType) {
    case OpenApiTypes.arm: {
      rulesetPayload = spectralRulesets.azARM
      break
    }
    case OpenApiTypes.dataplane: {
      rulesetPayload = spectralRulesets.azDataplane
      break
    }
    default: {
      rulesetPayload = spectralRulesets.azCommon
    }
  }

  return rulesetPayload
}

export function ifNotStagingRunDisableRulesInStagingOnly(
  initiator: IAutoRestPluginInitiator,
  namesOfRulesInStagingOnly: string[],
  isStagingRun: boolean,
  ruleset: Ruleset
) {
  if (isStagingRun) {
    initiator.Message({
      Channel: "information",
      Text: "Detected staging run. Running all enabled rules.",
    })
  } else {
    initiator.Message({
      Channel: "information",
      Text:
        "Detected production run. As a result, disabling all Spectral rules that are denoted to run only in staging. Names of rules being disabled: " +
        namesOfRulesInStagingOnly.join(", ") +
        ".",
    })
    disableRulesInRuleset(ruleset, namesOfRulesInStagingOnly)
  }
}

export function printRuleNames(
  initiator: IAutoRestPluginInitiator,
  ruleset: Ruleset,
  resolvedOpenapiType: OpenApiTypes,
  description: string
) {
  const ruleNames: string[] = Object.keys(ruleset.rules)
    // Case-insensitive sort.
    // Source: https://stackoverflow.com/a/60922998/986533
    .sort(Intl.Collator().compare)

  initiator.Message({
    Channel: "information",
    Text: `Loaded ${ruleNames.length} spectral rules, for OpenAPI type '${OpenApiTypes[resolvedOpenapiType]}' for ${description}:`,
  })
  for (const ruleName of ruleNames) {
    const severity: DiagnosticSeverity = ruleset.rules[ruleName].severity
    const sevStr: string = Number(severity) == -1 ? "DISABLED" : DiagnosticSeverity[severity]
    initiator.Message({
      Channel: "information",
      Text: (sevStr == "DISABLED" ? "DISABLED " : "").concat(`Spectral rule for ${description}, severity '${sevStr}': '${ruleName}'`),
    })
  }
}

export async function runSpectral(
  sendMessage: (m: Message) => void,
  spectral: Spectral,
  rulesetPayload: SpectralRulesetPayload,
  openApiSpecFilePath: string,
  openApiSpecYml: any
) {
  const mergedResults = []
  const convertSeverity = (severity: number) => {
    switch (severity) {
      case 0:
        return "error"
      case 1:
        return "warning"
      case 2:
        return "info"
      default:
        return "info"
    }
  }
  const convertRange = (range: any) => {
    return {
      start: {
        line: range.start.line + 1,
        column: range.start.character,
      },
      end: {
        line: range.end.line + 1,
        column: range.end.character,
      },
    }
  }

  // this function is added temporarily , should be remove after the autorest fix this issues.
  const removeXmsExampleFromPath = (paths: JsonPath) => {
    const index = paths.findIndex((item) => item === "x-ms-examples")
    if (index !== -1 && paths.length > index + 2) {
      return paths.slice(0, index + 2)
    }
    return paths
  }

  const formatAsLintResultMessage = (result: ISpectralDiagnostic, spec: string): LintResultMessage => {
    return {
      type: convertSeverity(result.severity),
      category: "",
      code: result.code,
      message: result.message,
      jsonpath: result.path && result.path.length ? removeXmsExampleFromPath(result.path) : [],
      sources: [`${spec}`],
      location: {
        line: result.range.start.line + 1,
        column: result.range.start.character,
      },
      range: convertRange(result.range),
      rpcGuidelineCode: rulesetPayload.rules[result.code]?.rpcGuidelineCode ?? "",
    } as LintResultMessage
  }

  // Newest source of spectral.run as of 2/23/2024
  // https://github.com/stoplightio/spectral/blob/ffa6ebeabbaa0441c8c967ef4e11a7a0a8c66aac/packages/core/src/spectral.ts#L79
  // Note: version used in this code is likely older than newest.
  const results: ISpectralDiagnostic[] = await spectral.run(openApiSpecYml)

  mergedResults.push(
    ...results.map((result: ISpectralDiagnostic) => formatAsLintResultMessage(result, createFileOrFolderUri(openApiSpecFilePath)))
  )

  for (const message of mergedResults) {
    sendMessage(convertLintMsgToAutoRestMsg(message))
  }

  return mergedResults
}

export function catchSpectralRunErrors(file: string, error: any, initiator: IAutoRestPluginInitiator): void {
  // Here 'error' may be AggregateError:
  // Spectral (from "@stoplight/spectral-core") may throw https://www.npmjs.com/package/es-aggregate-error
  // If so, we print out all the constituent errors.
  // For additional context, see: https://github.com/Azure/azure-sdk-tools/issues/6856

  // Initialize an array to collect error messages
  const errorMessages: string[] = [error]

  // Check if "error" contains the "errors" property
  if (error && error.errors && Array.isArray(error.errors)) {
    error.errors.forEach((error: any, index: number) => {
      // Push each error message into the array
      errorMessages.push(`Error ${index + 1}: ${error.message}`)
    })
  }

  // Combine all error messages with newlines
  const combinedErrorMessages = errorMessages.join("\n")

  // Call initiator.Message with the combined error message
  initiator.Message({
    Channel: "fatal",
    Text: `spectralPluginFunc: Failed validating: '${file}'. Errors encountered:\n${combinedErrorMessages}`,
  })
}
