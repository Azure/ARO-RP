import { xmsPaths } from "../util/constants";
import { resolveNestedDefinitionTransformer } from "./resolveNestedDefinitionTransformer";
import { SpecTransformer, TransformerType } from "./transformer";

export const xmsPathsTransformer: SpecTransformer = {
  type: TransformerType.Spec,
  before: [resolveNestedDefinitionTransformer],
  transform: (spec) => {
    const xPaths = spec[xmsPaths];
    if (xPaths !== undefined) {
      const paths = spec.paths;
      for (const pathTemplate of Object.keys(xPaths)) {
        paths[pathTemplate] = xPaths[pathTemplate];
      }
      spec[xmsPaths] = undefined;
    }
  },
};
