/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

import { AutoRestPluginHost } from "./jsonrpc/plugin-host"
import { openapiValidatorPluginFunc } from "./openapi-validator-plugin-func"
import { spectralPluginFunc } from "./spectral-plugin-func"

export const cachedFiles = new Map<string, string>()

async function main() {
  const pluginHost = new AutoRestPluginHost()
  pluginHost.Add("openapi-validator", openapiValidatorPluginFunc)
  pluginHost.Add("spectral", spectralPluginFunc)

  await pluginHost.Run()
}

main().then(
  () => {},
  () => {}
)
