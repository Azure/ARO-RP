import { Temporal } from "temporal-polyfill";
import { ignoreDiagnostics } from "../core/diagnostics.js";
import { getProperty } from "../core/semantic-walker.js";
import { isArrayModelType, isUnknownType } from "../core/type-utils.js";
import { getEncode, resolveEncodedName } from "./decorators.js";
/**
 * Error thrown when a value cannot be serialized.
 */
export class UnserializableValueError extends Error {
    reason;
    constructor(reason = "Cannot serialize value as JSON.") {
        super(reason);
        this.reason = reason;
        this.name = "UnserializableValueError";
    }
}
/**
 * Error thrown when a scalar value cannot be serialized because it uses an unsupported constructor.
 */
export class UnsupportedScalarConstructorError extends UnserializableValueError {
    scalarName;
    constructorName;
    supportedConstructors;
    constructor(scalarName, constructorName, supportedConstructors) {
        super(`Cannot serialize scalar '${scalarName}' with constructor '${constructorName}'. Supported constructors: ${supportedConstructors.join(", ")}`);
        this.scalarName = scalarName;
        this.constructorName = constructorName;
        this.supportedConstructors = supportedConstructors;
        this.name = "UnsupportedScalarConstructorError";
    }
}
/**
 * Serialize the given TypeSpec value as a JSON object using the given type and its encoding annotations.
 * The Value MUST be assignable to the given type.
 */
