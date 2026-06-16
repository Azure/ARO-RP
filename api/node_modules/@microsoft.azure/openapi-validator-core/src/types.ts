/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
export enum OpenApiTypes {
  "default" = 1 << 0,
  "arm" = 1 << 1,
  "dataplane" = 1 << 2,
  "rpaas" = 1 << 3,
}

export enum MergeStates {
  "individual",
  "composed",
}

export interface ValidationMessage {
  message: string
  location: JsonPath
}

export interface Rule {
  readonly id: string // see Rxxx/Sxxx codes on https://github.com/Azure/azure-rest-api-specs/blob/master/documentation/openapi-authoring-automated-guidelines.md
  readonly name: string // see same website as above
  readonly category: "ARMViolation" | "OneAPIViolation" | "SDKViolation" | "RPaaSViolation"
  readonly severity: "error" | "warning"
  readonly mergeState: MergeStates
  readonly openapiType: OpenApiTypes
  readonly appliesTo_JsonQuery?: string | string[] // see https://www.npmjs.com/package/jsonpath#jsonpath-syntax for syntax and samples
  run: IRuleFunctionLegacy
}

export const rules: Rule[] = []

export type RuleThen<T> = {
  execute: IRuleFunction<T>
  readonly options?: T
  fieldMatch?: string
}

export type RuleScope = "Global" | "File"

export interface IRule<T> {
  readonly id?: string // see rule ID
  readonly description?: string
  readonly category: "ARMViolation" | "OneAPIViolation" | "SDKViolation" | "RPaaSViolation"
  readonly openapiType: OpenApiTypes
  readonly mergeState?: MergeStates
  readonly severity: "error" | "warning"
  readonly scope?: RuleScope
  readonly given?: string | string[] // see https://github.com/JSONPath-Plus/JSONPath for syntax and samples , the strings to query data via jsonpath-plus.
  readonly then: RuleThen<T> // the rule procession steps
}

export type RulesObject = Record<string, IRule<unknown>>

export interface RuleContextLegacy {
  specPath: string
  inventory?: ISwaggerInventory
}

export interface RuleContext {
  document: any
  location: JsonPath
  specPath: string
  inventory?: ISwaggerInventory
}

export type IRuleFunctionLegacy = (
  document: any,
  openapiSection: any,
  location: JsonPath,
  ctx?: RuleContextLegacy
) => Iterable<ValidationMessage> | AsyncIterable<ValidationMessage>

export type IRuleFunction<T> = (
  openapiSection: any,
  options?: T,
  ctx?: RuleContext
) => Iterable<ValidationMessage> | AsyncIterable<ValidationMessage>

export interface ISwaggerInventory {
  referencesOf(specPath: string): Record<string, any>
  getDocuments(docPath?: string): Record<string, any> | any
}

export interface ISwaggerHelper {
  resolveSchema(schema: any | string): Promise<any>
}

export interface IRuleSet {
  documentationUrl: string
  rules: RulesObject
}

export type JsonPath = Array<string | number>

/* line: 1-based, column: 0-based */
export type Position = {
  line: number // 1-based
  column: number // 0-based
}
export interface SourceLocation {
  document: string
  Position: Position
}

export type MessageSeverity = "information" | "warning" | "error" | "debug" | "verbose" | "fatal"

export interface Message {
  Channel: MessageSeverity
  Key?: Iterable<string>
  Details?: any
  Text: string
  Source?: SourceLocation[]
}

export interface Range {
  start: Position
  end: Position
}

export interface LintResultMessage {
  type: MessageSeverity
  id?: string
  code: string
  message: string
  category: string
  jsonpath: JsonPath
  sources?: string[]
  range?: Range
  rpcGuidelineCode?: string
}

export interface IFileSystem {
  read(uri: string): string | Promise<string>
}
