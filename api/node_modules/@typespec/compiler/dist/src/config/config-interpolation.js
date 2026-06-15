import { createDiagnosticCollector } from "../core/diagnostics.js";
import { createDiagnostic } from "../core/messages.js";
import { NoTarget } from "../core/types.js";
export function expandConfigVariables(config, expandOptions) {
    const diagnostics = createDiagnosticCollector();
    const builtInVars = {
        "project-root": config.projectRoot,
        cwd: expandOptions.cwd,
    };
    const resolvedArgsParameters = diagnostics.pipe(resolveArgs(config.parameters, expandOptions.args));
    const commonVars = {
        ...builtInVars,
        ...resolvedArgsParameters,
        env: diagnostics.pipe(resolveArgs(config.environmentVariables, expandOptions.env, true)),
        "output-dir": expandOptions.outputDir ?? config.outputDir ?? "{cwd}/tsp-output",
    };
    const resolvedCommonVars = diagnostics.pipe(resolveValues(commonVars));
    const outputDir = resolvedCommonVars["output-dir"];
    const result = { ...config, outputDir };
    if (config.options) {
        const options = {};
        for (const [name, emitterOptions] of Object.entries(config.options)) {
            const emitterVars = { ...resolvedCommonVars, "output-dir": outputDir, "emitter-name": name };
            options[name] = diagnostics.pipe(resolveValues(emitterOptions, emitterVars));
        }
        result.options = options;
    }
    return diagnostics.wrap(result);
}
function resolveArgs(declarations, args, allowUnspecified = false) {
    const unmatchedArgs = new Set(Object.keys(args ?? {}));
    const result = {};
    if (declarations !== undefined) {
        for (const [name, definition] of Object.entries(declarations)) {
            unmatchedArgs.delete(name);
            result[name] = args?.[name] ?? definition.default;
        }
    }
    if (!allowUnspecified) {
        const diagnostics = [...unmatchedArgs].map((unmatchedArg) => {
            return createDiagnostic({
                code: "config-invalid-argument",
                format: { name: unmatchedArg },
                target: NoTarget,
            });
        });
        return [result, diagnostics];
    }
    return [result, []];
}
function hasNestedValues(value) {
    return (value && typeof value === "object" && !Array.isArray(value) && Object.keys(value).length > 0);
}
const VariableInterpolationRegex = /{([a-zA-Z-_.]+)}/g;
export function resolveValues(values, predefinedVariables = {}) {
    const diagnostics = [];
    const resolvedValues = {};
    const resolvingValues = new Set();
    function resolveValuePath(obj, path) {
        return path.reduce((acc, key) => (acc && typeof acc === "object" ? acc[key] : undefined), obj);
    }
    function resolveValue(keys) {
        const keyPath = keys.join(".");
        resolvingValues.add(keyPath);
        let value = resolveValuePath(values, keys);
        if (typeof value !== "string") {
            if (hasNestedValues(value)) {
                value = value;
                const resultObject = {};
                for (const [nestedKey] of Object.entries(value)) {
                    resultObject[nestedKey] = resolveValue(keys.concat(nestedKey));
                }
                resolvingValues.delete(keyPath);
                return resultObject;
            }
            resolvingValues.delete(keyPath);
            return value;
        }
        const replaced = value.replace(VariableInterpolationRegex, (match, expression) => {
            const resolved = resolveExpression(expression);
            return typeof resolved === "string" ||
                typeof resolved === "number" ||
                typeof resolved === "boolean"
                ? String(resolved)
                : match;
        });
        resolvingValues.delete(keyPath);
        return replaced;
    }
    function resolveExpression(expression) {
        if (resolvingValues.has(expression)) {
            diagnostics.push(createDiagnostic({
                code: "config-circular-variable",
                target: NoTarget,
                format: { name: expression },
            }));
            return undefined;
        }
        const segments = expression.split(".");
        return resolveValue(segments) ?? resolveValuePath(predefinedVariables, segments);
    }
    for (const key of Object.keys(values)) {
        if (key in resolvedValues) {
            continue;
        }
        resolvedValues[key] = resolveValue([key]);
    }
    return [resolvedValues, diagnostics];
}
//# sourceMappingURL=config-interpolation.js.map