export const TemplateWithMarkers = {
    is: (value) => {
        return typeof value === "object" && value !== null && "isTemplateWithMarkers" in value;
    },
};
/** Specify that this value is dynamic and needs to be interpolated with the given keys */
function code(strings, ...keys) {
    const markers = {};
    const result = [strings[0]];
    keys.forEach((key, i) => {
        if (typeof key === "string") {
            result.push(key);
        }
        else {
            result.push(`/*${key.name}*/${key.name}`);
            markers[key.name] = {
                entityKind: key.entityKind,
                name: key.name,
                kind: key.kind,
                valueKind: key.valueKind,
            };
        }
        result.push(strings[i + 1]);
    });
    return {
        isTemplateWithMarkers: true,
        code: result.join(""),
        markers: markers,
    };
}
function typeMarker(kind) {
    return (name) => {
        return {
            entityKind: "Type",
            kind,
            name,
        };
    };
}
function valueMarker(valueKind) {
    return (name) => {
        return {
            entityKind: "Value",
            valueKind,
            name,
        };
    };
}
/** TypeSpec template marker */
export const t = {
    /**
     * Define a marked code block
     *
     * @example
     * ```ts
     * const code = t.code`model ${t.model("Foo")} { bar: string }`;
     * ```
     */
    code: code,
    // -- Types --
    /** Mark any type */
    type: typeMarker(),
    /** Mark a model */
    model: typeMarker("Model"),
    /** Mark an enum */
    enum: typeMarker("Enum"),
    /** Mark an union */
    union: typeMarker("Union"),
    /** Mark an interface */
    interface: typeMarker("Interface"),
    /** Mark an operation */
    op: typeMarker("Operation"),
    /** Mark an enum member */
    enumMember: typeMarker("EnumMember"),
    /** Mark a model property */
    modelProperty: typeMarker("ModelProperty"),
    /** Mark a namespace */
    namespace: typeMarker("Namespace"),
    /** Mark a scalar */
    scalar: typeMarker("Scalar"),
    /** Mark a union variant */
    unionVariant: typeMarker("UnionVariant"),
    /** Mark a boolean literal */
    boolean: typeMarker("Boolean"),
    /** Mark a number literal */
    number: typeMarker("Number"),
    /** Mark a string literal */
    string: typeMarker("String"),
    // -- Values --
    /** Mark any value */
    value: valueMarker(),
    /** Mark an object value */
    object: valueMarker("ObjectValue"),
    /** Mark an array value */
    array: valueMarker("ArrayValue"),
    /** Mark a numeric value */
    numericValue: valueMarker("NumericValue"),
    /** Mark a string value */
    stringValue: valueMarker("StringValue"),
    /** Mark a boolean value */
    booleanValue: valueMarker("BooleanValue"),
    /** Mark a scalar value */
    scalarValue: valueMarker("ScalarValue"),
    /** Mark an enum value */
    enumValue: valueMarker("EnumValue"),
};
//# sourceMappingURL=marked-template.js.map