export function serializeValueAsJson(program, value, type, encodeAs, handlers, diagnosticTarget) {
    if (type.kind === "ModelProperty") {
        return serializeValueAsJson(program, value, type.type, encodeAs ?? getEncode(program, type), handlers, diagnosticTarget);
    }
    switch (value.valueKind) {
        case "NullValue":
            return null;
        case "BooleanValue":
        case "StringValue":
            return value.value;
        case "NumericValue":
            return value.value.asNumber();
        case "EnumValue":
            return value.value.value ?? value.value.name;
        case "ArrayValue":
            return value.values.map((v) => serializeValueAsJson(program, v, type.kind === "Model" && isArrayModelType(type)
                ? type.indexer.value
                : program.checker.anyType, 
            /* encodeAs: */ undefined, handlers, diagnosticTarget));
        case "ObjectValue":
            return serializeObjectValueAsJson(program, value, type, handlers, diagnosticTarget);
        case "ScalarValue":
            return serializeScalarValueAsJson(program, value, type, encodeAs, handlers);
        case "Function":
            throw new UnserializableValueError("Cannot serialize a function value as JSON.");
    }
}
/** Try to get the property of the type */
function getPropertyOfType(type, name) {
    switch (type.kind) {
        case "Model":
            return getProperty(type, name) ?? type.indexer?.value;
        case "Intrinsic":
            if (isUnknownType(type)) {
                return type;
            }
            else {
                return;
            }
        default:
            return undefined;
    }
}
function resolveUnions(program, value, type) {
    if (type.kind !== "Union") {
        return type;
    }
    const exactValueType = program.checker.getValueExactType(value);
    for (const variant of type.variants.values()) {
        if (ignoreDiagnostics(program.checker.isTypeAssignableTo(exactValueType ?? value.type, variant.type, value))) {
            // If the variant is itself a union, recursively resolve it
            if (variant.type.kind === "Union") {
                const resolvedNested = resolveUnions(program, value, variant.type);
                if (resolvedNested && resolvedNested !== variant.type) {
                    return resolvedNested;
                }
            }
            return variant.type;
        }
    }
    return type;
}
function serializeObjectValueAsJson(program, value, type, handlers, diagnosticTarget) {
    type = resolveUnions(program, value, type) ?? type;
    const obj = {};
    for (const propValue of value.properties.values()) {
        const definition = getPropertyOfType(type, propValue.name);
        if (definition) {
            const name = definition.kind === "ModelProperty"
                ? resolveEncodedName(program, definition, "application/json")
                : propValue.name;
            obj[name] = serializeValueAsJson(program, propValue.value, definition, 
            /* encodeAs: */ undefined, handlers, propValue.node);
        }
    }
    return obj;
}
function resolveKnownScalar(program, scalar) {
    const encode = getEncode(program, scalar);
    if (program.checker.isStdType(scalar)) {
        switch (scalar.name) {
            case "utcDateTime":
            case "offsetDateTime":
            case "plainDate":
            case "plainTime":
            case "duration":
                return { scalar: scalar, encodeAs: encode };
            case "unixTimestamp32":
                break;
            default:
                return undefined;
        }
    }
    if (scalar.baseScalar) {
        const result = resolveKnownScalar(program, scalar.baseScalar);
        return result && { scalar: result.scalar, encodeAs: encode };
    }
    return undefined;
}
function serializeScalarValueAsJson(program, value, type, encodeAs, handlers) {
    if (handlers?.serializeScalarValue) {
        return handlers.serializeScalarValue(value, type, encodeAs, serializeScalarValueAsJson.bind(null, program, value, type, encodeAs, undefined));
    }
    const result = resolveKnownScalar(program, value.scalar);
    if (result === undefined) {
        return undefined;
    }
    encodeAs = encodeAs ?? result.encodeAs;
    switch (result.scalar.name) {
        case "utcDateTime":
            if (value.value.name === "fromISO") {
                return ScalarSerializers.utcDateTime(value.value.args[0].value, encodeAs);
            }
            throw new UnsupportedScalarConstructorError("utcDateTime", value.value.name, ["fromISO"]);
        case "offsetDateTime":
            if (value.value.name === "fromISO") {
                return ScalarSerializers.offsetDateTime(value.value.args[0].value, encodeAs);
            }
            throw new UnsupportedScalarConstructorError("offsetDateTime", value.value.name, ["fromISO"]);
        case "plainDate":
            if (value.value.name === "fromISO") {
                return ScalarSerializers.plainDate(value.value.args[0].value);
            }
            throw new UnsupportedScalarConstructorError("plainDate", value.value.name, ["fromISO"]);
        case "plainTime":
            if (value.value.name === "fromISO") {
                return ScalarSerializers.plainTime(value.value.args[0].value);
            }
            throw new UnsupportedScalarConstructorError("plainTime", value.value.name, ["fromISO"]);
        case "duration":
            if (value.value.name === "fromISO") {
                return ScalarSerializers.duration(value.value.args[0].value, encodeAs);
            }
            throw new UnsupportedScalarConstructorError("duration", value.value.name, ["fromISO"]);
    }
}
const ScalarSerializers = {
    utcDateTime: (value, encodeAs) => {
        if (encodeAs === undefined || encodeAs.encoding === "rfc3339") {
            return value;
        }
        const date = new Date(value);
        switch (encodeAs.encoding) {
            case "unixTimestamp":
                return Math.floor(date.getTime() / 1000);
            case "rfc7231":
                return date.toUTCString();
            default:
                return date.toISOString();
        }
    },
    offsetDateTime: (value, encodeAs) => {
        if (encodeAs === undefined || encodeAs.encoding === "rfc3339") {
            return value;
        }
        const date = new Date(value);
        switch (encodeAs.encoding) {
            case "rfc7231":
                return date.toUTCString();
            default:
                return date.toISOString();
        }
    },
    plainDate: (value) => {
        return value;
    },
    plainTime: (value) => {
        return value;
    },
    duration: (value, encodeAs) => {
        const duration = Temporal.Duration.from(value);
        switch (encodeAs?.encoding) {
            case "seconds":
                if (isInteger(encodeAs.type)) {
                    return Math.floor(duration.total({ unit: "seconds" }));
                }
                else {
                    return duration.total({ unit: "seconds" });
                }
            default:
                return duration.toString();
        }
    },
};
function isInteger(scalar) {
    while (scalar.baseScalar) {
        scalar = scalar.baseScalar;
        if (scalar.name === "integer") {
            return true;
        }
    }
    return false;
}
//# sourceMappingURL=examples.js.map