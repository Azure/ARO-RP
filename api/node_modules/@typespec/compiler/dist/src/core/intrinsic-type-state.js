// Contains all intrinsic data setter or getter
// Anything that the TypeSpec check might should be here.
import { createStateSymbol } from "../lib/utils.js";
import { useStateMap } from "../utils/state-accessor.js";
import { isNumeric } from "./numeric.js";
const stateKeys = {
    minValues: createStateSymbol("minValues"),
    maxValues: createStateSymbol("maxValues"),
    minValueExclusive: createStateSymbol("minValueExclusive"),
    maxValueExclusive: createStateSymbol("maxValueExclusive"),
    minLength: createStateSymbol("minLengthValues"),
    maxLength: createStateSymbol("maxLengthValues"),
    minItems: createStateSymbol("minItems"),
    maxItems: createStateSymbol("maxItems"),
    docs: createStateSymbol("docs"),
    returnDocs: createStateSymbol("returnsDocs"),
    errorsDocs: createStateSymbol("errorDocs"),
    discriminator: createStateSymbol("discriminator"),
};
// #region @minValue
const [
/** Get the min value for numeric or scalar types like date times */
getMinValueRaw, setMinValue,] = useStateMap(stateKeys.minValues);
export { setMinValue };
/** Get the minimum value for a scalar type(datetime, duration, etc.). */
export function getMinValueForScalar(program, target) {
    const value = getMinValueRaw(program, target);
    return !isNumeric(value) ? value : undefined;
}
/** Get the minimum value for a numeric type. If the value cannot be represented as a JS number(Overflow) undefined will be returned */
export function getMinValueAsNumeric(program, target) {
    const value = getMinValueRaw(program, target);
    return isNumeric(value) ? value : undefined;
}
/**
 * Get the minimum value for a numeric type.
 * If the value cannot be represented as a JS number(Overflow) undefined will be returned
 * See {@link getMinValueAsNumeric} to get the precise value
 */
export function getMinValue(program, target) {
    return getMinValueAsNumeric(program, target)?.asNumber() ?? undefined;
}
// #endregion @minValue
// #region @maxValue
const [
/** Get the max value for numeric or scalar types like date times */
getMaxValueRaw, setMaxValue,] = useStateMap(stateKeys.maxValues);
export { setMaxValue };
/** Get the maximum value for a scalar type(datetime, duration, etc.). */
export function getMaxValueForScalar(program, target) {
    const value = getMaxValueRaw(program, target);
    return !isNumeric(value) ? value : undefined;
}
/** Get the maximum value for a numeric type. If the value cannot be represented as a JS number(Overflow) undefined will be returned */
export function getMaxValueAsNumeric(program, target) {
    const value = getMaxValueRaw(program, target);
    return isNumeric(value) ? value : undefined;
}
/**
 * Get the maximum value for a numeric type.
 * If the value cannot be represented as a JS number(Overflow) undefined will be returned
 * See {@link getMaxValueAsNumeric} to get the precise value
 */
export function getMaxValue(program, target) {
    return getMaxValueAsNumeric(program, target)?.asNumber() ?? undefined;
}
// #endregion @maxValue
// #region @minValueExclusive
const [
/** Get the min value exclusive for numeric or scalar types like date times */
getMinValueExclusiveRaw, setMinValueExclusive,] = useStateMap(stateKeys.minValueExclusive);
export { setMinValueExclusive };
/** Get the minimum value exclusive for a scalar type(datetime, duration, etc.). */
export function getMinValueExclusiveForScalar(program, target) {
    const value = getMinValueExclusiveRaw(program, target);
    return !isNumeric(value) ? value : undefined;
}
/** Get the minimum value exclusive for a numeric type. If the value cannot be represented as a JS number(Overflow) undefined will be returned */
export function getMinValueExclusiveAsNumeric(program, target) {
    const value = getMinValueExclusiveRaw(program, target);
    return isNumeric(value) ? value : undefined;
}
/**
 * Get the minimum value exclusive for a numeric type.
 * If the value cannot be represented as a JS number(Overflow) undefined will be returned
 * See {@link getMinValueExclusiveAsNumeric} to get the precise value
 */
