import { DiscriminatedOptions } from "../../generated-defs/TypeSpec.js";
import { type Numeric } from "./numeric.js";
import type { Program } from "./program.js";
import type { Model, ScalarValue, Type, Union } from "./types.js";
declare const setMinValue: (program: Program, type: Type, value: Numeric | ScalarValue) => void;
export { setMinValue };
/** Get the minimum value for a scalar type(datetime, duration, etc.). */
export declare function getMinValueForScalar(program: Program, target: Type): ScalarValue | undefined;
/** Get the minimum value for a numeric type. If the value cannot be represented as a JS number(Overflow) undefined will be returned */
export declare function getMinValueAsNumeric(program: Program, target: Type): Numeric | undefined;
/**
 * Get the minimum value for a numeric type.
 * If the value cannot be represented as a JS number(Overflow) undefined will be returned
 * See {@link getMinValueAsNumeric} to get the precise value
 */
export declare function getMinValue(program: Program, target: Type): number | undefined;
declare const setMaxValue: (program: Program, type: Type, value: Numeric | ScalarValue) => void;
export { setMaxValue };
/** Get the maximum value for a scalar type(datetime, duration, etc.). */
export declare function getMaxValueForScalar(program: Program, target: Type): ScalarValue | undefined;
/** Get the maximum value for a numeric type. If the value cannot be represented as a JS number(Overflow) undefined will be returned */
export declare function getMaxValueAsNumeric(program: Program, target: Type): Numeric | undefined;
/**
 * Get the maximum value for a numeric type.
 * If the value cannot be represented as a JS number(Overflow) undefined will be returned
 * See {@link getMaxValueAsNumeric} to get the precise value
 */
export declare function getMaxValue(program: Program, target: Type): number | undefined;
declare const setMinValueExclusive: (program: Program, type: Type, value: Numeric | ScalarValue) => void;
export { setMinValueExclusive };
/** Get the minimum value exclusive for a scalar type(datetime, duration, etc.). */
export declare function getMinValueExclusiveForScalar(program: Program, target: Type): ScalarValue | undefined;
/** Get the minimum value exclusive for a numeric type. If the value cannot be represented as a JS number(Overflow) undefined will be returned */
export declare function getMinValueExclusiveAsNumeric(program: Program, target: Type): Numeric | undefined;
/**
 * Get the minimum value exclusive for a numeric type.
 * If the value cannot be represented as a JS number(Overflow) undefined will be returned
 * See {@link getMinValueExclusiveAsNumeric} to get the precise value
 */
export declare function getMinValueExclusive(program: Program, target: Type): number | undefined;
declare const setMaxValueExclusive: (program: Program, type: Type, value: Numeric | ScalarValue) => void;
export { setMaxValueExclusive };
/** Get the maximum value exclusive for a scalar type(datetime, duration, etc.). */
export declare function getMaxValueExclusiveForScalar(program: Program, target: Type): ScalarValue | undefined;
/** Get the maximum value exclusive for a numeric type. If the value cannot be represented as a JS number(Overflow) undefined will be returned */
export declare function getMaxValueExclusiveAsNumeric(program: Program, target: Type): Numeric | undefined;
/**
 * Get the maximum value exclusive for a numeric type.
 * If the value cannot be represented as a JS number(Overflow) undefined will be returned
 * See {@link getMaxValueExclusiveAsNumeric} to get the precise value
 */
export declare function getMaxValueExclusive(program: Program, target: Type): number | undefined;
export declare function setMinLength(program: Program, target: Type, value: Numeric): void;
/**
 * Get the minimum length of a string type as a {@link Numeric} value.
 * @param program Current program
 * @param target Type with the `@minLength` decorator
 */
export declare function getMinLengthAsNumeric(program: Program, target: Type): Numeric | undefined;
export declare function getMinLength(program: Program, target: Type): number | undefined;
export declare function setMaxLength(program: Program, target: Type, value: Numeric): void;
/**
 * Get the minimum length of a string type as a {@link Numeric} value.
 * @param program Current program
 * @param target Type with the `@maxLength` decorator
 */
export declare function getMaxLengthAsNumeric(program: Program, target: Type): Numeric | undefined;
export declare function getMaxLength(program: Program, target: Type): number | undefined;
export declare function setMinItems(program: Program, target: Type, value: Numeric): void;
export declare function getMinItemsAsNumeric(program: Program, target: Type): Numeric | undefined;
export declare function getMinItems(program: Program, target: Type): number | undefined;
export declare function setMaxItems(program: Program, target: Type, value: Numeric): void;
export declare function getMaxItemsAsNumeric(program: Program, target: Type): Numeric | undefined;
export declare function getMaxItems(program: Program, target: Type): number | undefined;
export interface DocData {
    /**
     * Doc value.
     */
    value: string;
    /**
     * How was the doc set.
     * - `decorator` means the `@doc` decorator was used
     * - `comment` means it was set from a `/** comment * /`
     */
    source: "decorator" | "comment";
}
/**
 * Get the documentation information for the given type. In most cases you probably just want to use {@link getDoc}
 * @param program Program
 * @param target Type
 * @returns Doc data with source information.
 */
export declare function getDocData(program: Program, target: Type): DocData | undefined;
export interface Discriminator {
    readonly propertyName: string;
}
export declare function setDiscriminator(program: Program, entity: Type, discriminator: Discriminator): void;
export declare function getDiscriminator(program: Program, entity: Type): Discriminator | undefined;
export declare function getDiscriminatedTypes(program: Program): [Model | Union, Discriminator][];
export declare const getDiscriminatedOptions: (program: Program, type: Union) => Required<DiscriminatedOptions> | undefined, setDiscriminatedOptions: (program: Program, type: Union, value: Required<DiscriminatedOptions>) => void;
//# sourceMappingURL=intrinsic-type-state.d.ts.map