import * as path from "path";
import { appendFile, mkdir } from "fs/promises";
import {
  asyncWriteFile,
  readFile as vfsReadFile,
  pathJoin,
  pathResolve,
  urlParse,
} from "@azure-tools/openapi-tools-common";
import { inject, injectable } from "inversify";
import { TYPES } from "../inversifyUtils";
import { checkAndResolveGithubUrl } from "../util/utils";
import { Loader, setDefaultOpts } from "./loader";

export interface FileLoaderOption {
  fileRoot?: string;
  checkUnderFileRoot?: boolean;
}

@injectable()
export class FileLoader implements Loader<string> {
  private preloadCache = new Map<string, string>();

  public constructor(@inject(TYPES.opts) private opts: FileLoaderOption) {
    setDefaultOpts(opts, {
      checkUnderFileRoot: true,
    });

    if (this.opts.fileRoot) {
      this.opts.fileRoot = checkAndResolveGithubUrl(pathResolve(this.opts.fileRoot));
    }
  }

  public async load(filePath: string) {
    filePath = this.resolvePath(filePath);

    const preloadContent = this.preloadCache.get(filePath);
    if (preloadContent !== undefined) {
      return preloadContent;
    }

    return vfsReadFile(filePath);
  }

  public relativePath(filePath: string) {
    if (this.opts.fileRoot) {
      filePath = this.resolvePath(filePath);
      const url = urlParse(filePath);
      if (url) {
        const rootUrl = urlParse(this.opts.fileRoot);
        filePath = rootUrl === undefined ? filePath : path.relative(rootUrl.path, url.path);
      } else {
        filePath = path.relative(this.opts.fileRoot, filePath);
      }
    }
    return filePath;
  }

  public resolvePath(filePath: string) {
    const url = urlParse(filePath);
    if (url) {
      filePath = checkAndResolveGithubUrl(filePath);
    } else if (!path.isAbsolute(filePath)) {
      filePath = this.opts.fileRoot
        ? pathJoin(this.opts.fileRoot, filePath)
        : path.resolve(filePath);
    }
    if (
      this.opts.fileRoot &&
      this.opts.checkUnderFileRoot &&
      !filePath.startsWith(this.opts.fileRoot)
    ) {
      throw new Error(
        `Try to load file "${filePath}" outside of root folder ${this.opts.fileRoot}`
      );
    }
    return filePath;
  }

  public isUnderFileRoot(filePath: string) {
    if (!this.opts.fileRoot) {
      return true;
    }

    if (path.isAbsolute(filePath) && !filePath.startsWith(this.opts.fileRoot)) {
      return false;
    }

    filePath = pathJoin(this.opts.fileRoot, filePath);
    return filePath.startsWith(this.opts.fileRoot);
  }

  public async writeFile(filePath: string, content: string) {
    filePath = this.resolvePath(filePath);
    await mkdir(path.dirname(filePath), { recursive: true });
    return asyncWriteFile(filePath, content);
  }

  public async appendFile(filePath: string, content: string) {
    filePath = this.resolvePath(filePath);
    await appendFile(filePath, content);
  }

  public preloadExtraFile(filePath: string, content: string) {
    filePath = this.resolvePath(filePath);
    this.preloadCache.set(filePath, content);
  }
}
