import { array as topoSort } from "toposort";
import { SwaggerSpec } from "../swagger/swaggerTypes";
import { TransformContext } from "./context";

export enum TransformerType {
  Spec = "spec",
  Global = "global",
}

interface TransformerBase<T> {
  type: TransformerType;
  before?: T[];
  after?: T[];
}

export interface SpecTransformer extends TransformerBase<SpecTransformer> {
  type: TransformerType.Spec;
  transform: (spec: SwaggerSpec, ctx: TransformContext) => void;
}

export interface GlobalTransformer extends TransformerBase<GlobalTransformer> {
  type: TransformerType.Global;
  transform: (ctx: TransformContext) => void;
}

export type Transformer = SpecTransformer | GlobalTransformer;

export const sortTransformers = (transformers: Transformer[]) => {
  const specTransformers = new Set<SpecTransformer>();
  const globalTransformers = new Set<GlobalTransformer>();

  for (const t of transformers) {
    switch (t.type) {
      case TransformerType.Spec:
        specTransformers.add(t);
        break;

      case TransformerType.Global:
        globalTransformers.add(t);
        break;
    }
  }

  const sortTrans = <T extends TransformerBase<T>>(ts: Set<T>): T[] => {
    const edges: Array<[T, T]> = [];
    for (const t of ts) {
      if (t.after !== undefined) {
        for (const tAfter of t.after) {
          if (ts.has(tAfter)) {
            edges.push([tAfter, t]);
          }
        }
      }
      if (t.before !== undefined) {
        for (const tBefore of t.before) {
          if (ts.has(tBefore)) {
            edges.push([t, tBefore]);
          }
        }
      }
    }
    return topoSort([...ts], edges);
  };

  return {
    specTransformers: sortTrans(specTransformers),
    globalTransformers: sortTrans(globalTransformers),
  };
};

export const applySpecTransformers = (spec: SwaggerSpec, ctx: TransformContext) => {
  for (const t of ctx.specTransformers) {
    t.transform(spec, ctx);
  }
};

export const applyGlobalTransformers = (ctx: TransformContext) => {
  for (const t of ctx.globalTransformers) {
    t.transform(ctx);
  }
};
