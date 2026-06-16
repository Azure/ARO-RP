import { fileURLToPath } from "url"
import { resolveUri } from "@azure-tools/uri"
import { OpenApiTypes, getOpenapiType, isUriAbsolute } from "@microsoft.azure/openapi-validator-core"
import {
  SpectralRulesetPayload,
  deleteRulesPropertiesInPayloadNotValidForSpectralRules,
  disableRulesInRuleset,
  getNamesOfRulesInPayloadWithPropertySetToTrue,
  // spectralRulesets,
} from "@microsoft.azure/openapi-validator-rulesets"
import { Resolver } from "@stoplight/json-ref-resolver"
import { IResolver } from "@stoplight/json-ref-resolver/types"
import { Ruleset, Spectral } from "@stoplight/spectral-core"
import { load } from "js-yaml"
import { IAutoRestPluginInitiator } from "./jsonrpc/plugin-host"
import { getOpenapiTypeStr, isCommonTypes } from "./plugin-common"
import {
  catchSpectralRunErrors,
  getRulesetPayload,
  ifNotStagingRunDisableRulesInStagingOnly,
  printRuleNames,
  runSpectral,
} from "./spectral-plugin-utils"
import { cachedFiles } from "."

export async function spectralPluginFunc(initiator: IAutoRestPluginInitiator): Promise<void> {
  initiator.Message({
    Channel: "information",
    Text: `spectralPluginFunc: Enter`,
  })

  // openApiSpecFiles is an array of 'input-file' paths originating from given AutoRest README API version tag, for example:
  // https://github.com/Azure/azure-rest-api-specs/blob/b273d97aa44a84088bab144a8bcebeb07f827238/specification/storage/resource-manager/readme.md#tag-package-2023-01
  // Example value of suffix of one entry in openApiSpecFiles:
  // specification/storage/resource-manager/Microsoft.Storage/stable/2023-01-01/storage.json
  const openApiSpecFiles: string[] = (await initiator.ListInputs()).filter((f) => !isCommonTypes(f))
  const openapiType: string = await getOpenapiTypeStr(initiator)
  const isStagingRun: boolean = await initiator.GetValue("is-staging-run")

  const resolvedOpenapiType: OpenApiTypes = getOpenapiType(openapiType)

  const {
    rulesetPayload,
    rulesetForManualSpecs,
    rulesetForTypeSpecGeneratedSpecs,
  }: {
    rulesetPayload: SpectralRulesetPayload
    rulesetForManualSpecs: Ruleset
    rulesetForTypeSpecGeneratedSpecs: Ruleset
  } = await getRulesets(initiator, resolvedOpenapiType, isStagingRun)

  for (const openApiSpecFile of openApiSpecFiles) {
    await validateOpenApiSpecFileUsingSpectral(
      initiator,
      rulesetPayload,
      rulesetForManualSpecs,
      rulesetForTypeSpecGeneratedSpecs,
      openApiSpecFile,
    )
  }

  initiator.Message({
    Channel: "information",
    Text: `spectralPluginFunc: Return`,
  })
}

async function getRulesets(
  initiator: IAutoRestPluginInitiator,
  resolvedOpenapiType: OpenApiTypes,
  isStagingRun: boolean,
): Promise<{
  rulesetPayload: SpectralRulesetPayload
  rulesetForManualSpecs: Ruleset
  rulesetForTypeSpecGeneratedSpecs: Ruleset
}> {
  const rulesetPayload: SpectralRulesetPayload = await getRulesetPayload(initiator, resolvedOpenapiType)
  const namesOfRulesInStagingOnly: string[] = getNamesOfRulesInPayloadWithPropertySetToTrue(rulesetPayload, "stagingOnly")
  const namesOfRulesDisabledForTypespecDataPlane: string[] =
    resolvedOpenapiType === OpenApiTypes.dataplane
      ? getNamesOfRulesInPayloadWithPropertySetToTrue(rulesetPayload, "disableForTypeSpecDataPlane")
      : []

  // We need two of rulesetPayloads:
  // - The original, to prepare it as argument for spectral Rulesets. See deletePropertiesNotValidForSpectralRules for more.
  // - A copy, with all properties on it, so we can access the rpcGuidelineCode property of each rule when post-processing
  //   Spectral run results.
  // Note: the original, not copy, must be used as input to Ruleset constructor, as the way we
  // obtain the copy here makes it not suitable for use as input to Ruleset constructor.
  const rulesetPayloadCopy: SpectralRulesetPayload = JSON.parse(JSON.stringify(rulesetPayload))
  deleteRulesPropertiesInPayloadNotValidForSpectralRules(rulesetPayload)

  const rulesetForManualSpecs = new Ruleset(rulesetPayload, { severity: "recommended" })
  ifNotStagingRunDisableRulesInStagingOnly(initiator, namesOfRulesInStagingOnly, isStagingRun, rulesetForManualSpecs)
  printRuleNames(initiator, rulesetForManualSpecs, resolvedOpenapiType, "manually written OpenAPI specs")

  const rulesetForTypeSpecGeneratedSpecs = new Ruleset(rulesetPayload, { severity: "recommended" })
  ifNotStagingRunDisableRulesInStagingOnly(initiator, namesOfRulesInStagingOnly, isStagingRun, rulesetForTypeSpecGeneratedSpecs)
  disableRulesInRuleset(rulesetForTypeSpecGeneratedSpecs, namesOfRulesDisabledForTypespecDataPlane)
  printRuleNames(initiator, rulesetForTypeSpecGeneratedSpecs, resolvedOpenapiType, "TypeSpec-generated OpenAPI specs")

  return {
    rulesetPayload: rulesetPayloadCopy,
    rulesetForManualSpecs: rulesetForManualSpecs,
    rulesetForTypeSpecGeneratedSpecs,
  }
}

