import { LintResultMessage } from "@microsoft.azure/openapi-validator-core"
import { IAutoRestPluginInitiator } from "./jsonrpc/plugin-host"
import { Message } from "./jsonrpc/types"

export function convertLintMsgToAutoRestMsg(message: LintResultMessage): Message {
  // try to extract provider namespace and resource type
  const path = message.jsonpath?.[1] === "paths" && message.jsonpath[2]
  const pathComponents = typeof path === "string" && path.split("/")
  const pathComponentsProviderIndex = pathComponents && pathComponents.indexOf("providers")
  const pathComponentsTail =
    pathComponentsProviderIndex && pathComponentsProviderIndex >= 0 && pathComponents.slice(pathComponentsProviderIndex + 1)
  const pathComponentProviderNamespace = pathComponentsTail && pathComponentsTail[0]
  const pathComponentResourceType = pathComponentsTail && pathComponentsTail[1]
  const msg = {
    Channel: message.type,
    Text: message.message,
    Key: [message.code],
    Source: [
      {
        document: message?.sources?.[0] || "",
        Position: {
          path: message.jsonpath,
          //...message.range?.start as Position
        },
      },
    ],
    Details: {
      jsonpath: message.jsonpath,
      validationCategory: message.category,
      providerNamespace: pathComponentProviderNamespace,
      resourceType: pathComponentResourceType,
      rpcGuidelineCode: message.rpcGuidelineCode,
      range: message.range,
    },
  }
  return msg
}

export async function getOpenapiTypeStr(initiator: IAutoRestPluginInitiator) {
  let openapiType: string = await initiator.GetValue("openapi-type")
  let subType: string = await initiator.GetValue("openapi-subtype")
  subType = subType === "providerHub" ? "rpaas" : subType
  if (subType === "rpaas") {
    openapiType = "rpaas"
  }
  return openapiType
}

export function isCommonTypes(filePath: string) {
  const regex = new RegExp(/.*common-types\/resource-management\/v.*.json/)
  return regex.test(filePath)
}
