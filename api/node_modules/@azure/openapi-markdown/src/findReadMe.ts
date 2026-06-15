// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

import * as vfs from "@ts-common/virtual-fs"

/**
 * It may throw an exception if `dir` is URL and network is not available.
 *
 * @param dir
 */
export const findReadMe = async (dir: string): Promise<string | undefined> => {
    dir = vfs.pathResolve(dir)
    while (true) {
        const fileName = vfs.pathJoin(dir, "readme.md")
        if (await vfs.exists(fileName)) {
            return fileName
        }
        const newDir = vfs.pathDirName(dir)
        if (newDir === dir) {
            return undefined
        }
        const url = vfs.urlParse(newDir)
        if (url !== undefined) {
            const pathSplit = url.path.split("/")
            if (pathSplit.length === 2 && pathSplit[0] === "github.com") {
                // return undefined if it's a GitHub organization instead of a repository
                return undefined
            }
        }
        dir = newDir
    }
}
