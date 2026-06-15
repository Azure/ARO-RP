export const TYPES = {
  opts: Symbol("InversifyTYPES.opts"),
  emptyObject: Symbol("InversifyTYPES.emptyObject"),
  schemaValidator: Symbol("InversifyTYPES.schemaValidator"),
};

import { Container, type ContainerOptions, type Newable } from "inversify";
import { setDefaultOpts } from "./swagger/loader";

export const inversifyGetContainer = (opts: ContainerOptions = {}) => {
  setDefaultOpts(opts, {
    defaultScope: "Singleton",
    autobind: true,
  } as any);
  return new Container(opts);
};

export const inversifyGetInstance = <T, Opt = {}>(
  claz: Newable<T>,
  opts: Opt &
    ContainerOptions & {
      container?: Container;
    }
) => {
  if (opts.container === undefined) {
    opts.container = inversifyGetContainer(opts);
  }
  opts.container.bind(TYPES.opts).toConstantValue(opts);
  opts.container.bind(TYPES.emptyObject).toConstantValue({});
  const { AjvSchemaValidator } = require("./swaggerValidator/ajvSchemaValidator");
  opts.container.bind(TYPES.schemaValidator).to(AjvSchemaValidator);
  return opts.container.get(claz);
};
