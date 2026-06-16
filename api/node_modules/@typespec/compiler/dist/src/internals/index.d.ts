/**
 * This file is meant to export internal items from the TypeSpec compiler to some other tools that bundle them.
 * DO NOT USE it, it might change at any time with no warning.
 */
export { resolveCompilerOptions } from "../config/config-to-options.js";
export { NodeSystemHost } from "../core/node-system-host.js";
export { InitTemplateSchema } from "../init/init-template.js";
export { makeScaffoldingConfig, scaffoldNewProject } from "../init/scaffold.js";
export { resolveEntrypointFile } from "../server/entrypoint-resolver.js";
export { InternalCompileResult, ServerDiagnostic } from "../server/index.js";
//# sourceMappingURL=index.d.ts.map