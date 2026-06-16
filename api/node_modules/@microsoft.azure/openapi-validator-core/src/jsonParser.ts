import { findNodeAtLocation, getNodeValue, JSONPath, Node, parseTree } from "jsonc-parser"
import { JsonPath, Range } from "./types"

export interface JsonInstance {
  getLocation(path: JSONPath): Range
  getValue(): any
}

export interface IJsonParser {
  parse(text: string): JsonInstance
}

export class JsonParser implements IJsonParser {
  parse(text: string) {
    const errors :any[] = []
    const rootNode = parseTree(text, errors, { disallowComments: true })
    if (errors.length || rootNode == undefined) {
      throw new Error("Parser failed with errors:" + JSON.stringify(errors))
    }
    return {
      getLocation: (path: JsonPath) => {
        // JSONPath does not include the root '$', replace number string to number
        const targetPath = [...path]
        if (targetPath.length === 0) {
          return getRange(text, rootNode)
        }
        while (targetPath.length > 0) {
          const root = findNodeAtLocation(rootNode, targetPath)
          if (root) {
            return getRange(text, root)
          }
          targetPath.pop()
        }
        throw new Error("Invalid JSONPath:" + path.join("."))
      },
      getValue: () => {
        return Object.assign({}, getNodeValue(rootNode))
      }
    }
  }
}
function getLocation(text: string, offset: number) {
  let line = 1
  let column = 0

  for (let i = 0; i < offset; i++) {
    if (text[i] === "\n") {
      line++
      column = 0
    }
    column++
  }
  return {
    line,
    column
  }
}

function getRange(text: string, node: Node) {
  return {
    start: getLocation(text, node.offset),
    end: getLocation(text, node.offset + node.length)
  }
}
