import * as fs from "fs"
import * as path from "path"
import * as asyncIt from "@ts-common/async-iterator"
import * as util from "util"

export const readFile = util.promisify(fs.readFile)

export const writeFile = util.promisify(fs.writeFile)

export const exists = util.promisify(fs.exists)

export const readdir = util.promisify(fs.readdir)

export const mkdir = util.promisify(fs.mkdir)

export const rmdir = util.promisify(fs.rmdir)

export const unlink = util.promisify(fs.unlink)

export const recursiveReaddir = (dir: string): asyncIt.AsyncIterableEx<string> =>
    asyncIt.fromPromise(readdir(dir, { withFileTypes: true })).flatMap(
        f => {
            const p = path.join(dir, f.name)
            return f.isDirectory() ?  recursiveReaddir(p) : asyncIt.fromSequence(p)
        }
    )

export const recursiveRmdir = async (dir: string): Promise<void> => {
    const list = await readdir(dir, { withFileTypes: true })
    await Promise.all(list.map(async f => {
        const p = path.join(dir, f.name)
        if (f.isDirectory()) {
            await recursiveRmdir(p)
        } else {
            await unlink(p)
        }
    }))
    await rmdir(dir)
}