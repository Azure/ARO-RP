/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
import _ from "lodash"
import { LintRunner } from "./runner"
import { SwaggerInventory } from "./swaggerInventory"
import { IRuleSet, LintResultMessage, OpenApiTypes, RulesObject, IFileSystem } from "./types"

export type LintOptions = {
  ruleSet: IRuleSet
  openapiType: OpenApiTypes
  fileSystem?: IFileSystem
}
export type LintCallBack = (msg: LintResultMessage) => void

export async function lint(swaggerPaths: string[], options: LintOptions, cb?: LintCallBack): Promise<LintResultMessage[]> {
  const inventory = new SwaggerInventory(options?.fileSystem)
  const ruleLoader = { getRuleSet: () => options.ruleSet }
  const runner = new LintRunner(ruleLoader, inventory)
  const msgs = await runner.execute(swaggerPaths, options, cb)
  return msgs
}

export async function LintTester(
  sampleFilePath: string | string[],
  ruleSet: IRuleSet,
  ruleName?: string,
  fileSystem?: IFileSystem
): Promise<LintResultMessage[]> {
  const openapiType = OpenApiTypes.arm | OpenApiTypes.dataplane | OpenApiTypes.rpaas | OpenApiTypes.default
  let msgs: LintResultMessage[]
  const sampleFilePaths = Array.isArray(sampleFilePath) ? sampleFilePath : [sampleFilePath]
  if (ruleName) {
    const rules: RulesObject = {}
    if (!ruleSet.rules[ruleName]) {
      throw new Error(`Rule ${ruleName} was not found.`)
    }
    rules[ruleName] = ruleSet.rules[ruleName]
    const singleRuleSet: IRuleSet = { documentationUrl: "", rules }
    msgs = await lint(sampleFilePaths, { ruleSet: singleRuleSet, openapiType, fileSystem })
  } else {
    msgs = await lint(sampleFilePaths, { ruleSet, openapiType, fileSystem })
  }
  return msgs
}
