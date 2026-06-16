import * as fsp from "@ts-common/fs"
import * as it from "@ts-common/iterator"
import fetch from "node-fetch"
import * as path from "path"
import retry from "async-retry"

interface Url {
  readonly protocol: string
  readonly path: string
}

const protocolSeparator = "://"

const toUrlString = (url: Url) => url.protocol + protocolSeparator + url.path

export const urlParse = (dir: string): Url | undefined => {
  const split = dir.split(protocolSeparator)
  return split.length === 2
    ? {
      protocol: split[0],
      path: split[1]
    }
    : undefined
}

export const readFile = async (pathStr: string): Promise<string> => {
  const result = urlParse(pathStr)
  return result === undefined
    ? (await fsp.readFile(pathStr)).toString()
    : await getByUrl(pathStr)
}

const getByUrl = async (url: string) => {
  return retry<string>(
    async (bail, retryNr): Promise<string> => {
      try {
        const response = await fetch(url)
        const body = await response.text()

        if (response.status !== 200) {
          const msg = `StatusCode: "${
            response.status
            }", ResponseBody: "${body}."`

          bail(new Error(msg))
        }

        return body
      } catch (fetchError) {
        const message = `Request to ${url} failed with error ${fetchError} on retry number ${retryNr}.`

        throw new Error(message)
      }
    },
    {
      retries: 3,
      factor: 1
    }
  )
}

export const pathResolve = (dir: string): string =>
  urlParse(dir) !== undefined ? dir : path.resolve(dir)

export const pathJoin = (dir: string, value: string): string => {
  const url = urlParse(dir)
  return url !== undefined
    ? toUrlString({
      protocol: url.protocol,
      path: url.path.split("/").concat([value]).join("/")
    })
    : path.join(dir, value)
}

const IsStatusCodeRetryable = (statusCode: number): boolean => {
  if(statusCode >= 500 || statusCode === 408 || statusCode === 407) {
    return true
  }
  return false
}

export const exists = async (dir: string): Promise<boolean> => {
  if (urlParse(dir) !== undefined) {
    let retries = 0
    const retryTimes = 3
    const retryIntervals = [1000, 3000, 7000]
    while (retries < retryTimes) {
      try {
        const { status } = await fetch(dir, {
          method: "HEAD",
          timeout: 60 * 1000
        })
        if (status === 200) {
          return true
        } else if (!IsStatusCodeRetryable(status) || retries === retryTimes) {
          break
        }
        await new Promise(r => setTimeout(r, retryIntervals[retries++]))
      } catch (e) {
        if (retries === retryTimes) {
          throw new Error(e.message)
        }
        await new Promise(r => setTimeout(r, retryIntervals[retries++]))
      }
    }
    return false
  } else {
    return fsp.exists(dir)
  }
}

export const pathDirName = (dir: string): string => {
  const url = urlParse(dir)
  if (url === undefined) {
    return path.dirname(dir)
  }
  const split = url.path.split("/")
  return toUrlString({
    protocol: url.protocol,
    path: split.length <= 1 ? url.path : it.join(it.dropRight(split), "/")
  })
}