async function validateOpenApiSpecFileUsingSpectral(
  initiator: IAutoRestPluginInitiator,
  rulesetPayload: SpectralRulesetPayload,
  rulesetForManualSpecs: Ruleset,
  rulesetForTypeSpecGeneratedSpecs: Ruleset,
  openApiSpecFile: string,
) {
  if (openApiSpecFile.includes("common-types/resource-management")) {
    initiator.Message({
      Channel: "information",
      Text: `spectralPluginFunc: Ignoring file matching to 'common-types/resource-management': '${openApiSpecFile}'`,
    })
    return
  }

  try {
    const openApiSpecFilePath = openApiSpecFile.startsWith("file:///") ? fileURLToPath(openApiSpecFile) : openApiSpecFile
    const openApiSpecContent: string = await readFileUsingCache(initiator, openApiSpecFile)
    // load() documented at: https://github.com/nodeca/js-yaml/tree/4.1.0?tab=readme-ov-file#load-string---options-
    // Empirically confirmed the returned value type is object, not string.
    const openApiSpecYml: any = load(openApiSpecContent)
    // "x-typespec-generated" is expected to be found at JSONPath of $.info.x-typespec-generated.
    // Example: https://github.com/Azure/azure-rest-api-specs/blob/fca48bec19cc5aab0a45c0769bfca0f667164dbf/specification/edgemarketplace/resource-manager/Microsoft.EdgeMarketplace/stable/2023-08-01/operations.json#L7
    const specIsGeneratedFromTypeSpec = Boolean(openApiSpecYml.info["x-typespec-generated"]) // JSON.stringify(openApiSpecYml).includes("x-typespec-generated")

    initiator.Message({
      Channel: "information",
      Text: `spectralPluginFunc: Validating OpenAPI spec. TypeSpec-generated: ${specIsGeneratedFromTypeSpec}. Path: '${openApiSpecFile}'`,
    })

    const spectral = newSpectral(initiator, openApiSpecFile)
    spectral.setRuleset(specIsGeneratedFromTypeSpec ? rulesetForTypeSpecGeneratedSpecs : rulesetForManualSpecs)

    const sendMessage = initiator.Message.bind(initiator)

    await runSpectral(sendMessage, spectral, rulesetPayload, openApiSpecFilePath, openApiSpecYml)
  } catch (error: unknown) {
    catchSpectralRunErrors(openApiSpecFile, error, initiator)
  }
}

function newSpectral(initiator: IAutoRestPluginInitiator, openApiSpecFile: string) {
  const resolveFile: IResolver = {
    resolve: (ref: URI, ctx: any) => {
      const href = ref.href()
      const openApiSpecFileUri = isUriAbsolute(href) ? href : resolveUri(openApiSpecFile, href)
      const openApiSpecContent = readFileUsingCache(initiator, openApiSpecFileUri)
      return openApiSpecContent
    },
  }

  const jsonRefResolver = new Resolver({
    resolvers: {
      file: resolveFile,
      http: resolveFile,
      https: resolveFile,
    },
  })
  const spectral = new Spectral({ resolver: jsonRefResolver })
  return spectral
}

async function readFileUsingCache(initiator: IAutoRestPluginInitiator, fileUri: string): Promise<string> {
  let file: string | undefined = cachedFiles.get(fileUri)
  if (!file) {
    file = await initiator.ReadFile(fileUri)
    if (!file) {
      throw new Error(`Could not read file: ${fileUri} .`)
    }
    cachedFiles.set(fileUri, file)
  }
  return file
}
