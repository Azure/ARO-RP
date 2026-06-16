// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
import * as commonmark from "commonmark"
import { ReadMeBuilder } from "./readMeBuilder"
import { Logger } from "./logger"
import * as yaml from "js-yaml"
import { base64ToString } from "./gitHubUtils"
import { MarkDownEx, markDownExToString, parse } from "@ts-common/commonmark-to-markdown"
import * as sm from "@ts-common/string-map"
import * as it from "@ts-common/iterator"

/**
 * Examples:
 * - https://github.com/Azure/azure-rest-api-specs/blob/4c2be7a9983963a75e15c579e4fc8d17e547ea69/specification/guestconfiguration/resource-manager/readme.md#suppression
 * - https://github.com/Azure/azure-rest-api-specs/blob/32b0d873aa851d456dfde7d6ba1d89ff33f897f0/specification/azsadmin/resource-manager/user-subscriptions/readme.md#suppression
 */
export interface SuppressionItem {
    readonly suppress: string
    readonly reason?: string
    readonly where: string|readonly string[]
    readonly from?: string|readonly string[]
    readonly "text-matches"?: string
}

export interface Suppression {
    readonly directive: readonly SuppressionItem[];
}

export interface TagSettings {
    readonly "input-file": readonly string[] | string;
}

export const inputFile = (tagSettings: TagSettings) => {
    const f = tagSettings["input-file"]
    return typeof f === "string" ? [f] : f
}

/**
 * Provides operations that can be applied to readme files
 */
export class ReadMeManipulator {
    constructor(private logger: Logger, private readMeBuilder: ReadMeBuilder) { }

    /**
     * Updates the latest version tag of a readme
     */
    public readonly updateLatestTag = (markDownEx: MarkDownEx, newTag: string): string => {
        const startNode = markDownEx.markDown
        const codeBlockMap = getCodeBlocksAndHeadings(startNode)

        const latestHeader = "Basic Information"

        const lh = codeBlockMap[latestHeader]
        if (lh === undefined) {
            this.logger.error(`Couldn't parse code block`)
            throw new Error("")
        }

        const latestDefinition = yaml.load(lh.literal!) as
            | undefined
            | { tag: string }

        if (!latestDefinition) {
            this.logger.error(`Couldn't parse code block`)
            throw new Error("")
        }

        latestDefinition.tag = newTag

        lh.literal = yaml.dump(latestDefinition, {
            lineWidth: -1
        })

        return markDownExToString(markDownEx)
    }

    public readonly insertTagDefinition = (
        readmeContent: string,
        tagFiles: readonly string[],
        newTag: string
    ) => {
        const newTagDefinitionYaml = createTagDefinitionYaml(tagFiles)

        const toSplice = this.readMeBuilder.getVersionDefinition(
            newTagDefinitionYaml,
            newTag
        )

        return spliceIntoTopOfVersions(readmeContent, toSplice)
    }

    public readonly addSuppressionBlock = (readme: string) =>
        `${readme}\n\n${this.readMeBuilder.getSuppressionSection()}`


    /**
     * This function takes a markdown document and a list of file paths and
     * returns the list of tags that reference these file paths. It is meant to
     * work like https://github.com/Azure/azure-rest-api-specs/blob/master/test/linter.js
     */
    public readonly getTagsForFilesChanged = (
        markDownEx: MarkDownEx,
        specsChanged: readonly string[]
    ): readonly string[] => {
        const codeBlocks = getTagsToSettingsMapping(markDownEx.markDown);
        const tagsAffected = new Set<string>();

        for (const [tag, settings] of sm.entries(codeBlocks)) {
            // for every file in settings object, see if it matches one of the
            // paths changed
            const filesTouchedInTag = specsChanged.filter(
                spec => inputFile(settings).some(inputFile => spec.includes(inputFile))
            )

            if (filesTouchedInTag.length > 0) {
                tagsAffected.add(tag)
            }
        }
        return [...tagsAffected]
    }

    public readonly getAllTags = (
        markDownEx: MarkDownEx
    ): readonly string[] => {
        const codeBlocks = getTagsToSettingsMapping(markDownEx.markDown);
        const tags = new Set<string>();

        for (const [tag] of sm.entries(codeBlocks)) {
            tags.add(tag);
        }
        return [...tags]
    }
}

const isTagSettings = (obj: unknown): obj is TagSettings =>
    typeof obj === "object" &&
    obj !== null &&
    "input-file" in obj;

