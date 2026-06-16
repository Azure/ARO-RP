import { resolveCompilerOptions } from "../../../../config/config-to-options.js";
import { omitUndefined } from "../../../../utils/misc.js";
import { createDiagnosticCollector } from "../../../diagnostics.js";
import { createDiagnostic } from "../../../messages.js";
import { resolvePath } from "../../../path-utils.js";
import { NoTarget } from "../../../types.js";
import { parseCliArgsArgOption } from "../../utils.js";
export async function getCompilerOptions(host, entrypoint, cwd, args, env) {
    const diagnostics = createDiagnosticCollector();
    const pathArg = args["output-dir"] ?? args["output-path"];
    const cliOutputDir = pathArg
        ? pathArg.startsWith("{")
            ? pathArg
            : resolvePath(cwd, pathArg)
        : undefined;
    const cliOptions = diagnostics.pipe(resolveCliOptions(args));
    const resolvedOptions = diagnostics.pipe(await resolveCompilerOptions(host, {
        entrypoint,
        configPath: args["config"] && resolvePath(cwd, args["config"]),
        cwd,
        args: parseCliArgsArgOption(args.args),
        env,
        overrides: omitUndefined({
            outputDir: cliOutputDir,
            imports: args["import"],
            warnAsError: args["warn-as-error"],
            trace: args.trace,
            emit: args.emit,
            options: cliOptions.options,
        }),
    }));
    if (args["no-emit"]) {
        resolvedOptions.noEmit = true;
    }
    else if (args["list-files"]) {
        resolvedOptions.listFiles = true;
    }
    else if (args["dry-run"]) {
        resolvedOptions.dryRun = true;
    }
    else if (args["ignore-deprecated"]) {
        resolvedOptions.ignoreDeprecated = true;
    }
    return diagnostics.wrap(omitUndefined({
        ...resolvedOptions,
        miscOptions: cliOptions.miscOptions,
    }));
}
function resolveCliOptions(args) {
    const diagnostics = [];
    let miscOptions;
    const options = {};
    for (const option of args.options ?? []) {
        const optionParts = option.split("=");
        if (optionParts.length !== 2) {
            diagnostics.push(createDiagnostic({
                code: "invalid-option-flag",
                target: NoTarget,
                format: { value: option },
            }));
            continue;
        }
        const optionKeyParts = optionParts[0].split(".");
        if (optionKeyParts.length === 1) {
            const key = optionKeyParts[0];
            if (miscOptions === undefined) {
                miscOptions = {};
            }
            miscOptions[key] = optionParts[1];
            continue;
        }
        let current = options;
        for (let i = 0; i < optionKeyParts.length; i++) {
            const part = optionKeyParts[i];
            if (i === optionKeyParts.length - 1) {
                current[part] = optionParts[1];
            }
            else {
                if (!current[part]) {
                    current[part] = {};
                }
                current = current[part];
            }
        }
    }
    return [{ options, miscOptions }, diagnostics];
}
//# sourceMappingURL=args.js.map