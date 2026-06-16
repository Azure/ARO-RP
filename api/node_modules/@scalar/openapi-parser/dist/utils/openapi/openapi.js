import { loadCommand } from "./commands/loadCommand.js";
function openapi(globalOptions) {
  const queue = {
    input: null,
    options: globalOptions,
    tasks: []
  };
  return {
    load: (input, options) => loadCommand(queue, input, options)
  };
}
export {
  openapi
};
//# sourceMappingURL=openapi.js.map
