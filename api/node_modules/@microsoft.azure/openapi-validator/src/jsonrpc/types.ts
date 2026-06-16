/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
/* line: 1-based, column: 0-based */
export type Position =
  | {
      line: number // 1-based
      column: number // 0-based
    }
  | { path?: JsonPath }

export type JsonPath = Array<string | number>

export interface SourceLocation {
  document: string
  Position: Position
}

export interface Message {
  Channel: "information" | "warning" | "error" | "debug" | "verbose" | "fatal"
  Key?: Iterable<string>
  Details?: any
  Text: string
  Source?: SourceLocation[]
}

export interface RawSourceMap {
  file?: string
  sourceRoot?: string
  version: string
  sources: string[]
  names: string[]
  sourcesContent?: string[]
  mappings: string
}

export interface Mapping {
  generated: Position
  original: Position
  source: string
  name?: string
}
