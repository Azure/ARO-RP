"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.validateAlias = void 0;
const json_1 = require("@stoplight/json");
const lodash_1 = require("lodash");
const alias_1 = require("../../alias");
const formats_1 = require("../../formats");
const error_1 = require("./common/error");
const errors_1 = require("../errors");
function getOverrides(overrides, key) {
    if (!Array.isArray(overrides))
        return null;
    const index = Number(key);
    if (Number.isNaN(index))
        return null;
    if (index < 0 && index >= overrides.length)
        return null;
    const actualOverrides = overrides[index];
    return (0, json_1.isPlainObject)(actualOverrides) && (0, json_1.isPlainObject)(actualOverrides.aliases) ? actualOverrides.aliases : null;
}
function getExtended(extended, parsedPath) {
    if (!Array.isArray(extended))
        return null;
    const key = parsedPath[1];
    const index = Number(key);
    if (Number.isNaN(index))
        return null;
    if (index < 0 && index >= extended.length)
        return null;
    const item = extended[index];
    const isTuple = Array.isArray(item);
    const actualExtended = (isTuple ? item[0] : item);
    const aliases = (0, json_1.isPlainObject)(actualExtended) && (0, json_1.isPlainObject)(actualExtended.aliases) ? actualExtended.aliases : null;
    const overridesPathIndex = isTuple ? 3 : 2;
    if (parsedPath.length >= overridesPathIndex + 2 && parsedPath[overridesPathIndex] === 'overrides') {
        return {
            ...aliases,
            ...getOverrides(actualExtended.overrides, parsedPath[overridesPathIndex + 1]),
        };
    }
    if (parsedPath.length >= overridesPathIndex + 2 && parsedPath[overridesPathIndex] === 'extends') {
        if ((0, json_1.isPlainObject)(actualExtended) && Array.isArray(actualExtended.extends)) {
            const nestedAliases = getExtended(actualExtended.extends, parsedPath.slice(overridesPathIndex));
            return nestedAliases !== null && nestedAliases !== void 0 ? nestedAliases : aliases;
        }
    }
    return aliases;
}
function getResolvedAliases(parsedPath, ruleset) {
    if (parsedPath[0] === 'extends') {
        return getExtended(ruleset.extends, parsedPath);
    }
    else if (parsedPath[0] === 'overrides') {
        return {
            ...ruleset.aliases,
            ...getOverrides(ruleset.overrides, parsedPath[1]),
        };
    }
    else {
        return ruleset.aliases;
    }
}
function validateAlias(ruleset, alias, path) {
    const parsedPath = (0, error_1.toParsedPath)(path);
    try {
        const formats = (0, lodash_1.get)(ruleset, [...parsedPath.slice(0, parsedPath.indexOf('rules') + 2), 'formats']);
        const aliases = getResolvedAliases(parsedPath, ruleset);
        (0, alias_1.resolveAlias)(aliases !== null && aliases !== void 0 ? aliases : null, alias, Array.isArray(formats) ? new formats_1.Formats(formats) : null);
    }
    catch (ex) {
        if (ex instanceof ReferenceError) {
            return new errors_1.RulesetValidationError('undefined-alias', ex.message, parsedPath);
        }
        return (0, error_1.wrapError)(ex, path);
    }
}
exports.validateAlias = validateAlias;
//# sourceMappingURL=alias.js.map