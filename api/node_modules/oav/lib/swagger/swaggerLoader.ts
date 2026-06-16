import { relative as pathRelative, dirname } from "path";
import { inject, injectable } from "inversify";
import { TYPES } from "../inversifyUtils";
import { traverseSwagger } from "../transform/traverseSwagger";
import { xmsExamples } from "../util/constants";
import { getProviderFromSpecPath } from "../util/utils";
import { FileLoader, FileLoaderOption } from "./fileLoader";
import { JsonLoader, JsonLoaderOption } from "./jsonLoader";
import { Loader, setDefaultOpts } from "./loader";
import { SuppressionLoader, SuppressionLoaderOption } from "./suppressionLoader";
import { SwaggerExample, SwaggerSpec } from "./swaggerTypes";

export interface SwaggerLoaderOption
  extends SuppressionLoaderOption,
    JsonLoaderOption,
    FileLoaderOption {
  setFilePath?: boolean;
}

export interface ExampleUpdateEntry {
  swaggerPath: string;
  operationId: string;
  exampleName: string;
  exampleFilePath: string;
  exampleContent: SwaggerExample;
}

@injectable()
export class SwaggerLoader implements Loader<SwaggerSpec> {
  public constructor(
    @inject(TYPES.opts) private opts: SwaggerLoaderOption,
    private suppressionLoader: SuppressionLoader,
    private jsonLoader: JsonLoader,
    private siblingsJsonLoader: JsonLoader,
    private fileLoader: FileLoader
  ) {
    setDefaultOpts(opts, {
      setFilePath: true,
    });
  }

  public getResolvedJsonLoader() {
    return this.siblingsJsonLoader;
  }

  // TODO reportError
  public async load(specFilePath: string, keepRefSiblings?: boolean): Promise<SwaggerSpec> {
    const swaggerSpec = keepRefSiblings
      ? ((await (this.siblingsJsonLoader.load(
          specFilePath,
          false,
          true
        ) as unknown)) as SwaggerSpec)
      : ((await (this.jsonLoader.load(specFilePath) as unknown)) as SwaggerSpec);

    if (this.opts.setFilePath) {
      const pathProvider = getProviderFromSpecPath(this.fileLoader.resolvePath(specFilePath));
      swaggerSpec._filePath = this.fileLoader.relativePath(specFilePath);
      swaggerSpec._providerNamespace = pathProvider ? pathProvider.provider : "unknown";
    }

    await this.suppressionLoader.load(swaggerSpec);

    return swaggerSpec;
  }

  public async updateSwaggerAndExamples(entries: ExampleUpdateEntry[]) {
    const swaggerPaths = new Set<string>();
    const examplePaths = new Set<string>();
    const entriesGroupByOperationId = new Map<string, ExampleUpdateEntry[]>();

    for (const entry of entries) {
      swaggerPaths.add(entry.swaggerPath);
      if (examplePaths.has(entry.exampleFilePath)) {
        throw new Error(`Duplicated example path: ${entry.exampleFilePath}`);
      }
      examplePaths.add(entry.exampleFilePath);

      let entriesGroup = entriesGroupByOperationId.get(entry.operationId);
      if (entriesGroup === undefined) {
        entriesGroup = [];
        entriesGroupByOperationId.set(entry.operationId, entriesGroup);
      }
      entriesGroup.push(entry);
    }

    const toWait: Array<Promise<void>> = [];
    for (const swaggerPath of swaggerPaths) {
      const fileContent = await this.fileLoader.load(swaggerPath);
      const swaggerSpec = JSON.parse(fileContent) as SwaggerSpec;

      traverseSwagger(swaggerSpec, {
        onOperation: (operation) => {
          const operationId = operation.operationId ?? "";
          const entriesGroup = entriesGroupByOperationId.get(operationId);
          if (entriesGroup === undefined) {
            return;
          }

          let examples = operation[xmsExamples];
          if (examples === undefined) {
            examples = {};
            operation[xmsExamples] = examples;
          }

          for (const entry of entriesGroup) {
            let path = pathRelative(dirname(swaggerPath), entry.exampleFilePath).replace(
              /\\/g,
              "/"
            );
            if (!path.startsWith(".")) {
              path = `./${path}`;
            }
            examples[entry.exampleName] = {
              $ref: path,
            } as Partial<SwaggerExample> as SwaggerExample;
            toWait.push(
              this.fileLoader.writeFile(entry.exampleFilePath, formatJson(entry.exampleContent))
            );
          }
        },
      });

      toWait.push(this.fileLoader.writeFile(swaggerPath, formatJson(swaggerSpec)));
    }

    await Promise.all(toWait);
  }
}

const formatJson = (obj: any) => {
  return JSON.stringify(obj, undefined, 2);
};
