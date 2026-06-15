import { defineKit } from "../define-kit.js";
defineKit({
    literal: {
        create(value) {
            return this.program.checker.createLiteralType(value);
        },
        createString(value) {
            return this.program.checker.createLiteralType(value);
        },
        createNumeric(value) {
            return this.program.checker.createLiteralType(value);
        },
        createBoolean(value) {
            return this.program.checker.createLiteralType(value);
        },
        isBoolean(type) {
            return type.entityKind === "Type" && type.kind === "Boolean";
        },
        isString(type) {
            return type.entityKind === "Type" && type.kind === "String";
        },
        isNumeric(type) {
            return type.entityKind === "Type" && type.kind === "Number";
        },
        is(type) {
            return (this.literal.isBoolean(type) || this.literal.isNumeric(type) || this.literal.isString(type));
        },
    },
});
//# sourceMappingURL=literal.js.map