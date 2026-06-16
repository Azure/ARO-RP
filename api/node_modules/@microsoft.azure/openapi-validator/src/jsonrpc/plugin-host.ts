/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
import { createMessageConnection, NotificationType2, NotificationType4, RequestType0, RequestType1, RequestType2 } from "vscode-jsonrpc"
import { Mapping, Message, RawSourceMap } from "./types"

namespace IAutoRestPluginTarget_Types {
  export const GetPluginNames = new RequestType0<string[], Error, void>("GetPluginNames")
  export const Process = new RequestType2<string, string, boolean, Error, void>("Process")
}

namespace IAutoRestPluginInitiator_Types {
  export const ReadFile = new RequestType2<string, string, string, Error, void>("ReadFile")
  export const GetValue = new RequestType2<string, string, any, Error, void>("GetValue")
  export const ListInputs = new RequestType1<string, string[], Error, void>("ListInputs")
  export const WriteFile = new NotificationType4<string, string, string, Mapping[] | RawSourceMap | undefined, void>("WriteFile")
  export const Message = new NotificationType2<string, Message, void>("Message")
}
export interface IAutoRestPluginInitiator {
  ReadFile(filename: string): Promise<string>
  GetValue(key: string): Promise<any>
  ListInputs(): Promise<string[]>

  WriteFile(filename: string, content: string, sourceMap?: Mapping[] | RawSourceMap): void
  Message(message: Message): void
}

type AutoRestPluginHandler = (initiator: IAutoRestPluginInitiator) => Promise<void>

export class AutoRestPluginHost {
  private readonly plugins: { [name: string]: AutoRestPluginHandler } = {}

  public Add(name: string, handler: AutoRestPluginHandler): void {
    this.plugins[name] = handler
  }

  public async Run(): Promise<void> {
    // connection setup
    const channel = createMessageConnection(process.stdin, process.stdout, {
      error(message) {
        console.error("error: ", message)
      },
      info(message) {
        console.error("info: ", message)
      },
      log(message) {
        console.error("log: ", message)
      },
      warn(message) {
        console.error("warn: ", message)
      },
    })

    channel.onRequest(IAutoRestPluginTarget_Types.GetPluginNames, async () => Object.keys(this.plugins))
    channel.onRequest(IAutoRestPluginTarget_Types.Process, async (pluginName: string, sessionId: string) => {
      try {
        const handler = this.plugins[pluginName]
        if (!handler) {
          throw new Error(`Plugin host could not find requested plugin '${pluginName}'.`)
        }
        await handler({
          async ReadFile(filename: string): Promise<string> {
            return await channel.sendRequest(IAutoRestPluginInitiator_Types.ReadFile, sessionId, filename)
          },
          async GetValue(key: string): Promise<any> {
            return await channel.sendRequest(IAutoRestPluginInitiator_Types.GetValue, sessionId, key)
          },
          async ListInputs(): Promise<string[]> {
            return await channel.sendRequest(IAutoRestPluginInitiator_Types.ListInputs, sessionId)
          },

          WriteFile(filename: string, content: string, sourceMap?: Mapping[] | RawSourceMap): void {
            channel.sendNotification(IAutoRestPluginInitiator_Types.WriteFile, sessionId, filename, content, sourceMap)
          },
          Message(message: Message): void {
            channel.sendNotification(IAutoRestPluginInitiator_Types.Message, sessionId, message)
          },
        })
        return true
      } catch (e) {
        channel.sendNotification(IAutoRestPluginInitiator_Types.Message, sessionId, {
          Channel: "fatal" as any,
          Text: "" + e,
          Details: e,
        } as Message)
        return false
      }
    })

    // activate
    channel.listen()
  }
}
