import type { Program } from "../core/program.js";
import { DiagnosticTarget, NoTarget, type ScalarValue, type Type, type Value } from "../core/types.js";
import { type EncodeData } from "./decorators.js";
/**
 * Error thrown when a value cannot be serialized.
 */
export declare class UnserializableValueError extends Error {
    readonly reason: string;
    constructor(reason?: string);
}
/**
 * Error thrown when a scalar value cannot be serialized because it uses an unsupported constructor.
 */
export declare class UnsupportedScalarConstructorError extends UnserializableValueError {
    readonly scalarName: string;
    readonly constructorName: string;
    readonly supportedConstructors: readonly string[];
    constructor(scalarName: string, constructorName: string, supportedConstructors: readonly string[]);
}
export interface ValueJsonSerializers {
    /** Custom handler to serialize a scalar value
     * @param value The scalar value to serialize
     * @param type The type of the scalar value in the current context
     * @param encodeAs The encoding information for the scalar value, if any
     * @param originalFn The original serialization function to fall back to. Throws `UnsupportedScalarConstructorError` if the scalar constructor is not supported.
     * @returns The serialized value
     */
    serializeScalarValue?: (value: ScalarValue, type: Type, encodeAs: EncodeData | undefined, originalFn: (value: ScalarValue, type: Type, encodeAs: EncodeData | undefined) => unknown) => unknown;
}
/**
 * Serialize the given TypeSpec value as a JSON object using the given type and its encoding annotations.
 * The Value MUST be assignable to the given type.
 */
export declare function serializeValueAsJson(program: Program, value: Value, type: Type, encodeAs?: EncodeData, handlers?: ValueJsonSerializers, diagnosticTarget?: DiagnosticTarget | typeof NoTarget): unknown;
//# sourceMappingURL=examples.d.ts.map