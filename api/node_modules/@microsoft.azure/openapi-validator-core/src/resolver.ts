import { resolveUri } from "@azure-tools/uri";
import { isExample, traverse ,isUriAbsolute} from "./utils"

export class Resolver {
  private references = new Set<string>()
  constructor(private innerDoc: any, private currentFile: string) {}

  // return resolved doc
  async resolve() {
    const references = this.references
    const currentFile = this.currentFile
    await traverse(this.innerDoc, ["/"], new Set<any>(), {references,currentFile}, updateFileRefs)
  }

  getReferences() {
    return Array.from(this.references.values())
  }
}

const updateFileRefs = (node: any, path: string[], ctx: any) =>{
  if (typeof node === "object" && typeof node.$ref === "string") {
    const slices = node.$ref.split("#") as string[]
    if (slices.length === 2 && slices[0] && !isUriAbsolute(slices[0])) {
      const referenceFile = resolveUri(ctx.currentFile,slices[0])
      node.$ref = referenceFile + `#${slices[1]}`
      if (!isExample(referenceFile)) ctx.references.add(referenceFile)
    }
    return false
  }
  return true
}