export function getMinValueExclusive(program, target) {
    return getMinValueExclusiveAsNumeric(program, target)?.asNumber() ?? undefined;
}
// #endregion @minValueExclusive
// #region @maxValueExclusive
const [
/** Get the max value exclusive for numeric or scalar types like date times */
getMaxValueExclusiveRaw, setMaxValueExclusive,] = useStateMap(stateKeys.maxValueExclusive);
export { setMaxValueExclusive };
/** Get the maximum value exclusive for a scalar type(datetime, duration, etc.). */
export function getMaxValueExclusiveForScalar(program, target) {
    const value = getMaxValueExclusiveRaw(program, target);
    return !isNumeric(value) ? value : undefined;
}
/** Get the maximum value exclusive for a numeric type. If the value cannot be represented as a JS number(Overflow) undefined will be returned */
export function getMaxValueExclusiveAsNumeric(program, target) {
    const value = getMaxValueExclusiveRaw(program, target);
    return isNumeric(value) ? value : undefined;
}
/**
 * Get the maximum value exclusive for a numeric type.
 * If the value cannot be represented as a JS number(Overflow) undefined will be returned
 * See {@link getMaxValueExclusiveAsNumeric} to get the precise value
 */
export function getMaxValueExclusive(program, target) {
    return getMaxValueExclusiveAsNumeric(program, target)?.asNumber() ?? undefined;
}
// #endregion @maxValueExclusive
// #region @minLength
export function setMinLength(program, target, value) {
    program.stateMap(stateKeys.minLength).set(target, value);
}
/**
 * Get the minimum length of a string type as a {@link Numeric} value.
 * @param program Current program
 * @param target Type with the `@minLength` decorator
 */
export function getMinLengthAsNumeric(program, target) {
    return program.stateMap(stateKeys.minLength).get(target);
}
export function getMinLength(program, target) {
    return getMinLengthAsNumeric(program, target)?.asNumber() ?? undefined;
}
// #endregion @minLength
// #region @maxLength
export function setMaxLength(program, target, value) {
    program.stateMap(stateKeys.maxLength).set(target, value);
}
/**
 * Get the minimum length of a string type as a {@link Numeric} value.
 * @param program Current program
 * @param target Type with the `@maxLength` decorator
 */
export function getMaxLengthAsNumeric(program, target) {
    return program.stateMap(stateKeys.maxLength).get(target);
}
export function getMaxLength(program, target) {
    return getMaxLengthAsNumeric(program, target)?.asNumber() ?? undefined;
}
// #endregion @maxLength
// #region @minItems
export function setMinItems(program, target, value) {
    program.stateMap(stateKeys.minItems).set(target, value);
}
export function getMinItemsAsNumeric(program, target) {
    return program.stateMap(stateKeys.minItems).get(target);
}
export function getMinItems(program, target) {
    return getMinItemsAsNumeric(program, target)?.asNumber() ?? undefined;
}
// #endregion @minItems
// #region @minItems
export function setMaxItems(program, target, value) {
    program.stateMap(stateKeys.maxItems).set(target, value);
}
export function getMaxItemsAsNumeric(program, target) {
    return program.stateMap(stateKeys.maxItems).get(target);
}
export function getMaxItems(program, target) {
    return getMaxItemsAsNumeric(program, target)?.asNumber() ?? undefined;
}
/** @internal */
export function setDocData(program, target, key, data) {
    program.stateMap(getDocKey(key)).set(target, data);
}
function getDocKey(target) {
    switch (target) {
        case "self":
            return stateKeys.docs;
        case "returns":
            return stateKeys.returnDocs;
        case "errors":
            return stateKeys.errorsDocs;
    }
}
/**
 * @internal
 * Get the documentation information for the given type. In most cases you probably just want to use {@link getDoc}
 * @param program Program
 * @param target Type
 * @returns Doc data with source information.
 */
export function getDocDataInternal(program, target, key) {
    return program.stateMap(getDocKey(key)).get(target);
}
/**
 * Get the documentation information for the given type. In most cases you probably just want to use {@link getDoc}
 * @param program Program
 * @param target Type
 * @returns Doc data with source information.
 */
export function getDocData(program, target) {
    return getDocDataInternal(program, target, "self");
}
export function setDiscriminator(program, entity, discriminator) {
    program.stateMap(stateKeys.discriminator).set(entity, discriminator);
}
export function getDiscriminator(program, entity) {
    return program.stateMap(stateKeys.discriminator).get(entity);
}
export function getDiscriminatedTypes(program) {
    return [...program.stateMap(stateKeys.discriminator).entries()];
}
export const [getDiscriminatedOptions, setDiscriminatedOptions] = useStateMap(createStateSymbol("discriminated"));
// #endregion
//# sourceMappingURL=intrinsic-type-state.js.map