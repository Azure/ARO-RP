import { resolveNestedDefinitionTransformer } from "./resolveNestedDefinitionTransformer";
import { SpecTransformer, TransformerType } from "./transformer";
import { traverseSwagger } from "./traverseSwagger";

const defaultMime = ["application/json"];

const isDefaultMime = (mimes: string[]) => {
  return mimes.length === 1 && mimes[0] === defaultMime[0];
};

export const referenceFieldsTransformer: SpecTransformer = {
  type: TransformerType.Spec,
  before: [resolveNestedDefinitionTransformer],
  transform: (spec) => {
    if (spec.consumes === undefined || isDefaultMime(spec.consumes)) {
      spec.consumes = defaultMime;
    }
    if (spec.produces === undefined || isDefaultMime(spec.produces)) {
      spec.produces = defaultMime;
    }
    traverseSwagger(spec, {
      onPath: (path, pathTemplate) => {
        path._spec = spec;
        path._pathTemplate = pathTemplate;
      },
      onOperation: (operation, path, method) => {
        operation._path = path;
        operation._method = method;
        if (operation.consumes === undefined) {
          operation.consumes = spec.consumes;
        } else if (isDefaultMime(operation.consumes)) {
          operation.consumes = defaultMime;
        }
        if (operation.produces === undefined) {
          operation.produces = spec.produces;
        } else if (isDefaultMime(operation.produces)) {
          operation.produces = defaultMime;
        }
      },
    });
  },
};
