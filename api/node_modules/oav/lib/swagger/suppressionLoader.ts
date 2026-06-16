import { sep as pathSep } from "path";
import {
  getInfo,
  MutableStringMap,
  ObjectInfo,
  parseMarkdown,
} from "@azure-tools/openapi-tools-common";
import {
  findReadMe,
  getCodeBlocksAndHeadings,
  getYamlFromNode,
  SuppressionItem,
} from "@azure/openapi-markdown";
import { JSONPath } from "jsonpath-plus";
import { inject, injectable } from "inversify";
import { log } from "../util/logging";
import { TYPES } from "../inversifyUtils";
import { FileLoader, FileLoaderOption } from "./fileLoader";
import { Loader } from "./loader";
import { SwaggerSpec } from "./swaggerTypes";

export interface SuppressionLoaderOption extends FileLoaderOption {
  loadSuppression?: string[];
}

@injectable()
export class SuppressionLoader implements Loader<void, SwaggerSpec> {
  private suppressionCache = new Map<string, SuppressionItem[]>();
  private suppressionToLoad: Set<string>;

  constructor(@inject(TYPES.opts) opts: SuppressionLoaderOption, private fileLoader: FileLoader) {
    this.suppressionToLoad = new Set(opts?.loadSuppression ?? []);
  }

  public async load(spec: SwaggerSpec) {
    if (this.suppressionToLoad.size === 0) {
      return;
    }
    const filePath = this.fileLoader.resolvePath(spec._filePath);
    const readmePath = await findReadMe(filePath);
    if (readmePath === undefined) {
      return;
    }

    try {
      let items = this.suppressionCache.get(readmePath);
      if (items === undefined) {
        items = await this.getSuppression(readmePath);
        this.suppressionCache.set(readmePath, items);
      }

      for (const item of items) {
        if (!matchFileFrom(filePath, item.from)) {
          continue;
        }
        applySuppression(spec, item);
      }
    } catch (e) {
      const msg = `Error in loading suppression from readme:${readmePath}.\nDetails:${e.message}\n${e.stack}`;
      throw new Error(msg);
    }
  }

  private async getSuppression(readmePath: string): Promise<SuppressionItem[]> {
    if (!this.fileLoader.isUnderFileRoot(readmePath)) {
      return [];
    }
    const fileContent = await this.fileLoader.load(readmePath);

    const cmd = parseMarkdown(fileContent);
    const suppressionCodeBlock = getCodeBlocksAndHeadings(cmd.markDown).Suppression;
    if (suppressionCodeBlock === undefined) {
      return [];
    }

    const suppression = getYamlFromNode(suppressionCodeBlock);
    const items = suppression.directive as SuppressionItem[] | undefined;
    if (!Array.isArray(items)) {
      return [];
    }
    return items.filter((item) => this.suppressionToLoad.has(item.suppress));
  }
}

const matchFileFrom = (filePath: string, from: string | readonly string[] | undefined) => {
  if (from === undefined) {
    return true;
  }

  if (!Array.isArray(from) && typeof from === "string") {
    return endsWithPath(filePath, from);
  }

  for (const fromName of from) {
    if (endsWithPath(filePath, fromName)) {
      return true;
    }
  }

  return false;
};

const endsWithPath = (filePath: string, searchPath: string) => {
  const sep = filePath[filePath.length - searchPath.length - 1];
  return filePath.endsWith(searchPath) && (sep === pathSep || sep === "/");
};

const applySuppression = (spec: SwaggerSpec, item: SuppressionItem) => {
  for (const node of getNodesFromWhere(spec, item.where)) {
    const info = getInfo(node)?.position;
    if (info !== undefined) {
      if (info.directives === undefined) {
        (info as any).directives = {};
      }
      const directives = info.directives as MutableStringMap<string>;
      directives[item.suppress] = item["text-matches"] ?? ".*";
    }
  }
};

const getNodesFromWhere = (
  spec: SwaggerSpec,
  where: string | readonly string[] | undefined
): any[] => {
  if (where === undefined) {
    return [spec];
  }

  if (typeof where === "string") {
    where = [where];
  }

  const output = [];
  for (const wh of where) {
    try {
      output.push(...JSONPath<any[]>({ path: wh, json: spec, resultType: "value" }));
    } catch (e) {
      log.error(e);
    }
  }
  return output;
};

export const isSuppressed = (node: any, code: string, message: string) => {
  const info = getInfo(node);
  if (info === undefined) {
    return false;
  }

  const directives = info.position.directives;
  if (directives === undefined) {
    return false;
  }

  const messageRegex = directives[code] as string | undefined;
  if (messageRegex === undefined) {
    return false;
  }

  if (messageRegex === ".*") {
    return true;
  }

  return new RegExp(messageRegex).test(message);
};

const getParentNode = (info: ObjectInfo) => {
  return info.isChild ? getInfo(info.parent) : undefined;
};

export const isSuppressedInPath = (node: any, code: string, message: string) => {
  let info = getInfo(node);

  while (info !== undefined) {
    const directives = info.position.directives;
    if (directives === undefined) {
      info = getParentNode(info);
      continue;
    }

    const messageRegex = directives[code] as string | undefined;
    if (messageRegex === undefined) {
      info = getParentNode(info);
      continue;
    }

    if (messageRegex === ".*") {
      return true;
    }

    if (new RegExp(messageRegex).test(message)) {
      return true;
    }

    info = getParentNode(info);
  }

  return false;
};
