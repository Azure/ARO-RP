import * as _ from "lodash"
import { OpenapiDocument } from "./document"
import { JsonParser } from "./jsonParser"
import { ISwaggerInventory, IFileSystem } from "./types"
import { defaultFileSystem, normalizePath } from "./utils"
const DepGraph = require("dependency-graph").DepGraph
export class SwaggerInventory implements ISwaggerInventory {
  private inventory = new DepGraph({ circular: true })
  private referenceCache = new Map<string, OpenapiDocument>()
  private allDocs = new Map<string, any>()
  private docRecords: Record<string, any> | undefined = undefined
  constructor(private fileSystem: IFileSystem = defaultFileSystem) {}

  public getSingleDocument(specPath: string) {
    return this.getInternalDocument(specPath)?.getObj()
  }

  public getDocumentContent(specPath: string) {
    return this.getInternalDocument(specPath)?.getContent()
  }

  public getInternalDocument(specPath: string) {
    const urlPath = normalizePath(specPath)
    if (this.referenceCache.has(urlPath)) {
      return this.referenceCache.get(urlPath)
    }
    throw new Error(`No cached file:${specPath}`)
  }

  public referencesOf(specPath: string): Record<string, any> {
    const result: Record<string, any> = {}
    const references = this.inventory.dependantsOf(normalizePath(specPath))
    for (const ref of references) {
      result[ref] = this.getSingleDocument(ref)
    }
    return result
  }

  public getDocuments(docPath?: string): Record<string, any> | any {
    if (docPath) {
      return this.getSingleDocument(docPath)
    }
    if (!this.docRecords) {
      this.docRecords = {}
      for (const [key, value] of this.allDocs.entries()) {
        this.docRecords[key] = value
      }
    }
    return this.docRecords
  }

  async loadDocument(specPath: string): Promise<any> {
    const urlPath = normalizePath(specPath)
    let cache = this.referenceCache.get(urlPath)
    if (cache) {
      return cache
    }
    cache = await this.cacheDocument(urlPath)
    return cache
  }

  async cacheDocument(specPath: string) {
    const cache = this.allDocs.get(specPath)
    if (cache) {
      return cache
    }
    const parser = new JsonParser()
    const document = new OpenapiDocument(specPath, parser, this.fileSystem)
    await document.resolve()
    this.referenceCache.set(specPath, document)
    this.allDocs.set(specPath, document.getObj())
    this.inventory.addNode(specPath)
    const references = document.getReferences()
    for (const ref of references) {
      if (!this.allDocs.has(ref)) {
        this.inventory.addNode(ref)
        await this.cacheDocument(ref)
      }
      this.inventory.addDependency(specPath, ref)
    }
    return document
  }
}
