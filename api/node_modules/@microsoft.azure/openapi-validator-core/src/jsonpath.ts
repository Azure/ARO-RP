import * as JSONPath from "jsonpath-plus"

export function nodes(obj: any, pathExpression: string) {
  try {
    const result = JSONPath.JSONPath({ json: obj, path: pathExpression, resultType: "all" })
    return result.map((v:any) => ({ path: JSONPath.JSONPath.toPathArray(v.path), value: v.value, parent: v.parent }))
  }
  catch(e) {
   // throw new Error(`Encountered exception when run jsonpath ${pathExpression}, ${JSON.stringify(e)}`)
   return []
  }

}

export function stringify(path: string[]) {
  const pathWithRoot = ["$", ...path]
  return JSONPath.JSONPath.toPathString(pathWithRoot)
}
