import { Program } from "./program.js";
import type { MarshalledValue, TemplateValue, Type, Value } from "./types.js";
export declare function marshalTypeForJs<T extends Value | TemplateValue>(value: T, valueConstraint: Type | undefined): MarshalledValue<T>;
export declare function canNumericConstraintBeJsNumber(type: Type | undefined): boolean;
export declare function unmarshalJsToValue(program: Program, value: unknown, onInvalid: (value: unknown) => void): Value | null;
//# sourceMappingURL=js-marshaller.d.ts.map