export const getTagsToSettingsMapping = (
    startNode: commonmark.Node
): { readonly [keg: string]: TagSettings|undefined } =>
    getAllCodeBlockNodes(startNode).fold(
        (accumulator, node) => {
            if (node && node.literal && node.info) {
                let settings: unknown
                try {
                    settings = yaml.safeLoad(node.literal, { });
                } catch (e) {
                    return accumulator
                }
                // tag matching from
                // https://github.com/Azure/azure-rest-api-specs/blob/45e82e67d42ee347edbdb8b15807473b5aaf3a06/test/linter.js#L37
                const matchTag = /\$\(tag\)[^'"]*(?:['"](.*?)['"])/;
                const matches = matchTag.exec(node.info);

                if (isTagSettings(settings) && matches) {
                    const [, tag] = matches;
                    return { ...accumulator, [tag]: settings };
                }
            }
            return accumulator;
        },
        {}
    );

export const getInputFiles = (startNode: commonmark.Node) : it.IterableEx<string> =>
    sm.values(getTagsToSettingsMapping(startNode)).flatMap(inputFile)

/**
 * Get input files listed for a given tag
 * @returns array of file path or null if the tag doesn't exists
 */
export const getInputFilesForTag = (startNode: commonmark.Node, tag: string): readonly string[] | undefined => {
    const tagMapping = getTagsToSettingsMapping(startNode);
    const foo = tagMapping[tag];
    return foo !== undefined ? inputFile(foo) : undefined;
}

export const addSuppression = (
    startNode: commonmark.Node,
    item: SuppressionItem
): void => {
    const mapping = getCodeBlocksAndHeadings(startNode)
    const suppressionNode = mapping.Suppression
    if (suppressionNode === undefined) {
        // probably it's a bug
        return
    }
    const suppressionBlock = getYamlFromNode(suppressionNode)
    const updatedSuppressionBlock = {
        ...suppressionBlock,
        directive: [...suppressionBlock.directive, item]
    }
    updateYamlForNode(suppressionNode, updatedSuppressionBlock)
}

export const base64ToMarkDownEx = (base: string): MarkDownEx => {
    const str = base64ToString(base)
    return parse(str)
}

export const getYamlFromNode = (node: commonmark.Node) => {
    const infoYaml: any = yaml.load(node.literal!)
    return infoYaml
}

const updateYamlForNode = (node: commonmark.Node, yamlObject: any): void => {
    node.literal = yaml.dump(yamlObject, { lineWidth: -1 })
}

const spliceIntoTopOfVersions = (file: string, splice: string) => {
    const index = file.indexOf("### Tag")
    return file.slice(0, index) + splice + file.slice(index)
}

const createTagDefinitionYaml = (files: readonly string[]) => ({
    ["input-file"]: files
})

export const hasSuppressionBlock = (startNode: commonmark.Node) => {
    const mapping = getCodeBlocksAndHeadings(startNode)
    return !!mapping.Suppression
}

export interface CodeBlocksAndHeadings {
    readonly Suppression?: commonmark.Node
    readonly [key: string]: commonmark.Node|undefined
}

export const getCodeBlocksAndHeadings = (
    startNode: commonmark.Node
): CodeBlocksAndHeadings =>
    getAllCodeBlockNodes(startNode).fold(
        (acc, curr) => {
            const headingNode = nodeHeading(curr)

            if (!headingNode) {
                return { ...acc }
            }

            const headingLiteral = getHeadingLiteral(headingNode);

            if (!headingLiteral) {
                return { ...acc }
            }

            return { ...acc, [headingLiteral]: curr }
        },
        {}
    )

const getHeadingLiteral = (heading: commonmark.Node): string => {
    const headingNode = walkToNode(
        heading.walker(),
        n => n.type === "text"
    )

    return headingNode && headingNode.literal ? headingNode.literal : ""
}

const getAllCodeBlockNodes = (startNode: commonmark.Node) =>
    it.iterable(function *() {
        const walker = startNode.walker()
        while (true) {
            const a = walkToNode(walker, n => n.type === "code_block")
            if (!a) {
                break
            }
            yield a
        }
    })

const nodeHeading = (startNode: commonmark.Node): commonmark.Node | null => {
    let resultNode: commonmark.Node | null = startNode

    while (resultNode != null && resultNode.type !== "heading") {
        resultNode = resultNode.prev || resultNode.parent
    }

    return resultNode
}

/**
 * walks a markdown tree until the callback provided returns true for a node
 */
const walkToNode = (
    walker: commonmark.NodeWalker,
    cb: (node: commonmark.Node) => boolean
): commonmark.Node | undefined => {
    let event = walker.next()

    while (event) {
        const curNode = event.node
        if (cb(curNode)) {
            return curNode
        }
        event = walker.next()
    }
    return undefined
}
