import { $ } from "../typekit/index.js";
import { compilerAssert } from "./diagnostics.js";
import { numericRanges } from "./numeric-ranges.js";
import { Numeric } from "./numeric.js";
export function marshalTypeForJs(value, valueConstraint) {
    switch (value.valueKind) {
        case "BooleanValue":
        case "StringValue":
            return value.value;
        case "NumericValue":
            return numericValueToJs(value, valueConstraint);
        case "ObjectValue":
            return objectValueToJs(value);
        case "ArrayValue":
            return arrayValueToJs(value);
        case "NullValue":
        case "TemplateValue":
            return null;
        case "ScalarValue":
        case "EnumValue":
        case "Function":
            return value;
        default:
            compilerAssert(false, `Cannot marshal value of kind '${value.valueKind}' to JS.`);
    }
}
function isNumericScalar(scalar) {
    let current = scalar;
    while (current) {
        if (current.name === "numeric" && current.namespace?.name === "TypeSpec") {
            return true;
        }
        current = current.baseScalar;
    }
    return false;
}
export function canNumericConstraintBeJsNumber(type) {
    if (type === undefined)
        return true;
    switch (type.kind) {
        case "Scalar":
            if (isNumericScalar(type)) {
                return numericRanges[type.name]?.[2].isJsNumber;
            }
            else {
                return true;
            }
        case "Union":
            return [...type.variants.values()].every((x) => canNumericConstraintBeJsNumber(x.type));
        default:
            return true;
    }
}
function numericValueToJs(type, valueConstraint) {
    const canBeANumber = canNumericConstraintBeJsNumber(valueConstraint);
    if (canBeANumber) {
        const asNumber = type.value.asNumber();
        compilerAssert(asNumber !== null, `Numeric value '${type.value.toString()}' is not a able to convert to a number without losing precision.`);
        return asNumber;
    }
    return type.value;
}
function objectValueToJs(type) {
    const result = {};
    for (const [key, value] of type.properties) {
        result[key] = marshalTypeForJs(value.value, undefined);
    }
    return result;
}
function arrayValueToJs(type) {
    return type.values.map((x) => marshalTypeForJs(x, undefined));
}
export function unmarshalJsToValue(program, value, onInvalid) {
    if (typeof value === "object" &&
        value !== null &&
        "entityKind" in value &&
        value.entityKind === "Value") {
        return value;
    }
    if (value === null || value === undefined) {
        return {
            entityKind: "Value",
            valueKind: "NullValue",
            value: null,
            type: program.checker.nullType,
        };
    }
    else if (typeof value === "boolean") {
        const boolean = program.checker.getStdType("boolean");
        return {
            entityKind: "Value",
            valueKind: "BooleanValue",
            value,
            type: boolean,
            scalar: boolean,
        };
    }
    else if (typeof value === "string") {
        const string = program.checker.getStdType("string");
        return {
            entityKind: "Value",
            valueKind: "StringValue",
            value,
            type: string,
            scalar: string,
        };
    }
    else if (typeof value === "number") {
        const numeric = Numeric(String(value));
        const numericType = program.checker.getStdType("numeric");
        return {
            entityKind: "Value",
            valueKind: "NumericValue",
            value: numeric,
            type: $(program).literal.create(value),
            scalar: numericType,
        };
    }
    else if (Array.isArray(value)) {
        const values = [];
        const uniqueTypes = new Set();
        for (const item of value) {
            const itemValue = unmarshalJsToValue(program, item, onInvalid);
            if (itemValue) {
                values.push(itemValue);
                uniqueTypes.add(itemValue.type);
            }
        }
        return {
            entityKind: "Value",
            valueKind: "ArrayValue",
            type: $(program).array.create($(program).union.create([...uniqueTypes])),
            values,
        };
    }
    else if (typeof value === "object" && !("entityKind" in value)) {
        const properties = new Map();
        for (const [key, val] of Object.entries(value)) {
            const propertyValue = unmarshalJsToValue(program, val, onInvalid);
            if (propertyValue) {
                properties.set(key, { name: key, value: propertyValue });
            }
        }
        return {
            entityKind: "Value",
            valueKind: "ObjectValue",
            properties,
            type: $(program).model.create({
                properties: Object.fromEntries([...properties.entries()].map(([k, v]) => [k, $(program).modelProperty.create({ name: k, type: v.value.type })])),
            }),
        };
    }
    else {
        onInvalid(value);
        return null;
    }
}
//# sourceMappingURL=js-marshaller.js.map